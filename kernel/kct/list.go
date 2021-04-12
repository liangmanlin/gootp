package kct

// 提供一种类似erlang：list的数据结构,但是从尾部插入
// 创建的时候需要传入一个比较函数

type equalFun func(interface{}, interface{}) bool

type foldlFun func(e, acc interface{}) interface{}

type Klist struct {
	arr   []interface{}
	eqFun equalFun
}

func NewList(f equalFun) *Klist {
	return &Klist{eqFun: f}
}

func (l *Klist) Append(e interface{}) int {
	l.arr = append(l.arr, e)
	return len(l.arr) - 1
}

func (l *Klist) Len() int {
	return len(l.arr)
}

func (l *Klist) Delete(e interface{}) {
	size := len(l.arr)
	for i := 0; i < size; i++ {
		if l.eqFun(e, l.arr[i]) {
			l.arr = append(l.arr[:i], l.arr[i+1:]...)
			break
		}
	}
}

func (l *Klist) Nth(idx int) interface{} {
	if idx < 0 || idx >= len(l.arr) {
		return nil
	}
	return l.arr[idx]
}

func (l *Klist) Reverse() {
	for i, j := 0, len(l.arr)-1; i < j; i, j = i+1, j-1 {
		l.arr[i], l.arr[j] = l.arr[j], l.arr[i]
	}
}

func (l *Klist) Foreach(f func(e interface{})) {
	for _, v := range l.arr {
		f(v)
	}
}

func (l *Klist) ForeachReverse(f func(e interface{})) {
	for i := len(l.arr) - 1; i >= 0; i-- {
		f(l.arr[i])
	}
}

func (l *Klist) Fold(f foldlFun, acc interface{}) interface{} {
	for _, v := range l.arr {
		acc = f(v, acc)
	}
	return acc
}

// 删除指定索引,并返回
func (l *Klist) Take(idx int) interface{} {
	if idx < 0 || idx >= len(l.arr) {
		return nil
	}
	tmp := l.arr[idx]
	l.arr = append(l.arr[:idx], l.arr[idx+1:]...)
	return tmp
}

func (l *Klist) All() []interface{} {
	return l.arr
}
