package ringbuffer

import (
	"runtime"
	"sync/atomic"
)

const is_expand = 1 << 31

/*
	暂时没有实现，不要使用

	通用ring buffer，这里是实现无锁队列
	因为每个队列成员都多了一个uint32，所以内存会比单线程的大
	和channel不同,会实现动态扩容，不会通知消费者有消息(TODO 后续可以研究实现)
*/

type wait struct{}

type bufCache struct {
	count uint32      // 这里使用一个标记，告诉消费者，这里是否开始设置值
	value interface{} // TODO 这里可以换成unsafe.Pointer
}

type RingBuffer struct {
	tail  uint32
	head  uint32
	cap   uint32
	max   uint32
	cache []bufCache
}

// 参数必须是2的指数倍
func NewRingBuffer(size, maxSize int) *RingBuffer {
	if !isPower(size) {
		return nil
	}
	s := &RingBuffer{
		cap:   uint32(size),
		cache: make([]bufCache, size, size),
		max:   uint32(maxSize),
	}
	return s
}

func (r *RingBuffer) Put(v interface{}) {
	for {
		tail := atomic.LoadUint32(&r.tail)
		// 借用了tail的最高位，标记当前是否正在扩容
		if (tail & is_expand) > 0 {
			runtime.Gosched() // TODO 可以考虑不让出cpu
			continue
		}
		head := atomic.LoadUint32(&r.head)
		if (head & is_expand) > 0 {
			runtime.Gosched() // TODO 可以考虑不让出cpu
			continue
		}
		cap := atomic.LoadUint32(&r.cap)
		next := (tail + 1) & (cap - 1)
		if next == head {
			// 满了，扩容
			//r.expand(tail, next, cap)
			continue
		}
		tmp := &r.cache[tail]
		if !atomic.CompareAndSwapUint32(&r.tail, tail, next) {
			runtime.Gosched() // TODO 可以考虑不让出cpu
			continue
		}
		for {
			if atomic.LoadUint32(&tmp.count) == 0 {
				tmp.value = v
				atomic.StoreUint32(&tmp.count, 1)
				break
			}
		}
		break
	}
}

func (r *RingBuffer) Pop() (rs interface{}) {
	for {
		tail := atomic.LoadUint32(&r.tail) // 消费者应该只有一个，所以这里先读取tail索引是可以的
		if (tail & is_expand) > 0 {
			tail = tail - is_expand
		}
		head := atomic.LoadUint32(&r.head)
		// 判断是否扩容,通常应该很快，所以可能下一刻就好了
		if (head & is_expand) > 0 {
			continue
		}
		if head == tail {
			continue
		}
		// 这里只考虑单一消费者的情况
		cap := atomic.LoadUint32(&r.cap)
		newHead := (head + 1) & (cap - 1)
		tmp := &r.cache[head]
		if !atomic.CompareAndSwapUint32(&r.head, head, newHead) {
			continue
		}
		for {
			// 判断是否已经写入
			if atomic.LoadUint32(&tmp.count) == 1 {
				rs = tmp.value
				tmp.value = nil                   // gc
				atomic.StoreUint32(&tmp.count, 0) // TODO 这里是不是可以不需要使用 atomic
				break
			}
		}
		return rs
	}
}

func (r *RingBuffer) Cache() []bufCache {
	return r.cache
}

func (r *RingBuffer) checkComplete(start, end uint32) {
	var tmp *bufCache
check:
	for i := start; i < end; i++ {
		tmp = &r.cache[i]
		// 判断是否已经写入
		if atomic.LoadUint32(&tmp.count) == 0 {
			runtime.Gosched()
			start = i
			goto check
		}
	}
}

func (r *RingBuffer) expand(tail, next, cap uint32) {
	// 设定一个最大值，通常都不应该这么大，如果无限增大，可能会使得系统oom
	if cap >= r.max {
		runtime.Gosched()
		return
	}
	newTail := tail | is_expand
	if atomic.CompareAndSwapUint32(&r.tail, tail, newTail) {
		var head uint32
		// 需要锁住读取
		for {
			head = atomic.LoadUint32(&r.head)
			// 如果消费者没有阻塞，可能还在消费
			if next != head {
				atomic.StoreUint32(&r.tail, tail)
				goto finish
			}
			newHead := head | is_expand
			if atomic.CompareAndSwapUint32(&r.head, head, newHead) {
				break
			}
		}
		// 判断当前位置是否已经写入完成
		if head != tail {
			if tail > head {
				r.checkComplete(head, tail)
			} else {
				r.checkComplete(0, tail)
				r.checkComplete(head, r.cap)
			}
		}
		// 开始扩容
		cap := r.cap
		newCap := cap << 1 // 2倍扩容
		cache := make([]bufCache, newCap, newCap)
		// 复制数据
		if head > tail {
			idx := cap - head
			newTail = idx + tail
			copy(cache, r.cache[head:])
			copy(cache[idx:], r.cache[0:tail])
		} else {
			newTail = tail - head
			copy(cache, r.cache[head:tail])
		}
		r.cache = cache
		atomic.StoreUint32(&r.cap, newCap)
		atomic.StoreUint32(&r.head, 0)
		atomic.StoreUint32(&r.tail, newTail)
	}
finish:
}
