
package netcore
import (
	"net"
	"utility"
)

type ClientInitHandler func(net.Conn, *utility.LogModule)
func TcpServer(addr string, port int, log *utility.LogContext, clientInitHandler ClientInitHandler){
	ip := net.ParseIP(addr)
	servAddr := net.TCPAddr{IP:ip, Port: port}
	listener, err := net.ListenTCP("tcp", &servAddr)
	if err != nil{
		panic(err.Error())
	}
	defer utility.CatchPanic(log, nil)
	for {
		conn, err := listener.Accept()
		if err != nil{
			log.LogWarn("accept error on addr:%v", err)
		}
		log.LogInfo("new connection:%s arrived on addr:%s", conn.RemoteAddr().String(), servAddr.String())
		clientInitHandler(conn, log.GetHandle())	
	}
}
