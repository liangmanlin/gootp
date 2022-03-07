package kernel

import (
	"log"
)

var Env = &env{
	WriteLogStd:        true,
	LogPath:            "", //如果为空，则不会输出到文件,默认不输出日志
	ActorChanCacheSize: 100,
	timerMinTick:       100,
	TimerProcNum:       3,
}

type env struct {
	timerMinTick       int64 `command:"timer_min_tick"`
	TimerProcNum       int `command:"timer_proc_num"`
	ActorChanCacheSize int	`command:"actor_chan_cache_size"`
	LogPath            string `command:"log_path"`
	WriteLogStd        bool `command:"write_log_std"`
}

func (e *env) SetTimerMinTick(tick int64) {
	if Millisecond%tick != 0 {
		log.Panicf("TimerMinTick error,value:%d ", tick)
	}
	e.timerMinTick = tick
}
func (e *env) TimerMinTick() int64 {
	return e.timerMinTick
}
