package core

//Protocol  protocol is an abstract layer
//the concret protocol implement this interface
type Protocol interface{
	Output ([]byte)(int, error)
	Input ([]byte)(int, []byte, error)
}