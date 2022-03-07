package httpd

import (
	"errors"
	"log"
	"strings"
)

var (
	errorNoHandler       = errors.New("no handler")
	errorInterceptorStop = errors.New("interceptor stop")
)

func routerHandler(eg *Engine, r *Request) (handle *handler, err error) {
	if r.Method == "POST" {
		return routerHandlerReq(eg.postRouter, r)
	}
	return routerHandlerReq(eg.getRouter, r)
}

func routerHandlerReq(rt router, r *Request) (handle *handler, err error) {
	p := buildPathsReq(r.URL.Path)
	for ; ; p.next() {
		if h, ok := rt[p.now]; ok {
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
			break
		}
	}
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

func insertPathsGroup(group router, paths []string, h handlerFunc) *handler {
	endIdx := len(paths) - 1
	var hd *handler
	var ok bool
	for i, path := range paths {
		if hd, ok = group[path]; ok {
			if i == endIdx {
				if hd.f != nil && h != nil {
					log.Panic("duplicate uri", paths)
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
			} else {
				hd.r = router{}
			}
			group[path] = hd
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
		p.idx = size
		p.now = uri[startIdx:]
	}
}

func (p *pathReader) isEnd() bool {
	return p.idx >= len(p.uri)
}
