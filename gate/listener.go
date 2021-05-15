package gate

import (
	"fmt"
	"github.com/liangmanlin/gootp/kernel"
	"log"
	"net"
	"unsafe"
)

type listener struct {
	ls net.Listener
}

var listenerActor = &kernel.Actor{
	Init: func(context *kernel.Context,pid *kernel.Pid, args ...interface{}) unsafe.Pointer {
		flag := args[0].(string)
		handler := args[1].(*kernel.Actor)
		port := args[2].(int)
		clientSup := args[3].(*kernel.Pid)
		childSup := args[4].(*kernel.Pid)
		opt := args[5].([]interface{})
		optStruct := parseOpt(opt)
		l := listener{}
		ls, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
		if err != nil {
			kernel.ErrorLog(err.Error())
			log.Panic(err)

		}
		// 假如port=0，那么端口是随机的，记录一下
		addrMap[flag] = ls.Addr().String()
		l.ls = ls
		l.startAccept(ls, handler, clientSup, childSup, optStruct)
		return unsafe.Pointer(&l)
	},
	HandleCast: func(context *kernel.Context, msg interface{}) {
	},
	HandleCall: func(context *kernel.Context, request interface{}) interface{} {
		return nil
	},
	Terminate: func(context *kernel.Context, reason *kernel.Terminate) {
		(*listener)(context.State).ls.Close()
	},
	ErrorHandler: func(context *kernel.Context, err interface{}) bool {
		return true
	},
}

func (l *listener) startAccept(ls net.Listener, handler *kernel.Actor, clientSup *kernel.Pid, childSup *kernel.Pid, opt *optStruct) {
	for i := 0; i < opt.acceptNum; i++ {
		pid, err := kernel.Start(acceptorActor, clientSup, ls, handler, opt.clientArgs)
		if err != nil {
			log.Panic(err)
		}
		kernel.Cast(pid, true)
		pid.SetDie()
	}
}

func parseOpt(opt []interface{}) *optStruct {
	df := &optStruct{acceptNum: 10, clientArgs: nil}
	for _, v := range opt {
		switch v2 := v.(type) {
		case AcceptNum:
			if v2 > 0 {
				df.acceptNum = int(v2)
			}
		case ClientArgs:
			df.clientArgs = v2
		}
	}
	return df
}

type optStruct struct {
	acceptNum  int
	clientArgs ClientArgs
}
