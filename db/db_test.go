package db

import (
	"github.com/liangmanlin/gootp/kernel"
	"testing"
	"time"
)

type Account struct {
	Account			string
	AgentID			int32
	ServerID		int32
	LastRole		int64
	LastOffline     int32
	BanTime         int32
	OT				[]int32
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
		config := Config{Host: "192.168.24.128", Port: 3306, User: "tttt", PWD: "tttt"}
		Start(config,tabSlice,"pkfr2","pkfr2_log")
		if rs := ModSelectRow(GameDB,"account2","b");rs == nil {
			ModInsert(GameDB,"account2",&Account{Account: "b",OT: []int32{1,2}})
		}else{
			kernel.ErrorLog("%#v",rs)
			rs.(*Account).AgentID = 1
			rs.(*Account).OT = []int32{1,2,3}
			ModUpdate(GameDB,"account2",rs)
		}
	},nil)
}
