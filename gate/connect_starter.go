package gate

import (
	"github.com/liangmanlin/gootp/kernel"
	"net"
)

type starterState struct {
	clientSup  *kernel.Pid
	handler *kernel.Actor
	clientArgs []interface{}
}

var starterActor = &kernel.Actor{
	Init: func(ctx *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
		state := starterState{}
		state.handler = args[0].(*kernel.Actor)
		state.clientSup = args[1].(*kernel.Pid)
		state.clientArgs = args[2].([]interface{})
		return &state
	},
	HandleCast: func(ctx *kernel.Context, msg interface{}) {
		state := ctx.State.(*starterState)
		switch m := msg.(type) {
		case net.Conn:
			initArgs := append(state.clientArgs, m)
			child := &kernel.SupChild{ChildType: kernel.SupChildTypeWorker, ReStart: false, Svr: state.handler, InitArgs: initArgs}
			e, pid := kernel.SupStartChild(state.clientSup, child)
			if e != nil {
				kernel.ErrorLog("start client error: %#v", e)
			} else {
				kernel.Cast(pid, true)
			}
		}
	},

}
