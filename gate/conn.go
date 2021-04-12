package gate

import (
	"fmt"
	"github.com/liangmanlin/gootp/kernel"
	"io"
	"net"
	"time"
)

type Conn struct {
	Conn    net.Conn
	head    int
	reading bool
	ctl     chan interface{}
	headBuf []byte
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{Conn: conn, head: 0, ctl: make(chan interface{}, 2)}
}

func (c *Conn) Send(buf []byte) error {
	if c.Conn == nil {
		return fmt.Errorf("socket closed")
	}
	head := c.head
	size := len(buf)
	sendBuf := make([]byte, head, size+head)
	WriteSize(sendBuf, head, size)
	sendBuf = append(sendBuf, buf...)
	_, err := c.Conn.Write(sendBuf)
	return err
}

// 这个函数用来发送经过pb打包的数据,可以减少一次分配内存
func (c *Conn) SendBufHead(buf []byte) error {
	if c.Conn == nil {
		return fmt.Errorf("socket closed")
	}
	_, err := c.Conn.Write(buf)
	return err
}

func (c *Conn)SetHead(head int)  {
	if head != 2 && head != 4 {
		kernel.ErrorLog("start reader error: head = %d", head)
		return
	}
	c.head = head
	c.headBuf = make([]byte,head,head)
}
func (c *Conn)GetHead() int {
	return c.head
}

func (c *Conn) Recv(len int, timeOutMS int) (error, []byte) {
	var err error
	if timeOutMS > 0 {
		err = c.Conn.SetReadDeadline(time.Now().Add(time.Duration(timeOutMS) * time.Millisecond))
		if err != nil {
			return err, nil
		}
	}
	if c.head == 0 {
		buf := make([]byte, len)
		_, err = io.ReadAtLeast(c.Conn, buf, len)
		return err, buf
	}
	_, err = io.ReadAtLeast(c.Conn, c.headBuf, c.head)
	if err != nil {
		return err, nil
	}
	packSize := ReadHead(c.head, c.headBuf)
	pack := make([]byte, packSize)
	_, err = io.ReadAtLeast(c.Conn, pack, packSize)
	return err, pack
}

// 开始异步接收数据,该函数尽量只调用一次
func (c *Conn) StartReader(dest *kernel.Pid) {
	c.StartReaderDecode(dest,nil)
}
func (c *Conn) StartReaderDecode(dest *kernel.Pid,decoder func([]byte)(int,interface{})) {
	if c.head != 2 && c.head != 4 {
		kernel.ErrorLog("start reader error: head = %d", c.head)
		return
	}
	if c.reading {
		c.ctl <- dest
	} else {
		go startReader(dest, c,decoder)
		c.reading = true
	}
}

func (c *Conn) Close() {
	if c.Conn == nil {
		return
	}
	c.ctl <- 1
	_ = c.Conn.Close()
	c.Conn = nil
	close(c.ctl)
}
