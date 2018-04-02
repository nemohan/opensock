package teachserver

import (
	"core"
	"encoding/json"
	"errors"
	"net"
	"netcore"
	"sync/atomic"
	"time"
	"utility"
)

/********************
redis store format:
account:   uid:name:account

**********************/
const (
	errOK uint16 = iota
	errNoData
	errAccountAlreadyExist
	ErrInternal
)

const (
	stateOffline = iota
	stateLogin
	stateOnline
	stateWaitingExchangeKey
	stateKeyExchangeDone
)
const (
	cmdTypeReq = 1 << iota
	cmdTypeReply
	cmdTypeSend
)

const (
	constHeaderLen       = 13 + 4
	unencryptHeaderLen   = 13
	maxRequestSize       = 1024
	NET_ID_LEN           = 4
	MSG_TYPE_LEN         = 1
	connectionStateNone  = 1
	moduleMulticastID    = 1
	constProtocolVersion = 2
)

const cmdBacklog = 1024
const invalidUID = 0

type groupCMDCtx struct {
	gid       uint32
	groupName string
	echo      bool
}

type Account struct {
	DevID string `json:"devid"` //this may be useless
	UID   uint32 `json:"uid"`
	Name  string `json:"Name"`
}

const (
	cmdNone = iota
	cmdCreateGroup
	cmdJoinGroup
	cmdExitGroup
	cmdDestroyGroup // cmd about groups
	cmdUpdateInfo   //5
	cmdMulticast
	cmdHeartbeat
	cmdSubTest
	cmdOffline
	cmdLogin //10
	cmdCreateGroupCallback
	cmdJoinGroupCallback
	cmdMemberOnline
	cmdMemberOffline
	cmdSearchGroup //15
)

var cmdTable = map[string]uint16{
	"createGroup":          cmdCreateGroup,
	"joinGroup":            cmdJoinGroup,
	"exitGroup":            cmdExitGroup,
	"destroyGroup":         cmdDestroyGroup,
	"update":               cmdUpdateInfo,
	"heartbeat":            cmdHeartbeat,
	"multicast":            cmdMulticast,
	"sub":                  cmdSubTest,
	"offline":              cmdOffline,
	"login":                cmdLogin,
	"createGroupCallback":  cmdCreateGroupCallback,
	"cmdJoinGroupCallback": cmdJoinGroupCallback,
	"cmdMemberOnline":      cmdMemberOnline,
	"cmdMemberOffline":     cmdMemberOffline,
	"cmdSearchGroup":       cmdSearchGroup,
}

var cmdInvertTable map[int]string

func init() {
	cmdInvertTable = make(map[int]string, len(cmdTable))
	for k, v := range cmdTable {
		cmdInvertTable[int(v)] = k
	}
}

type GroupInfo struct {
	OwnerID  uint64 `json:"ownerid"`
	Name     string `json:"name"`
	GID      uint32 `json:"gid"`
	MemberID uint32 `json:"memberid"`
	Info     string `json:"info"`
}
type Client struct {
	id             uint64
	uuid           uint64 // server internal use
	conn           *netcore.Connection
	cmdChan        chan core.CmdContainer
	exitChan       chan bool
	bfake          bool
	group          GroupInfo
	lastActiveTime time.Time
	state          int
	disableTimeout bool
	requetIDGen    int
	pendingRequest map[int]*Request
	log            *utility.LogContext
	rc4            utility.RC4Simple
	dh64           utility.DH64
	session        *core.Session
	db             *core.DBRedis
	PirvData       interface{} //for test purpose
}

type cmdHandlerFunc func(*Client, *Request) ([]byte, error)
type cmdCallbackFunc func(*Client, *clientCmd) ([]byte, error)

var funcTable = map[int]cmdHandlerFunc{
	cmdCreateGroup:  handleCreateGroup,
	cmdJoinGroup:    handleJoinGroup,
	cmdExitGroup:    handleExitGroup,
	cmdDestroyGroup: handleDestroyGroup,
	cmdUpdateInfo:   handleUpdateState,
	cmdHeartbeat:    handleHeartbeat,
	cmdSubTest:      handleJoinGroup,
	cmdLogin:        handleLogin,
	cmdMulticast:    handleMulticast,
}

var callbackCMDTable = map[int]cmdCallbackFunc{
	cmdCreateGroupCallback: createGroupCallback,
	cmdJoinGroupCallback:   joinGroupCallback,
	cmdMulticast:           multicastCallback,
	cmdMemberOnline:        memberOnlineCallback,
	cmdMemberOffline:       memberOfflineCallback,
	cmdUpdateInfo:          updateCallback,
	cmdSearchGroup:         searchGroupCallback,
}

var clientIDSource uint64 = 10000

func getCMDID() uint64 {
	//return atomic.AddUint64(&cmd_id_src, 1)
	return 0
}

