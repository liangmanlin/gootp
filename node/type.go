package node

import (
	"github.com/liangmanlin/gootp/kernel"
)

type NodeEnv struct {
	cookie   string
	nodeName string
	PingTick int64 `command:"ping_tick"`// 毫秒
	Port     int `command:"node_port"` //可以指定一个端口，而不是随机
}

type ping struct {}
type pong struct {}

type call struct {
	callID int64
	reply  interface{}
}

type monitorNode struct {
	pid  *kernel.Pid
	node string
}

type deMonitorNode struct {
	pid  *kernel.Pid
	node string
}

type NodeOP struct {
	Name string
	OP   OPType
}

type OPType = int

type RpcFunc *func(interface{})interface{}

const (
	OPConnect OPType = 1 << iota
	OPDisConnect
)

const (
	M_TYPE_PING byte = iota
	M_TYPE_CAST
	M_TYPE_CALL
	M_TYPE_CALL_RESULT
	M_TYPE_CAST_NAME
	M_TYPE_CALL_NAME
	M_TYPE_CAST_KMSG
	M_TYPE_CAST_NAME_KMSG
	M_TYPE_PONG
)

type app struct {
	register bool
}
