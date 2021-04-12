package kernel

import (
	"fmt"
	"time"
	"unsafe"
)

type timerType int

const (
	TimerTypeForever timerType = 1 << iota
	TimerTypeOnce
)

const (
	Millisecond     int64 = 1000
	tenMillisecond        = 10 * Millisecond
	minMillisecond        = 60 * Millisecond
	hourMillisecond       = 60 * minMillisecond
)

var rc = make(chan *actorTimer, 1000)

var timerServerMap = make(map[int64]*Pid)

var secM, tenM, minM, hourM int64

func initTimer() {
	secM = Millisecond / Env.timerMinTick
	tenM = tenMillisecond / Env.timerMinTick
	minM = minMillisecond / Env.timerMinTick
	hourM = hourMillisecond / Env.timerMinTick
	for i := 1; i <= Env.TimerProcNum; i++ {
		name := fmt.Sprintf("timer_%d", i)
		_, pid := SupStartChild("kernel", &SupChild{Name: name, Svr: timerActor, InitArgs: []interface{}{i}})
		timerServerMap[pid.id] = pid
		addToKernelMap(pid)
	}
	ErrorLog("timer service started,min tick: %dms", Env.timerMinTick)
}

type timerList struct {
	pre  *timerList
	next *timerList
	data *actorTimer
}

type actorTimer struct {
	timerType timerType
	d         int64 //毫秒
	inv       int64
	pid       *Pid
	msg       interface{}
}

func TimerStart(timerType timerType, pid *Pid, inv int64, msg interface{}) {
	ti := &actorTimer{timerType: timerType, inv: inv, pid: pid, msg: msg, d: Now2() + inv}
	rc <- ti
}

// 目前只提供最小精度为100ms的定时器

type aTimer struct {
	tick int64
	t0   *timerList // 少于1秒
	t1   *timerList // 少于10秒
	t2   *timerList // 超过10秒的时间
	t3   *timerList // 超过1分钟的时间
	t4   *timerList // 超过1小时的时间
}

var timerActor = &Actor{
	Init: func(context *Context,self *Pid, args ...interface{}) unsafe.Pointer {
		t := aTimer{}
		t.tick = 0
		t.t0 = nil
		t.t1 = nil
		t.t2 = nil
		t.t3 = nil
		t.t4 = nil
		go startReceiver(self, rc)
		go tLoop(self,args[0].(int))
		return unsafe.Pointer(&t)
	},
	HandleCast: func(context *Context, msg interface{}) {
		t := (*aTimer)(context.State)
		switch tm := msg.(type) {
		case *actorTimer:
			t.insertTimer(&timerList{pre: nil, next: nil, data: tm}, tm.inv)
		case bool:
			t.loopTimer()
		}
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

// 遍历所有定时器
func (t *aTimer) loopTimer() {
	now := Now2()
	t.t0 = t.loopTimerCheck(t.t0, now)
	t.tick++
	if t.tick%secM == 0 {
		t.t1 = t.actTimer(t.t1, now, Millisecond)
	}
	if t.tick%tenM == 0 {
		t.t2 = t.actTimer(t.t2, now, tenMillisecond)
	}
	if t.tick%minM == 0 {
		t.t3 = t.actTimer(t.t3, now, minMillisecond)
	}
	if t.tick == hourM {
		t.t4 = t.actTimer(t.t4, now, hourMillisecond)
		t.tick = 0
	}
}

func (t *aTimer) actTimer(list *timerList, now int64, inv int64) *timerList {
	for e := list; e != nil; {
		v := e
		e = e.next
		dif := v.data.d - now
		if dif <= inv {
			removeTimer(&list, v)
			t.insertTimer(v, dif)
		}
	}
	return list
}

func (t *aTimer) loopTimerCheck(list *timerList, now int64) *timerList {
	for e := list; e != nil; {
		v := e
		e = e.next
		if now >= v.data.d {
			if v.data.pid.IsAlive() {
				Cast(v.data.pid, v.data.msg)
				if v.data.timerType == TimerTypeOnce || v.data.inv > Millisecond {
					removeTimer(&list, v)
				}
				if v.data.timerType == TimerTypeForever && v.data.inv > Millisecond {
					v.data.d += v.data.inv
					t.insertTimer(v, v.data.inv)
				} else if v.data.timerType == TimerTypeForever {
					v.data.d = v.data.d + v.data.inv
				}
			} else {
				removeTimer(&list, v)
			}
		}
	}
	return list
}

func (t *aTimer) insertTimer(ti *timerList, inv int64) {
	if inv <= Millisecond {
		t.t0 = insertTimer(t.t0, ti)
	} else if inv <= tenMillisecond {
		t.t1 = insertTimer(t.t1, ti)
	} else if inv <= minMillisecond {
		t.t2 = insertTimer(t.t2, ti)
	} else if inv <= hourMillisecond {
		t.t3 = insertTimer(t.t3, ti)
	} else {
		t.t4 = insertTimer(t.t4, ti)
	}
}

func insertTimer(list *timerList, ti *timerList) *timerList {
	ti.next = list
	ti.pre = nil
	if list != nil {
		list.pre = ti
	}
	return ti
}

func startReceiver(father *Pid, rc chan *actorTimer) {
	for {
		ti := <-rc
		father.c <- ti
	}
}

func tLoop(father *Pid,i int) {
	time.Sleep(time.Duration(i)*10*time.Millisecond)
	c := time.Tick(time.Duration(Env.timerMinTick) * time.Millisecond)
	for {
		<-c
		Cast(father, true)
	}
}

func removeTimer(list **timerList, e *timerList) {
	if e.next == nil {
		if *list == e {
			*list = nil
		} else {
			e.pre.next = nil
		}
	} else {
		if *list == e {
			*list = e.next
			if *list != nil {
				(*list).pre = nil
			}
		} else {
			e.pre.next = e.next
			e.next.pre = e.pre
		}
	}
}
