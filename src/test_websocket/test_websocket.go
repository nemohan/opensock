package main

import(
	//"encoding/json"
	"net"
	//"websocket"
	"utility"
	"time"
)

type CmdT struct{
	Type string `json:"type"`
	Key string `json:"key"`
}

type ContentT struct{
	Username string `json:"username"`
	Password string `json:"password"`
}

type LobbyContent struct{
	UID string 	`json:"uid"`
}

type Login struct{
	Cmd CmdT `json:"cmd"`
	Content ContentT `json:"content"` 
}

type LobbyCmd struct{
	Cmd CmdT `json:"cmd"`
	Content LobbyContent `json:"content"` 
}

const gateway_addr = "119.29.161.170"
func test(log *utility.LogModule){
	
	net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(gateway_addr), Port:20000})
	//time.Sleep(time.Second)
	/*
	session := websocket.NewWebsocketSession(conn, log, 300)

	login := Login{Cmd:CmdT{Type:"query", Key:"login"}, 
	Content:ContentT{Username:"lumos", Password:"123456"},
}

	req, _ := json.Marshal(login)
	session.Send(req)

	time.Sleep(time.Second)
	cmd := LobbyCmd{
		Cmd:CmdT{Type: "query", Key:"profile_get"},
			Content:LobbyContent{UID:"10000"},
	}
	req, _ = json.Marshal(cmd)
	session.Send(req)
	*/
}

func main(){
	log := utility.NewLog("test", "DBG", 500000, "")
	for i := 0; i < 50000; i++{
		time.Sleep(time.Minute)
		for j := 0; j < 10; j++{
			time.Sleep(time.Second)
			go test(log)
		}
		
	}
	for{
		time.Sleep(time.Second)
	}
}