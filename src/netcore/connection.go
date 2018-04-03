package netcore

import (
	"core"
	"errors"
	"net"
	"time"
	"utility"
)

type DataBuffer struct {
	data    []byte
	size    int
	dataLen int
}
type Connection struct {
	//conn          *net.TCPConn
	conn 		  net.Conn
	rbuf          *DataBuffer
	wbuf          *DataBuffer
	addr          string
	err           error
	state         int
	enableEncrypt bool
	log           *utility.LogContext
	//session       *core.Session
	session 	  core.Session
	recvTimeout   time.Duration
	protocol      core.Protocol
	closeSignal   chan bool
}

const (
	maxBufferSize      = 16384 * 2
	maxRequestSize     = 16384
	DefaultRecvTimeout = 50
)

func NewConnection(conn net.Conn, session core.Session, log *utility.LogContext) *Connection {
	//conn.SetNoDelay(true)
	myConn := &Connection{
		conn:        conn,
		rbuf:        &DataBuffer{make([]byte, maxBufferSize), maxBufferSize, 0},
		wbuf:        &DataBuffer{make([]byte, maxBufferSize), maxBufferSize, 0},
		addr:        conn.RemoteAddr().String(),
		state:       0,
		session:     session,
		log:         log,
		recvTimeout: time.Duration(DefaultRecvTimeout) * time.Millisecond,
		closeSignal: make(chan bool, 1),
	}

	go myConn.IOHandler()
	return myConn
}

func (c *Connection) AddProtocol(protocol core.Protocol){
	c.protocol = protocol
}
//SetSession set a new session
func (c *Connection) SetSession(session core.Session) {
	c.session = session
}

/*
func logErrFrame(data []byte, ctx *Client) {
	var id uint32
	errData, frameLen := Read_uint32(data)
	errData, id = Read_uint32(errData)
	frameType := errData[0]
	ctx.log(LOG_WARN, "error frame. frameLen:%d, id:%d type:%d content:%v", frameLen, id, frameType, errData[1:])
}
*/
//return false when encouter some error
func (conn *Connection) handleReply(data []byte, err error) bool {
	log := conn.log
	if err != nil {
		log.LogWarn("err:%v", err)
		return false
	}
	if data == nil {
		return true
	}
	c := conn.conn
	c.SetWriteDeadline(time.Now().Add(time.Millisecond * 500))
	wBytes, err := c.Write(data)
	if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
		log.LogWarn("Write err:%s", err.Error())
		return false
	} else if err != nil {
		log.LogWarn("%s", err.Error())
		return true
	}
	log.LogDebug("Write bytes:%d", wBytes)
	return true
}

func (conn *Connection) UpdateRecvTimeout(timeout int) {
	conn.recvTimeout = time.Duration(timeout) * time.Millisecond
}

func (conn *Connection) Output(sendData []byte)(int, error){
	log := conn.log
	c := conn.conn
	c.SetWriteDeadline(time.Now().Add(time.Millisecond * 500))
	wBytes, err := c.Write(sendData)
	if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
		log.LogWarn("Write err:%s", err.Error())
		return wBytes, err
	} else if err != nil {
		log.LogWarn("%s", err.Error())
		return wBytes, err
	}
	log.LogDebug("Write bytes:%d", wBytes)
	return wBytes, err
}

func (conn *Connection) Input(){

}

func (conn *Connection) Close(){
	conn.closeSignal <- true
}


//IOHandler handle io stuff, erlang style
func (conn *Connection) IOHandler() {
	log := conn.log
	onExit := func() {
		conn.conn.Close()
		conn.err = errors.New("")
		/*
		if conn.session.Clean != nil {
			conn.session.Clean(conn.session.GetPrivData())
		}
		*/

		log.LogInfo("client exit:%s ", conn.addr)
	}
	defer utility.CatchPanic(log, onExit)
	for {
		select {
		case <-conn.closeSignal:
			return
		default:
			rBuf := conn.rbuf
			conn.conn.SetReadDeadline(time.Now().Add(conn.recvTimeout))
			rBytes, err := conn.conn.Read(rBuf.data[rBuf.dataLen:])
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				if !conn.handleReply(conn.session.UpdateProc()){
					return
				}
				continue
			}

		if rBytes == 0 || err != nil {
			log.LogInfo("client close the connection read bytes:%d err:%v", rBytes, err)
			return
		}

		log.LogInfo("read %d bytes", rBytes)

		rBuf.dataLen += rBytes
		left := rBuf.dataLen
		totalDecode := 0
		for left > 0 {
			decodeLen, resp, err := conn.session.ReadProc(rBuf.data[totalDecode:rBuf.dataLen])
			log.LogDebug("decode len:%d", decodeLen)
			if err != nil {
				log.LogWarn("Decode error on connection:%s err:%s", conn.addr, err.Error())
				return
			}
			if decodeLen == 0 {
				break
			}
			
			if !conn.handleReply(resp, nil) {
				return
			}
			totalDecode += decodeLen
			left -= decodeLen
		}
		if (totalDecode < rBuf.dataLen) && totalDecode != 0 {
			copy(rBuf.data, rBuf.data[totalDecode:rBuf.dataLen])
		}
		rBuf.dataLen -= totalDecode
		log.LogDebug("buf len:%d. data size in recv buffer:%d", len(rBuf.data), rBuf.dataLen)
		}
		
	}
}

/*
//IOHandler handle io stuff
func (conn *Connection) IOHandler() {
	log := conn.log
	onExit := func() {
		conn.conn.Close()
		conn.err = errors.New("")
		if conn.session.Clean != nil {
			conn.session.Clean(conn.session.GetPrivData())
		}

		log.LogInfo("client exit:%s ", conn.addr)
	}
	defer utility.CatchPanic(log, onExit)
	for {
		rBuf := conn.rbuf
		conn.conn.SetReadDeadline(time.Now().Add(conn.recvTimeout))
		rBytes, err := conn.conn.Read(rBuf.data[rBuf.dataLen:])
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			if !conn.handleReply(conn.session.Update(conn.session.GetPrivData())) {
				return
			}
			continue
		}

		if rBytes == 0 || err != nil {
			log.LogInfo("client close the connection read bytes:%d err:%v", rBytes, err)
			return
		}

		log.LogInfo("read %d bytes", rBytes)

		rBuf.dataLen += rBytes
		left := rBuf.dataLen
		totalDecode := 0
		for left > 0 {
			decodeLen, frame, err := conn.session.Decode(rBuf.data[totalDecode:rBuf.dataLen], conn.session.GetPrivData())
			log.LogDebug("decode len:%d", decodeLen)
			if err != nil {
				log.LogWarn("Decode error on connection:%s", conn.addr)
				return
			}
			if decodeLen == 0 {
				break
			}
			if !conn.handleReply(conn.session.HandleInput(frame, conn.session.GetPrivData())) {
				return
			}
			totalDecode += decodeLen
			left -= decodeLen
		}
		if (totalDecode < rBuf.dataLen) && totalDecode != 0 {
			copy(rBuf.data, rBuf.data[totalDecode:rBuf.dataLen])
		}
		rBuf.dataLen -= totalDecode
		log.LogDebug("buf len:%d. data size in recv buffer:%d", len(rBuf.data), rBuf.dataLen)
	}
}
*/