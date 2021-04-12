package pfun

import "unsafe"

type PtrFun = func(pointer unsafe.Pointer, offset uintptr) unsafe.Pointer
