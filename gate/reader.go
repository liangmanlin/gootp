package gate

import (
	"github.com/liangmanlin/gootp/kernel"
	"go/types"
	"io"
	"time"
)

func startReader(dest *kernel.Pid, conn *Conn, decoder func([]byte) (int, interface{})) {
	defer kernel.Catch() // catch the error
	c := conn.Conn
	if err := c.SetReadDeadline(time.Time{}); err != nil { // 规避超时
		kernel.Cast(dest, &TcpError{Err: err})
		return
	}
	head := conn.head
	headBuf := make([]byte, head)
	var pack []byte
	var packSize int
	for {
		select {
		case ctl := <-conn.ctl:
			if handleCtl(conn, ctl, &head, &dest) != 0 {
				goto end
			}
		default:
			_, err := io.ReadAtLeast(c, headBuf, head)
			if err != nil {
				if len(conn.ctl) == 0 {
					kernel.Cast(dest, &TcpError{Err: err})
				}
				goto end
			}
			packSize = ReadHead(head, headBuf)
			if packSize > 0 {
				if decoder == nil || len(pack) < packSize { // todo 这里可以通过闭包提高性能
					pack = make([]byte, packSize) // TODO 后续需要解决频繁申请内存的垃圾回收问题
				}
				_, err = io.ReadAtLeast(c, pack, packSize)
				if err != nil {
					kernel.Cast(dest, &TcpError{Err: err})
					goto end
				}
				if decoder != nil {
					protoID, proto := decoder(pack)
					// 这里没有使用指针，减少一些gc对象
					kernel.Cast(dest, Pack{ProtoID: protoID, Proto: proto})
				} else {
					kernel.Cast(dest, pack)
				}
			} else {
				kernel.Cast(dest, []byte{})
			}
		}
	}
end:
}

func ReadHead(head int, buf []byte) (packSize int) {
	switch head {
	case 2:
		packSize = (int(buf[0]) << 8) + int(buf[1])
	case 4:
		packSize = (int(buf[0]) << 24) + (int(buf[1]) << 16) + (int(buf[2]) << 8) + int(buf[3])
	}
	return
}

func handleCtl(conn *Conn, ctl interface{}, headPtr *int, destPtr **kernel.Pid) int {
	if ctl == nil {
		return 1
	}
	switch c := ctl.(type) {
	case int:
		return c
	case *kernel.Pid:
		*headPtr = conn.head
		*destPtr = c
	case types.Nil:
		return 1
	}
	return 0
}
