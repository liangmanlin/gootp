package gate

import (
	"fmt"
	"github.com/lesismal/nbio"
	"github.com/liangmanlin/gootp/kernel"
	"log"
	"net"
	"strconv"
)

type listener struct {
	isUseNbio bool
	g         *nbio.Gopher

	ls net.Listener
}

var listenerActor = &kernel.Actor{
	Init: func(context *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
		name := args[0].(string)
		handler := args[1].(*kernel.Actor)
		port := args[2].(int)
		clientSup := args[3].(*kernel.Pid)
		childSup := args[4].(*kernel.Pid) // 二级根sup
		opt := args[5].([]optFun)
		optStruct := parseOpt(opt)
		l := listener{}
		if optStruct.isUseNbio {
			l.isUseNbio = true
			connectStarter, _ := kernel.Start(starterActor, handler, clientSup, optStruct.clientArgs)
			g := nbio.NewGopher(nbio.Config{
				Name:           name,
				Network:        "tcp",
				Addrs:          []string{":" + strconv.Itoa(port)},
				ReadBufferSize: 1024,
				EpollMod:       nbio.EPOLLET, // 边缘模式
			})
			l.g = g
			g.OnOpen(func(c *nbio.Conn) {
				conn := NewNbConn(c)
				c.SetSession(conn)
				connectStarter.Cast(conn)
			})
			g.OnClose(func(c *nbio.Conn, err error) {
				kernel.ErrorLog("on closed :%s",c.RemoteAddr().String())
				if c.Session() != nil {
					s := c.Session().(*ConnNbio)
					if s.mod == nb_mode_active { // 暂时不需要考虑race
						s.recvPid.Cast(&TcpError{ErrType: ErrClosed})
					}else{
						s.Close()
					}
				}
			})
			g.OnRead(func(c *nbio.Conn) {
				c.Session().(*ConnNbio).OnRead(c)
			})
			g.Start()
			kernel.ErrorLog("gate start on nbio")
		} else {
			ls, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
			if err != nil {
				kernel.ErrorLog(err.Error())
				log.Panic(err)

			}
			// 假如port=0，那么端口是随机的，记录一下
			addrMap[name] = ls.Addr().String()
			l.ls = ls
			l.startAccept(ls, handler, clientSup, childSup, optStruct)
		}
		return &l
	},
	HandleCast: func(context *kernel.Context, msg interface{}) {
	},
	HandleCall: func(context *kernel.Context, request interface{}) interface{} {
		state := context.State.(*listener)
		switch request.(type) {
		case kernel.AppStopType:
			if state.isUseNbio {
				state.g.Stop()
				state.g = nil
			}else {
				// 停止接收连接
				state.ls.Close()
				state.ls = nil
			}
		}
		return nil
	},
	Terminate: func(context *kernel.Context, reason *kernel.Terminate) {
		state := context.State.(*listener)
		if state.g !=nil {
			state.g.Stop()
		}
		if state.ls != nil {
			state.ls.Close()
		}
	},
	ErrorHandler: func(context *kernel.Context, err interface{}) bool {
		return true
	},
}

func (l *listener) startAccept(ls net.Listener, handler *kernel.Actor, clientSup *kernel.Pid, childSup *kernel.Pid, opt *optStruct) {
	for i := 0; i < opt.acceptNum; i++ {
		pid, err := kernel.Start(acceptorActor, clientSup, ls, handler, opt.clientArgs)
		if err != nil {
			log.Panic(err)
		}
		kernel.Cast(pid, true)
	}
}

func parseOpt(opt []optFun) *optStruct {
	df := &optStruct{acceptNum: 10, clientArgs: nil}
	for _, f := range opt {
		f(df)
	}
	return df
}

type optStruct struct {
	isUseNbio  bool
	acceptNum  int
	clientArgs []interface{}
}
