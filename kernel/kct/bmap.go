package kct

import "container/list"

type BMap struct {
	l *list.List
	m map[int64]*list.Element
}

func NewBMap() *BMap {
	return &BMap{l: list.New(), m: make(map[int64]*list.Element)}
}

func (b *BMap) Insert(key int64, value interface{}) {
	e := b.l.PushFront(value)
	b.m[key] = e
}

func (b *BMap) Lookup(key int64) interface{} {
	if e, ok := b.m[key]; ok {
		return e.Value
	}
	return nil
}

func (b *BMap) Delete(key int64) {
	if e, ok := b.m[key]; ok {
		b.l.Remove(e)
		delete(b.m, key)
	}
}

func (b *BMap) Foreach(f func(interface{})) {
	for e := b.l.Front(); e != nil; e = e.Next() {
		f(e.Value)
	}
}
