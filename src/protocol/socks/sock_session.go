package socks
import (
	"core"
	"utility"
	"crypto/rc4"
	"net"
	"netcore"
	//"strings"
	"time"
//	"errors"
)



type Upstream struct {
	tcpConn *net.TCPConn
	udpConn *net.UDPConn
	rbuf    []byte
	conn *netcore.Connection
	log *utility.LogContext
}

const(
	sessionStateNone = iota
	sessionStateUpstream
	sessionStateData
)
type Sock5Session struct {
	state        int
	upstream     *Upstream
	readDeadLine int
	serverMode   bool
	clientMode   bool
	cipher       *rc4.Cipher
	sockNodeAddr *net.TCPAddr
	log  		*utility.LogContext
	con         *netcore.Connection
	session     *core.Session
	protocol    *Sock5
}



func init() {
	//sockServerStat.lock = new(sync.Mutex)
}

func NewSock5Session(conn *net.TCPConn, logHandle *utility.LogModule) *Sock5Session{
	s := &Sock5Session{
		log : utility.NewLogContext(0, logHandle),
		state : sessionStateUpstream,
	}
	s.session = core.NewSession(nil, nil, inputHandler, nil, updateHandler, nil)
	s.session.Init(s)
	s.con = netcore.NewConnection(conn, s.session, s.log)
	s.protocol = NewSock5(s.log)
	s.con.AddProtocol(s.protocol)
	return s
}


func updateHandler(v interface{})([]byte, error){
	session := v.(*Sock5Session)
	if session.upstream == nil{
		return nil, nil
	}
	up := session.upstream
	conn := session.upstream.tcpConn
	log := session.log
	for {
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(50)))
		size, err := conn.Read(up.rbuf)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				log.LogDebug("upstream timeout")
				break
			}
			log.LogWarn("%s", err.Error())
			conn.Close()
			return nil, err
		}

		log.LogDebug("read size from upstream:%d ", size)
		session.protocol.Output(up.rbuf[:size])
	}
	return nil, nil
}

func inputHandler(v interface{}, ctx interface{})([]byte, error){
	frame := v.(*Sock5Frame)
	s := ctx.(*Sock5Session)
	up := s.upstream
	if s.upstream == nil{
		s.upstream = s.protocol.upstream
	}
	up.tcpConn.Write(frame.data)
	return nil, nil
}

/*

func ClientInit(c *netcore.TcpClient, cfg *netcore.ConfigContext) {
	sock := &Sock5{owner: c}
	c.UpdateHandler = updateHandler
	c.FrameHandler = frameHandler
	c.SetCodec(encode, decode)
	c.SetTimeout(0)
	c.SetMinFrameSize(1)
	c.SetPrivData(sock)
	sock.readDeadLine = cfg.ReadDeadLine
	c.Log.LogDebug("upstream read deadline:%d", sock.readDeadLine)
	cipher, err := rc4.NewCipher([]byte(cfg.PrivKey))
	if err != nil {
		c.Log.LogDebug("%s", err.Error())
		c.Close()
		return
	}
	sock.cipher = cipher
	if strings.Compare("server", cfg.Mode) == 0 {
		sock.serverMode = true
	} else if strings.Compare("client", cfg.Mode) == 0 {
		sock.clientMode = true
		sock.sockNodeAddr = &net.TCPAddr{IP: net.ParseIP(cfg.NodeAddr), Port: cfg.NodePort}
		tcpConn, err := net.DialTCP("tcp", nil, sock.sockNodeAddr)
		if err != nil {
			c.Log.LogDebug("%s", err)
			c.Close()
			return
		}
		sock.upstream.tcpConn = tcpConn
		sock.upstream.rbuf = make([]byte, 65536)
	}
}*/



