package node

import (
	"github.com/liangmanlin/gootp/kernel"
	"unsafe"
)

var rpcSvr = &kernel.Actor{
	Init: func(context *kernel.Context,pid *kernel.Pid, args ...interface{}) unsafe.Pointer {
		return nil
	},
	HandleCast: func(context *kernel.Context, msg interface{}) {

	},
	HandleCall: func(context *kernel.Context, request interface{}) interface{} {
		switch r := request.(type) {
		case *RpcCallArgs:
			var args interface{}
			if len(r.Args) > 0 {
				_,args = coder.Decode(r.Args)
			}else{
				args = nil
			}
			return callFunc(r.Fun, args)
		}
		return nil
	},
	Terminate: func(context *kernel.Context, reason *kernel.Terminate) {

	},
	ErrorHandler: func(context *kernel.Context, err interface{}) bool {
		return true
	},
}
