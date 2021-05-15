package httpc

import (
	"container/list"
	"github.com/liangmanlin/gootp/kernel"
	"sync/atomic"
	"time"
	"unsafe"
)

var server *kernel.Pid

var started = false

var mangerActor = &kernel.Actor{
	Init: func(ctx *kernel.Context, pid *kernel.Pid, args ...interface{}) unsafe.Pointer {
		server = pid
		state := manager{queue: list.New()}
		workerPid := StartWorker(pid)
		state.idle = []*kernel.Pid{workerPid}
		state.poolSize = 1
		state.maxPoolSize = 5
		kernel.SendAfter(kernel.TimerTypeForever, pid, 5*1000, true)
		return unsafe.Pointer(&state)
	},
	HandleCast: func(ctx *kernel.Context, msg interface{}) {
		state := (*manager)(ctx.State)
		switch m := msg.(type) {
		case *requestData:
			size := len(state.idle)
			if size > 0 {
				workerPid := state.idle[size-1]
				state.idle = state.idle[:size-1]
				kernel.Cast(workerPid, m)
			} else {
				state.queue.PushBack(m)
			}
		case bool:
			if e := state.queue.Front(); e != nil && state.poolSize < state.maxPoolSize {
				workerPid := StartWorker(ctx.Self())
				ctx.CastSelf(workerPid)
			}
		case *kernel.Pid:
			if e := state.queue.Front(); e != nil {
				kernel.Cast(m, e.Value)
				state.queue.Remove(e)
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
			(*manager)(ctx.State).maxPoolSize = int32(r)
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

func request(pid *kernel.Pid,method, url, body, contentType string, timeOut int32, ssl bool) ([]byte, bool) {
	confirmStart()
	recv := make(chan response, 1)
	req := &requestData{method: method, url: url, body: body, contentType: contentType, ssl: ssl, ch: recv, close: 0}
	kernel.Cast(pid, req)
	t := time.After(time.Duration(timeOut) * time.Second)
	defer close(recv)
	select {
	case r := <-recv:
		return r.rsp, true
	case <-t:
		atomic.AddInt32(&req.close, 1) // 给与worker判断是否丢弃
		return nil, false
	}
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
