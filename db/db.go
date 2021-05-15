package db

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/liangmanlin/gootp/kernel"
	"log"
)

var GameDB *sql.DB
var LogDB *sql.DB

func Start(DBConfig Config,defSlice []*TabDef, gameDB string,logDB string) {
	Env.dbConfig = DBConfig
	initDef(defSlice)
	cn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8",DBConfig.User,DBConfig.PWD,DBConfig.Host,DBConfig.Port, gameDB) // 主库
	db,err := sql.Open("mysql",cn)
	GameDB = db
	if err != nil {
		log.Panic(err)
	}
	rows,err := db.Query("show tables;")
	if err !=nil {
		log.Panic(err)
	}
	rows.Close()
	db.SetMaxOpenConns(Env.ConnNum)
	db.SetMaxIdleConns(Env.ConnNum)
	startSync(db)
	// 检查数据库表版本号
	tableCheck(db)
	kernel.ErrorLog("db app start on database: %s",gameDB)
	// 连接日志库
	if logDB != "" {
		cnLog := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8",DBConfig.User,DBConfig.PWD,DBConfig.Host,DBConfig.Port,logDB) // 主库
		dbLog,err2 := sql.Open("mysql",cnLog)
		LogDB = dbLog
		if err2 !=nil {
			log.Panic(err2)
		}
	}
}
