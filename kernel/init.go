package kernel

import (
	"github.com/liangmanlin/gootp/kernel/kct"
)

const initServerName = "init"

var initServerPid *Pid = nil

var isStop = false

var stopCB func() = nil

// 停止整个服务
func InitStop() {
	ErrorLog("system going to init stop")
	if !isStop && stopCB != nil {
		stopCB()
	}
	// 先停止各种app
	CallTimeOut(appPid, &initStop{}, 600)
	CallTimeOut(initServerPid, &initStop{}, 600)
}

type initStop struct {
	reply  bool
	callID int64
	recCh  chan interface{}
}

type initState struct {
	started *kct.BMap
}

var initActor = &Actor{
	Init: func(context *Context, pid *Pid, args ...interface{}) interface{} {
		ErrorLog("%s %s started", initServerName, pid)
		initServerPid = pid
		addToKernelMap(pid)
		return &initState{started: kct.NewBMap()}
	},
	HandleCast: func(context *Context, msg interface{}) {
		state := context.State.(*initState)
		switch m := msg.(type) {
		case *Pid:
			state.started.Insert(m.id, m)
		case *PidExit:
			state.started.Delete(m.Pid.id)
		}
	},
	HandleCall: func(context *Context, request interface{}) interface{} {
		state := context.State.(*initState)
		switch r := request.(type) {
		case *initStop:
			if !isStop {
				isStop = true
				initStopF(state, context)
			}
			return nil
		case stopFunc:
			stopCB = r
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
			iStop := &initStop{callID: callID, recCh: context.self.callResult}
			Cast(pid, &actorOP{iStop})
			for rl := true; rl; {
				succ, rs := context.recResult(callID, context.self.callResult, 3)
				if succ {
					rl = false
				} else {
					switch r := rs.(type) {
					case *CallError:
						// 应该只有数据库持久进程会超时
						if r.ErrType == CallErrorTypeTimeOut && pid.IsAlive() {
							if name := TryGetName(pid); name != "" {
								ErrorLog("stop %s timeout", name)
							}
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
