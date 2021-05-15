package node

import "github.com/liangmanlin/gootp/kernel"

func (a *app) Name() string {
	return "NodeApp"
}

func (a *app) Start(bootType kernel.AppBootType) *kernel.Pid {
	kernel.SetSelfNodeName(Env.nodeName)
	// 启动监控数
	_, supPid := kernel.SupStartChild("kernel", &kernel.SupChild{ChildType: kernel.SupChildTypeSup, Name: "NodeSup"})
	kernel.SupStartChild(supPid, &kernel.SupChild{ChildType: kernel.SupChildTypeWorker, Name: "NodeMonitor", Svr: monitorActor, ReStart: true})
	kernel.SupStartChild(supPid, &kernel.SupChild{ChildType: kernel.SupChildTypeWorker, Name: "RPC", Svr: rpcSvr, ReStart: true})
	start(Env.nodeName)
	return supPid
}

func (a *app)Stop(stopType kernel.AppStopType){

}

func (a *app)SetEnv(key string,value interface{}){

}

func (a *app)GetEnv(key string)interface{}{
	return nil
}