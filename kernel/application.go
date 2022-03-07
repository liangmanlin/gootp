package kernel

import (
	"container/list"
	"errors"
	"runtime/debug"
	"sync"
)

var _appMaps sync.Map
var _appPid2Name sync.Map

var appPid *Pid

var ErrAppAlreadyStarted = errors.New("app already started ")
var ErrAppStart = errors.New("app start catch error ")
var ErrAppNotStart = errors.New("app not Start ")

type app struct {
	l *list.List
}

func AppStart(app Application) (err error) {
	var ok bool
	defer func() {
		if !ok {
			err = ErrAppStart
			p := recover()
			ErrorLog("catch error:%s,Stack:%s", p, debug.Stack())
		}
	}()
	if AppInfo(app.Name()) != nil {
		return ErrAppAlreadyStarted
	}
	pid := app.Start(APP_BOOT_TYPE_START)
	ai := &appInfo{app: app, pid: pid}
	_appMaps.Store(app.Name(), ai)
	_appPid2Name.Store(pid.id, app.Name())
	ok = true
	Cast(appPid, ai)
	return
}

func AppStop(name string) {
	CallTimeOut(appPid, name, 20)
}

func AppRestart(name string) (err error) {
	app := AppInfo(name)
	if app == nil {
		return ErrAppNotStart
	}
	var ok bool
	defer func() {
		if !ok {
			err = ErrAppStart
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
		return app.(*appInfo).app
	}
	return nil
}

var appSvr = &Actor{
	Init: func(ctx *Context, pid *Pid, args ...interface{}) interface{} {
		ErrorLog("application %s started", pid)
		addToKernelMap(pid)
		appPid = pid
		state := app{
			l: list.New(),
		}
		return &state
	},
	HandleCast: func(ctx *Context, msg interface{}) {
		switch m := msg.(type) {
		case *appInfo:
			m.e =ctx.State.(*app).l.PushFront(m.app.Name())
			Link(ctx.self, m.pid)
		case *PidExit:
			if appName, ok := _appPid2Name.Load(m.Pid.id); ok {
				_appPid2Name.Delete(m.Pid.id)
				if ai, ok := _appMaps.Load(appName); ok {
					ctx.State.(*app).l.Remove(ai.(*appInfo).e)
					_appMaps.Delete(appName)
				}
			}
		}

	},
	HandleCall: func(ctx *Context, request interface{}) (rs interface{}) {
		switch r := request.(type) {
		case string: // stop
			if ai, ok := _appMaps.Load(r); ok {
				aInf := ai.(*appInfo)
				app_stop(aInf)
			}
		case *initStop:
			l := ctx.State.(*app).l
			for e := l.Front(); e != nil; e = e.Next() {
				if ai, ok := _appMaps.Load(e.Value); ok {
					aInf := ai.(*appInfo)
					if aInf.pid.IsAlive() {
						app_stop(aInf)
					}
				}
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

func app_stop(ai *appInfo) {
	ai.app.Stop(APP_STOP_TYPE_NORMAL)
	SupStop(ai.pid) // 这里不删除数据，等待PidExit消息处理
}
