package db

import (
	"database/sql"
	"github.com/liangmanlin/gootp/kernel"
	"log"
	"unsafe"
)

var syncPool []*kernel.Pid

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
func SyncSelectRow(context *kernel.Context, tab string, indexKey int64, key ...interface{}) interface{} {
	proc := syncPool[indexKey%Env.SyncNum]
	var ok bool
	var rs interface{}
	q := &selectRow{index: indexKey, table: tab, key: key}
	if context != nil {
		ok, rs = context.Call(proc, q)
	} else {
		ok, rs = kernel.Call(proc, q)
	}
	if ok {
		return rs
	}
	kernel.ErrorLog("select %s error:%#v", tab, rs)
	return nil
}

// 查询多条记录
func SyncSelect(context *kernel.Context, tab string, indexKey int64, key ...interface{}) []interface{} {
	proc := syncPool[indexKey%Env.SyncNum]
	var ok bool
	var rs interface{}
	q := &selectAll{index: indexKey, table: tab, key: key}
	if context != nil {
		ok, rs = context.Call(proc, q)
	} else {
		ok, rs = kernel.Call(proc, q)
	}
	if ok {
		return rs.([]interface{})
	}
	kernel.ErrorLog("select %s error:%#v", tab, rs)
	return nil
}

func SyncUpdate(tab string, indexKey int64, data interface{}) {
	proc := syncPool[indexKey%Env.SyncNum]
	kernel.Cast(proc, &updateRow{index: indexKey, table: tab, data: kernel.DeepCopy(data)})
}

func SyncInsert(tab string, indexKey int64, data interface{}) {
	proc := syncPool[indexKey%Env.SyncNum]
	kernel.Cast(proc, &insertRow{index: indexKey, table: tab, data: kernel.DeepCopy(data)})
}

func SyncDelete(tab string, indexKey int64, data interface{}) {
	proc := syncPool[indexKey%Env.SyncNum]
	kernel.Cast(proc, &deleteRow{index: indexKey, table: tab, data: kernel.DeepCopy(data)})
}

func SyncDeletePKey(tab string, indexKey int64, pkey ...interface{}) {
	proc := syncPool[indexKey%Env.SyncNum]
	kernel.Cast(proc, &deletePKey{index: indexKey, table: tab, pkey: pkey})
}

func startSync(db *sql.DB) {
	syncPool = make([]*kernel.Pid, Env.SyncNum)
	for i := int64(0); i < Env.SyncNum; i++ {
		pid, err := kernel.Start(dbSyncActor, db)
		if err != nil {
			log.Panic(err)
		}
		syncPool[i] = pid
	}
}

type dbSync struct {
	db *sql.DB
}

var dbSyncActor = &kernel.Actor{
	Init: func(context *kernel.Context, pid *kernel.Pid, args ...interface{}) unsafe.Pointer {
		d := dbSync{}
		d.db = args[0].(*sql.DB)
		return unsafe.Pointer(&d)
	},
	HandleCast: func(context *kernel.Context, msg interface{}) {
		d := (*dbSync)(context.State)
		switch m := msg.(type) {
		case *updateRow:
			ModUpdate(d.db, m.table, m.data)
		case *insertRow:
			ModInsert(d.db, m.table, m.data)
		case *deleteRow:
			ModDelete(d.db, m.table, m.data)
		case *deletePKey:
			ModDeletePKey(d.db, m.table, m.pkey...)
		}
	},
	HandleCall: func(context *kernel.Context, request interface{}) interface{} {
		d := (*dbSync)(context.State)
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
	return ModSelectRow(d.db, tab, key...)
}
func (d *dbSync) selectAll(tab string, key []interface{}) interface{} {
	return ModSelectAll(d.db, tab, key...)
}
