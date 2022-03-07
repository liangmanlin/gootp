package kernel

import (
	"fmt"
	"github.com/liangmanlin/gootp/kernel/kct"
)

type op uint

const (
	opWhichChild op = 1 << iota
)

const (
	SupChildTypeWorker = iota
	SupChildTypeSup
)

type SupChild struct {
	ChildType int
	ReStart   bool
	Name      string
	Svr       *Actor
	InitOpt   []interface{}
	InitArgs  []interface{}
}

type SupChildInfo struct {
	*SupChild
	Pid *Pid
}

func SupStartChild(sup interface{}, child *SupChild) (interface{}, *Pid) {
	switch s := sup.(type) {
	case string:
		name := sup.(string)
		pid := WhereIs(name)
		if pid != nil {
			return startChild(pid, child)
		}
		return fmt.Sprintf("sup Name:%s not founded\n", name), nil
	case *Pid:
		return startChild(s, child)
	}
	return fmt.Errorf("unknow arg:%#v", sup), nil
}

func SupStart(name string, initChild ...*SupChild) *Pid {
	pid, _ := StartName(name, supervisorActor, name, initChild)
	return pid
}

func SupStop(sup interface{}) {
	switch s := sup.(type) {
	case string:
		name := sup.(string)
		pid := WhereIs(name)
		if pid != nil {
			_, _ = CallTimeOut(pid, stop(ExitReasonNormal), 86400)
		}
	case *Pid:
		_, _ = CallTimeOut(s, stop(ExitReasonNormal), 86400)
	}
}

func SupWhichChild(sup interface{}) []*SupChildInfo {
	var pid *Pid
	switch t := sup.(type) {
	case string:
		pid = WhereIs(t)
	case *Pid:
		pid = t
	}
	if pid == nil {
		return nil
	}
	_, rs := Call(pid, opWhichChild)
	if t, ok := rs.([]*SupChildInfo); ok {
		return t
	}
	return nil
}

func startChild(supPid *Pid, child *SupChild) (interface{}, *Pid) {
	ok, rs := CallTimeOut(supPid, child, 86400)
	if ok {
		rs2 := rs.(*initResult)
		if rs2.ok {
			return nil, rs2.err.(*Pid)
		}
		return rs2.err, nil
	}
	return rs, nil
}

var supervisorActor *Actor

func init() {
	supervisorActor = &Actor{
		Init:       supInit,
		HandleCast: supHandleCast,
		HandleCall: supHandleCall,
		Terminate: func(context *Context, reason *Terminate) {
			context.State.(*supervisor).stopAllChild(context)
		},
		ErrorHandler: func(context *Context, err interface{}) bool {
			return true
		},
	}
}

func supInit(context *Context, pid *Pid, args ...interface{}) interface{} {
	s := supervisor{}
	s.name = args[0].(string)
	ErrorLog("%s sup %s started", s.name, pid)
	s.initChild = args[1].([]*SupChild)
	s.child = kct.NewBMap()
	s.cache = make(map[int64]*SupChildInfo, 1000)
	for _, child := range s.initChild {
		opt := buildOpt(child, pid)
		childPid, _ := StartOpt(child.Svr, opt, child.InitArgs...)
		s.addChild(&SupChildInfo{child, childPid})
	}
	return &s
}

func supHandleCast(context *Context, msg interface{}) {
	s := context.State.(*supervisor)
	switch m := msg.(type) {
	case *PidExit:
		if m.Reason.Reason != ExitReasonNormal {
			child := s.cache[m.Pid.id].SupChild
			ErrorLog("sup [%s] child %s%s exit,Reason:%s", s.name, child.Name, m.Pid, m.Reason.Reason)
			if child.ReStart {
				// restart
				ErrorLog("%s restart %#v", s.name, child)
				s.startChild(context, child)
			}
		}
		s.childExit(m.Pid)
	}
}

func supHandleCall(context *Context, request interface{}) interface{} {
	s := context.State.(*supervisor)
	switch r := request.(type) {
	case *SupChild: //StartOpt a new service
		return s.startChild(context, r)
	case op:
		return s.callOP(r)
	case stop:
		ErrorLog("%s stop", context.name)
		reason := string(r)
		s.stopAllChild(context)
		context.Exit(reason)
		return nil
	}
	return fmt.Errorf("no case match")
}

type supervisor struct {
	name      string
	initChild []*SupChild
	child     *kct.BMap
	cache     map[int64]*SupChildInfo
}

func (s *supervisor) startChild(context *Context, child *SupChild) interface{} {
	var pid *Pid
	var err interface{}
	if child.ChildType == SupChildTypeWorker {
		if child.Name != "" {
			pid, err = context.StartLinkOpt(child.Svr, append(child.InitOpt, regName(child.Name)), child.InitArgs...)
		} else {
			pid, err = context.StartLinkOpt(child.Svr, child.InitOpt, child.InitArgs...)
		}
	} else if child.ChildType == SupChildTypeSup {
		var sc []*SupChild
		for _, c := range child.InitArgs {
			sc = append(sc, c.(*SupChild))
		}
		pid = SupStart(child.Name, sc...)
	}
	s.addChild(&SupChildInfo{child, pid})
	if err != nil {
		return &initResult{ok: false, err: err}
	}
	return &initResult{ok: true, err: pid}
}

func (s *supervisor) addChild(child *SupChildInfo) {
	id := child.Pid.id
	s.child.Insert(id, id)
	s.cache[id] = child
}
func (s *supervisor) childExit(pid *Pid) {
	id := pid.id
	s.child.Delete(id)
	delete(s.cache, id)
}

func (s *supervisor) stopAllChild(context *Context) {
	f := func(e interface{}) {
		child := s.cache[e.(int64)].Pid
		if child.IsAlive() {
			callID := makeCallID()
			iStop := &initStop{callID: callID, recCh: context.self.callResult}
			Cast(child, &actorOP{iStop})
			context.recResult(callID, context.self.callResult, 100)
		}
	}
	s.child.Foreach(f)
}

func (s *supervisor) callOP(op op) interface{} {
	switch op {
	case opWhichChild:
		var rs []*SupChildInfo
		f := func(e interface{}) {
			child := s.cache[e.(int64)]
			rs = append(rs, child)
		}
		s.child.Foreach(f)
		return rs
	}
	return nil
}

func buildOpt(child *SupChild, father *Pid) []interface{} {
	opt := append(child.InitOpt, &link{pid: father})
	if child.Name != "" {
		opt = append(opt, regName(child.Name))
	}
	return opt
}
