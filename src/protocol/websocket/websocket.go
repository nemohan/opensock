package websocket
import(
	"core"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"utility"
	"errors"
	"netcore"
)

const(
	stateHandshake = iota
	stateData
)
const(
	stateMethod = iota
	stateURLBegin 
	stateURL
	stateURLDone
	stateVersionBegin
	stateVersion
	stateHeader
	stateValueBegin
	stateValue
	stateDone
)

const(
	FlagFinMask = 0x8000
	FlagFinShift = 15
	FlagRsvMask = 0x7000
	FlagRsvShift = 12
	FlagOpcodeMask = 0x0f00
	FlagOpcodeShift = 8
	FlagMask = 0x80
	FlagMaskShift = 7
	FlagLenMask = 0x7f
	FlagFin = 0x80
	FlagRsv = 0x00
)
const(
	ErrHandshake = "handshake error"
)
const(
	opcodeCon = 0
	opcodeTxt = 1
	opcodeBin = 2
	opcodeClose = 8
	opcodePing = 9
	opcodePong = 0xa
	opcodeHandshake = 100
)

const(
	modeClient = 0
	modeServer = 1
)

const guid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type websocketFrame struct{
	opcode int 
	data []byte
}

type Websocket struct{
	state int
	log *utility.LogContext
	respKey string
	mode int
	conn *netcore.Connection
	session *core.Session
}


func NewWebsocket(log *utility.LogContext, conn *netcore.Connection, session *core.Session) *Websocket{
	ws := &Websocket{
		log:log,
		state :stateHandshake,
		mode:modeServer,
		conn :conn,
		session:session,
	}
	return ws
}


//Input handle the input data from connection
func (ws *Websocket) Input(recvData []byte)(int, []byte, error){
	switch ws.state{
	case stateHandshake:
		size, _, err := ws.DecodeHandshake(recvData)
		if err != nil{
			return 0, nil, err
		}
		hs := ws.ConstuctHandshake()
		ws.state = stateData
		return size, hs, err
	case stateData:
		size, frame, err := ws.DecodeFrame(recvData)
		if size == 0 || err != nil{
			return size, nil, err
		}
		if frame.opcode == opcodeClose{
			if len(frame.data) >= 2{
				rest, errCode := utility.ReadUint16(frame.data)
				if len(rest) >= 0{
					ws.log.LogInfo("err code:%d reason:%s", errCode, string(rest))
				}
			}
			
			resp := ws.Close()
			ws.conn.Output(resp)
			return size, nil, errors.New("normal close")
		}
		resp, err := ws.session.HandleInput(frame, ws.session.GetPrivData())
		return size, resp, err
	}
	return 0, nil, nil
}

func (ws *Websocket) Output(sendData []byte)(int, error){
	frame := ws.EncodeTxt(sendData)
	return ws.conn.Output(frame)
}


func (ws *Websocket)ConstuctHandshake()([]byte){
	ws.log.LogDebug("key:%s", ws.respKey)
	hash := sha1.Sum([]byte(ws.respKey))
	h := make([]byte, 20)
	for i, v := range hash{
		h[i] = v
	}
	key := base64.StdEncoding.EncodeToString(h)
	
	acc := fmt.Sprintf("Sec-WebSocket-Accept: %s\r\n\r\n", key)
	resp := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"
	resp += acc

	ws.log.LogDebug("handshake:%s", resp)
	return []byte(resp)
}

func (ws *Websocket) DecodeHandshake(data []byte) (int, interface{}, error){
	dataLen := len(data)
	state := stateMethod
	uriBegin := 0

	ws.log.LogDebug("dump data:%s", string(data))
	versionBegin := 0
	headers := make(map[string]string, 0)
	headerBegin := 0
	valueBegin := 0
	header := ""
	uri := ""
	version := ""
	reqLineEnd := false
	for i := 0; i < dataLen; i++{
		c := data[i]
		switch state{
		case stateMethod:
			switch c{
			case ' ':
			default:
				if data[i] != 'G' && data[i+1] != 'E' && data[i+2] != 'T'{
					return 0, nil, errors.New(ErrHandshake)
				}
				i+= 3
				state = stateURLBegin
			}
		case stateURLBegin:
			switch c{
			case ' ':
			default:
				uriBegin = i
				state = stateURL
			}
		case stateURL:
			switch c{
			case ' ':
				uri = string(data[uriBegin: i])
				state = stateVersionBegin
			default:
			}
		case stateVersionBegin:
			switch c{
			case ' ':
			default:
				versionBegin = i
				state = stateVersion
			}
		case stateVersion:
			switch c{
			case ' ':
				if !reqLineEnd{
					version = string(data[versionBegin:i])
					reqLineEnd = true
				}
			case '\r':
				if !reqLineEnd{
					version = string(data[versionBegin:i])
				}
			case '\n':
				state = stateHeader
			}
			
		case stateHeader:
			switch c{
			case ':':
				header = string(data[headerBegin:i])
				state = stateValueBegin
			case '\r':
			case '\n':
				headerBegin = i
			default:
				if headerBegin == 0{
					headerBegin = i
				}
			}

		case stateValueBegin:
			switch c{
			case ' ':
			default:
				headerBegin = 0
				valueBegin = i
				state = stateValue
			}
		case stateValue:
			switch c{
			case '\r':
				headers[header] = string(data[valueBegin:i])
			case '\n':
				state = stateDone

			}
		case stateDone:
			switch c{
			case '\r':
				goto out
			case '\n':
				goto out
			default:
				state = stateHeader
				i--
			}

		}
	}

	out:
	ws.log.LogDebug("uri:%s version:%s header len:%d", uri, version, len(headers))
	for h, v := range headers{
		ws.log.LogDebug("%s:%s", h, v)
	}
	ws.state = stateData
	frame := &websocketFrame{opcode:opcodeHandshake}
	ws.respKey = headers["Sec-WebSocket-Key"] + guid
	return dataLen, frame, nil
}



