package node

import (
	"github.com/liangmanlin/gootp/kernel"
)

type empty struct {}

var monitorPid *kernel.Pid

func Monitor(nodeName string,pid *kernel.Pid) {
	kernel.Cast(monitorPid,&monitorNode{pid: pid,node: nodeName})
}

func DeMonitor(nodeName string,pid *kernel.Pid)  {
	kernel.Cast(monitorPid,&deMonitorNode{pid: pid,node: nodeName})
}

type monitor struct {
	node2pid map[string]map[int64]*kernel.Pid
	id2node  map[int64]map[string]empty
}

var monitorActor = &kernel.Actor{
	Init: func(context *kernel.Context,pid *kernel.Pid, args ...interface{}) interface{} {
		monitorPid = pid
		state := monitor{
			node2pid: make(map[string]map[int64]*kernel.Pid),
			id2node:  make(map[int64]map[string]empty),
		}
		kernel.ErrorLog("NodeMonitor %s started",pid)
		return &state
	},
	HandleCast: func(context *kernel.Context, msg interface{}) {
		state := context.State.(*monitor)
		switch m := msg.(type) {
		case *kernel.PidExit:
			id := m.Pid.GetID()
			if s, ok := state.id2node[id]; ok {
				delete(state.id2node, id)
				for sv, _ := range s {
					if mm, ok := state.node2pid[sv]; ok {
						delete(mm, id)
					}
				}
			}
		case *monitorNode:
			id := m.pid.GetID()
			if sl, ok := state.id2node[id]; ok {
				sl[m.node] = empty{}
			} else {
				sl := make(map[string]empty)
				state.id2node[id] = sl
				sl[m.node] = empty{}
			}
			if mm, ok := state.node2pid[m.node]; ok {
				mm[id] = m.pid
			} else {
				mm := make(map[int64]*kernel.Pid)
				state.node2pid[m.node] = mm
				mm[id] = m.pid
			}
		case *NodeOP:
			if mm, ok := state.node2pid[m.Name]; ok {
				for _, pid := range mm {
					kernel.Cast(pid,m)
				}
			}
		case *deMonitorNode:
			if mm, ok := state.node2pid[m.node]; ok {
				id := m.pid.GetID()
				if _,ok:= mm[id];ok{
					delete(mm,id)
					if sl,ok := state.id2node[id];ok{
						delete(sl,m.node)
						if len(sl) == 0 {
							delete(state.id2node,id)
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
