package kernel

import (
	"github.com/liangmanlin/routine"
	"runtime/debug"
	"sync/atomic"
	"time"
	"unsafe"
)

type initResult struct {
	ok  bool
	err interface{}
}

var actorID int64 = 0
var callIndex int64 = 1

func makePid() int64 {
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

func Self() *Pid {
	ctx := routine.Get[Context]()
	if ctx != nil {
		return ctx.self
	}
	return nil
}

func Ctx() *Context {
	ctx := routine.Get[Context]()
	if ctx != nil {
		return ctx
	}
	return nil
}

func ActorOpt(opt ...interface{}) []interface{} {
	return opt
}

func Start(newActor *Actor, args ...interface{}) (*Pid, interface{}) {
	return StartOpt(newActor, nil, args...)
}

func StartName(name string, newActor *Actor, args ...interface{}) (*Pid, interface{}) {
	opt := []interface{}{regName(name)}
	return StartOpt(newActor, opt, args...)
}

func StartNameOpt(name string, newActor *Actor, opt []interface{}, args ...interface{}) (*Pid, interface{}) {
	opt = append(opt, regName(name))
	return StartOpt(newActor, opt, args...)
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
	defer func() { recover() }()
	if p, ok := GetNodeNetWork(dn); ok {
		m := &NodeMsgName{Dest: name, Msg: msg}
		p.c <- m
	}
}

func CastName(name string, msg interface{}) {
	if pid := WhereIs(name); pid != nil {
		Cast(pid, msg)
	}
}

func Cast(pid *Pid, msg interface{}) {
	// 浪费一点性能，使得发送不会因为对端退出而阻塞，或者panic
	defer func() { recover() }()
	if pid.node != nil {
		if p, ok := GetNodeNetWork(pid.node); ok {
			m := &NodeMsg{Dest: pid, Msg: msg}
			p.c <- m
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
	return false, &CallError{ErrType: CallErrorTypeNoProc}
}

func CallNameNode(name string, node interface{}, request interface{}) (bool, interface{}) {
	c := make(chan interface{})
	f := recResult
	if ctx := Ctx(); ctx != nil {
		f = ctx.recResult
	}
	return callNameNode(name, node, request, c, true, f)
}

func CallTimeOut(pid *Pid, request interface{}, timeOut time.Duration) (bool, interface{}) {
	c := make(chan interface{})
	f := recResult
	if ctx := Ctx(); ctx != nil {
		f = ctx.recResult
	}
	return callTimeOut(pid, request, timeOut, c, true, f)
}

func callTimeOut(pid *Pid, request interface{}, timeOut time.Duration, rc chan interface{}, closeRC bool,
	recvFun func(int64, chan interface{}, time.Duration) (bool, interface{})) (bool, interface{}) {
	// 浪费一点性能，使得发送不会因为对端退出而阻塞，或者panic
	defer func() { recover() }()
	if closeRC {
		defer close(rc)
	}
	callID := makeCallID()
	if pid.node != nil {
		// 其他节点，需要构造额外信息
		if p, ok := GetNodeNetWork(pid.node); ok {
			ci := &NodeCall{Dest: pid, Req: request, CallID: callID, Ch: rc}
			p.c <- ci
			ok, result := recvFun(callID, rc, timeOut)
			return ok, result
		}
		return false, &CallError{ErrType: CallErrorTypeNodeNotConnect}
	} else {
		ci := &CallInfo{RecCh: rc, CallID: callID, Request: request}
		pid.c <- ci
		ok, result := recvFun(callID, rc, timeOut)
		return ok, result
	}
}

func callNameNode(name string, node interface{}, request interface{}, rc chan interface{}, closeRC bool,
	recvFun func(int64, chan interface{}, time.Duration) (bool, interface{})) (bool, interface{}) {
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
		if pid := WhereIs(name); pid != nil {
			return callTimeOut(pid, request, 5, rc, closeRC, recvFun)
		}
		return false, &CallError{ErrType: CallErrorTypeNoProc}
	}
	defer func() { recover() }()
	if closeRC {
		defer close(rc)
	}
	if p, ok := GetNodeNetWork(dn); ok {
		callID := makeCallID()
		ci := &NodeCallName{Dest: name, Req: request, CallID: callID, Ch: rc}
		p.c <- ci
		ok, result := recvFun(callID, rc, 5)
		return ok, result
	}
	return false, &CallError{ErrType: CallErrorTypeNodeNotConnect}
}

func StartOpt(actor *Actor, opt []interface{}, args ...interface{}) (*Pid, interface{}) {
	c := make(chan interface{}, getCacheSize(opt))
	id := makePid()
	pid := &Pid{0, id, c, make(chan interface{}, 1), nil}
	ok, err := startGO(pid, actor, opt, args...)
	if ok {
		return pid, nil
	}
	close(pid.c)
	close(pid.callResult)
	return nil, err
}

func startGO(pid *Pid, actor *Actor, opt []interface{}, args ...interface{}) (ok bool, err interface{}) {
	context := &Context{self: pid, actor: actor, callMode: call_mode_normal}
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
		context.Link(initRegister(pid))
	}
	context.State = actor.Init(context, pid, args...)
	addAliveMap(pid)
	atomic.AddInt32(&pid.isAlive, 1) // 用于判断进程存活，可以快速判断，不需要全局锁
	ok = true
	go loop(pid, context)
	// 修改为在同一个进程中执行初始化逻辑，减少启动进程的栈浪费，并且defer不会执行
	return
}

func getCacheSize(opt []interface{}) int {
	for _, op := range opt {
		switch o := op.(type) {
		case ActorChanCacheSize:
			return int(o)
		}
	}
	return Env.ActorChanCacheSize
}

func loop(pid *Pid, context *Context) {
	routine.Set(unsafe.Pointer(context))
	var iStop *initStop = nil
	defer exitFinal(context, &iStop)
	for {
		recMsg(pid, context, &iStop)
		if iStop != nil {
			break
		}
	}
}

func recMsg(pid *Pid, ctx *Context, stop **initStop) {
	defer func() {
		if err := recover(); err != nil {
			ErrorLog("catch error Reason: %s,Stack: %s", err, debug.Stack())
			if ctx.actor.ErrorHandler == nil || !ctx.actor.ErrorHandler(ctx, err) {
				ctx.terminateReason = &Terminate{Reason: "error"}
				*stop = &initStop{reply: false}
			}
		}
	}()
	var msg interface{}
	for {
		if msg = ctx.msgQ.Pop(); msg != nil {
		} else {
		rec:
			select {
			case msg = <-pid.c:
			case msg = <-pid.callResult: // 损失一些性能，防止call通道阻塞，导致对端阻塞
				ErrorLog("un handle callResult Result:%#v", msg)
				goto rec
			}
		}
		switch m := msg.(type) {
		case *CallInfo:
			ctx.handleCall(m)
		case *actorOP:
			code, reason := ctx.handleOP(m.op)
			switch code {
			case actorCodeExit:
				ctx.terminateReason = reason
				*stop = &initStop{reply: false}
				return
			case actorCodeInitStop:
				ctx.terminateReason = reason
				t := m.op.(*initStop)
				t.reply = true
				*stop = t
				return
			default:
			}
		default:
			ctx.actor.HandleCast(ctx, msg)
		}
	}
}

func exitFinal(context *Context, stop **initStop) {
	defer func() {
		err := recover()
		if err != nil {
			ErrorLog("catch error Reason: %s,Stack: %s", err, debug.Stack())
		}
		close(context.self.callResult) //之所以要关闭，是为了防止对端无辜阻塞
		close(context.self.c)
	}()
	removeAliveMap(context.self)
	context.self.SetDie()
	var reason *Terminate
	if context.terminateReason != nil {
		reason = context.terminateReason
	}
	p := recover()
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
	if context.actor.Terminate != nil {
		CatchFun(func() { context.actor.Terminate(context, reason) })
	}
	if context.name != "" {
		UnRegister(context.name)
	}
	if *stop != nil && (*stop).reply {
		Reply((*stop).recCh, (*stop).callID, true)
	}
}

func recResult(callID int64, c chan interface{}, timeOut time.Duration) (bool, interface{}) {
	t := time.NewTimer(timeOut * time.Second)
rec:
	select {
	case result := <-c:
		r := result.(*CallResult)
		if r.ID == callID {
			t.Stop()
			return true, r.Result
		}
		ErrorLog("not match callResult ID,%d,%d", r.ID, callID)
		goto rec
	case <-t.C:
		ErrorLog("rec callResult timeout")
		return false, &CallError{CallErrorTypeTimeOut, nil}
	}
}

func makeCallID() int64 {
	return atomic.AddInt64(&callIndex, 1)
}

func Reply(recCh chan interface{}, callID int64, result interface{}) {
	defer func() { recover() }() // 理论上可以预见问题
	r := &CallResult{callID, result}
	recCh <- r
}
