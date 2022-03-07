package httpc

import (
	"crypto/tls"
	"github.com/liangmanlin/gootp/kernel"
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"time"
)

func StartWorker(father *kernel.Pid) *kernel.Pid {
	pid, _ := kernel.Start(workerActor, father)
	return pid
}

var workerActor = &kernel.Actor{
	Init: func(ctx *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		var father *kernel.Pid
		if args[0] != nil {
			father = args[0].(*kernel.Pid)
			ctx.Link(father)
		}
		state := worker{
			father:    father,
			client:    &http.Client{},
			sslClient: &http.Client{Transport: tr},
		}
		return &state
	},
	HandleCast: func(ctx *kernel.Context, msg interface{}) {
		state := ctx.State.(*worker)
		switch m := msg.(type) {
		case *requestData:
			if atomic.LoadInt32(&m.close) == 0 {
				client := state.client
				if m.ssl {
					client = state.sslClient
				}
				client.Timeout = time.Duration(m.timeOut) * time.Second
				req, err := http.NewRequest(m.method, m.url, nil)
				if err != nil {
					kernel.ErrorLog("new request error:%s",err)
					responseFunc(nil, err, m)
					return
				}
				if m.method == "POST"{
					if m.contentType == "" {
						m.contentType = "application/ejson"
					}
					req.Header.Set("Content-Type",m.contentType)
				}
				addHeader(req,m)
				rsp, err := client.Do(req)
				responseFunc(rsp, err, m)
			}
			if state.father != nil {
				kernel.Cast(state.father, ctx.Self())
			}
		}
	},
	HandleCall: func(ctx *kernel.Context, request interface{}) interface{} {
		return nil
	},
	Terminate: func(ctx *kernel.Context, reason *kernel.Terminate) {

	},
	ErrorHandler: func(ctx *kernel.Context, err interface{}) bool {
		return true
	},
}

func responseFunc(rsp *http.Response, err error, m *requestData) {
	if err != nil {
		kernel.ErrorLog("httpc %s err:%#v", m.url, err)
		m.ch <- response{ok: false, rsp: nil}
	} else {
		defer rsp.Body.Close()
		if rsp.StatusCode == 200 {
			body, _ := ioutil.ReadAll(rsp.Body)
			m.ch <- response{ok: true, rsp: body}
		} else {
			kernel.ErrorLog("httpc %s rsp code:%d",m.url,rsp.StatusCode)
			m.ch <- response{ok: false, rsp: nil}
		}
	}
}

func addHeader(req *http.Request,m *requestData)  {
	for _,v := range m.headers{
		req.Header.Set(v.key,v.value)
	}
}
