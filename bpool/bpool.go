package bpool

import (
	"io"
	"math/bits"
	"sync"
)

/*
	提供一个用于接收网络数据的缓冲池
	小于64k的数据将会被重用

	为什么是64k？
	因为uint16的最大值是64k
*/
const (
	min_size  = 32
	max_size  = 64 * 1024
	pool_size = 12 //32,64,128,256,512,1k,2k,4k,8k,16k,32k,64k
)

var pool [pool_size]sync.Pool

type Buff struct {
	b       []byte
	poolIdx int8
}

func init() {
	for i := 0; i < pool_size; i++ {
		size := getSize(i)
		idx := i
		pool[i].New = func() interface{} {
			return &Buff{poolIdx: int8(idx), b: make([]byte, size)}
		}
	}
}

func New(size int) *Buff {
	if size >= max_size {
		// 理论上很少这么大的数据,重用意义不大，所以，直接申请
		b := make([]byte, 0, size)
		return &Buff{poolIdx: -1, b: b}
	}
	idx := getIndex(size)
	buf := pool[idx].Get().(*Buff)
	buf.b = buf.b[0:0]
	return buf
}

func NewBuf(buf []byte) *Buff {
	size := len(buf)
	b := New(size)
	copy(b.b[0:size], buf)
	b.b = b.b[:size]
	return b
}

func getIndex(size int) int {
	if size < min_size {
		return 0
	}
	return bits.Len32(uint32(size-1)) - 5
}

// 调用该方法后，不能继续使用buff，否则有不可预料的bug
func (b *Buff) Free() {
	if b.poolIdx < 0 {
		return
	}
	pool[b.poolIdx].Put(b)
}

func (b *Buff) Size() int {
	return len(b.b)
}

func (b *Buff) Cap() int {
	return cap(b.b)
}

func (b *Buff) Reset() {
	b.b = b.b[0:0]
}

func (b *Buff) Read(r io.Reader, size int) (n int, err error) {
	if cap(b.b) < size {
		return 0, io.ErrShortBuffer
	}
	b.b = b.b[0:size]
	for n < size && err == nil {
		var nn int
		nn, err = r.Read(b.b[n:])
		n += nn
	}
	if n >= size {
		err = nil
	} else if n > 0 && err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return
}

func (b *Buff) Append(buf ...byte) *Buff {
	totalSize := len(buf) + b.Size()
	if totalSize > b.Cap() {
		newCache := New(totalSize)
		newCache = newCache.Append(b.b...).Append(buf...)
		b.Free()
		return newCache
	}
	b.b = append(b.b, buf...)
	return b
}

func (b *Buff) ToBytes() []byte {
	return b.b
}

func (b *Buff) Copy() (buf []byte) {
	return append(buf, b.b...)
}

func (b *Buff) SetSize(size int) {
	b.b = b.b[:size]
}

func getSize(i int) int {
	return min_size << i
}

func ReadAll(r io.Reader, size int) (bp *Buff, err error) {
	const maxAppendSize = 1024 * 1024 * 4
	b := New(size)
	var n int
	for {
		n, err = r.Read(b.b[len(b.b):cap(b.b)])
		if n > 0 {
			b.b = b.b[:len(b.b)+n]
		}
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}
		if len(b.b) == cap(b.b) {
			l := len(b.b)
			al := l
			if al > maxAppendSize {
				al = maxAppendSize
			}
			tmp := b
			b = New(l + al).Append(b.b...)
			tmp.Free()
		}
	}
}
