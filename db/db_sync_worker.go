package db

import (
	"database/sql/driver"
	"errors"
	"github.com/liangmanlin/gootp/gutil"
	"github.com/liangmanlin/gootp/kernel"
)

type syncWorker struct {
	father *kernel.Pid
	g      *Group
}

var dbSyncWorker = &kernel.Actor{
	Init: func(ctx *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
		state := syncWorker{}
		state.father = args[0].(*kernel.Pid)
		state.g = args[1].(*Group)
		return &state
	},
	HandleCast: func(ctx *kernel.Context, msg interface{}) {
		state := ctx.State.(*syncWorker)
		switch m := msg.(type) {
		case syncData:
			state.multiInsert(m.tab,m.dataList)
		}
	},
	HandleCall: func(ctx *kernel.Context, request interface{}) interface{} {
		return nil
	},
	ErrorHandler: func(ctx *kernel.Context, err interface{}) bool {
		return true
	},
	Terminate: func(ctx *kernel.Context, reason *kernel.Terminate) {
		kernel.ErrorLog("sync worker terminate")
	},
}

func (s *syncWorker)multiInsert(tab string,dataList []interface{})  {
	size := len(dataList)
	if size == 0 {
		return
	}
	// 最多300条插入
	const div_size = 300
	var start, end int32
	var fail []interface{}
	div := gutil.Ceil(float32(size) / div_size)
	for i := int32(0); i < div; i++ {
		start = i * div_size
		end = gutil.MinInt32((i+1)*div_size, int32(size))
		tmp := dataList[start:end]
		_, err := s.g.ModMultiInsert(tab, tmp)
		if err != nil && errors.Is(err, driver.ErrBadConn) {
			// 说明断开了，我们等待重连
			fail = append(fail, tmp...)
		}
	}
	if len(fail) > 0 {
		s.father.Cast(syncData{tab: tab,dataList: fail})
	}
}
