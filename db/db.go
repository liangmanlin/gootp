package db

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/liangmanlin/gootp/kernel"
	"log"
)

var groups []*Group

func Start(idx int,DBConfig Config,defSlice []*TabDef, dbName string,syncNum int,mode Mode) *Group {
	g := newGroup(idx)
	g.mode = mode
	Env.dbConfig = DBConfig
	initDef(g,defSlice)
	cn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8",DBConfig.User,DBConfig.PWD,DBConfig.Host,DBConfig.Port, dbName)
	db,err := sql.Open("mysql",cn)
	g.db = db
	if err != nil {
		log.Panic(err)
	}
	rows,err := db.Query("show tables;")
	if err !=nil {
		log.Panic(err)
	}
	rows.Close()
	connNum := DBConfig.ConnNum
	if connNum == 0 {
		connNum = Env.ConnNum
	}
	db.SetMaxOpenConns(connNum)
	db.SetMaxIdleConns(connNum)
	startSync(g,syncNum)
	// 检查数据库表版本号
	tableCheck(g)
	kernel.ErrorLog("db start on database: %s",dbName)
	return g
}

func newGroup(idx int) *Group {
	if cap(groups) <= idx{
		groups = append(groups,&Group{})
	}
	if groups[idx].db != nil{
		log.Panicf("duplicate db idx:%d",idx)
	}
	return groups[idx]
}

func GetGroup(idx int) *Group {
	return groups[idx]
}