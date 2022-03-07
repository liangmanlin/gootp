package rand

import (
	"github.com/liangmanlin/gootp/pfun"
	"math/rand"
	"reflect"
	"time"
	"unsafe"
)

var defaultR = Rand{rand.New(rand.NewSource(time.Now().UnixNano()))}

func Random(min, max int32) int32 {
	return defaultR.Random(min,max)
}

func Int32(n int32) int32 {
	return defaultR.Int32(n)
}

func Int64(n int64) int64 {
	return defaultR.Int64(n)
}

// 给定一个范围，随机若干个数
func RandomNum(min, max, num int32) []int32 {
	return defaultR.RandomNum(min,max,num)
}

/*
从一个slice里面，根据权重，随机出value

可以重复的参数比不可重复效率略高

list := []struct{int32,interface{}}

切记不要修改返回值里面的内容
*/
func RandomQSlice(list interface{}, num int32, canRepeated bool) interface{} {
	return defaultR.RandomQSlice(list,num,canRepeated)
}

func New() Rand {
	return Rand{rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (r Rand) Random(min, max int32) int32 {
	return r.rand.Int31n(max-min+1) + min
}

func (r Rand) Int32(n int32) int32 {
	return r.rand.Int31n(n)
}

func (r Rand) Int64(n int64) int64 {
	return r.rand.Int63n(n)
}

// 给定一个范围，随机若干个数
func (r Rand) RandomNum(min, max, num int32) []int32 {
	dc := max - min + 1
	if dc < num {
		num = dc
	}
	rs := make([]int32, num)
	m := make(map[int32]mv, num+1)
	var k, l int32
	m[k] = mv{min, max}
	for i := int32(0); i < num; i++ {
		l = int32(len(m))
		k = r.rand.Int31n(l)
		tmp := m[k]
		if tmp.min == tmp.max {
			rs[i] = tmp.min
			l--
			m[k] = m[l]
			delete(m, l)
		} else {
			v := r.rand.Int31n(tmp.max-tmp.min+1) + tmp.min
			rs[i] = v
			if v == tmp.min {
				tmp.min++
				m[k] = tmp
			} else if v == tmp.max {
				tmp.max--
				m[k] = tmp
			} else {
				m[l] = mv{v + 1, tmp.max}
				tmp.max = v - 1
				m[k] = tmp
			}
		}
	}
	return rs
}

/*
从一个slice里面，根据权重，随机出value

可以重复的参数比不可重复效率略高

list := []struct{int32,interface{}}

切记不要修改返回值里面的内容
 */
func (r Rand) RandomQSlice(list interface{}, num int32, canRepeated bool) interface{} {
	num2 := int(num)
	vf := reflect.ValueOf(list)
	if vf.Kind() != reflect.Slice {
		return nil
	}
	ptr := pfun.Ptr(list)
	size := pfun.SliceSize(ptr)
	rsf := vf.Type().Elem()
	inc := rsf.Size()
	sliceDataPtr := *(*unsafe.Pointer)(ptr)
	getPtrF := pfun.KindFun(&rsf)
	var total int32 = 1
	// 复制一个数组，规避远数组被修改
	ql := make([]qi, size)
	for i := 0; i < size; i++ {
		p := getPtrF(sliceDataPtr, inc*uintptr(i))
		q := *(*int32)(p)
		ql[i] = qi{q, i}
		total += q
	}
	vvt := rsf.Field(1).Type
	rs := reflect.MakeSlice(reflect.SliceOf(vvt), num2, num2)
	for i := 0; i < num2; i++ {
		random := r.rand.Int31n(total)
		for j := 0; j < size; j++ {
			tmp := ql[j]
			q := tmp.q
			if q >= random {
				f := vf.Index(tmp.idx)
				if f.Kind() == reflect.Ptr {
					f = f.Elem()
				}
				rs.Index(i).Set(f.Field(1))
				if !canRepeated {
					total -= q
					size--
					ql[j] = ql[size]
				}
				break
			}
			random -= q
		}
	}
	return rs.Interface()
}

