package kernel

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"unsafe"
)

const (
	ExitReasonNormal = "normal"
	ExitReasonError  = "error"
)

var mainCh = make(chan int, 1)

var kernelPid = make(map[int64]*Pid)

var kernelAliveMap sync.Map

func KernelStart(start func(), stop func()) {
	StartName(initServerName, initActor)
	kernelChild := []*SupChild{
		{Name: selfServerName, Svr: selfSenderActor},
	}
	kernel := SupStart("kernel", kernelChild)
	addToKernelMap(kernel)
	initTimer()
	startLogger()
	if stop != nil {
		Call(initServerPid, stopFunc(stop))
	}
	ErrorLog("kernel start complete")
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

func Catch() {
	p := recover()
	if p != nil {
		ErrorLog("catch error:%s,Stack:%s", p, debug.Stack())
	}
}

func CatchNoPrint() {
	recover()
}

func CatchFun(f func()) {
	defer Catch()
	f()
}

func init() {
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
		Init: func(context *Context,pid *Pid, args ...interface{}) unsafe.Pointer {
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
