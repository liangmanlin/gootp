package httpd

import (
	"github.com/lesismal/nbio"
	"github.com/liangmanlin/gootp/kernel"
	"runtime/debug"
)

type workerState struct {
	manager *kernel.Pid
	engine  *Engine
}

var workerActor = &kernel.Actor{
	Init: func(ctx *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
		state := workerState{}
		state.manager = args[0].(*kernel.Pid)
		state.engine = args[1].(*Engine)
		return &state
	},
	HandleCast: func(ctx *kernel.Context, msg interface{}) {
		switch r := msg.(type) {
		case *Request:
			state := ctx.State.(*workerState)
			defer kernel.Cast(state.manager, ctx.Self())
			var ok bool
			defer func() {
				if !ok {
					p := recover()
					if p != nil {
						kernel.ErrorLog("catch error:%s,Stack:%s", p, debug.Stack())
					}
					r.Conn.Close()
				}
			}()
			if r.f != nil {
				r.f(ctx, r)
				r.replyNormal()
			} else {
				if c, _ := r.Conn.(*nbio.Conn).IsClosed(); c {
					ok = true
					kernel.ErrorLog("connect closed: %s uri: %s", r.RemoteIP(), r.RequestURI)
					return
				}
				//kernel.DebugLog("worker:%s %s",ctx.Self(),r.RequestURI)
				if h,err := routerHandler(state.engine, r); err == nil {
					h.f(ctx,r)
					r.replyNormal()
				} else {
					kernel.ErrorLog("err: %s uri: %s", err.Error(), r.RequestURI)
					r.reply404()
				}
			}
			ok = true
		}

	},
	HandleCall: func(ctx *kernel.Context, request interface{}) interface{} {
		return nil
	},
	ErrorHandler: func(ctx *kernel.Context, err interface{}) bool {
		return true
	},
}
