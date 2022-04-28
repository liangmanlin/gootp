package gate

import (
	"errors"
	"github.com/lesismal/nbio"
	"github.com/liangmanlin/gootp/bpool"
	"github.com/liangmanlin/gootp/kernel"
	"io"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var (
	ErrSocketActiveMode = errors.New("active mode")
)

type ConnNbio struct {
	mux sync.Mutex
	*nbio.Conn
	recvPid *kernel.Pid

	recvState bool
	recvChan  chan kernel.Empty

	head      int
	totalSize int
	buffer    *bpool.Buff
}

func NewNbConn(conn *nbio.Conn) Conn {
	return &ConnNbio{Conn: conn, recvChan: make(chan kernel.Empty, 1)}
}

func (c *ConnNbio) getHandler() *kernel.Pid {
	return (*kernel.Pid)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&c.recvPid))))
}

// 仅仅被动模式可用
func (c *ConnNbio) Read(buf []byte) (n int, err error) {
	if c.getHandler() != nil {
		err = ErrSocketActiveMode
		return
	}
	c.mux.Lock()
	size := len(buf)
	var rn int
	for n < size {
		if c.Conn == nil {
			err = ErrSocketClosed
			break
		}
		rn, err = c.Conn.Read(buf[n:])
		if rn > 0 {
			n += rn
			// 再继续尝试
		}
		if err != nil {
			if errors.Is(err, syscall.EINTR) {
				continue
			}
			if errors.Is(err, syscall.EAGAIN) {
				// 这里需要阻塞等待数据
				c.recvState = true
				c.mux.Unlock()
				<-c.recvChan
				c.mux.Lock()
				if c.Conn == nil {
					err = ErrSocketClosed
					break
				}
				continue
			} else {
				break
			}
		}
	}
	c.mux.Unlock()
	return
}

func (c *ConnNbio) Recv(len int, timeOutMS int) ([]byte, error) {
	var err error
	if c.getHandler() != nil {
		return nil, ErrSocketActiveMode
	}

	if timeOutMS > 0 {
		err = c.SetReadDeadline(time.Now().Add(time.Duration(timeOutMS) * time.Millisecond))
		if err != nil {
			return nil, err
		}
	}
	if c.head == 0 {
		buf := make([]byte, len)
		_, err = io.ReadAtLeast(c, buf, len)
		return buf, err
	}
	headBuf := make([]byte, c.head)
	_, err = io.ReadAtLeast(c, headBuf, c.head)
	if err != nil {
		return nil, err
	}
	packSize := ReadHead(c.head, headBuf)
	pack := make([]byte, packSize)
	_, err = io.ReadAtLeast(c, pack, packSize)
	c.SetReadDeadline(time.Time{})
	return pack, err
}

func (c *ConnNbio) Send(buf []byte) (int, error) {
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
func (c *ConnNbio) SendBufHead(buf []byte) error {
	if c.Conn == nil {
		return ErrSocketClosed
	}
	_, err := c.Conn.Write(buf)
	return err
}

func (c *ConnNbio) SetHead(head int) {
	if head != 2 && head != 4 {
		kernel.ErrorLog("not support head = %d", head)
		return
	}
	c.head = head
}
func (c *ConnNbio) GetHead() int {
	return c.head
}

// 开始异步接收数据,该函数尽量只调用一次,并且确保没有调用Recv
func (c *ConnNbio) StartReader(dest *kernel.Pid) {
	if c.head != 2 && c.head != 4 {
		kernel.ErrorLog("not support head = %d", c.head)
		return
	}
	if c.Conn == nil {
		kernel.ErrorLog("start reader on close conn")
		return
	}
	c.mux.Lock()
	defer c.mux.Unlock()
	c.recvState = false
	close(c.recvChan)
	c.recvChan = nil
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&c.recvPid)), unsafe.Pointer(dest))
	// 考虑到可能缓冲区还有数据，这里尝试读取一次数据
	c.onRead(c.Conn, dest)
}

// epoll 触发
func (c *ConnNbio) OnRead(conn *nbio.Conn) {
	recvPid := c.getHandler()
	if recvPid == nil {
		c.mux.Lock()
		if c.recvState {
			c.recvState = false
			c.recvChan <- kernel.Empty{}
		}
		c.mux.Unlock()
	} else {
		c.onRead(conn, recvPid)
	}
}

func (c *ConnNbio) onRead(conn *nbio.Conn, recvPid *kernel.Pid) {
	if c.buffer == nil {
		c.buffer = bpool.New(4 * 1024)
	}
	buf := c.buffer.ToBytes()[:c.buffer.Cap()]
	for {
		bSize := c.buffer.Cap() - c.buffer.Size()
		n, err := conn.Read(buf[c.buffer.Size():])
		if n > 0 {
			c.buffer.SetSize(c.buffer.Size() + n)
			buf = c.readBuf(buf, recvPid)
		}
		if errors.Is(err, syscall.EINTR) {
			continue
		}
		if errors.Is(err, syscall.EAGAIN) {
			break
		}
		if err != nil {
			c.CloseWithError(err)
		}
		if n < bSize {
			break
		}
	}
}

func (c *ConnNbio) readBuf(buf []byte, recvPid *kernel.Pid) []byte {
	for {
		if c.totalSize == 0 && c.buffer.Size() >= c.head {
			c.totalSize = ReadHead(c.head, buf) + c.head
			if c.buffer.Cap() < c.totalSize {
				c.buffer = bpool.New(c.totalSize).Append(buf[:c.buffer.Size()]...)
				buf = c.buffer.ToBytes()[:c.buffer.Cap()]
			}
		}
		size := c.buffer.Size()
		if c.totalSize > 0 && size >= c.totalSize {
			if c.totalSize == size {
				recvPid.Cast(c.buffer)
				c.buffer = bpool.New(4 * 1024)
				buf = c.buffer.ToBytes()[:c.buffer.Cap()]
				c.totalSize = 0
				break
			} else {
				tmp := c.buffer
				c.buffer = bpool.NewBuf(buf[c.totalSize:size])
				buf = c.buffer.ToBytes()[:c.buffer.Cap()]
				tmp.SetSize(c.totalSize)
				c.totalSize = 0
				recvPid.Cast(tmp)
			}
		} else {
			break
		}
	}
	return buf
}

func (c *ConnNbio) Close() error {
	if c.Conn == nil {
		return nil
	}
	err := c.Conn.Close()
	c.Conn = nil
	c.mux.Lock()
	if c.recvState {
		c.recvState = false
		close(c.recvChan)
	}
	c.mux.Unlock()
	return err
}
