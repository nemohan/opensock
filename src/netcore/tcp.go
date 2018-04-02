
package netcore
import (
	"net"
)
func tcpServer(addr string, port int){
	ip := net.ParseIP(addr)
	servAddr := net.TCPAddr{IP:ip, Port: port}

	err := net.ListenTCP("tcp", &servAddr)
}