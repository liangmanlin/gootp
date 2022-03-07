package ringbuffer

/*
	通用ring buffer，主要是想解决链表带来的垃圾产生，以及减少go的gc对象
*/

// 这里是单线程使用的，无锁
type SingleRingBuffer struct {
	head   int
	tail   int
	cap    int
	maxCap int           // 当队列空闲的时候，会判断是否超过最大值
	cache  []interface{} // TODO 这里可以考虑使用unsafe
}

// 参数必须是2的指数倍
func NewSingleRingBuffer(size, maxSize int) *SingleRingBuffer {
	if !isPower(size) || !isPower(maxSize) {
		return nil
	}
	s := &SingleRingBuffer{
		cap:    size,
		maxCap: maxSize,
		cache:  make([]interface{}, size, size),
	}
	return s
}

func (s *SingleRingBuffer) Size() int {
	if s.head > s.tail {
		return s.cap - s.head + s.tail
	}
	return s.tail - s.head
}

// 放入队列，如果满了，会扩容
func (s *SingleRingBuffer) Put(value interface{}) {
	next := (s.tail + 1) & (s.cap-1)
	if next == s.head {
		// 满了，扩容
		s.expand()
		// 需要从新计算
		next = (s.tail + 1) & (s.cap-1)
	}
	s.cache[s.tail] = value
	s.tail = next
}

func (s *SingleRingBuffer) Pop() interface{} {
	if s == nil || s.tail == s.head {
		return nil
	}
	rs := s.cache[s.head]
	s.cache[s.head] = nil // gc
	s.head = (s.head + 1) & (s.cap-1)
	if s.tail == s.head && s.cap > s.maxCap {
		// 空了，缩小
		s.narrow()
	}
	return rs
}

func (s *SingleRingBuffer) expand() {
	newCap := s.cap * 2
	cache := make([]interface{}, newCap, newCap)
	var size int
	// 拷贝到新cache
	if s.head > s.tail {
		idx := s.cap - s.head
		size = idx + s.tail
		copy(cache, s.cache[s.head:])
		copy(cache[idx:], s.cache[0:s.tail])
	} else {
		size = s.tail - s.head
		copy(cache, s.cache[s.head:s.tail])
	}
	s.head = 0
	s.tail = size
	s.cache = cache
	s.cap = newCap
}

func (s *SingleRingBuffer) narrow() {
	s.cache = make([]interface{}, s.maxCap, s.maxCap)
	s.head = 0
	s.tail = 0
}

func isPower(v int) bool {
	return v > 0 && v&(v-1) == 0
}
