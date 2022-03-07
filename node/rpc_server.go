package node

import (
	"github.com/liangmanlin/gootp/kernel"
)

var rpcSvr = kernel.NewActor(
	kernel.InitFunc(func(ctx *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
		ctx.ChangeCallMode()
		return nil
	}),
	kernel.HandleCallFunc(func(context *kernel.Context, request interface{}) interface{} {
		go rpcHandleCall(request.(*kernel.CallInfo))
		return nil
	}))

func rpcHandleCall(call *kernel.CallInfo) {
	var result interface{}
	switch r := call.Request.(type) {
	case *RpcCallArgs:
		var args interface{}
		if len(r.Args) > 0 {
			_, args = coder.Decode(r.Args)
		} else {
			args = nil
		}
		result = callFunc(r.Fun, args)
	}
	kernel.Reply(call.RecCh,call.CallID,result)
}