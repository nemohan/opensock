package core

/*****


*/
type Module interface{
	Init() bool
	Destroy() bool
	Send() 
	Reconfig() 
}