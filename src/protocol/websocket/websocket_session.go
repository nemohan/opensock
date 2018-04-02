package websocket

import (
	//"encoding/json"
	"core"
	"net"
	"utility"
	"sync/atomic"
	"netcore"
	"errors"
)

var websocketSessionID uint32

type Uid struct{
	UID int `json:"uid"`
}

type Info struct{
	User string `json:"user"`
	Password string `json:"password"`
}

type wsCmd struct{
	
}
func getSessionID() uint32{
	return atomic.AddUint32(&websocketSessionID, 1)
}

const(
	sessionLogin = iota
	sessionMsg
	sessionNoAck
)

type WebsocketSession struct{
	log *utility.LogContext
	sessionID uint32
	conn *netcore.Connection
	session *core.Session
	protocol *Websocket
	state int
}

//TODO: 
func NewWebsocketSession(conn *net.TCPConn, logHandle *utility.LogModule, timeout int) *WebsocketSession{
	ws := &WebsocketSession{
		log: utility.NewLogContext(getSessionID(), logHandle),
		state: sessionLogin,
	}

	ws.session = core.NewSession(nil, nil, inputHandler, nil, updateHandler, cleanHandler)
	ws.session.Init(ws)
	ws.conn = netcore.NewConnection(conn, ws.session, ws.log)
	
	ws.protocol = NewWebsocket(ws.log, ws.conn, ws.session)
	ws.conn.AddProtocol(ws.protocol)
	ws.protocol.HandShake()
	return ws
}

func (ws *WebsocketSession) GetSessionID() uint32{
	return ws.sessionID
}
/*************
1 session layer just return the data which will be sent by connection layer,
2 session layer call protocol's output interface to output data
which is better
************/
func inputHandler(v interface{}, ctx interface{})([]byte, error){
	c := ctx.(*WebsocketSession)
	//proto := c.protocol
	frame := v.(*websocketFrame)
	c.log.LogDebug("data:%s", string(frame.data))
	switch c.state{
	case sessionLogin:
		/*
		uid := &Uid{UID:int(c.sessionID)}	
		bj, _ := json.Marshal(uid)
		resp := proto.EncodeTxt(bj)
		c.state = sessionMsg
		return resp, nil
		*/
	case sessionMsg:
		/*
		resp := proto.EncodeTxt([]byte("ok"))
		c.state = sessionNoAck
		return resp, nil
		*/
	case sessionNoAck:
		return nil, nil
	}

	return nil, nil
}


func cleanHandler(ctx interface{}){

}

func updateHandler(ctx interface{})([]byte, error){

	return nil, nil
}

func (ws *WebsocketSession)Send(v interface{})(int, error){
	switch req := v.(type){
	case string:
		return ws.protocol.Output([]byte(req))
	case []byte:
		return ws.protocol.Output(req)
	}

	return 0, errors.New("only support string and []byte")
	
}