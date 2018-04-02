package core

import (
	"sync"
	"sync/atomic"
	"utility"
)

const (
	routeCmdNone = iota
	routeCmdAddClient
	routeCmdRemoveClient
	routeCmdUnicast
	routeCmdBroadcast
	routeCmdAddIdleClient
)

//ClientContainer
type ClientContainer interface {
	GetID() uint32
	GetUID() uint32
	SendMsg(CmdContainer)
	CloseChannel()
	IsFake() bool
}

const (
	ROUTE_EXIT_SIG      = 1
	ROUTE_NOTIFY_SIG    = 2
	CMD_OFFLINE         = 3
	CMD_EXIT            = 4
	ROUTE_CMD_BROADCAST = 5
)
const (
	NOTIFY_TYPE_ONLINE = 1 << iota
	NOTIFY_TYPE_OFFLINE
)
const backlog = 1024

type NotifyContext struct {
	id     uint64
	noType int
	//req    *Request
}

type notifyFunc func(*NotifyContext)

type RouteMsg struct {
	from     uint32
	to       uint32
	cmd      uint16
	msg      interface{}
	id       uint64
	routeCmd int
	group    []uint64
}

type RouteContext struct {
	exitChan       chan int
	clientTable    map[uint32]ClientContainer
	idleClient     map[uint32]ClientContainer
	cmdChan        chan CmdContainer
	mapUIDToUUID   map[uint32]uint64
	notifyHandlers map[string]notifyFunc
	moduleTable    map[uint32]ClientContainer
	cmdPool        CmdList
	registerChan   chan *routeCmd
	log            *utility.LogContext
	waitGroup      *sync.WaitGroup
}

type routeCmd struct {
	cmdType int
	client  ClientContainer
	id      uint32
	module  bool
}

var msgIDSrc uint64
var routeCtx *RouteContext

//NewRouteMsg create a route message
func NewRouteMsg(from, to uint32, cmd uint16, msg interface{}, route_cmd int) *RouteMsg {
	return &RouteMsg{
		from:     from,
		to:       to,
		msg:      msg,
		cmd:      cmd,
		id:       genMsgID(),
		routeCmd: route_cmd,
		group:    make([]uint64, 0),
	}
}

func NewBroadcastMsg(from uint32, cmd uint16, msg interface{}) *RouteMsg {
	return &RouteMsg{
		from:     from,
		to:       0,
		msg:      msg,
		cmd:      cmd,
		id:       genMsgID(),
		routeCmd: ROUTE_CMD_BROADCAST,
		group:    make([]uint64, 0),
	}
}

func genMsgID() uint64 {
	return atomic.AddUint64(&msgIDSrc, 1)
}

//NewRoute create an new route instance
func NewRoute(log *utility.LogModule) *RouteContext {
	routeCtx = &RouteContext{
		exitChan:       make(chan int),
		clientTable:    make(map[uint32]ClientContainer, backlog),
		idleClient:     make(map[uint32]ClientContainer),
		moduleTable:    make(map[uint32]ClientContainer, 10),
		cmdChan:        make(chan CmdContainer, backlog),
		notifyHandlers: make(map[string]notifyFunc, 10),
		mapUIDToUUID:   make(map[uint32]uint64),
		registerChan:   make(chan *routeCmd, 1024),
		log:            nil,
	}

	moduleID := utility.AllocModuleID()
	routeCtx.log = utility.NewLogContext(moduleID, log)
	routeCtx.log.LogInfo("route module id:%d", moduleID)
	return routeCtx
}

//Init start an new goroutine
func (r *RouteContext) Init(waitGroup *sync.WaitGroup) {
	r.waitGroup = waitGroup
	waitGroup.Add(1)
	go routeCtx.route()
}

func Route_close_all_client() {
	routeCtx.exitChan <- ROUTE_NOTIFY_SIG
}

func RouteExit() {
	routeCtx.exitChan <- ROUTE_EXIT_SIG
}

//  RouteRegisteradd client to route
func RouteRegister(client ClientContainer) {
	id := client.GetUID()
	cmd := new(routeCmd)
	cmd.client = client
	cmd.cmdType = routeCmdAddClient
	cmd.id = id
	routeCtx.registerChan <- cmd
	routeCtx.log.LogDebug("call route register client:%d", id)
}

func RouteRegisterModule(client ClientContainer) {
	id := client.GetID()
	cmd := new(routeCmd)
	cmd.client = client
	cmd.cmdType = routeCmdAddClient
	cmd.id = uint32(id)
	cmd.module = true
	routeCtx.registerChan <- cmd
	routeCtx.log.LogDebug("call route register client:%d", id)
}

