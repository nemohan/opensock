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
	"strconv"
	"strings"
)

const(
	sessionStateNone = iota
	sessionStateUpstream
	sessionStateData
)

type Sock5Session struct {
	state        int
	upstream     *Upstream
	cipher       *rc4.Cipher
	sockNodeAddr []*net.TCPAddr
	log  		*utility.LogContext
	con         *netcore.Connection
	session     *core.Session
	protocol    *socks.Sock5
	mode 		int
}


func ClientInit(conn net.Conn, logHandle *utility.LogModule){
	NewSock5Session(conn, logHandle, serverConfig)
}

//NewSock5Session create a new session
func NewSock5Session(conn net.Conn, logHandle *utility.LogModule, cfg *ServerConfig) *Sock5Session{
	s := &Sock5Session{
		log : utility.NewLogContext(0, logHandle),
		state : sessionStateUpstream,
	}
	if cfg.Mode == "server"{
		s.mode = modeServer
		s.cipher, _ = rc4.NewCipher([]byte(cfg.Key))
	}else if cfg.Mode == "client"{
		s.mode = modeClient
		token := strings.Split(cfg.ServerIP, ":") 
		ip := net.ParseIP(token[0])
		port, _:= strconv.Atoi(token[1])
		s.sockNodeAddr = make([]*net.TCPAddr, 1)
		s.sockNodeAddr[0] = &net.TCPAddr{IP:ip, Port: port}
		s.upstream = NewUpstream(0, s.sockNodeAddr, s.log)
		if s.upstream == nil{
			return nil
		}
		s.cipher, _ = rc4.NewCipher([]byte(cfg.Key))
		
	}else if cfg.Mode == "standard"{
		s.mode = modeStandard
	}
	s.con = netcore.NewConnection(conn, s, s.log)
	s.protocol = socks.NewSock5(s.log)
	return s
}

//ReadProc process the data from connection
func (s *Sock5Session) ReadProc(data []byte)(int, []byte, error){
	pro := s.protocol
	state := s.protocol.GetCurrentState()
	decodeSize := 0
	if s.mode == modeClient{
			size := len(data)
			dst := make([]byte, size + 4) 
			utility.WriteUint32(dst, uint32(size))
			s.cipher.XORKeyStream(dst[4:], data)
			s.upstream.SendMsg(dst)
			return size, nil, nil
	}
	if s.mode == modeServer{
		_, size := utility.ReadUint32(data)
		if int(size) < len(data) - 4{
			return 0, nil, nil
		}
		data = data[4:]
		s.cipher.XORKeyStream(data[:size], data[:size])
		decodeSize += 4
	}

	var resp []byte
	size := 0
	var err error
	switch state{
	case socks.StateMethodNegotiation:
		size, resp, err = pro.MethodNego(data)	
		if err != nil{
			return size, resp, err
		}
		pro.SetState(socks.StateRequest)
	case socks.StateRequest:
		size, resp, err = pro.HandleRequest(data)
		if err != nil{
			return size, resp, err
		}
		pro.SetState(socks.StateDataForward)
		s.upstream = NewUpstream(0, pro.GetDestAddr(), s.log)
		if s.upstream == nil{
			return size, nil, errors.New("failed to connect server")
		}	
	case socks.StateDataForward:
		msg := make([]byte, len(data))
		copy(msg, data)
		s.upstream.SendMsg(msg)
		size = len(data)
	}
	return size + decodeSize, resp, err
}

func (s *Sock5Session) UpdateProc()([]byte, error){
	if s.upstream == nil{
		return nil, nil
	}
	msg, err := s.upstream.RecvMsg()
	if err == errTimeout{
		return nil, nil
	}
	if err != nil{

	}	
	s.log.LogDebug("recv mesg from upstream size:%d", len(msg))
	return msg, nil
}


