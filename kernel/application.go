package kernel

import (
	"fmt"
	"runtime/debug"
	"sync"
	"unsafe"
)

var _appMaps sync.Map
var _appPid2Name sync.Map

var appPid *Pid

var AppErrAlreadyStarted = fmt.Errorf("app already started ")
var AppStartErr = fmt.Errorf("error ")
var AppErrNotStart = fmt.Errorf("app not start ")

func AppStart(app Application) (err error) {
	var ok bool
	defer func() {
		if !ok {
			err = AppStartErr
			p := recover()
			ErrorLog("catch error:%s,Stack:%s", p, debug.Stack())
		}
	}()
	if AppInfo(app.Name()) != nil {
		return AppErrAlreadyStarted
	}
	pid := app.Start(APP_BOOT_TYPE_START)
	ai := &appInfo{app: app, pid: pid}
	_appMaps.Store(app.Name(), ai)
	_appPid2Name.Store(pid.id, app.Name())
	ok = true
	Cast(appPid, ai)
	return nil
}

func AppStop(name string) {
	Call(appPid, name)
}

func AppRestart(name string)(err error) {
	app := AppInfo(name)
	if app == nil {
		return AppErrNotStart
	}
	var ok bool
	defer func() {
		if !ok {
			err = AppStartErr
			p := recover()
			ErrorLog("catch error:%s,Stack:%s", p, debug.Stack())
		}
	}()
	app.Stop(APP_STOP_TYPE_RESTART)
	app.Start(APP_BOOT_TYPE_RESTART)
	return
}

func AppInfo(name string) Application {
	if app, ok := _appMaps.Load(name); ok {
		return app.(appInfo).app
	}
	return nil
}

var appSvr = &Actor{
	Init: func(ctx *Context, pid *Pid, args ...interface{}) unsafe.Pointer {
		ErrorLog("application %s started", pid)
		addToKernelMap(pid)
		appPid = pid
		return nil
	},
	HandleCast: func(ctx *Context, msg interface{}) {
		switch m := msg.(type) {
		case *appInfo:
			Link(ctx.self, m.pid)
		case *PidExit:
			if appName, ok := _appPid2Name.Load(m.Pid.id); ok {
				ai, _ := _appMaps.Load(appName)
				aInf := ai.(*appInfo)
				aInf.app.Stop(APP_STOP_TYPE_NORMAL)
				_appPid2Name.Delete(m.Pid.id)
				_appMaps.Delete(appName)
			}
		}

	},
	HandleCall: func(ctx *Context, request interface{}) (rs interface{}) {
		switch r := request.(type) {
		case string: // stop
			if ai, ok := _appMaps.Load(r); ok {
				aInf := ai.(*appInfo)
				SupStop(aInf.pid) // 这里不删除数据，等待PidExit消息处理
			}

		}
		return
	},
	Terminate: func(ctx *Context, reason *Terminate) {

	},
	ErrorHandler: func(ctx *Context, err interface{}) bool {
		return true
	},
}
