package kernel

import (
	"testing"
	"time"
)

func TestKernelStart(t *testing.T) {
	go func() {
		time.Sleep(3 * time.Second)
		ErrorLog("test init stop now")
		InitStop()
	}()
	Env.LogPath = ""
	KernelStart(func() {
		for i := 0; i < 10; i++ {
			A := DefaultActor()
			A.Init = func(context *Context, pid *Pid, args ...interface{}) interface{} {
				SendAfter(TimerTypeForever, pid, 100, 1)
				ErrorLog("Start :%s", pid)
				return nil
			}
			A.Terminate = func(context *Context, reason *Terminate) {
				ErrorLog("stop :%s", Ctx().Self())
			}
			Start(A)
		}
	}, nil)
}
