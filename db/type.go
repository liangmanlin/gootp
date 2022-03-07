package db

import (
	"database/sql"
	"github.com/liangmanlin/gootp/kernel"
)

type env struct {
	dbConfig    Config
	ConnNum     int  `command:"db_conn_num"`
	IsOpenCache bool `command:"db_is_open_cache"`
}

type Config struct {
	Host    string
	Port    int
	User    string
	PWD     string
	ConnNum int
}

type Mode int

const (
	MODE_NORMAL       Mode = iota + 1
	MODE_MULTI_INSERT      //适用是日志库，有批量插入缓存
)

type Group struct {
	db       *sql.DB
	dbTabDef map[string]*TabDef
	mode     Mode
	syncNum  int64
	syncPool []*kernel.Pid
}

type Call func(pid *kernel.Pid, req interface{}) (bool, interface{})

type syncData struct {
	tab      string
	dataList []interface{}
}
