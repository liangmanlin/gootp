package httpc

import (
	"crypto/tls"
	"github.com/liangmanlin/gootp/kernel"
	"io/ioutil"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

func StartWorker(father *kernel.Pid) *kernel.Pid {
	pid,_ := kernel.Start(workerActor,father)
	return pid
}

var workerActor = &kernel.Actor{
	Init: func(ctx *kernel.Context, pid *kernel.Pid, args ...interface{}) unsafe.Pointer {
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		var father *kernel.Pid
		if args[0] != nil {
			father = args[0].(*kernel.Pid)
			ctx.Link(father)
		}
		state := worker{
			father: father,
			client: &http.Client{},
			sslClient:&http.Client{Transport: tr},
		}
		return unsafe.Pointer(&state)
	},
	HandleCast: func(ctx *kernel.Context, msg interface{}) {
		state := (*worker)(ctx.State)
		switch m:=msg.(type) {
		case *requestData:
			if atomic.LoadInt32(&m.close) == 0 {
				client := state.client
				if m.ssl {
					client = state.sslClient
				}
				client.Timeout = 3*time.Second
				if m.method == "GET" {
					rsp, err := client.Get(m.url)
					responseFunc(rsp,err,m)
				}else if m.method == "POST" {
					if m.contentType == "" {
						m.contentType = "application/json"
					}
					rsp, err := client.Post(m.url,m.contentType,strings.NewReader(m.body))
					responseFunc(rsp,err,m)
				}
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

func responseFunc(rsp *http.Response,err error,m *requestData)  {
	if err != nil {
		kernel.ErrorLog("httpc %s err:%#v",m.url,err)
		m.ch <- response{ok: false,rsp: nil}
	}else {
		defer rsp.Body.Close()
		if rsp.StatusCode == 200 {
			body, _ := ioutil.ReadAll(rsp.Body)
			m.ch <- response{ok: true, rsp: body}
		}else{
			m.ch <- response{ok: false, rsp: nil}
		}
	}
}