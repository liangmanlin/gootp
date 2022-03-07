package gate

import (
	"github.com/liangmanlin/gootp/bpool"
	"github.com/liangmanlin/gootp/kernel"
	"testing"
	"time"
)

type stateEcho struct {
	conn Conn
}

func TestEpollStart(t *testing.T) {
	go func() {
		time.Sleep(30 * time.Second)
		kernel.ErrorLog("test init stop now")
		kernel.InitStop()
	}()
	kernel.Env.LogPath = ""
	kernel.KernelStart(func() {
		kernel.SetLogLevel(1)
		echo := kernel.DefaultActor()
		echo.Init = func(context *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
			state := stateEcho{}
			state.conn = args[0].(Conn)
			kernel.ErrorLog("connect")
			return &state
		}
		echo.HandleCast = func(context *kernel.Context, msg interface{}) {
			state := context.State.(*stateEcho)
			switch m := msg.(type) {
			case bool:
				// start
				kernel.ErrorLog("recv")
				state.conn.SetHead(2)
				buf, err := state.conn.Recv(0, 0)
				if err == nil {
					kernel.ErrorLog("%s", string(buf))
					state.conn.StartReader(context.Self())
				}else{
					context.Exit("normal")
				}
			case *TcpError:
				context.Exit("normal")
			case int:
				context.Exit("normal")
			case *bpool.Buff:
				if _, err := state.conn.Send(m.ToBytes()[2:]); err != nil {
					context.Exit("normal")
				}
				kernel.ErrorLog("%s", string(m.ToBytes()[2:]))
				m.Free()
			default:
				kernel.ErrorLog("un handle msg: %#v", m)
			}
		}
		echo.Terminate = func(context *kernel.Context, reason *kernel.Terminate) {
			kernel.ErrorLog("client exit :%s", reason.Reason)
			context.State.(*stateEcho).conn.Close()
		}
		gatePort := 8000
		Start("echo", echo, gatePort, WithUseEpoll())
	}, func() {
		Stop("echo")
	})
}
