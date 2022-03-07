package gate

import (
	"github.com/liangmanlin/gootp/bpool"
	"github.com/liangmanlin/gootp/kernel"
	"io"
	"runtime/debug"
	"time"
)

func startReader(conn *ConnNet) {
	defer func() {
		p := recover()
		if p != nil {
			kernel.ErrorLog("catch error:%s,Stack:%s", p, debug.Stack())
		}
	}() // catch the error
	c := conn.Conn
	if err := c.SetReadDeadline(time.Time{}); err != nil { // 规避超时
		kernel.Cast(conn.handler, &TcpError{ErrType: ErrDeadLine, Err: err})
		return
	}
	head := conn.head
	headBuf := make([]byte, head)
	pack := bpool.New(100) // 先默认申请一个小的
	var packSize int
	for {
		_, err := io.ReadAtLeast(c, headBuf, head)
		if err != nil {
			kernel.Cast(conn.getHandler(), &TcpError{ErrType: ErrReadErr, Err: err})
			break
		}
		packSize = ReadHead(head, headBuf)
		if packSize > 0 {
			pack = bpool.New(packSize)
			_, err = pack.Read(c, packSize)
			handler := conn.getHandler()
			if err != nil {
				kernel.Cast(handler, &TcpError{ErrType: ErrReadErr, Err: err})
				goto end
			}
			kernel.Cast(handler, pack)
		} else {
			kernel.Cast(conn.getHandler(), []byte{})
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
