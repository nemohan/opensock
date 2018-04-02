package socks

import (
	"utility"
	"core"
	"crypto/rc4"
	"errors"
	"net"
	"netcore"
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
	stateMethodNegotiation = iota
	stateRequest
	stateDataForward
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
	upstream     *Upstream
	readDeadLine int
	serverMode   bool
	clientMode   bool
	cipher       *rc4.Cipher
	sockNodeAddr *net.TCPAddr
	encrypt    bool
	conn        *netcore.Connection
	session     *core.Session
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
	}
}


func (s *Sock5) methodNego(data []byte) {
	c := s.conn
	ver := data[0]
	if ver != sockVersion5 {
		c.Close()
		return
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
		c.Close()
		return
	}

	resp := make([]byte, 2)
	resp[0] = sockVersion5
	resp[1] = methodNoAuth
	if s.serverMode {
		s.cipher.XORKeyStream(resp, resp)
		c.Output(resp)
		addTxBytes(2)
		return
	}
	c.Output(resp)
	s.log.LogDebug("version:%d method number:%d", ver, nmethod)
}


func resolveIPPort(cmd int, data []byte, log *utility.LogContext) ([]net.IP, int, error) {
	if cmd == sockAddrV4 {
		ip := net.IPv4(data[0], data[1], data[2], data[3])
		if ip == nil {
			return nil, 0, errors.New("not invalid address")
		}
		ips := make([]net.IP, 0)
		ips = append(ips, ip)
		port := int(data[4]) & 0xff
		port = (port << 8) | int(data[5]&0xff)
		return ips, port, nil
	}
	size := data[0]
	if cmd == sockAddrDomainName {
		name := string(data[1 : size+1])
		log.LogDebug("resolve hostname: %s data:%v", name, data[1:size+1])
		ips, err := net.LookupIP(name)
		size++
		port := int(data[size]) & 0xff
		size++
		port = (port << 8) | int(data[size])
		return ips, port, err
	}
	return nil, 0, errors.New("invalid cmd")
}

func (s *Sock5) handleRequest(data []byte) error{
	s.log.LogDebug("call handleRequest")
	if data[0] != sockVersion5 || data[2] != sockReserved || (data[3] != sockAddrV4 && data[3] != sockAddrDomainName) {
		s.log.LogWarn("invalid version:%d cmd:%d r:%d addr:%d", data[0], data[1], data[2], data[3])
		return errors.New("")
	}

	if data[1] != sockCmdConnect && data[1] != sockCmdUDP {
		s.log.LogWarn("invalid cmd:%d", data[1])
		return errors.New("")
	}

	ipAddrs, port, err := resolveIPPort(int(data[3]), data[4:], s.log)
	if err != nil {
		s.log.LogWarn("%s", err.Error())
		return errors.New("")
	}
	s.log.LogDebug("addrs:%v port:%d", ipAddrs, port)
	s.upstream = NewUpstream(int(data[1]), ipAddrs, port, s.log)
	resp := make([]byte, 10)
	resp[0] = sockVersion5
	resp[1] = sockRepOK
	resp[2] = 0
	resp[3] = sockAddrV4
	for i := 4; i < 10; i++ {
		resp[i] = data[i]
	}

	if !s.encrypt{
		s.conn.Output(resp)
		return nil
	}
	
	s.cipher.XORKeyStream(resp, resp)
	s.conn.Output(resp)
	addTxBytes(uint32(len(resp)))
	return nil
}

func (s *Sock5) Output(sendData []byte)(int, error){
	if s.encrypt{
		encryptData := make([]byte, len(sendData))
		s.cipher.XORKeyStream(encryptData, sendData)
		s.conn.Output(encryptData)
	}
	s.conn.Output(sendData)
	return 0, nil
}

func (s *Sock5) Input(recvData []byte)(int, []byte, error){
	if s.encrypt{
		data := recvData
		plainTxt := make([]byte, len(data))
		s.cipher.XORKeyStream(plainTxt, data)
		addRxBytes(uint32(len(data)))
		rxBytes := getRxBytes()
		txBytes := getTxBytes()
		s.log.LogDebug("txBytes:%d  rxBytes:%d txM:%d rxM:%d", txBytes, rxBytes, txBytes>>20, rxBytes>>20)
	}

	switch s.state {
	case stateMethodNegotiation:
		s.methodNego(recvData)
		s.state = stateRequest
		return len(recvData), nil, nil
	case stateRequest:
		frame := s.handleRequest(recvData)
		s.state = stateDataForward
		s.session.HandleInput(frame, s.session.GetPrivData())
		return len(recvData), nil, nil
	case stateDataForward:
		frame := &Sock5Frame{data:recvData, frameType:frameData}
		resp, err := s.session.HandleInput(frame, s.session.GetPrivData())
		return len(recvData), nil, nil
	}
	return 0, nil, nil
}


func  NewUpstream(cmd int, addrs []net.IP, port int, log *utility.LogContext) *Upstream {
	up := &Upstream{}
	if cmd == sockCmdConnect {
		for _, ip := range addrs {
			dstAddr := net.TCPAddr{IP: ip, Port: port}
			log.LogDebug("dest addr:%s", dstAddr.String())
			tcpConn, err := net.DialTCP("tcp", nil, &dstAddr)
			if err != nil {
				log.LogWarn("%s", err.Error())
				continue
			}

			up.tcpConn = tcpConn
			up.rbuf = make([]byte, 65536)
			return up
		}
	}

	return nil
}
