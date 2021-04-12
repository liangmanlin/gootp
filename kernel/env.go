package kernel

import (
	"fmt"
	"log"
)

var Env = &env{
	WriteLogStd:        true,
	LogPath:            "./", //如果为空，则不会输出到文件
	ActorChanCacheSize: 100,
	timerMinTick:       100,
	TimerProcNum:       3,
}

type env struct {
	timerMinTick       int64
	TimerProcNum       int
	ActorChanCacheSize int
	LogPath            string
	WriteLogStd        bool
}

func (e *env) SetTimerMinTick(tick int64) {
	if Millisecond%tick != 0 {
		log.Panic(fmt.Errorf("TimerMinTick error,value:%d ", tick))
	}
	e.timerMinTick = tick
}
func (e *env) TimerMinTick() int64 {
	return e.timerMinTick
}
