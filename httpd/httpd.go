package httpd

import (
	"github.com/lesismal/nbio"
	"github.com/liangmanlin/gootp/gutil"
	"github.com/liangmanlin/gootp/httpd/websocket"
	"github.com/liangmanlin/gootp/kernel"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

var (
	DefaultHTTPReadLimit = 20 * 1024 * 1024
)

/*
	快速的http server
	仅仅支持http协议，如果需要https，可以通过nginx做反向代理
*/
func New(name string, port int, c ...config) *Engine {
	cpuNum := runtime.NumCPU()
	d := &Engine{
		name:         name,
		port:         port,
		readLimit:    DefaultHTTPReadLimit,
		tcpReadBuff:  16 * 1024, // 缺省16k
		managerNum:   cpuNum,
		maxWorkerNum: cpuNum * 1024,
		getRouter:    &router{rt: map[string]*handler{}},
		postRouter:   &router{rt: map[string]*handler{}},
	}
	for _, f := range c {
		f(d)
	}
	return d
}

func (e *Engine) Run() error {
	return kernel.AppStart(&eapp{e})
}

func (e *Engine) start(sup *kernel.Pid) {
	// 先启动必要的进程，再进行网关启动
	dev := int(gutil.Ceil(float32(e.maxWorkerNum) / float32(e.managerNum)))
	for i := 0; i < e.managerNum; i++ {
		_, managerPid := kernel.SupStartChild(sup, &kernel.SupChild{Svr: manager, InitArgs: kernel.MakeArgs(dev, e)})
		e.manager = append(e.manager, managerPid)
	}
	addrs := e.buildAddr()
	g := nbio.NewGopher(nbio.Config{
		Name:           e.name,
		Network:        "tcp",
		Addrs:          addrs,
		ReadBufferSize: e.tcpReadBuff,
		EpollMod:       nbio.EPOLLET,
	})
	h := e.handler
	if e.hasWebSocket {
		if e.wsConfig == nil {
			e.wsConfig = websocket.DefaultConfig()
		}
		e.wsConfig.Confirm()
		h = e.handlerWebSocket
		g.OnClose(func(c *nbio.Conn, err error) {
			if s, ok := c.Session().(*websocket.Conn); ok {
				kernel.DebugLog("websocket closed: %s", c.RemoteAddr())
				s.OnClose()
			}
		})
	} else {
		if e.balancing == 1 {
			h = e.handlerRand
		}
	}
	g.OnOpen(func(c *nbio.Conn) {
		parser := NewParser(h, c, e.readLimit)
		c.SetSession(parser)
		kernel.DebugLog("new c:%d", c.Hash())
	})
	g.OnData(func(c *nbio.Conn, data []byte) {
		var err error
		switch p := c.Session().(type) {
		case *Parser:
			err = p.Read(data)
		case *websocket.Conn:
			err = p.Read(data)
		}
		if err != nil {
			kernel.ErrorLog("handle error:%s", err)
			c.CloseWithError(err)
		}
	})
	err := g.Start()
	if err != nil {
		log.Panic(err)
	}
	e.engine = g
	for _, addr := range addrs {
		kernel.ErrorLog("httpd [%s] start on [%s]", e.name, addr)
	}
}

func (e *Engine) handler(conn *nbio.Conn, req *http.Request) {
	// 移交到队列处理
	// 根据fd hash
	r := newRequest(req, conn)
	e.manager[conn.Hash()%e.managerNum].Cast(r)
}

func (e *Engine) handlerRand(conn *nbio.Conn, req *http.Request) {
	// 移交到队列处理
	r := newRequest(req, conn)
	e.manager[rand.Intn(e.managerNum)].Cast(r)
}

// 性能会有所损失
// 为了提升性能，尽量用最短的uri
func (e *Engine) handlerWebSocket(conn *nbio.Conn, req *http.Request) {
	r := newRequest(req, conn)
	var ok bool
	defer func() {
		if !ok {
			p := recover()
			if p != nil {
				kernel.ErrorLog("catch error:%s,Stack:%s", p, debug.Stack())
			}
			conn.Close()
		}
	}()
	if h, err := routerHandler(e, r); err == nil {
		if h.isWs {
			reqCopy := *req
			if c, err := websocket.Upgrade(e.wsConfig, r.ResponseWriter(), conn, req); err != nil {
				kernel.ErrorLog("upgrade error: %s", err)
				// 再一次关闭，防止内部逻辑没有close
				conn.Close()
			} else {
				args := append(kernel.MakeArgs(c, &reqCopy), h.actorArgs...)
				if pid, err := kernel.Start(h.actor, args...); err == nil {
					c.SetHandler(pid)
					conn.SetSession(c)
				} else {
					kernel.ErrorLog("start handler error:%s", err)
					conn.Close()
				}
			}
		} else {
			r.f = h.f
			if e.balancing == 1 {
				e.manager[rand.Intn(e.managerNum)].Cast(r)
			} else {
				e.manager[conn.Hash()%e.managerNum].Cast(r)
			}
		}
	} else {
		kernel.ErrorLog("router error:%s %s", err, req.RequestURI)
		r.reply404()
	}
	ok = true
}

func (e *Engine) buildAddr() []string {
	if e.port == 0 {
		e.port = 8080
	}
	if len(e.addr) == 0 {
		return []string{":" + strconv.Itoa(e.port)}
	}
	rs := make([]string, 0, len(e.addr))
	for _, addr := range e.addr {
		if strings.Index(addr, ":") >= 0 {
			rs = append(rs, addr)
		} else {
			rs = append(rs, addr+":"+strconv.Itoa(e.port))
		}
	}
	return rs
}

// 单纯是代码好看一点
func none(ctx *kernel.Context, request *Request) {

}
