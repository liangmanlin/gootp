package httpc

import (
	"github.com/liangmanlin/gootp/kernel"
	"github.com/liangmanlin/gootp/ringbuffer"
	"sync/atomic"
	"time"
)

var server *kernel.Pid

var started = false

var mangerActor = &kernel.Actor{
	Init: func(ctx *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
		server = pid
		state := manager{queue: ringbuffer.NewSingleRingBuffer(10, 100)}
		workerPid := StartWorker(pid)
		state.idle = []*kernel.Pid{workerPid}
		state.poolSize = 1
		state.maxPoolSize = 5
		kernel.SendAfterForever(pid, 5*1000, kernel.Loop{})
		kernel.ErrorLog("httpc_manager started :%s", pid)
		return &state
	},
	HandleCast: func(ctx *kernel.Context, msg interface{}) {
		state := ctx.State.(*manager)
		switch m := msg.(type) {
		case *requestData:
			size := len(state.idle)
			if size > 0 {
				workerPid := state.idle[size-1]
				state.idle = state.idle[:size-1]
				kernel.Cast(workerPid, m)
			} else {
				state.queue.Put(m)
			}
		case kernel.Loop:
			if state.queue.Size() > 0 && state.poolSize < state.maxPoolSize {
				workerPid := StartWorker(ctx.Self())
				ctx.CastSelf(workerPid)
			}
		case *kernel.Pid:
			if req := state.queue.Pop();req !=nil {
				kernel.Cast(m, req)
			} else {
				state.idle = append(state.idle, m)
			}
		case *kernel.PidExit:
			state.poolSize--
			state.idle = delPid(state.idle, m.Pid)
		}
	},
	HandleCall: func(ctx *kernel.Context, request interface{}) interface{} {
		switch r := request.(type) {
		case maxPoolSize:
			ctx.State.(*manager).maxPoolSize = int32(r)
		}
		return nil
	},
	Terminate: func(ctx *kernel.Context, reason *kernel.Terminate) {

	},
	ErrorHandler: func(ctx *kernel.Context, err interface{}) bool {
		return true
	},
}

func SetMaxPoolSize(num int32) {
	confirmStart()
	kernel.Call(server, maxPoolSize(num))
}

func confirmStart() {
	if started {
		return
	}
	kernel.SupStartChild("kernel", &kernel.SupChild{
		ChildType: kernel.SupChildTypeSup,
		ReStart:   true,
		Name:      "httpc_sup",
	})
	kernel.SupStartChild("httpc_sup", &kernel.SupChild{
		ChildType: kernel.SupChildTypeWorker,
		ReStart:   true,
		Name:      "httpc_manager",
		Svr:       mangerActor,
	})
	started = true
}

func Request(pid *kernel.Pid, method, url string, param ...interface{}) ([]byte, bool) {
	confirmStart()
	if pid == nil {
		pid = server
	}
	recv := make(chan response, 1)
	req := &requestData{method: method, url: url, ch: recv, close: 0}
	timeout := transParam(req, param)
	kernel.Cast(pid, req)
	t := time.After(time.Duration(timeout)*time.Second + 100*time.Millisecond)
	defer close(recv)
	select {
	case r := <-recv:
		return r.rsp, true
	case <-t:
		atomic.AddInt32(&req.close, 1) // 给与worker判断是否丢弃
		return nil, false
	}
}

func transParam(req *requestData, param []interface{}) (timeout int32) {
	timeout = 3
	for _, v := range param {
		switch t := v.(type) {
		case useSSL:
			req.ssl = true
		case contentType:
			req.contentType = string(t)
		case bodyType:
			req.body = string(t)
		case timeOut:
			timeout = int32(t)
			req.timeOut = timeout
		case header:
			req.headers = append(req.headers, t)
		}
	}
	return
}

func delPid(list []*kernel.Pid, pid *kernel.Pid) []*kernel.Pid {
	size := len(list)
	for i := 0; i < size; i++ {
		if list[i].GetID() == pid.GetID() {
			list[i] = list[size-1]
			list[size-1] = nil // GC
			list = list[:size-1]
			break
		}
	}
	return list
}
