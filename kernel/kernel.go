package kernel

import (
	"github.com/liangmanlin/gootp/args"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
)

const (
	ExitReasonNormal = "normal"
	ExitReasonError  = "error"
)

var mainCh = make(chan int, 1)

var kernelPid = make(map[int64]*Pid)

var kernelAliveMap sync.Map

func KernelStart(start func(), stop func()) {
	StartNameOpt(initServerName, initActor,ActorOpt(ActorChanCacheSize(10000)))
	kernelChild := []*SupChild{
		{Name: selfServerName, Svr: selfSenderActor, ReStart: true,InitOpt: ActorOpt(ActorChanCacheSize(10000))},
		{Name: "application", Svr: appSvr, ReStart: true},
	}
	kernel := SupStart("kernel", kernelChild...)
	addToKernelMap(kernel)
	initTimer()
	startLogger()
	if stop != nil {
		Call(initServerPid, stopFunc(stop))
	}
	ErrorLog("kernel Start OTP complete")
	start()
	// block the main goroutine
	<-mainCh
	ErrorLog("kernel stopped")
}

func kernelStop() {
	mainCh <- 1
}

func addToKernelMap(pid *Pid) {
	kernelPid[pid.id] = pid
}

func addAliveMap(pid *Pid) () {
	kernelAliveMap.Store(pid.id, pid)
}

func removeAliveMap(pid *Pid) {
	kernelAliveMap.Delete(pid.id)
}

// return args in slice
func MakeArgs(args ...interface{}) []interface{} {
	return args
}

func CatchFun(f func()) {
	defer func() {
		p := recover()
		if p != nil {
			ErrorLog("catch error:%s,Stack:%s", p, debug.Stack())
		}
	}()
	f()
}

func init() {
	// 支持命令行参数
	args.FillEvn(Env)
	if v, ok := args.GetInt("timer_min_tick"); ok {
		Env.SetTimerMinTick(int64(v))
	}
	if v, ok := args.GetInt("log_level"); ok {
		logLevel = LogLevel(v)
	}
	DebugLog("env :%#v",Env)
}

var mainRoot string

// 获取二进制执行文件所在目录
func GetMainRoot() string {
	if mainRoot != "" {
		return mainRoot
	}
	m := os.Args[0]
	// 获取main的目录
	mainRoot = filepath.Dir(m)
	return mainRoot
}

// 纯粹方便测试用，不用每次编写一堆相同代码
func DefaultActor() *Actor {
	actor := &Actor{
		Init: func(context *Context, pid *Pid, args ...interface{}) interface{} {
			return nil
		},
		HandleCast: func(context *Context, msg interface{}) {
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
	return actor
}

func Link(srcPid, destPid *Pid) {
	Cast(destPid, &actorOP{op: &link{pid: srcPid}})
}

// 减少代码规模，但是不应该用在需要高性能的地方
func NewActor(funList ...interface{}) *Actor {
	actor := DefaultActor()
	for _, f := range funList {
		switch fun := f.(type) {
		case InitFunc:
			actor.Init = fun
		case HandleCastFunc:
			actor.HandleCast = fun
		case HandleCallFunc:
			actor.HandleCall = fun
		case TerminateFunc:
			actor.Terminate = fun
		case ErrorHandleFunc:
			actor.ErrorHandler = fun
		}
	}
	return actor
}
