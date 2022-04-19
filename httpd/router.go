package httpd

import (
	"errors"
	"github.com/liangmanlin/gootp/kernel"
	"log"
	"strings"
)

var (
	errorNoHandler       = errors.New("no handler")
	errorInterceptorStop = errors.New("interceptor stop")
)

func (e *Engine) Get(uri string, handler func(ctx *kernel.Context, request *Request)) {
	paths := buildPaths(uri)
	insertPathsGroup(e.getRouter, nil, paths, handler)
}

func (e *Engine) Post(uri string, handler func(ctx *kernel.Context, request *Request)) {
	paths := buildPaths(uri)
	insertPathsGroup(e.postRouter, nil, paths, handler)
}

func (e *Engine) GetWebsocket(uri string, handler *kernel.Actor, args ...interface{}) {
	paths := buildPaths(uri)
	h := insertPathsGroup(e.getRouter, nil, paths, none)
	h.isWs = true
	h.actor = handler
	h.actorArgs = args
	e.hasWebSocket = true
}

// GetGroup 返回一个url组
// 用法
// e := New("web",8080)
// g := e.Group("/v1") // 组为根目录 host/v1
// {
//		g.Get("/login",handler)  // 响应url： host/v1/login
// }
func (e *Engine) GetGroup(uriGroup string) *GetGroup {
	r := insertPathsGroup(e.getRouter, nil, buildPaths(uriGroup), nil)
	return &GetGroup{h: r, eg: e}
}

func (e *Engine) PostGroup(uriGroup string) *PostGroup {
	r := insertPathsGroup(e.postRouter, nil, buildPaths(uriGroup), nil)
	return &PostGroup{h: r, eg: e}
}

type Group struct {
	h  *handler
	eg *Engine
}

type GetGroup Group
type PostGroup Group

// GET
func (g *GetGroup) Get(uri string, handler handlerFunc) {
	paths := buildPaths(uri)[1:]
	insertPathsGroup(g.h.r, g.h, paths, handler)
}

func (g *GetGroup) GetWebsocket(uri string, handler *kernel.Actor) {
	paths := buildPaths(uri)[1:]
	h := insertPathsGroup(g.h.r, g.h, paths, none)
	h.isWs = true
	h.actor = handler
	g.eg.hasWebSocket = true
}

func (g *GetGroup) Group(uriGroup string) *GetGroup {
	paths := buildPaths(uriGroup)[1:]
	r := insertPathsGroup(g.h.r, g.h, paths, nil)
	return &GetGroup{h: r, eg: g.eg}
}

// 设置拦截器
func (g *GetGroup) SetInterceptor(f func(r *Request) bool) {
	g.h.interceptor = f
}

// POST
func (g *PostGroup) Post(uri string, handler handlerFunc) {
	paths := buildPaths(uri)[1:]
	insertPathsGroup(g.h.r, g.h, paths, handler)
}

func (g *PostGroup) Group(uriGroup string) *PostGroup {
	paths := buildPaths(uriGroup)[1:]
	r := insertPathsGroup(g.h.r, g.h, paths, nil)
	return &PostGroup{h: r, eg: g.eg}
}

// 设置拦截器
func (g *PostGroup) SetInterceptor(f func(r *Request) bool) {
	g.h.interceptor = f
}

func routerHandler(eg *Engine, r *Request) (handle *handler, err error) {
	if r.Method == "POST" {
		return routerHandlerReq(eg.postRouter, r)
	}
	return routerHandlerReq(eg.getRouter, r)
}

