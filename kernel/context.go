package kernel

import (
	"github.com/liangmanlin/gootp/ringbuffer"
	"time"
)

type Context struct {
	self            *Pid
	actor           *Actor
	name            string
	links           []*Pid
	terminateReason *Terminate
	msgQ            *ringbuffer.SingleRingBuffer
	callMode        callMode
	State           interface{}
}

func (c *Context) Self() *Pid {
	return c.self
}

// 如果自身注册了名字，返回
func (c *Context) Name() string {
	return c.name
}

func (c *Context) CastName(name string, msg interface{}) {
	CastName(name, msg)
}

func (c *Context) CastNameNode(name string, node interface{}, msg interface{}) {
	CastNameNode(name, node, msg)
}

func (c *Context) Cast(pid *Pid, msg interface{}) {
	Cast(pid, msg)
}

func (c *Context) CallName(name string, request interface{}) (bool, interface{}) {
	if pid := WhereIs(name); pid != nil {
		return c.Call(pid, request)
	}
	return false, nil
}

func (c *Context) Call(pid *Pid, request interface{}) (bool, interface{}) {
	return callTimeOut(pid, request, 5, c.self.callResult, false, c.recResult)
}

func (c *Context) CallNameNode(name string, node interface{}, request interface{}) (bool, interface{}) {
	return callNameNode(name, node, request, c.self.callResult, false, c.recResult)
}

func (c *Context) StartLink(newActor *Actor, args ...interface{}) (*Pid, interface{}) {
	pid, err := c.StartLinkOpt(newActor, nil, args...)
	return pid, err
}
func (c *Context) StartNameLink(name string, newActor *Actor, args ...interface{}) (*Pid, interface{}) {
	opt := []interface{}{regName(name)}
	pid, err := c.StartLinkOpt(newActor, opt, args...)
	return pid, err
}

func (c *Context) StartLinkOpt(newActor *Actor, opt []interface{}, args ...interface{}) (*Pid, interface{}) {
	opt = append(opt, &link{pid: c.self})
	pid, err := StartOpt(newActor, opt, args...)
	return pid, err
}

func (c *Context) Exit(reason string) {
	m := &actorOP{&Terminate{Reason: reason}}
	c.CastSelf(m)
}

func (c *Context) Link(pid *Pid) {
	c.links = append(c.links, pid)
}

func (c *Context) CastSelf(msg interface{}) {
	if len(c.self.c) == 0 {
		c.recMsg(msg)
	} else {
		Cast(selfSenderPid, &routerMsg{to: c.Self(), msg: msg})
	}
}

func (c *Context)ChangeCallMode()  {
	c.callMode = call_mode_no_reply
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
	if c.callMode == call_mode_normal {
		result := c.actor.HandleCall(c, ci.Request)
		Reply(ci.RecCh, ci.CallID, result)
	}else{
		c.actor.HandleCall(c, ci)
	}
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
	t := time.NewTimer(timeOut * time.Second)
rec:
	select {
	case result := <-rec:
		r := result.(*CallResult)
		if r.ID == callID {
			t.Stop()
			return true, r.Result
		}
		ErrorLog("not match callResult ID,%d,%d", r.ID, callID)
		goto rec
	case msg := <-c.self.c: // 这里修改为：如果是一个Actor进程，那么在call的时候，也会接收消息，放在队列里面
		c.recMsg(msg)
		goto rec
	case <-t.C:
		ErrorLog("rec callResult timeout")
		return false, &CallError{CallErrorTypeTimeOut, nil}
	}
}

func (c *Context) recMsg(msg interface{}) {
	if c.msgQ == nil {
		c.msgQ = ringbuffer.NewSingleRingBuffer(4, 32)
	}
	c.msgQ.Put(msg)
}
