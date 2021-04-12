package node

import (
	"github.com/liangmanlin/gootp/kernel"
)

type NodeEnv struct {
	cookie   string
	nodeName string
	PingTick int64 // 毫秒
	Port     int //可以指定一个端口，而不是随机
}

type ping int32

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
