package kernel

const selfServerName = "selfSender"

var selfSenderPid *Pid

var selfSenderActor = &Actor{
	Init: func(context *Context,pid *Pid, args ...interface{}) interface{} {
		ErrorLog("%s %s started", selfServerName, pid)
		selfSenderPid = pid
		addToKernelMap(pid)
		return nil
	},
	HandleCast: func(context *Context, msg interface{}) {
		switch m := msg.(type) {
		case *routerMsg:
			Cast(m.to, m.msg)
		}
	},
	HandleCall: func(context *Context, request interface{}) interface{} {
		return nil
	},
	Terminate: func(context *Context, reason *Terminate) {

	},
	ErrorHandler: func(context *Context, err interface{}) bool {
		return true
	},
}
