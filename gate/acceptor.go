package gate

import (
	"github.com/liangmanlin/gootp/kernel"
	"net"
)

type acceptor struct {
	clientSup  *kernel.Pid
	listener   net.Listener
	handler    *kernel.Actor
	clientArgs []interface{}
}

var acceptorActor = &kernel.Actor{
	Init: func(context *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
		a := acceptor{}
		a.clientSup = args[0].(*kernel.Pid)
		a.listener = args[1].(net.Listener)
		a.handler = args[2].(*kernel.Actor)
		a.clientArgs = args[3].([]interface{})
		return &a
	},
	HandleCast: func(context *kernel.Context, msg interface{}) {
		switch msg.(type) {
		case bool:
			context.State.(*acceptor).accept(context)
		}
	},
	HandleCall: func(context *kernel.Context, request interface{}) interface{} {
		return nil
	},
	Terminate: func(context *kernel.Context, reason *kernel.Terminate) {

	},
	ErrorHandler: func(context *kernel.Context, err interface{}) bool {
		return true
	},
}

func (a *acceptor) accept(context *kernel.Context) {
	conn, err := a.listener.Accept()
	if err == nil {
		svr := a.handler
		c := NewConn(conn)
		initArgs := append(a.clientArgs, c)
		child := &kernel.SupChild{ChildType: kernel.SupChildTypeWorker, ReStart: false, Svr: svr, InitArgs: initArgs}
		e, pid := kernel.SupStartChild(a.clientSup, child)
		if e != nil {
			kernel.ErrorLog("accept error: %#v", e)
		} else {
			kernel.Cast(pid, true)
		}
		context.CastSelf(true)
	} else {
		if e, ok := err.(*net.OpError); ok && e.Op == "accept" {

		} else {
			kernel.ErrorLog("accept error: %s", err.Error())
		}
	}
}
