package gate

import (
	"errors"
	"github.com/liangmanlin/gootp/kernel"
	"io"
	"net"
	"sync/atomic"
	"time"
	"unsafe"
)

type ConnNet struct {
	net.Conn

	head    int
	handler *kernel.Pid
	headBuf []byte
}

var (
	ErrSocketClosed  = errors.New("socket closed")
	ErrSocketReading = errors.New("socket is reading")
)

func NewConn(conn net.Conn) Conn {
	return &ConnNet{Conn: conn, head: 0}
}

func (c *ConnNet) Send(buf []byte) (int, error) {
	if c.Conn == nil {
		return 0, ErrSocketClosed
	}
	if c.head == 0 {
		n, err := c.Conn.Write(buf)
		return n, err
	}
	head := c.head
	size := len(buf)
	sendBuf := make([]byte, head, size+head)
	WriteSize(sendBuf, head, size)
	sendBuf = append(sendBuf, buf...)
	n, err := c.Conn.Write(sendBuf)
	return n, err
}

// 这个函数用来发送经过pb打包的数据,可以减少一次分配内存
func (c *ConnNet) SendBufHead(buf []byte) error {
	if c.Conn == nil {
		return ErrSocketClosed
	}
	_, err := c.Conn.Write(buf)
	return err
}

func (c *ConnNet) SetHead(head int) {
	if head != 2 && head != 4 {
		kernel.ErrorLog("not support head = %d", head)
		return
	}
	c.head = head
	c.headBuf = make([]byte, head, head)
}
func (c *ConnNet) GetHead() int {
	return c.head
}

func (c *ConnNet) Recv(len int, timeOutMS int) ([]byte, error) {
	if c.handler != nil {
		return nil, ErrSocketReading
	}
	var err error
	if timeOutMS > 0 {
		err = c.Conn.SetReadDeadline(time.Now().Add(time.Duration(timeOutMS) * time.Millisecond))
		if err != nil {
			return nil, err
		}
	}
	if c.head == 0 {
		buf := make([]byte, len)
		_, err = io.ReadAtLeast(c.Conn, buf, len)
		return buf, err
	}
	_, err = io.ReadAtLeast(c.Conn, c.headBuf, c.head)
	if err != nil {
		return nil, err
	}
	packSize := ReadHead(c.head, c.headBuf)
	pack := make([]byte, packSize)
	_, err = io.ReadAtLeast(c.Conn, pack, packSize)
	return pack, err
}

// 开始异步接收数据,该函数尽量只调用一次
// 没有考虑并发的情况，使用者自行规避
func (c *ConnNet) StartReader(dest *kernel.Pid) {
	if c.head != 2 && c.head != 4 {
		kernel.ErrorLog("start reader error: head = %d", c.head)
		return
	}
	if c.handler !=nil {
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&c.handler)),unsafe.Pointer(dest))
	} else {
		c.handler = dest
		go startReader(c)
	}
}

func (c *ConnNet) Close() error {
	if c.Conn == nil {
		return nil
	}
	return c.Conn.Close()
}

func (c *ConnNet)getHandler() *kernel.Pid {
	return (*kernel.Pid)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&c.handler))))
}