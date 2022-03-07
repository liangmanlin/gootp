package gate

import (
	"fmt"
	"github.com/liangmanlin/gootp/kernel"
	"log"
)

func (a *app)Name() string {
	return a.name
}

func (a *app)Start(bootType kernel.AppBootType) *kernel.Pid {
	supName := fmt.Sprintf("gate_child_sup_%s", a.name)
	child := &kernel.SupChild{Name: supName, ReStart: false, ChildType: kernel.SupChildTypeSup}
	_, childSup := kernel.SupStartChild(gateSupName, child)
	clientSup := fmt.Sprintf("gate_client_sup_%s", a.name)
	child = &kernel.SupChild{Name: clientSup, ReStart: false, ChildType: kernel.SupChildTypeSup}
	_, csPid := kernel.SupStartChild(supName, child)
	// 启动侦听进程
	listenerName := fmt.Sprintf("gate_listener_%s", a.name)
	args := kernel.MakeArgs(a.name, a.handler, a.port, csPid, childSup, a.opt)
	child = &kernel.SupChild{Name: listenerName, ReStart: true, ChildType: kernel.SupChildTypeWorker, Svr: listenerActor, InitArgs: args}
	err, _ := kernel.SupStartChild(supName, child)
	if err != nil {
		kernel.ErrorLog("%#v", err)
		log.Panic(err)
	}
	if a.port > 0 {
		kernel.ErrorLog("[%s] listening on port: [0.0.0.0:%d]", a.name, a.port)
	}
	return childSup
}

func (a *app)Stop(stopType kernel.AppStopType)  {
	listenerName := fmt.Sprintf("gate_listener_%s", a.name)
	kernel.CallName(listenerName,stopType)
	kernel.ErrorLog("gate %s stop",a.name)
}

func (a *app)SetEnv(key string,value interface{})  {

}

func (a *app)GetEnv(key string) interface{} {
	return nil
}