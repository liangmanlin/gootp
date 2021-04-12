package gutil

import (
	"github.com/liangmanlin/gootp/pfun"
	"reflect"
	"sort"
	"unsafe"
)

type s struct {
	cmp  cmpFunc
	swap swapFunc
	ptr  unsafe.Pointer
}

func (s *s) Len() int {
	return pfun.SliceSize(s.ptr)
}

func (s *s) Less(i, j int) bool {
	return s.cmp(s.ptr, i, j)
}

func (s *s) Swap(i, j int) {
	s.swap(s.ptr, i, j)
}

/*
	排序

	只是一个sort.Sort函数的封装，使得不需要使用接口

	由于包装了一层，所以效率会比sort.Sort慢2.5倍左右
 */
func Sort(list interface{}, cmp cmpFunc, swap swapFunc) {
	if reflect.ValueOf(list).Kind() != reflect.Slice{
		return
	}
	ptr := pfun.Ptr(list)
	sort.Sort(&s{cmp: cmp, swap: swap, ptr: ptr})
}