func getClientID() uint64 {
	return atomic.AddUint64(&clientIDSource, 1)
}

//NewFakeClient create an new fake client instance for internal use only
func NewFakeClient() *Client {
	return &Client{id: 0,
		uuid:    0,
		bfake:   true,
		cmdChan: make(chan core.CmdContainer, 1024),
	}
}

//NewClient create a new client instance represent an connection
func NewClient(conn *net.TCPConn, log *utility.LogModule, db *core.DBRedis) *Client {
	client := &Client{id: 0,
		uuid:           getClientID(),
		conn:           nil,
		cmdChan:        make(chan core.CmdContainer, 256),
		exitChan:       make(chan bool, 1),
		bfake:          false,
		lastActiveTime: time.Now(),
		state:          stateKeyExchangeDone,
		db:             db,
	}

	session := core.NewSession(DecodeMsg, nil, handleInput, nil, updateHandler, cleanHandler)
	session.Init(client)
	client.session = session
	client.log = utility.NewLogContext(uint32(client.uuid), log)
	client.conn = netcore.NewConnection(conn, session, client.log)
	//core.RouteRegister(client)
	return client
}

func (c *Client) Init(session *core.Session) {
	//session.Init(c)
	c.session = session
	c.conn.SetSession(session)

}

//GetID return the client's internal instance id
func (c *Client) GetID() uint64 {
	return c.uuid
}

//GetUID return the client's player ID
func (c *Client) GetUID() uint64 {
	return c.id
}

//CloseChannel close message channel
func (c *Client) CloseChannel() {
}

//IsFake whether the client is not a real client
func (c *Client) IsFake() bool {
	return c.bfake
}

func (c *Client) getRequestID() int {
	return 0
}

//SendMsg  send message to me
func (c *Client) SendMsg(cmd core.CmdContainer) {
	defer func() {
		if r := recover(); r != nil {
			//c.log(LOG_WARN, "the channel of client:%d closed reason:%v", c.id, r)
		}
	}()
	c.cmdChan <- cmd
}

func (c *Client) updateTime() {
	c.lastActiveTime = time.Now()
}

func (c *Client) timeoutCheck() bool {
	now := time.Now()
	if now.After(c.lastActiveTime.Add(30000 * time.Second)) {
		c.log.LogInfo("client:%d uuid:%d timeout", c.id, c.uuid)
		return true
	}
	return false

}

func handleInput(v interface{}, ctx interface{}) ([]byte, error) {
	c := ctx.(*Client)
	req := v.(*Request)
	c.log.LogDebug("call handleInput")
	if c.state == stateKeyExchangeDone {
		cmd := req.cmd
		handler, ok := funcTable[int(cmd)]
		if !ok {
			c.log.LogWarn("invliad command:%s", req.hd.Cmd)
			return nil, errors.New("invalid command")
		}
		c.updateTime()
		reply, err := handler(c, req)
		return reply, err
	}

	return c.handleExchangeKey(req)
}

func cleanHandler(ctx interface{}) {
	c := ctx.(*Client)
	c.log.LogDebug("do some clean stuff")
	if c.group.GID == constInvalidGroupID {
		return
	}
	cmd := newClientCmd(nil, nil, nil, c.id, moduleMulticastID, cmdOffline)
	cmd.groupCtx = new(groupCMDCtx)
	//There is only one group for now
	cmd.groupCtx.groupName = c.group.Name
	cmd.groupCtx.gid = c.group.GID
	core.RouteUnregister(c)
	core.ForwardCmd(cmd)
}

func objToJSON(src interface{}) []byte {
	if bytes, err := json.Marshal(src); err == nil {
		return bytes
	}
	return nil
}

func (c *Client) closeChannel() {

	/* 	close(c.exit_ch)
	   	close(c.cmd_ch) */
}

//case 1: register
//case 2: already registered
func handleLogin(c *Client, req *Request) ([]byte, error) {
	account := new(Account)
	if req.dataLen > 0 {
		if err := json.Unmarshal(req.data, account); err != nil {
			return nil, err
		}
	}
	//TODO: fix the race condition
	accKey := utility.Format("uid:%d:name:%s", account.UID, account.Name)
	data, err := c.db.Get(accKey)
	if err != nil {
		c.log.LogDebug("get account:%v err:%v ", account, err)
		return c.getReply(ErrInternal, req), err
	}
	if data != nil {
		info := new(Account)
		json.Unmarshal(data, info)
		c.id = uint64(info.UID)
		cmd := newClientCmd(nil, c, nil, c.id, moduleMulticastID, cmdSearchGroup)
		core.ForwardCmd(cmd)
		c.log.LogInfo("old user login. account:%s", string(data))
		r := c.replyWithData(data, req)
		return r, nil
	}

	c.log.LogInfo("new user register. info:%v", account)
	uid, _ := c.db.GetCounter()
	accKey = utility.Format("uid:%d:name:%s", uid, account.Name)
	c.db.Set(accKey, req.data)
	c.id = uid

	// register route
	core.RouteRegister(c)
	//get group information
	cmd := newClientCmd(nil, c, nil, c.id, moduleMulticastID, cmdSearchGroup)
	core.ForwardCmd(cmd)
	return c.getReply(errOK, req), nil
}

