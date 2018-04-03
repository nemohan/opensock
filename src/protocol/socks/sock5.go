package socks

import (
	"utility"
	//"core"
	"crypto/rc4"
	"errors"
	"net"
	//"netcore"
	"sync/atomic"
	//"time"
)

const (
	sockVersion5 = 5
)
const (
	methodNoAuth = iota
	methodGSSAPI
	methodUserPasswd
	methodIANA
	methodReserved = 0x80
	methodNoAccept = 0xff
)
const (
	sockCMDNone = iota
	sockCmdConnect
	sockCmdBind
	sockCmdUDP
)
const (
	sockRepOK = iota
	sockRepErr
	sockRepNotAllowed
	sockRepHostUnreachable
	sockRepConnRefused
)
const (
	sockAddrV4         = 1
	sockAddrDomainName = 3
	sockAddrV6         = 4
)
const (
	upstreamNone = iota
	upstreamUDP  = 1
	upstreamTCP  = 2
)
const sockReserved = 0
const (
	StateMethodNegotiation = iota
	StateRequest
	StateDataForward
)

const (
	frameNone = iota
	frameUDPAddr
	frameTCPAddr
	frameData
)

type Sock5Frame struct{
	data []byte
	frameType int
	cmd int
}
type sockStat struct {
	rxBytes uint32
	txBytes uint32
}
type Sock5 struct {
	state        int
	serverMode   bool
	clientMode   bool
	cipher       *rc4.Cipher
	destAddrs []*net.TCPAddr
	log    		*utility.LogContext
}

var sockServerStat sockStat

func addRxBytes(bytes uint32) {
	atomic.AddUint32(&sockServerStat.rxBytes, bytes)
}

func addTxBytes(bytes uint32) {
	atomic.AddUint32(&sockServerStat.txBytes, bytes)
}

func getRxBytes() uint32 {
	return atomic.LoadUint32(&sockServerStat.rxBytes)
}
func getTxBytes() uint32 {
	return atomic.LoadUint32(&sockServerStat.txBytes)
}

func NewSock5(log *utility.LogContext) *Sock5{
	return &Sock5{
		log: log,
		state : StateMethodNegotiation,
	}
}

func (s *Sock5)SetState(state int){
	s.state = state
}
func (s *Sock5) GetCurrentState()int{
	return s.state
}
func (s *Sock5) GetDestAddr()[]*net.TCPAddr{
	return s.destAddrs
}
func (s *Sock5) MethodNego(data []byte) (int, []byte, error){
	ver := data[0]
	if ver != sockVersion5 {
		s.log.LogWarn("invalid sock version:%d", ver)
		return 0, nil, errors.New("invalid sock version")
	}
	nmethod := data[1]
	
	find := false
	for _, method := range data[2:] {
		if method == methodNoAuth {
			find = true
			break
		}
	}
	
	if !find {
		s.log.LogDebug("does not find no authentication method in method list. method num:%d version:%d",
			nmethod, ver)
		return 0, nil, errors.New("not supported method")
	}

	resp := make([]byte, 2)
	resp[0] = sockVersion5
	resp[1] = methodNoAuth
	/*
	if s.serverMode {
		s.cipher.XORKeyStream(resp, resp)
		c.Output(resp)
		addTxBytes(2)
		return
	}
	c.Output(resp)
	s.log.LogDebug("version:%d method number:%d", ver, nmethod)
	*/
	return 2 + int(nmethod), resp, nil
}


func resolveIPPort(cmd int, data []byte, log *utility.LogContext) ([]*net.TCPAddr, int, error) {
	if cmd != sockAddrV4 && cmd != sockAddrDomainName{
		return nil, 0, errors.New("invalid cmd")
	}
	addrs := make([]*net.TCPAddr, 0)
	size := 0
	var err error
	switch cmd{
	case sockAddrV4:
		ip := net.IPv4(data[0], data[1], data[2], data[3])
		if ip == nil {
			return nil, 0, errors.New("not invalid address")
		}
		port := int(data[4]) & 0xff
		port = (port << 8) | int(data[5]&0xff)
		addrs = append(addrs, &net.TCPAddr{IP: ip, Port: port})
		size = 6
	case sockAddrDomainName:
		size = int(data[0])
		name := string(data[1 : size+1])
		log.LogDebug("resolve hostname: %s data:%v", name, data[1:size+1])
		ips, err := net.LookupIP(name)
		if err != nil{
			break
		}
		size++
		port := int(data[size]) & 0xff
		size++
		port = (port << 8) | int(data[size])
		for _, ip := range ips{
			addrs = append(addrs, &net.TCPAddr{IP: ip, Port: port})
			//log.LogInfo("ip:%s for name")
		}
		size += 1
	}
	return addrs, size, err
}

//HandleRequest
func (s *Sock5) HandleRequest(data []byte) (int, []byte, error){
	s.log.LogDebug("call handleRequest")
	if data[0] != sockVersion5 || data[2] != sockReserved || (data[3] != sockAddrV4 && data[3] != sockAddrDomainName) {
		s.log.LogWarn("invalid version:%d cmd:%d r:%d addr:%d", data[0], data[1], data[2], data[3])
		return 0, nil, errors.New("")
	}

	if data[1] != sockCmdConnect && data[1] != sockCmdUDP {
		s.log.LogWarn("invalid cmd:%d", data[1])
		return 0, nil, errors.New("")
	}

	addrs, size, err := resolveIPPort(int(data[3]), data[4:], s.log)
	if err != nil {
		s.log.LogWarn("%s", err.Error())
		return 0, nil, err 
	}
	s.destAddrs = addrs
	resp := make([]byte, 10)
	resp[0] = sockVersion5
	resp[1] = sockRepOK
	resp[2] = 0
	resp[3] = sockAddrV4
	for i := 4; i < 10; i++ {
		resp[i] = data[i]
	}
	return 4 + size, resp, nil
}


