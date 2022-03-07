package gate

import (
	"github.com/liangmanlin/gootp/bpool"
	"github.com/liangmanlin/gootp/kernel"
	"testing"
	"time"
)

type stateNet struct {
	Conn
}

func TestStart(t *testing.T) {
	go func() {
		time.Sleep(30 * time.Second)
		kernel.ErrorLog("test init stop now")
		kernel.InitStop()
	}()
	kernel.Env.LogPath = ""
	kernel.KernelStart(func() {
		echo := kernel.DefaultActor()
		echo.Init = func(context *kernel.Context,pid *kernel.Pid, args ...interface{}) interface{} {
			state := stateNet{}
			state.Conn = args[0].(Conn)
			kernel.ErrorLog("connect")
			return &state
		}
		echo.HandleCast = func(context *kernel.Context, msg interface{}) {
			state := context.State.(*stateNet)
			switch m := msg.(type) {
			case bool:
				// start
				state.SetHead(2)
				state.StartReader(context.Self())
			case *TcpError:
				context.Exit("normal")
			case int:
				context.Exit("normal")
			case *bpool.Buff:
				if _,err := state.Write(m.ToBytes());err != nil {
					context.Exit("normal")
				}
				kernel.ErrorLog("%s",string(m.ToBytes()))
				m.Free()
			default:
				kernel.ErrorLog("un handle msg: %#v", m)
			}
		}
		echo.Terminate = func(context *kernel.Context, reason *kernel.Terminate) {
			state := context.State.(*stateNet)
			kernel.ErrorLog("exit:%s",state.RemoteAddr().String())
			state.Conn.Close()
		}
		gatePort := 8000
		Start("echo", echo, gatePort, WithAcceptNum(5))
	}, func() {
		Stop("echo")
	})
}
