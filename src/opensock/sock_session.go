package opensock
import (
	"core"
	"utility"
	"crypto/rc4"
	"net"
	"netcore"
	//"strings"
	//"time"
	"protocol/socks"
	"errors"
)




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
	protocol    *socks.Sock5
}


func ClientInit(conn net.Conn, logHandle *utility.LogModule){
	NewSock5Session(conn, logHandle)
}

//NewSock5Session create a new session
func NewSock5Session(conn net.Conn, logHandle *utility.LogModule) *Sock5Session{
	s := &Sock5Session{
		log : utility.NewLogContext(0, logHandle),
		state : sessionStateUpstream,
	}
	s.con = netcore.NewConnection(conn, s, s.log)
	s.protocol = socks.NewSock5(s.log)
	return s
}

//ReadProc process the data from connection
func (s *Sock5Session) ReadProc(data []byte)(int, []byte, error){
	pro := s.protocol
	state := s.protocol.GetCurrentState()
	switch state{
	case socks.StateMethodNegotiation:
		size, resp, err := pro.MethodNego(data)	
		if err != nil{
			return size, resp, err
		}
		pro.SetState(socks.StateRequest)
		return size, resp, err
	case socks.StateRequest:
		size, resp, err := pro.HandleRequest(data)
		if err != nil{
			return size, resp, err
		}
		pro.SetState(socks.StateDataForward)
		s.upstream = NewUpstream(0, pro.GetDestAddr(), s.log)
		if s.upstream == nil{
			return size, nil, errors.New("failed to connect server")
		}	
		return size, resp, err
	case socks.StateDataForward:
		msg := make([]byte, len(data))
		copy(msg, data)
		s.upstream.SendMsg(msg)
		return len(data), nil,nil
	}
	return 0, nil, nil
}

func (s *Sock5Session) UpdateProc()([]byte, error){
	msg, err := s.upstream.RecvMsg()
	if err == errTimeout{
		return nil, nil
	}
	if err != nil{

	}	
	s.log.LogDebug("recv mesg from upstream size:%d", len(msg))
	return msg, nil
}


