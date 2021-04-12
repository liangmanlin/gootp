package kernel

import (
	"github.com/liangmanlin/gootp/kernel/kct"
	"unsafe"
)

const initServerName = "init"

var initServerPid *Pid = nil

var isStop = false

// 停止整个服务
func InitStop() {
	ErrorLog("init stop")
	CallTimeOut(initServerPid, &initStop{}, 600)
}

type initStop struct {
	flag   bool
	callID int64
	recCh  chan interface{}
}

type initState struct {
	started *kct.BMap
	stopCB  func()
}

var initActor = &Actor{
	Init: func(context *Context,pid *Pid, args ...interface{}) unsafe.Pointer {
		ErrorLog("%s %s started", initServerName, pid)
		initServerPid = pid
		addToKernelMap(pid)
		return unsafe.Pointer(&initState{started: kct.NewBMap(), stopCB: nil})
	},
	HandleCast: func(context *Context, msg interface{}) {
		state := (*initState)(context.State)
		switch m := msg.(type) {
		case *Pid:
			state.started.Insert(m.id, m)
		case *PidExit:
			state.started.Delete(m.Pid.id)
		}
	},
	HandleCall: func(context *Context, request interface{}) interface{} {
		state := (*initState)(context.State)
		switch r := request.(type) {
		case *initStop:
			if !isStop {
				if state.stopCB != nil {
					state.stopCB()
				}
				isStop = true
				initStopF(state, context)
			}
			return nil
		case stopFunc:
			state.stopCB = r
		}
		return nil
	},
	Terminate: func(context *Context, reason *Terminate) {
	},
	ErrorHandler: func(context *Context, err interface{}) bool {
		return true
	},
}

func initStopF(state *initState, context *Context) {
	f := func(e interface{}) {
		pid := e.(*Pid)
		if _, ok := kernelPid[pid.id]; !ok && pid.IsAlive() {
			callID := makeCallID()
			iStop := &initStop{callID: callID, recCh: context.self.call}
			Cast(pid, &actorOP{iStop})
			for rl := true; rl; {
				succ, rs := context.recResult(callID, context.self.call, 3)
				if succ {
					rl = false
				} else {
					switch r := rs.(type) {
					case *CallError:
						// 应该只有数据库持久进程会超时
						if r.ErrType == CallErrorTypeTimeOut && pid.IsAlive() {
						} else {
							rl = false
						}
					default:
						rl = false
					}
				}
			}
		}
	}
	state.started.Foreach(f)
	ErrorLog("kernel going to stop")
	kernelStop()
}

func initRegister(pid *Pid) *Pid {
	Cast(initServerPid, pid)
	return initServerPid
}
