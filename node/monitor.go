package node

import (
	"github.com/liangmanlin/gootp/kernel"
	"unsafe"
)

var monitorPid *kernel.Pid

func Monitor(nodeName string,pid *kernel.Pid) {
	kernel.Cast(monitorPid,&monitorNode{pid: pid,node: nodeName})
}

func DeMonitor(nodeName string,pid *kernel.Pid)  {
	kernel.Cast(monitorPid,&deMonitorNode{pid: pid,node: nodeName})
}

type monitor struct {
	n2pid map[string]map[int64]*kernel.Pid
	id2n  map[int64]map[string]bool
}

var monitorActor = &kernel.Actor{
	Init: func(context *kernel.Context,pid *kernel.Pid, args ...interface{}) unsafe.Pointer {
		monitorPid = pid
		state := monitor{
			n2pid: make(map[string]map[int64]*kernel.Pid),
			id2n:  make(map[int64]map[string]bool),
		}
		kernel.ErrorLog("NodeMonitor %s started",pid)
		return unsafe.Pointer(&state)
	},
	HandleCast: func(context *kernel.Context, msg interface{}) {
		state := (*monitor)(context.State)
		switch m := msg.(type) {
		case *kernel.PidExit:
			id := m.Pid.GetID()
			if s, ok := state.id2n[id]; ok {
				delete(state.id2n, id)
				for sv, _ := range s {
					if mm, ok := state.n2pid[sv]; ok {
						delete(mm, id)
					}
				}
			}
		case *monitorNode:
			id := m.pid.GetID()
			if sl, ok := state.id2n[id]; ok {
				sl[m.node] = true
			} else {
				sl := make(map[string]bool)
				state.id2n[id] = sl
				sl[m.node] = true
			}
			if mm, ok := state.n2pid[m.node]; ok {
				mm[id] = m.pid
			} else {
				mm := make(map[int64]*kernel.Pid)
				state.n2pid[m.node] = mm
				mm[id] = m.pid
			}
		case *NodeOP:
			if mm, ok := state.n2pid[m.Name]; ok {
				for _, pid := range mm {
					kernel.Cast(pid,m)
				}
			}
		case *deMonitorNode:
			if mm, ok := state.n2pid[m.node]; ok {
				id := m.pid.GetID()
				if _,ok:= mm[id];ok{
					delete(mm,id)
					if sl,ok := state.id2n[id];ok{
						delete(sl,m.node)
						if len(sl) == 0 {
							delete(state.id2n,id)
						}
					}
				}
			}
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
