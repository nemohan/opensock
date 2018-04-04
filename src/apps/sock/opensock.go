package main

import(
	"utility"
	"opensock"
	"time"
)
func main(){

	log := utility.NewLog("opensock", "DBG", 500000, "")
	/*
	logCtx := utility.NewLogContext(0, log)
	go netcore.TcpServer("0.0.0.0", 22222, logCtx, opensock.ClientInit)
	*/
	server := opensock.NewSockServer(log)
	server.Main()
	for{
		time.Sleep(time.Second * 10)
	}

}