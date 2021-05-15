package gate

import "github.com/liangmanlin/gootp/kernel"

type TcpError struct {
	Err error
}

type AcceptNum int

type ClientArgs []interface{}

type Pack struct {
	ProtoID int
	Proto   interface{}
}

type app struct {
	name    string
	port    int
	handler *kernel.Actor
	opt     []interface{}
}
