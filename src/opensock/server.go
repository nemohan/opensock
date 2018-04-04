package opensock

import(
	"encoding/json"
	"io/ioutil"
	"netcore"
	"time"
	"utility"
	"strings"
	"strconv"

)

const (
	modeClient = iota
	modeServer
	modeStandard
)
type ServerConfig struct{
	ServerIP string `json:"serverip"`
	BindAddr string `json:"bindaddr"`
	Mode     string `json:"mode"`
	Key 	 string `json:"key"`
}

type SockServer struct{
	mode int
	log *utility.LogModule 
	logCtx *utility.LogContext
}

var serverConfig *ServerConfig
func NewSockServer(log *utility.LogModule)*SockServer{
	return &SockServer{
		mode:modeStandard,
		log:log,
		logCtx: utility.NewLogContext(0, log),
	}
}
func LoadConfig(log *utility.LogContext) *ServerConfig{
	data, err := ioutil.ReadFile("opensock.cfg")	
	if err != nil{
		log.LogWarn("load config error:%v", err)
		return nil
	}
	cfg := &ServerConfig{}
	if err := json.Unmarshal(data, cfg); err != nil{
		log.LogWarn("%v", err)
		return nil
	}
	return cfg	
}
func (serv *SockServer) Main(){
	cfg :=  LoadConfig(serv.logCtx)	
	if cfg == nil{
		panic("invalid config file")
	}
	serverConfig = cfg
	addrPair := strings.Split(cfg.BindAddr, ":")	
	if len(addrPair) != 2{
		panic("invalid ip address")
	}

	port, err := strconv.Atoi(addrPair[1])
	if err != nil{
		panic("invalid port")
	}
	go netcore.TcpServer(addrPair[0], port, serv.logCtx, ClientInit)
	for{
		time.Sleep(time.Second * 10)
	}
}