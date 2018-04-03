package core

type Frame interface {
}


type Session interface{
	ReadProc([]byte)(int, []byte, error)
	UpdateProc()([]byte, error)
}