package httpd

import (
	"github.com/lesismal/nbio"
	"github.com/liangmanlin/gootp/bpool"
	"github.com/liangmanlin/gootp/httpd/websocket"
	"github.com/liangmanlin/gootp/kernel"
	"net"
	"net/http"
)

type Engine struct {
	name         string
	port         int
	readLimit    int
	tcpReadBuff  int
	maxWorkerNum int
	managerNum   int
	manager      []*kernel.Pid
	balancing    int
	engine       *nbio.Gopher
	addr         []string
	getRouter    router
	postRouter   router
	hasWebSocket bool
	wsConfig     *websocket.Config
}

type config func(*Engine)

type Request struct {
	*http.Request
	Conn         net.Conn
	response     *http.Response
	responseBody *bpool.Buff
	f            handlerFunc
	isReply      bool
}

type router map[string]*handler

type handler struct {
	isWs        bool
	actor       *kernel.Actor
	actorArgs   []interface{}
	interceptor func(request *Request) bool // 拦截器
	f           handlerFunc
	r           router
}

type handlerFunc func(ctx *kernel.Context, request *Request)