func routerHandlerReq(rt *router, r *Request) (handle *handler, err error) {
	p := buildPathsReq(r.URL.Path)
	for ; ; p.next() {
		if h, ok := rt.rt[p.now]; ok {
			if p.isEnd() {
				if h.f != nil {
					return h, nil
				}
				return nil, errorNoHandler
			} else {
				// 不应该存在最终url的拦截器，所以，非终点的拦截器才会被执行
				if h.interceptor != nil {
					if !h.interceptor(r) {
						// 停止
						return nil, errorInterceptorStop
					}
				}
				rt = h.r
			}
		} else {
			switch rt.rType {
			case '*':
				r.addHeader(rt.rName, p.remain())
				return rt.h, nil
			case ':':
				r.addHeader(rt.rName, p.now)
				if p.isEnd() {
					if rt.h.f != nil {
						return rt.h, nil
					}
					return nil, errorNoHandler
				} else if rt.h.r == nil {
					return nil, errorNoHandler
				}
				rt = rt.h.r
			default:
				goto end
			}
		}
	}
end:
	return nil, errorNoHandler
}

func buildPaths(uri string) []string {
	if len(uri) == 0 {
		return []string{"/"}
	} else if uri[0] == '/' {
		uri = uri[1:]
	}
	pl := strings.Split(uri, "/")
	return append([]string{"/"}, pl...)
}

func insertPathsGroup(group *router, hd *handler, paths []string, h handlerFunc) *handler {
	endIdx := len(paths) - 1
	var ok bool
	for i, path := range paths {
		if group == nil {
			group = &router{rt: map[string]*handler{}}
			if hd != nil {
				hd.r = group
			}
		}
		paramType, paramName := readParam(path)
		if paramType == '*' || paramType == ':' {
			if group.rType == '*' || (group.rType != 0 && (group.rType != paramType || group.rName != paramName)) || (paramType == '*' && len(group.rt) != 0) {
				log.Panic("duplicate uri /", strings.Join(paths, "/"))
			}
			group.rName = paramName
			group.rType = paramType
			if group.h == nil {
				hd = &handler{}
				group.h = hd
			}
			hd = group.h
			if i == endIdx {
				if hd.f != nil {
					log.Panic("duplicate uri /", strings.Join(paths, "/"))
				}
				hd.f = h
			}
			if hd.r == nil {
				hd.r = &router{rt: map[string]*handler{}}
			}
			group = hd.r
			continue
		}
		if group.rType == '*' {
			log.Panic("duplicate uri /", strings.Join(paths, "/"))
		}
		if hd, ok = group.rt[path]; ok {
			if paramType == '*' {
				log.Panic("duplicate uri /", strings.Join(paths, "/"))
			}
			if i == endIdx {
				if hd.f != nil && h != nil {
					log.Panic("duplicate uri /", strings.Join(paths, "/"))
				} else if h != nil {
					hd.f = h
				}
			} else {
				group = hd.r
			}
		} else {
			hd = &handler{}
			if i == endIdx {
				hd.f = h
			}
			group.rt[path] = hd
			group = hd.r
		}
	}
	return hd
}

type pathReader struct {
	uri string
	now string
	idx int
}

func readParam(path string) (byte, string) {
	var paramType byte = 0
	var paramName string
	switch path[0] {
	case ':':
		paramType = ':'
		paramName = path[1:]
	case '*':
		paramType = '*'
		paramName = path[1:]
	}
	return paramType, paramName
}

var root = "/"

func buildPathsReq(uri string) *pathReader {
	if len(uri) == 0 {
		return &pathReader{now: root}
	} else if uri[0] == '/' {
		uri = uri[1:]
	}
	p := &pathReader{now: root, uri: uri}
	return p
}

func (p *pathReader) next() {
	uri := p.uri
	size := len(uri)
	if p.idx < size {
		startIdx := p.idx
		for i := startIdx; i < size; i++ {
			if uri[i] == '/' {
				p.now = uri[startIdx:i]
				p.idx = i + 1
				return
			}
		}
		p.idx = size + 1
		p.now = uri[startIdx:]
	}
}

func (p *pathReader) remain() string {
	return p.now + p.uri[p.idx-1:]
}

func (p *pathReader) isEnd() bool {
	return p.idx >= len(p.uri)
}
