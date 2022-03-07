package main

import (
	"github.com/liangmanlin/gootp/args"
	"github.com/liangmanlin/gootp/bpool"
	"github.com/liangmanlin/gootp/httpd"
	"github.com/liangmanlin/gootp/httpd/websocket"
	"github.com/liangmanlin/gootp/kernel"
)

type wsState struct {
	conn *websocket.Conn
	room *kernel.Pid
}

var chat = &kernel.Actor{
	Init: func(ctx *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
		state := wsState{}
		state.conn = args[0].(*websocket.Conn)
		state.room = args[2].(*kernel.Pid)
		state.room.Cast(pid)
		ctx.Link(state.room)
		return &state
	},
	HandleCast: func(ctx *kernel.Context, msg interface{}) {
		state := ctx.State.(*wsState)
		switch m := msg.(type) {
		case *bpool.Buff:
			kernel.ErrorLog("%s recv: %s",state.conn.RemoteAddr(), string(m.ToBytes()))
			state.room.Cast(m)
		case []byte:
			state.conn.WriteMessage(websocket.TextMessage, m)
		case *websocket.WsError:
			// 基本上都是close
			kernel.ErrorLog("close :%s",ctx.Self())
			ctx.Exit(kernel.ExitReasonNormal)
		}
	},
	HandleCall: func(ctx *kernel.Context, request interface{}) interface{} {
		return nil
	},
	Terminate: func(ctx *kernel.Context, reason *kernel.Terminate) {
		kernel.DebugLog("terminate %s", reason.Reason)
	},
}

type roomState struct {
	m map[int64]*kernel.Pid
}

var room = &kernel.Actor{
	Init: func(ctx *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
		state := roomState{
			m: make(map[int64]*kernel.Pid),
		}
		return &state
	},
	HandleCall: func(ctx *kernel.Context, request interface{}) interface{} {
		return nil
	},
	HandleCast: func(ctx *kernel.Context, msg interface{}) {
		state := ctx.State.(*roomState)
		switch m := msg.(type) {
		case *bpool.Buff:
			buf := m.Copy()
			m.Free()
			for _, pid := range state.m {
				pid.Cast(buf)
			}
		case *kernel.Pid:
			state.m[m.GetID()] = m
		case *kernel.PidExit:
			kernel.DebugLog("pid:%s exit", m.Pid)
			delete(state.m, m.Pid.GetID())
		}
	},
}

func main() {
	kernel.Env.LogPath = ""
	kernel.SetLogLevel(1)
	kernel.KernelStart(func() {
		port, _ := args.GetInt("port")
		if port == 0 {
			port = 8080
		}
		const managerNum = 8
		eg := httpd.New("test", port,
			httpd.WithManagerNum(managerNum),
			httpd.WithMaxWorkerNum(managerNum*2048),
			httpd.WithWsConfig(websocket.Config{
				EnableCompression:      true,
				EnableWriteCompression: true,
				CompressionLevel:       1,
				Origin:                 false,
			}),
		)
		roomPid,_ := kernel.Start(room)
		// websocket uri
		eg.GetWebsocket("/ws/chat", chat,roomPid)

		g := eg.GetGroup("/2")
		g2 := g.Group("/e")
		{
			g2.Get("/e", func(ctx *kernel.Context, request *httpd.Request) {
				request.AddBody([]byte("hello e"))
			})
		}
		g = g.Group("/2")
		{
			g.Get("/2", func(ctx *kernel.Context, request *httpd.Request) {
				request.AddBody([]byte("hello " + request.Lookup("name")))
			})
			g.Get("/1", func(ctx *kernel.Context, request *httpd.Request) {
				request.AddBody([]byte("hello " + request.Lookup("name")))
			})
		}
		eg.Run()
	}, nil)
}
