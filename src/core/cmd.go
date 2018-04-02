package core

type CmdContainer interface {
	GetFromID() uint32
	GetToID() uint32
	GetType() uint32
	String() string
}
