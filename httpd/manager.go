package httpd

import (
	"github.com/liangmanlin/gootp/gutil"
	"github.com/liangmanlin/gootp/kernel"
	"github.com/liangmanlin/gootp/ringbuffer"
	"math/bits"
)

type managerState struct {
	workerNum    int
	maxWorkerNum int
	engine       *Engine
	worker       *ringbuffer.SingleRingBuffer
	wait         *ringbuffer.SingleRingBuffer
	//requestCount uint64
}

var manager = &kernel.Actor{
	Init: func(ctx *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
		state := managerState{}
		state.maxWorkerNum = args[0].(int)
		state.engine = args[1].(*Engine)
		const minW = 128
		max := 1 << bits.Len(uint(state.maxWorkerNum-1))
		max = int(gutil.MaxInt32(minW, int32(max)))
		min := int(gutil.MinInt32(minW, int32(max)))
		kernel.ErrorLog("manager [%s] max worker num : %d",pid, max)
		state.worker = ringbuffer.NewSingleRingBuffer(min, max)
		state.wait = ringbuffer.NewSingleRingBuffer(minW, 2048)
		return &state
	},
	HandleCast: func(ctx *kernel.Context, msg interface{}) {
		state := ctx.State.(*managerState)
		switch m := msg.(type) {
		case *Request:
			state.newRequest(m, ctx)
		case *kernel.Pid:
			if req := state.wait.Pop(); req != nil {
				m.Cast(req)
			} else {
				state.worker.Put(m)
			}
		}
	},
	HandleCall: func(ctx *kernel.Context, request interface{}) interface{} {
		return nil
	},
	ErrorHandler: func(ctx *kernel.Context, err interface{}) bool {
		return true
	},
	Terminate: func(ctx *kernel.Context, reason *kernel.Terminate) {
		kernel.ErrorLog("httpd manager [%s] terminate", ctx.State.(*managerState).engine.name)
	},
}

func (m *managerState) newRequest(req *Request, ctx *kernel.Context) {
	if m.worker.Size() > 0 {
		m.worker.Pop().(*kernel.Pid).Cast(req)
	} else if m.workerNum < m.maxWorkerNum {
		m.newWorker(ctx.Self()).Cast(req)
	} else {
		m.wait.Put(req)
	}
}

func (m *managerState) newWorker(self *kernel.Pid) *kernel.Pid {
	worker, _ := kernel.Start(workerActor, self,m.engine)
	return worker
}
