package timer

import (
	"github.com/liangmanlin/gootp/kernel"
	"reflect"
	"unsafe"
)

func NewTimerPointer() unsafe.Pointer {
	t := NewTimer()
	return unsafe.Pointer(t)
}

func NewTimer() *Timer {
	return &Timer{m: make(map[TimerKey]*timer)}
}

func Start(timer unsafe.Pointer, key TimerKey, inv, times int32, f interface{}, args ...interface{}) {
	t := (*Timer)(timer)
	t.Add(key, inv, times, f, args...)
}

func Loop(timer unsafe.Pointer, state interface{}, now2 int64) {
	(*Timer)(timer).Loop(state, now2)
}

// inv <= 0 表示永久定时器
func (t *Timer) Add(key TimerKey, inv, times int32, f interface{}, args ...interface{}) {
	now2 := kernel.Now2()
	tm := &timer{
		time:  now2 + int64(inv),
		times: times,
		f:     f,
		arg:   args,
	}
	if t.isLooping {
		t.tmp[key] = tm
	} else {
		t.m[key] = tm
	}
}

func (t *Timer)Del(key TimerKey)  {
	delete(t.m,key)
}

// TODO 后续考虑增加多级时间分类，减少遍历长度
func (t *Timer) Loop(state interface{}, now2 int64) {
	t.isLooping = true
	for k, tm := range t.m {
		if now2 >= tm.time {
			tm.time += int64(tm.inv)
			// 激活回调函数
			if tm.times > 0 {
				tm.times--
				if tm.times == 0 {
					// 先删除
					delete(t.m, k)
				}
			}
			actFun(state, tm.f, tm.arg)
		}
	}
	if len(t.tmp) > 0 {
		for k, tm := range t.tmp {
			t.m[k] = tm
			delete(t.tmp, k)
		}
	}
	t.isLooping = false
}

func actFun(state interface{}, f interface{}, arg []interface{}) {
	defer kernel.Catch()
	// 浪费一些性能，使用反射执行
	vf := reflect.ValueOf(f).Elem()
	size := len(arg) + 1
	in := make([]reflect.Value, size)
	in[0] = reflect.ValueOf(state)
	for i := 0; i < len(arg); i++ {
		in[i+1] = reflect.ValueOf(arg[i])
	}
	vf.Call(in)
}