func (ws *Websocket) isValidFrame(frame *websocketFrame){

}

//DecodeFrame decode 
func (ws *Websocket) DecodeFrame(data []byte)(int, *websocketFrame, error){
	size := len(data)
	ws.log.LogDebug("data size:%d", size)
	if size < 2{
		ws.log.LogInfo("data is less than 2")
		return 0, nil, nil
	}

	buf, hd := utility.ReadUint16(data)
	fin := (hd & FlagFinMask) >> FlagFinShift
	rsv := (hd & FlagRsvMask) >> FlagRsvShift
	opcode := (hd & FlagOpcodeMask) >> FlagOpcodeShift
	mask := (hd & FlagMask) >> FlagMaskShift
	frameLen := uint64(hd & FlagLenMask)
	maskSize := 0
	frameLenSize := 0
	if mask == 1{
		maskSize = 4
	}
	if (frameLen == 126){
		frameLenSize = 2
	}
	if frameLen == 127{
		frameLenSize = 8
	}

	hdSize := maskSize + frameLenSize
	if size < hdSize{
		return 0, nil, nil
	}

	if frameLen == 126{
		payloadLen := uint16(0)
		buf, payloadLen = utility.ReadUint16(buf)
		frameLen = uint64(payloadLen)
	}
	if frameLen == 127{
		buf, frameLen = utility.ReadUint64(buf)
	}

	ws.log.LogDebug("fin:%d rsv:%d opcode:%d mask:%d len:%d frame size:%d", 
		fin, rsv, opcode, mask, frameLen, hdSize + 2 + int(frameLen))
	hdSize += 2
	size -= hdSize
	if uint64(size) < frameLen{
		ws.log.LogDebug("data is not enough, size:%d", size)
		return 0, nil, nil
	}
	maskValue := buf[:4]
	dst := make([]byte, frameLen)
	payload := buf[4: frameLen + 4]
	for i, b := range payload{
		j := i % 4
		dst[i] = maskValue[j] ^ b
	}

	frame := new(websocketFrame)
	frame.opcode = int(opcode)
	frame.data = dst
	return hdSize + int(frameLen), frame, nil
}

func (ws *Websocket) decodeFrame(frame *websocketFrame){

}

func (ws *Websocket) encodeHeader(opcode int, size int)(int, []byte){
	hd := 0
	hd |= FlagFin
	hd |= FlagRsv
	hd |= (opcode & 0xff)
	
	hdSize := 2
	frameLenSize := 0
	sizeInBytes := make([]byte, 8)
	if size <= 125{
		frameLenSize = size & 0xff
	}else if size >= 126 && size <= 65535{
		frameLenSize = 126 & 0xff
		utility.WriteUint16(sizeInBytes, uint16(size))
		hdSize += 2
	}else{
		frameLenSize = 127 & 0xff
		utility.WriteUint64(sizeInBytes, uint64(size))
		hdSize += 8
	}

	buf := make([]byte, size + hdSize)
	buf[0] = byte(hd & 0xff)
	buf[1] = byte(frameLenSize & 0xff)
	copy(buf[2:], sizeInBytes[:hdSize - 2])
	return hdSize, buf
}

//EncodeTxt send txt frame to another peer
func (ws *Websocket) EncodeTxt(data []byte)[]byte{
	size := len(data)
	hdSize, buf := ws.encodeHeader(opcodeTxt, size)
	copy(buf[hdSize:], data)
	return buf
}

//EncodeBin encode bin frame
func (ws *Websocket) EncodeBin(data []byte)[]byte{
	size := len(data)
	hdSize, buf := ws.encodeHeader(opcodeBin, size)
	copy(buf[hdSize:], data)
	return buf
}


//Close close frame to another peer and close the connection
func (ws *Websocket) Close()[]byte{
	_, buf := ws.encodeHeader(opcodeClose, 0)
	return buf
}

//Pong response pong to the ping
func (ws *Websocket) Pong(){

}

//Ping send ping frame to another peer
func (ws *Websocket) Ping(){

}

func (ws *Websocket) HandShake(){
	header := fmt.Sprintf("GET /chat HTTP/1.1\r\n")
	header += "Host: tenios.lumosgame.com\r\n"
	header += "Upgrade: websocket\r\n"
	header += "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n"
	header += "Sec-WebSocket-Protocol: chat\r\n"
	header += "Sec-WebSocket-Version: 13\r\n\r\n"
	ws.conn.Output([]byte(header))
}