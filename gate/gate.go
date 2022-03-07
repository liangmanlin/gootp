package gate

import (
	"github.com/liangmanlin/gootp/kernel"
	"log"
)

const gateSupName = "gate_sup"

var addrMap = make(map[string]string)

func Start(name string, handler *kernel.Actor, port int, opt ...optFun) {
	start(name,handler,port,opt)
}

func start(name string, handler *kernel.Actor, port int, opt []optFun)  {
	ensureSupStart()
	a := &app{name: name,handler: handler,port: port,opt: opt}
	kernel.AppStart(a)
}

func Stop(name string) {
	kernel.AppStop(name)
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
