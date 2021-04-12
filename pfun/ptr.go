package pfun

import (
	"reflect"
	"unsafe"
)

func KindFun(v *reflect.Type) PtrFun {
	if (*v).Kind() == reflect.Ptr {
		*v = (*v).Elem()
		return getPtrPtr
	}
	return getPtr
}

func getPtr(pointer unsafe.Pointer, offset uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(pointer) + offset)
}

func getPtrPtr(pointer unsafe.Pointer, offset uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(uintptr(pointer) + offset))
}

var _i = int(0)

const CpuSize = unsafe.Sizeof(&_i) // 预先计算出int指针的大小

func Ptr(i interface{}) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(uintptr(unsafe.Pointer(&i)) + CpuSize)) // TODO 这里是根据interface 的内部实现写死的地址
}

func SliceSize(ptr unsafe.Pointer) int {
	return *(*int)(unsafe.Pointer(uintptr(ptr) + CpuSize))                             // slice的size是第二个字段
}

func GetInt32(ptr unsafe.Pointer,inc uintptr) int32 {
	return *(*int32)(unsafe.Pointer(uintptr(ptr)+inc))
}

func GetInt64(ptr unsafe.Pointer,inc uintptr) int64 {
	return *(*int64)(unsafe.Pointer(uintptr(ptr)+inc))
}