package httpc

import (
	"fmt"
	"github.com/liangmanlin/gootp/kernel"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	go func() {
		time.Sleep(3 * time.Second)
		kernel.ErrorLog("test init stop now")
		kernel.InitStop()
	}()
	kernel.Env.LogPath = ""
	kernel.KernelStart(func() {
		if body,ok := Get("http://daohang.4399om.com/",nil,3,false);ok{
			fmt.Println(string(body))
		}else{
			t.Errorf("error")
		}
		if body,ok := GetSSL("https://daohang.4399om.com/",nil,3,false);ok{
			fmt.Println(string(body))
		}else{
			t.Errorf("error")
		}
	},nil)
}