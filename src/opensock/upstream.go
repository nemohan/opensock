package opensock
import(
	"net"
	"utility"
	"netcore"
	"errors"
	"bytes"
)

type Upstream struct {
	tcpConn *net.TCPConn
	udpConn *net.UDPConn
	conn *netcore.Connection
	log *utility.LogContext
	msgChan chan []byte	
	outMsgChan chan []byte
}
var(
errTimeout = errors.New("timeout")
)

func  NewUpstream(cmd int, addrs []*net.TCPAddr, log *utility.LogContext) *Upstream {
	up := &Upstream{}
	for _, addr := range addrs {
		tcpConn, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			log.LogWarn("%s", err.Error())
			continue
		}

		up.log = utility.NewLogContext(0, log.GetHandle())
		up.tcpConn = tcpConn
		up.conn = netcore.NewConnection(tcpConn, up, up.log)	
		up.msgChan = make(chan []byte, 256)
		up.outMsgChan = make(chan []byte, 128)
		up.log.LogInfo("new upstream for connection:%d addr:%s", log.GetID(), tcpConn.LocalAddr().String())
		return up
	}
	return nil
}

func (u *Upstream) ReadProc(data[]byte) (int, []byte, error){
	msg := make([]byte, len(data))
	copy(msg, data)
	u.outMsgChan <- msg
	return len(data), nil, nil
}

func (u *Upstream) UpdateProc()([]byte, error){
	select {
	case msg := <- u.msgChan:
		return msg, nil	
	default:
	}
	return nil, nil
}

//RecvMsg return a byte slice 
func (u *Upstream) RecvMsg()([]byte, error){
	buf := bytes.NewBuffer(make([]byte, 0, 2048))
	more := true
	size := 0
	for more{
		select{
		case msg := <-u.outMsgChan:
			if msg == nil{
				return nil, errors.New("closed channel") 
			}
			buf.Write(msg)
			size += len(msg)
		default:
			more = false
		}
	}
	if size == 0{
		return nil, errTimeout
	}
	return buf.Bytes()[:size], nil
}

func (u *Upstream) SendMsg(msg []byte){
	u.log.LogDebug("send message to upstream size:%d", len(msg))
	u.msgChan <- msg
}