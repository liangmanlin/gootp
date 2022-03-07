package gate

import (
	"github.com/liangmanlin/gootp/kernel"
	"net"
)

const (
	ErrDeadLine = iota + 1
	ErrReadErr
	ErrClosed
)

type TcpError struct {
	ErrType int
	Err     error
}

type Pack struct {
	ProtoID int
	Proto   interface{}
}

type app struct {
	name    string
	port    int
	handler *kernel.Actor
	opt     []optFun
}

type Conn interface {
	net.Conn
	Send(buf []byte) (int, error)
	SendBufHead(buf []byte) error
	SetHead(head int)
	GetHead() int
	Recv(len int, timeOutMS int) ([]byte,error)
	StartReader(dest *kernel.Pid)
}

type optFun func(*optStruct)