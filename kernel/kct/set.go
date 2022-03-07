package kct

/*
	等1.18泛型重写
 */

type empty struct {}

type Set map[interface{}]empty

func NewSet(size int) Set {
	if size <= 0 {
		size = 1
	}
	return make(map[interface{}]empty,size)
}

func (s Set)Insert(val interface{})  {
	s[val] = empty{}
}

func (s Set)Erase(val interface{})  {
	delete(s,val)
}

func (s Set)Size() int {
	return len(s)
}

func (s Set)Has(v interface{}) bool {
	_,ok := s[v]
	return ok
}

func (s Set)Foreach(f func(v interface{})){
	for k,_ := range s{
		f(k)
	}
}