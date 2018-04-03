package main

import(
	"netcore"
	"utility"
	"opensock"
	"time"
)
func main(){

	log := utility.NewLog("opensock", "DBG", 500000, "")
	logCtx := utility.NewLogContext(0, log)
	go netcore.TcpServer("0.0.0.0", 22222, logCtx, opensock.ClientInit)
	for{
		time.Sleep(time.Second * 10)
	}

}