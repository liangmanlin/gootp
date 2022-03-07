package kernel

import (
	"fmt"
	"log"
	"sync"
)

//保全全局名字
var (
	nameMap sync.Map
	id2Name sync.Map
)

func Register(name string, pid *Pid) {
	nameMap.Store(name, pid)
	id2Name.Store(pid.id,name)
	pid.c <- regName(name)
}

func RegisterNotExist(name string, pid *Pid) {
	registerNotExist(name, pid)
	pid.c <- regName(name)
}

func UnRegister(name string) {
	if pid, ok := nameMap.Load(name); ok {
		nameMap.Delete(name)
		id2Name.Delete(pid.(*Pid).id)
	}
}

func WhereIs(name string) *Pid {
	if pid, ok := nameMap.Load(name); ok {
		return pid.(*Pid)
	}
	return nil
}

func registerNotExist(name string, pid *Pid) {
	if p, ok := nameMap.Load(name);ok && p.(*Pid).IsAlive() {
		log.Panic(fmt.Errorf("Name :[%s] is register ", name))
	}
	nameMap.Store(name, pid)
	id2Name.Store(pid.id,name)
}

func TryGetName(pid *Pid) string {
	if n,ok := id2Name.Load(pid.id);ok{
		return n.(string)
	}
	return ""
}
