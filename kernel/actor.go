package kernel

import (
	"runtime/debug"
	"sync/atomic"
	"time"
)

type initResult struct {
	ok  bool
	err interface{}
}

var actorID int64 = 0
var callIndex int64 = 1

func makeID() int64 {
start:
	id := atomic.AddInt64(&actorID, 1)
	// 损失一点点性能，判断重复
	if _, ok := kernelAliveMap.Load(id); ok {
		goto start
	}
	return id
}

type CallInfo struct {
	RecCh   chan interface{}
	CallID  int64
	Request interface{}
}

type CallResult struct {
	ID     int64
	Result interface{}
}

func Start(newActor *Actor, args ...interface{}) (*Pid, interface{}) {
	return start(newActor, nil, args...)
}

func StartName(name string, newActor *Actor, args ...interface{}) (*Pid, interface{}) {
	opt := []interface{}{regName(name)}
	return start(newActor, opt, args...)
}

func CastNameNode(name string, node interface{}, msg interface{}) {
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

func CastName(name string, msg interface{}) {
	if pid := WhereIs(name); pid != nil {
		Cast(pid, msg)
	}
}

func Cast(pid *Pid, msg interface{}) {
	// 浪费一点性能，使得发送不会因为对端退出而阻塞，或者panic
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

func Call(pid *Pid, request interface{}) (bool, interface{}) {
	return CallTimeOut(pid, request, 5)
}

func CallName(name string, request interface{}) (bool, interface{}) {
	if pid := WhereIs(name); pid != nil {
		return CallTimeOut(pid, request, 5)
	}
	return false, nil
}

func CallNameNode(name string, node interface{}, request interface{}) (bool, interface{}) {
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
		return CallName(name, request)
	}
	defer CatchNoPrint()

	if p, ok := nodeNetWork.Load(dn.id); ok {
		c := make(chan interface{})
		defer close(c)
		callID := makeCallID()
		ci := &NodeCallName{Dest: name, Req: request, CallID: callID, Ch: c}
		p.(*Pid).c <- ci
		ok, result := recResult(callID, c, 5)
		return ok, result
	}
	return false, nil
}

func CallTimeOut(pid *Pid, request interface{}, timeOut time.Duration) (bool, interface{}) {
	// 浪费一点性能，是的发送不会因为对端退出而阻塞，或者panic
	defer CatchNoPrint()
	c := make(chan interface{})
	defer close(c)
	callID := makeCallID()
	if pid.node != nil {
		// 其他节点，需要构造额外信息
		if p, ok := nodeNetWork.Load(pid.node.id); ok {
			ci := &NodeCall{Dest: pid, Req: request, CallID: callID, Ch: c}
			p.(*Pid).c <- ci
			ok, result := recResult(callID, c, timeOut)
			return ok, result
		}
		return false, nil
	} else {
		ci := &CallInfo{RecCh: c, CallID: callID, Request: request}
		pid.c <- ci
		ok, result := recResult(callID, c, timeOut)
		return ok, result
	}
}

func start(actor *Actor, opt []interface{}, args ...interface{}) (*Pid, interface{}) {
	c := make(chan interface{}, Env.ActorChanCacheSize)
	id := makeID()
	pid := &Pid{id, c, make(chan interface{}, 1), nil, 0}
	ok, err := startGO(pid, actor, opt, args...)
	if ok {
		return pid, nil
	}
	close(pid.c)
	return nil, err
}

func startGO(pid *Pid, actor *Actor, opt []interface{}, args ...interface{}) (ok bool, err interface{}) {
	context := &Context{self: pid, actor: actor}
	defer func() {
		if !ok {
			err = recover()
			ErrorLog("catch error:%s,Stack:%s", err, debug.Stack())
			context.initExit(opt)
		}
	}()
	context.parseOP(opt) // 在init之前执行，仅仅是为了注册名字
	// 向init注册启动
	if initServerPid != nil {
		context.links = append(context.links, initRegister(pid))
	}
	context.State = actor.Init(context,pid, args...)
	addAliveMap(pid)
	pid.isAlive = 1 // 用于判断进程存活，可以快速判断，不需要全局锁
	ok = true
	go loop(pid, context)
	// 修改为在同一个进程中执行初始化逻辑，减少启动进程的栈浪费，并且defer不会执行
	return
}

func loop(pid *Pid, context *Context) {
	var iStop *initStop = nil
	defer exitFinal(context, &iStop)
	for {
		recMsg(pid, context, &iStop)
		if iStop != nil {
			goto exit
		}
	}
exit:
}

func recMsg(pid *Pid, context *Context, stop **initStop) {
	defer actorCatch(context, stop)
	var msg interface{}
	for {
		if context.msgQ != nil {
			msg = context.msgQ.msg
			context.msgQ = context.msgQ.next
		} else {
		rec:
			select {
			case msg = <-pid.c:
			case msg = <-pid.call: // 损失一些性能，防止call通道阻塞，导致对端阻塞
				ErrorLog("un handle call Result:%#v", msg)
				goto rec
			}
		}
		switch m := msg.(type) {
		case *CallInfo:
			context.handleCall(m)
		case *actorOP:
			code, reason := context.handleOP(m.op)
			switch code {
			case actorCodeExit:
				context.terminateReason = reason
				*stop = &initStop{flag: false}
				goto exit
			case actorCodeInitStop:
				context.terminateReason = reason
				t := m.op.(*initStop)
				t.flag = true
				*stop = t
				goto exit
			default:
			}
		default:
			context.actor.HandleCast(context, msg)
		}
	}
exit:
}

func actorCatch(context *Context, stop **initStop) {
	if err := recover(); err != nil {
		ErrorLog("catch error Reason: %s,Stack: %s", err, debug.Stack())
		if !context.actor.ErrorHandler(context, err) {
			context.terminateReason = &Terminate{Reason: "error"}
			*stop = &initStop{flag: false}
		}
	}
}

func exitFinal(context *Context, stop **initStop) {
	defer func() {
		err := recover()
		if err != nil {
			ErrorLog("catch error Reason: %s,Stack: %s", err, debug.Stack())
		}
		close(context.self.call) //之所以要关闭，是为了防止对端无辜阻塞
		close(context.self.c)
	}()
	removeAliveMap(context.self)
	context.self.exit()
	p := recover()
	var reason *Terminate
	if context.terminateReason != nil {
		reason = context.terminateReason
	}
	if p != nil {
		ErrorLog("actor exit,Reason:%s,Stack:%s", p, debug.Stack())
		reason = &Terminate{Reason: ExitReasonError}
	}
	if len(context.links) > 0 {
		msg := &PidExit{Pid: context.self, Reason: reason}
		for _, pid := range context.links {
			Cast(pid, msg)
		}
	}
	CatchFun(func() { context.actor.Terminate(context, reason) })
	if context.name != "" {
		UnRegister(context.name)
	}
	if *stop != nil && (*stop).flag {
		reply((*stop).recCh, (*stop).callID, true)
	}
}

func recResult(callID int64, c chan interface{}, timeOut time.Duration) (bool, interface{}) {
	t := time.After(timeOut * time.Second)
rec:
	select {
	case result := <-c:
		r := result.(*CallResult)
		if r.ID == callID {
			return true, r.Result
		}
		ErrorLog("not match call ID,%d,%d", r.ID, callID)
		goto rec
	case <-t:
		ErrorLog("rec call timeout")
		return false, &CallError{1, nil}
	}
}

func makeCallID() int64 {
	return atomic.AddInt64(&callIndex, 1)
}

func reply(recCh chan interface{}, callID int64, result interface{}) {
	defer CatchNoPrint() // 理论上可以预见问题
	r := &CallResult{callID, result}
	recCh <- r
}
