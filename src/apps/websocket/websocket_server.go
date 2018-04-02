package main


import (
	"core"
	"net"
	"sync"
	"time"
	"utility"
	"protocol/websocket"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
)

var chanNotify chan bool
var nodeExitChan chan bool
var netExit bool

type WebsocketServer struct {
	log        *utility.LogContext
	chanNotify chan bool
	listener   *net.TCPListener
	db         *core.DBRedis
	waitGroup *sync.WaitGroup
	timeout   int
}

//NewTeachServer create an new server instance
func NewWebsocketServer(log *utility.LogModule, db *core.DBRedis, timeout int) *WebsocketServer {
	return &WebsocketServer{log: utility.NewLogContext(utility.AllocModuleID(), log),
		chanNotify: make(chan bool),
		db:         db,
		timeout:   timeout}
}

func (s *WebsocketServer) Init(waitGroup *sync.WaitGroup) {
	s.waitGroup = waitGroup
	waitGroup.Add(1)
}

//Start listen on specific address and port
func (s *WebsocketServer) Start(addr string, port int) {
	ipAddr := net.ParseIP(addr)
	log := s.log
	if ipAddr == nil {
		log.LogFatal("Invalid ip address:%s", addr)
	}
	if port < 0 || port > 65535 {
		log.LogFatal("Invalid port:%d", port)
	}

	servAddr := &net.TCPAddr{IP: ipAddr, Port: port}
	listener, err := net.ListenTCP("tcp", servAddr)
	if err != nil {
		log.LogFatal("Failed to listen on addr %s:%d reson:%s", addr, port, err.Error())
	}
	s.listener = listener
	log.LogInfo("chat listen on addr: %s:%d", addr, port)
	go s.loop()
}

func (s *WebsocketServer) Stop() {
	s.chanNotify <- true
}

func (s *WebsocketServer) loop() {
	log := s.log
	defer utility.CatchPanic(log, nil)
	for {
		select {
		case <-s.chanNotify:
			log.LogInfo("listen exit")
			s.listener.Close()
			s.waitGroup.Done()
			return
		default:
			s.listener.SetDeadline(time.Now().Add(200 * time.Millisecond))
			conn, err := s.listener.AcceptTCP()
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			}
			if err != nil {
				log.LogWarn("listener error:%s", err.Error())
			}
			client := websocket.NewWebsocketSession(conn, s.log.GetHandle(), s.timeout)
			log.LogDebug("new connection arrived:%s on addr:%s id:%d", conn.RemoteAddr().String(),
			 conn.LocalAddr().String(), client.GetSessionID())
		}
	}

}

func main(){
	hash := sha1.Sum([]byte("m3cXeKcuPBs/CuC/Ca0n/w==258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	h := make([]byte, 20)
	for i, v := range hash{
		h[i] = v
	}
	key := base64.StdEncoding.EncodeToString(h)
	fmt.Printf("key:%s\n", key)
}