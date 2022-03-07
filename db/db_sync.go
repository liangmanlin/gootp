package db

import (
	"github.com/liangmanlin/gootp/args"
	"github.com/liangmanlin/gootp/kernel"
	"log"
	"time"
)

type selectRow struct {
	index int64
	table string
	key   []interface{}
}

type selectAll struct {
	index int64
	table string
	key   []interface{}
}

type updateRow struct {
	index int64
	table string
	data  interface{}
}

type insertRow struct {
	index int64
	table string
	data  interface{}
}

type deleteRow struct {
	index int64
	table string
	data  interface{}
}
type deletePKey struct {
	index int64
	table string
	pkey  []interface{}
}

// 查询单行
func (g *Group) SyncSelectRow(c Call, tab string, indexKey int64, key ...interface{}) interface{} {
	proc := g.syncPool[indexKey%g.syncNum]
	var ok bool
	var rs interface{}
	q := &selectRow{index: indexKey, table: tab, key: key}
	ok, rs = c(proc, q)
	if ok {
		return rs
	}
	kernel.ErrorLog("select %s error:%#v", tab, rs)
	return nil
}

// 查询多条记录
func (g *Group) SyncSelect(c Call, tab string, indexKey int64, key ...interface{}) []interface{} {
	proc := g.syncPool[indexKey%g.syncNum]
	var ok bool
	var rs interface{}
	q := &selectAll{index: indexKey, table: tab, key: key}
	ok, rs = c(proc, q)
	if ok {
		return rs.([]interface{})
	}
	kernel.ErrorLog("select %s error:%#v", tab, rs)
	return nil
}

func (g *Group) SyncUpdate(tab string, indexKey int64, data interface{}) {
	proc := g.syncPool[indexKey%g.syncNum]
	kernel.Cast(proc, &updateRow{index: indexKey, table: tab, data: kernel.DeepCopy(data)})
}

func (g *Group) SyncInsert(tab string, indexKey int64, data interface{}) {
	proc := g.syncPool[indexKey%g.syncNum]
	if g.mode != MODE_MULTI_INSERT {
		data = kernel.DeepCopy(data)
	}
	kernel.Cast(proc, &insertRow{index: indexKey, table: tab, data: data})
}

func (g *Group) SyncDelete(tab string, indexKey int64, data interface{}) {
	proc := g.syncPool[indexKey%g.syncNum]
	kernel.Cast(proc, &deleteRow{index: indexKey, table: tab, data: kernel.DeepCopy(data)})
}

func (g *Group) SyncDeletePKey(tab string, indexKey int64, pkey ...interface{}) {
	proc := g.syncPool[indexKey%g.syncNum]
	kernel.Cast(proc, &deletePKey{index: indexKey, table: tab, pkey: pkey})
}

func startSync(group *Group, num int) {
	group.syncPool = make([]*kernel.Pid, num)
	group.syncNum = int64(num)
	for i := 0; i < num; i++ {
		pid, err := kernel.Start(dbSyncActor, group, i)
		if err != nil {
			log.Panic(err)
		}
		group.syncPool[i] = pid
	}
}

type dbSync struct {
	g      *Group
	worker *kernel.Pid
	cache  map[string][]interface{}
}

var dbSyncActor = &kernel.Actor{
	Init: func(context *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
		d := dbSync{}
		d.g = args[0].(*Group)
		if d.g.mode == MODE_MULTI_INSERT {
			syncTime := getSyncTime()
			go func() {
				i := args[1].(int)
				time.Sleep(time.Duration(i+1) * 5 * time.Second)
				kernel.SendAfterForever(pid, syncTime, kernel.Loop{})
			}()
			d.cache = make(map[string][]interface{}, 10)
			// start child
			d.worker, _ = context.StartLinkOpt(dbSyncWorker, kernel.ActorOpt(kernel.ActorChanCacheSize(1000)), pid, d.g)
		}
		return &d
	},
	HandleCast: func(context *kernel.Context, msg interface{}) {
		d := context.State.(*dbSync)
		switch m := msg.(type) {
		case *updateRow:
			d.g.ModUpdate(m.table, m.data)
		case *insertRow:
			if d.g.mode == MODE_MULTI_INSERT {
				// 缓存起来，后续批量插入
				d.insertCache(m.table, m.data)
			} else {
				d.g.ModInsert(m.table, m.data)
			}
		case *deleteRow:
			d.g.ModDelete(m.table, m.data)
		case *deletePKey:
			d.g.ModDeletePKey(m.table, m.pkey...)
		case kernel.Loop:
			d.multiInsert()
		case syncData:
			if d.g.mode == MODE_MULTI_INSERT {
				// 缓存起来，后续批量插入
				d.insertCache(m.tab, m.dataList...)
			}
		}
	},
	HandleCall: func(context *kernel.Context, request interface{}) interface{} {
		d := context.State.(*dbSync)
		switch req := request.(type) {
		case *selectRow:
			return d.selectRow(req.table, req.key)
		case *selectAll:
			return d.selectAll(req.table, req.key)
		}
		return nil
	},
	Terminate: func(context *kernel.Context, reason *kernel.Terminate) {

	},
	ErrorHandler: func(context *kernel.Context, err interface{}) bool {
		return true
	},
}

func (d *dbSync) selectRow(tab string, key []interface{}) interface{} {
	return d.g.ModSelectRow(tab, key...)
}

func (d *dbSync) selectAll(tab string, key []interface{}) interface{} {
	return d.g.ModSelectAll(tab, key...)
}

func (d *dbSync) insertCache(tab string, data ...interface{}) {
	s := d.cache[tab]
	s = append(s, data...)
	d.cache[tab] = s
}
func (d *dbSync) multiInsert() {
	for tab, dl := range d.cache {
		if len(dl) > 0 {
			d.worker.Cast(syncData{tab: tab, dataList: dl})
		}
		delete(d.cache, tab)
	}
}

func getSyncTime() int64 {
	if v, ok := args.GetInt("db_sync_time"); ok {
		return int64(v) * 1000
	}
	return 30 * 1000
}
