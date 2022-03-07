package node

import "github.com/liangmanlin/gootp/kernel"

type Connect struct {
	Time     int64
	Sign     string
	Name     string
	DestName string
}

type ConnectSucc struct {
	Self *kernel.Pid
	Name string
}

type RpcCallArgs struct {
	Fun  string
	Args []byte
}