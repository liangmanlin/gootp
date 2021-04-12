package gate

import (
	"fmt"
	"github.com/liangmanlin/gootp/kernel"
	"log"
)

const gateSupName = "gate_sup"

var addrMap = make(map[string]string)

func Start(name string, handler *kernel.Actor, port int, opt ...interface{}) {
	ensureSupStart()
	supName := fmt.Sprintf("gate_child_sup_%s", name)
	child := &kernel.SupChild{Name: supName, ReStart: false, ChildType: kernel.SupChildTypeSup}
	_, childSup := kernel.SupStartChild(gateSupName, child)
	clientSup := fmt.Sprintf("gate_client_sup_%s", name)
	child = &kernel.SupChild{Name: clientSup, ReStart: false, ChildType: kernel.SupChildTypeSup}
	_, csPid := kernel.SupStartChild(supName, child)
	// 启动侦听进程
	listenerName := fmt.Sprintf("gate_listener_%s", name)
	args := kernel.MakeArgs(name, handler, port, csPid, childSup, opt)
	child = &kernel.SupChild{Name: listenerName, ReStart: true, ChildType: kernel.SupChildTypeWorker, Svr: listenerActor, InitArgs: args}
	err, _ := kernel.SupStartChild(supName, child)
	if err != nil {
		kernel.ErrorLog("%#v", err)
		log.Panic(err)
	}
	if port > 0 {
		kernel.ErrorLog("[%s] listening on port: [0.0.0.0:%d]", name, port)
	}
}

func Stop() {
	kernel.SupStop(gateSupName)
	kernel.ErrorLog("gate stopped")
}

func ensureSupStart() {
	if kernel.WhereIs(gateSupName) == nil {
		child := &kernel.SupChild{Name: gateSupName, ReStart: true, ChildType: kernel.SupChildTypeSup}
		if err, _ := kernel.SupStartChild("kernel", child); err != nil {
			log.Panic(err)
		}
	}
}

func WriteSize(buf []byte, head int, size int) {
	switch head {
	case 2:
		buf[0] = uint8(size >> 8)
		buf[1] = uint8(size)
	case 4:
		buf[0] = uint8(size >> 24)
		buf[1] = uint8(size >> 16)
		buf[2] = uint8(size >> 8)
		buf[3] = uint8(size)
	}
}

func GetAddr(flag string) string {
	return addrMap[flag]
}