//RegisterNotifyHandler register handler about some event
func RegisterNotifyHandler(name string, f notifyFunc) {
	routeCtx.notifyHandlers[name] = f
	routeCtx.log.LogInfo("register handler:%s to route", name)
}

//RouteUnregister remove client from route
func RouteUnregister(c ClientContainer) {
	cmd := new(routeCmd)
	cmd.id = c.GetUID()
	cmd.cmdType = routeCmdRemoveClient
	cmd.client = c
	routeCtx.registerChan <- cmd
}

func (r *RouteContext) routeHandleUnicast(cmd CmdContainer, table map[uint32]ClientContainer) {
	to, ok := table[cmd.GetToID()]
	if !ok {
		r.log.LogInfo("cmd:%s from:%d to:%d is offline", cmd.String(), cmd.GetFromID(), cmd.GetToID())
		return
	}
	to.SendMsg(cmd)
	r.log.LogDebug("route msg from:%d to :%d", cmd.GetFromID(), cmd.GetToID())
}

/*
func routeHandleBroadcast(cmd *Cmd, table map[uint64]ClientContainer, log *utility.LogModule) {
	log.LogDebug("call broadcast handler")
	for _, v := range cmd.group {
		to, ok := table[v]
		if !ok {
			log.LogInfo("can not find uid:%d in route table size:%d", v, len(table))
			continue
		}
		//log.LogDebug("broadcast from:%d to:%d cmd:%s", cmd.fromID, to.id, cmd_name_table[int(cmd.cmd_type)])
		toCmd := NewCmd(nil, cmd.from, to, cmd.fromID, to.GetID(), cmd.cmdType)
		//toCmd.req = cmd.req
		to.SendMsg(toCmd)

	}
}
*/
func ForwardCmd(cmd CmdContainer) {
	routeCtx.cmdChan <- cmd
}

/*
func (r *RouteContext) handleNotify(cmd *Cmd) {

		for _, handler := range r.notify_handlers {
			handler(&NotifyContext{id: cmd.fromID, noType: NOTIFY_TYPE_ONLINE, req: cmd.req})
		}

}*/

/*
func (r *RouteContext) remove(cmd *Cmd) {
	delete(r.clientTable, cmd.fromID)
	//delete(r.idleClient, cmd.from.GetUID())
	//cmd.from.closeChannel()
	r.log.LogDebug("remove client:%d from route", cmd.fromID)

	for _, v := range r.moduleTable {
		offlineCmd := NewCmd(nil, cmd.from, nil, cmd.fromID, v.GetID(), CMD_OFFLINE)
		v.SendMsg(offlineCmd)
	}
}
*/
func (r *RouteContext) notifyClientExit(sig int) bool {
	/*
		log := r.log
		if sig != ROUTE_NOTIFY_SIG {
			log.LogInfo("Route exit")
			return true
		}
		log.LogDebug("notify all client to exit")
		for _, c := range r.idleClient {
			delete(r.clientTable, uint64(c.GetID()))
			log.LogInfo("idle table: notify client:%d uuid:%d to exit", c.GetID(), c.GetUID())

		}
		for _, c := range r.clientTable {
			log.LogDebug("client table: notify client:%d uuid:%d to exit", c.GetID(), c.GetUID())
		}*/
	return false
}

func (r *RouteContext) route() {
	defer utility.CatchPanic(r.log, nil)
	log := r.log
	for {
		select {
		case <-r.exitChan:
			log.LogInfo("route prepare to exit")
			r.waitGroup.Done()
			return
		case cmd := <-r.cmdChan:
			log.LogDebug("got new cmd from:%d to:%d", cmd.GetFromID(), cmd.GetToID())
			if e, ok := r.moduleTable[cmd.GetToID()]; ok {
				e.SendMsg(cmd)
				break
			}
			r.routeHandleUnicast(cmd, r.clientTable)
		case cmd := <-r.registerChan:
			switch cmd.cmdType {
			case routeCmdAddClient:
				if !cmd.module {
					r.clientTable[cmd.id] = cmd.client
					log.LogDebug("add client:%d to route", cmd.id)
					break
				}
				r.moduleTable[cmd.id] = cmd.client

			case routeCmdRemoveClient:
				delete(r.clientTable, cmd.id)
				log.LogInfo("remove client:%d from route", cmd.id)
			}

		}
	}

}

func (r *RouteContext) Stop() {
	r.exitChan <- 1
}