func handleCreateGroup(c *Client, req *Request) ([]byte, error) {
	c.log.LogDebug("call handleCreateGroup")
	c.id = req.hd.From

	cmd := newClientCmd(nil, c, nil, req.hd.From, moduleMulticastID, cmdCreateGroup)
	cmd.req = req
	groupCmd := new(groupCMDCtx)
	groupCmd.groupName = req.hd.Group
	cmd.groupCtx = groupCmd
	core.ForwardCmd(cmd)
	return nil, nil
}

func handleJoinGroup(c *Client, req *Request) ([]byte, error) {
	c.log.LogDebug("call handleJoinGroup")
	c.id = req.hd.From
	cmd := newClientCmd(nil, c, nil, req.hd.From, moduleMulticastID, cmdJoinGroup)
	cmd.req = req
	groupCmd := new(groupCMDCtx)
	groupCmd.groupName = req.hd.Group
	cmd.groupCtx = groupCmd
	core.ForwardCmd(cmd)
	return nil, nil
}

func handleMulticast(c *Client, req *Request) ([]byte, error) {
	c.log.LogDebug("call handleMulticast")
	c.id = req.hd.From

	cmd := newClientCmd(nil, nil, nil, c.id, moduleMulticastID, cmdMulticast)
	groupCmd := new(groupCMDCtx)
	groupCmd.groupName = req.hd.Group
	groupCmd.echo = req.hd.Echo
	cmd.groupCtx = groupCmd
	cmd.req = req
	core.ForwardCmd(cmd)
	return c.getReply(errOK, req), nil
}

func handleHeartbeat(c *Client, req *Request) ([]byte, error) {
	c.updateTime()
	c.log.LogDebug("call handle_heartbeat of client:%d", c.id)
	return c.getReply(errOK, req), nil
}

func handleExitGroup(c *Client, req *Request) ([]byte, error) {
	return nil, nil
}
func handleDestroyGroup(c *Client, req *Request) ([]byte, error) {

	return nil, nil
}

func handleUpdateState(c *Client, req *Request) ([]byte, error) {
	log := c.log
	if c.group.OwnerID == c.id {
		cmd := newClientCmd(nil, c, nil, c.id, moduleMulticastID, cmdUpdateInfo)
		core.ForwardCmd(cmd)
		log.LogDebug("multicast to students")
		return c.getReply(errOK, req), nil
	}
	cmd := newClientCmd(nil, c, nil, c.id, c.group.OwnerID, cmdUpdateInfo)
	core.ForwardCmd(cmd)
	return c.getReply(errOK, req), nil
}

func (c *Client) handleExchangeKey(req *Request) ([]byte, error) {
	log := c.log
	log.LogDebug("handleExchangeKey")
	data := req.data
	_, clientPubKey := utility.ReadUint64(data)
	c.dh64.Init()
	privateKey, pubKey := c.dh64.DH64KeyPair()
	secretKey := c.dh64.Secret(privateKey, clientPubKey)

	log.LogDebug("key from client:%d secret key:%d server public key:%d", clientPubKey, secretKey, pubKey)
	byteKey := make([]byte, 8)
	for i := 0; i < 8; i++ {
		byteKey[i] = byte((secretKey >> uint32(i*8)) & 0xff)
	}
	c.rc4.Init(byteKey)

	msg := new(MsgBuf)
	totalLen := 13 + 8 + 8
	payloadLen := 8
	msg.begin = make([]byte, totalLen)
	utility.WriteUint32(msg.begin[unencryptHeaderLen:], 0)
	utility.WriteUint32(msg.begin[unencryptHeaderLen+4:], uint32(payloadLen))

	utility.WriteUint64(msg.begin[unencryptHeaderLen+8:], pubKey)
	EncodeMsg(msg, req.netID, constProtoReply)
	c.session.Decode = encryptDecode
	c.state = stateKeyExchangeDone
	return msg.begin, nil
}

/*****************************************************/

const (
	CMDNone = iota
	CMDCreateGroup
	CMDJoinGroup
	CMDExitGroup
	CMDDestroyGroup // cmd about groups
	CMDUpdateInfo   //5
	CMDMulticast
	CMDHeartbeat
	CMDSubTest
	CMDOffline
	CMDLogin //10
	CMDCreateGroupCallback
	CMDJoinGroupCallback
	CMDMemberOnline
	CMDMemberOffline
)

func CMDToString(cmd int) string {
	return cmdInvertTable[cmd]
}
