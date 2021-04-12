package gutil

import (
	"github.com/liangmanlin/gootp/pfun"
	"math"
	"reflect"
	"unsafe"
)

func MaxInt32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func MinInt32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func MaxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func MinFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func Ceil(v float32) int32 {
	 return int32(math.Ceil(float64(v)))
}

func Round(v float32) int32 {
	return int32(math.Round(float64(v)))
}

func Trunc(v float32) int32 {
	return int32(v)
}

func SliceDelInt32(arr []int32, value int32) []int32 {
	for i, v := range arr {
		if value == v {
			return append(arr[:i], arr[i+1:]...)
		}
	}
	return arr
}

func SliceDelInt64(arr []int64, value int64) []int64 {
	for i, v := range arr {
		if value == v {
			return append(arr[:i], arr[i+1:]...)
		}
	}
	return arr
}

// 根据一个范围，获取到值
// list := []struct{int32,int32,interface{}}
func FindRangeValue(list interface{}, value int32) interface{} {
	vf := reflect.ValueOf(list)
	if vf.Kind() != reflect.Slice {
		return nil
	}
	rsf := vf.Type().Elem()
	inc := rsf.Size()
	getPtrF := pfun.KindFun(&rsf)
	if !(rsf.NumField() == 3 && rsf.Field(0).Type.Kind() == reflect.Int32 && rsf.Field(1).Type.Kind() == reflect.Int32) {
		return nil
	}
	ptr := pfun.Ptr(list)
	sliceDataPtr := *(*unsafe.Pointer)(ptr)
	size := pfun.SliceSize(ptr)
	for i := 0; i < size; i++ {
		p := getPtrF(sliceDataPtr, inc*uintptr(i))
		min := *(*int32)(p)
		max := pfun.GetInt32(p, 4)
		if min <= value && max >= value {
			return vf.Index(i).Field(2).Interface()
		}
	}
	return nil
}
