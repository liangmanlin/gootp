package kernel

import (
	"fmt"
	"log"
	"sync"
)

//保全全局名字
var nameMap sync.Map

func Register(name string, pid *Pid) {
	nameMap.Store(name, pid)
	pid.c <- regName(name)
}

func RegisterNotExist(name string, pid *Pid) {
	registerNotExist(name, pid)
	pid.c <- regName(name)
}

func UnRegister(name string) {
	nameMap.Delete(name)
}

func WhereIs(name string) *Pid {
	if pid, ok := nameMap.Load(name); ok {
		return pid.(*Pid)
	}
	return nil
}

func registerNotExist(name string, pid *Pid) {
	p, ok := nameMap.Load(name)
	if ok && p.(*Pid).IsAlive() {
		log.Panic(fmt.Errorf("Name :[%s] is register ", name))
	}
	nameMap.Store(name, pid)
}
