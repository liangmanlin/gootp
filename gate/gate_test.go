package gate

import (
	"github.com/liangmanlin/gootp/kernel"
	"testing"
	"time"
	"unsafe"
)

func TestStart(t *testing.T) {
	go func() {
		time.Sleep(3 * time.Second)
		kernel.ErrorLog("test init stop now")
		kernel.InitStop()
	}()
	kernel.Env.LogPath = ""
	kernel.KernelStart(func() {
		echo := kernel.DefaultActor()
		echo.Init = func(context *kernel.Context,pid *kernel.Pid, args ...interface{}) unsafe.Pointer {
			c := *args[0].(*Conn)
			kernel.ErrorLog("connect")
			return unsafe.Pointer(&c)
		}
		echo.HandleCast = func(context *kernel.Context, msg interface{}) {
			c := (*Conn)(context.State)
			switch m := msg.(type) {
			case bool:
				// start
				c.SetHead(2)
				c.StartReader(context.Self())
			case *TcpError:
				context.Exit("normal")
			case int:
				context.Exit("normal")
			case []byte:
				if err := c.Send(m);err != nil {
					context.Exit("normal")
				}
			default:
				kernel.ErrorLog("un handle msg: %#v", m)
			}
		}
		echo.Terminate = func(context *kernel.Context, reason *kernel.Terminate) {
			(*Conn)(context.State).Close()
		}
		gatePort := 8000
		Start("echo", echo, gatePort, AcceptNum(5))
	}, func() {
		Stop()
	})
}
