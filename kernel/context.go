package kernel

import (
	"time"
	"unsafe"
)

type Context struct {
	self            *Pid
	actor           *Actor
	name            string
	links           []*Pid
	terminateReason *Terminate
	msgQ            *msgQueue
	State           unsafe.Pointer // 之所以没有用interface，是因为可以少一次类型转换
}

func (c *Context) Self() *Pid {
	return c.self
}

// 如果自身注册了名字，返回
func (c *Context)Name() string {
	return c.name
}

func (c *Context) CastName(name string, msg interface{}) {
	if pid := WhereIs(name); pid != nil {
		c.Cast(pid, msg)
	}
}

func (c *Context) CastNameNode(name string, node interface{}, msg interface{}) {
	var dn *Node
	switch n := node.(type) {
	case string:
		dn = GetNode(n)
	case *Node:
		dn = n
	default:
		ErrorLog("badarg:%#v", node)
		return
	}
	if dn.Equal(SelfNode()) {
		CastName(name, msg)
		return
	}
	defer CatchNoPrint()
	if p, ok := nodeNetWork.Load(dn.id); ok {
		m := &NodeMsgName{Dest: name, Msg: msg}
		p.(*Pid).c <- m
	}
}

func (c *Context) Cast(pid *Pid, msg interface{}) {
	defer CatchNoPrint()
	if pid.node != nil {
		if p, ok := nodeNetWork.Load(pid.node.id); ok {
			m := &NodeMsg{Dest: pid, Msg: msg}
			p.(*Pid).c <- m
		}
		return
	}
	pid.c <- msg
}

func (c *Context) CallName(name string, request interface{}) (bool, interface{}) {
	if pid := WhereIs(name); pid != nil {
		return c.Call(pid, request)
	}
	return false, nil
}

func (c *Context) Call(pid *Pid, request interface{}) (bool, interface{}) {
	defer CatchNoPrint()
	if pid == nil {
		return false, nil
	}
	callID := makeCallID()
	if pid.node != nil {
		// 其他节点，需要构造额外信息
		if p, ok := nodeNetWork.Load(pid.node.id); ok {
			ci := &NodeCall{Dest: pid, Req: request, CallID: callID, Ch: c.self.call}
			p.(*Pid).c <- ci
			ok, result := recResult(callID, c.self.call, 5)
			return ok, result
		}
		return false, nil
	}
	ci := &CallInfo{RecCh: c.self.call, CallID: callID, Request: request}
	pid.c <- ci
	return c.recResult(callID, c.self.call, 5)
}

func (c *Context) CallNameNode(name string, node interface{}, request interface{}) (bool, interface{}) {
	var dn *Node
	switch n := node.(type) {
	case string:
		dn = GetNode(n)
	case *Node:
		dn = n
	default:
		ErrorLog("badarg:%#v", node)
		return false, nil
	}
	if dn.Equal(SelfNode()) {
		return c.CallName(name, request)
	}
	defer CatchNoPrint()
	if p, ok := nodeNetWork.Load(dn.id); ok {
		callID := makeCallID()
		ci := &NodeCallName{Dest: name, Req: request, CallID: callID, Ch: c.self.call}
		p.(*Pid).c <- ci
		ok, result := recResult(callID, c.self.call, 5)
		return ok, result
	}
	return false, nil
}

func (c *Context) StartLink(newActor *Actor, args ...interface{}) (*Pid, interface{}) {
	pid, err := start(newActor, []interface{}{&link{pid: c.self}}, args...)
	return pid, err
}
func (c *Context) StartNameLink(name string, newActor *Actor, args ...interface{}) (*Pid, interface{}) {
	opt := []interface{}{&link{pid: c.self}, regName(name)}
	pid, err := start(newActor, opt, args...)
	return pid, err
}

func (c *Context) Exit(reason string) {
	c.CastSelf(&actorOP{&Terminate{Reason: reason}})
}

func (c *Context) Link(pid *Pid) {
	c.links = append(c.links, pid)
}

func (c *Context) handleOP(op interface{}) (int, *Terminate) {
	switch m := op.(type) {
	case regName:
		c.name = string(m)
	case *link:
		c.links = append(c.links, m.pid)
	case *Terminate:
		return actorCodeExit, m
	case *initStop:
		return actorCodeInitStop, &Terminate{Reason: ExitReasonNormal}
	}
	return actorCodeNone, nil
}

func (c *Context) handleCall(ci *CallInfo) {
	result := c.actor.HandleCall(c, ci.Request)
	reply(ci.RecCh, ci.CallID, result)
}

func (c *Context) parseOP(opt []interface{}) {
	if len(opt) > 0 {
		for _, op := range opt {
			switch o := op.(type) {
			case *link:
				c.links = append(c.links, o.pid)
			case regName:
				c.name = string(o)
				registerNotExist(c.name, c.self)
			}
		}
	}
}

// 这个函数是对应parseOP的，用来回滚一些处理
func (c *Context) initExit(opt []interface{}) {
	if len(opt) > 0 {
		for _, op := range opt {
			switch o := op.(type) {
			case *link:
				c.links = nil
			case regName:
				c.name = string(o)
				UnRegister(c.name)
				c.name = ""
			}
		}
	}
	msg := &PidExit{Pid: c.self, Reason: &Terminate{Reason: ExitReasonError}}
	Cast(initServerPid, msg)
}

// 由于chan会阻塞，为了消除在call的时候，对方阻塞，或者是，对方正在发送消息给自己导致阻塞
func (c *Context) recResult(callID int64, rec chan interface{}, timeOut time.Duration) (bool, interface{}) {
	t := time.After(timeOut * time.Second)
rec:
	select {
	case result := <-rec:
		r := result.(*CallResult)
		if r.ID == callID {
			return true, r.Result
		}
		ErrorLog("not match call ID,%d,%d", r.ID, callID)
		goto rec
	case msg := <-c.self.c: // 这里修改为：如果是一个Actor进程，那么在call的时候，也会接收消息，放在队列里面
		c.recMsg(msg)
		goto rec
	case <-t:
		ErrorLog("rec call timeout")
		return false, &CallError{CallErrorTypeTimeOut, nil}
	}
}

func (c *Context) recMsg(msg interface{}) {
	m := &msgQueue{next: nil, msg: msg}
	if c.msgQ == nil {
		c.msgQ = m
		return
	}
	q := c.msgQ
	for q.next != nil {
		q = q.next
	}
	q.next = m
}
