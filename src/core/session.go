package core

type Frame interface {
}

type DecodeHandler func(data []byte, ctx interface{}) (int, interface{}, error)
type EncodeHandler func(data []byte, ctx interface{}) int
type InputHandler func(v interface{}, ctx interface{}) ([]byte, error)
type OutputHandler func(v interface{}, ctx interface{}) error
type UpdateHandler func(ctx interface{}) ([]byte, error)
type CleanHandler func(ctx interface{})

//NewSession create an new session instance
func NewSession(decode DecodeHandler, encode EncodeHandler,
	inputHandler InputHandler, outputHandler OutputHandler,
	updateHandler UpdateHandler, cleanHandler CleanHandler) *Session {
	return &Session{
		Decode:       decode,
		Encode:       encode,
		HandleInput:  inputHandler,
		HandleOutput: outputHandler,
		Update:       updateHandler,
		Clean:        cleanHandler,
	}
}

func (s *Session) Init(data interface{}) {
	s.PrivData = data
}

func (s *Session) GetPrivData() interface{} {
	return s.PrivData
}

//Session one intermediate layer between connection and client
type Session struct {
	Decode       DecodeHandler
	Encode       EncodeHandler
	HandleInput  InputHandler
	HandleOutput OutputHandler
	Update       UpdateHandler
	Clean        CleanHandler
	PrivData     interface{}
}
