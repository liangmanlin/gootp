package kernel

import "unsafe"

type actorOP struct {
	op interface{}
}

type regName string

type link struct {
	pid *Pid
}

type PidExit struct {
	Pid    *Pid
	Reason *Terminate
}

type routerMsg struct {
	to  *Pid
	msg interface{}
}

type Terminate struct {
	Reason string
}

type msgQueue struct {
	next *msgQueue
	msg  interface{}
}

const (
	actorCodeNone int = 1 << iota
	actorCodeExit
	actorCodeInitStop
)

type callErrorType int

type CallError struct {
	ErrType callErrorType
	err     interface{}
}

const (
	CallErrorTypeTimeOut = 1 << iota
)

type stop string

type stopFunc func()

type InitFunc func(ctx *Context, pid *Pid, args ...interface{}) unsafe.Pointer
type HandleCastFunc func(ctx *Context, msg interface{})
type HandleCallFunc func(ctx *Context, request interface{}) interface{}
type TerminateFunc func(ctx *Context, reason *Terminate)
type ErrorHandleFunc func(ctx *Context, err interface{}) bool

type Actor struct {
	// 初始化回调
	Init InitFunc
	// 接收消息
	HandleCast HandleCastFunc
	// 接受同步调用
	HandleCall HandleCallFunc
	// actor退出回调
	Terminate TerminateFunc
	// 当发生catch错误时调用，如果返回false，那么进程将会退出
	ErrorHandler ErrorHandleFunc
}

type Node struct {
	id   int32
	name string
}

type NodeMsg struct {
	Dest *Pid
	Msg  interface{}
}

type NodeMsgName struct {
	Dest string
	Msg  interface{}
}

type NodeCall struct {
	Dest   *Pid
	Req    interface{}
	CallID int64
	Ch     chan interface{}
}

type NodeCallName struct {
	Dest   string
	Req    interface{}
	CallID int64
	Ch     chan interface{}
}

type KMsg struct {
	ModID int32
	Msg   interface{}
}

type Application interface {
	Name() string
	Start(bootType AppBootType) *Pid // return the supervisor pid
	Stop(stopType AppStopType)
	SetEnv(Key string,value interface{})
	GetEnv(key string) interface{}
}

type appInfo struct {
	app Application
	pid *Pid
}

type AppBootType int
type AppStopType int

const (
	APP_BOOT_TYPE_START = iota + 1
	APP_BOOT_TYPE_RESTART
)

const (
	APP_STOP_TYPE_NORMAL = iota + 1
	APP_STOP_TYPE_RESTART
)


