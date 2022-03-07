package db

import (
	"github.com/liangmanlin/gootp/kernel"
	"testing"
	"time"
)

type Account struct {
	Account     string
	AgentID     int32
	ServerID    int32
	LastRole    int64
	LastOffline int32
	BanTime     int32
	OT          []int32
}

var tabSlice = []*TabDef{
	{Name: "account2", DataStruct: &Account{}, Pkey: []string{"Account"}, Keys: []string{"AgentID"}},
}

func TestStart(t *testing.T) {
	go func() {
		time.Sleep(3 * time.Second)
		kernel.ErrorLog("test init stop now")
		kernel.InitStop()
	}()
	kernel.Env.LogPath = ""
	kernel.KernelStart(func() {
		config := Config{Host: "127.0.0.1", Port: 3306, User: "tttt", PWD: "tttt"}
		g := Start(1, config, tabSlice, "pkfr2", 3, MODE_NORMAL)
		if rs := g.ModSelectRow("account2", "b"); rs == nil {
			g.ModInsert("account2", &Account{Account: "b", OT: []int32{1, 2}})
		} else {
			kernel.ErrorLog("%#v", rs)
			rs.(*Account).AgentID = 1
			rs.(*Account).OT = []int32{1, 2, 3}
			g.ModUpdate("account2", rs)
		}
	}, nil)
}
