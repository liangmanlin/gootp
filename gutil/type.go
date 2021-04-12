package gutil

import "unsafe"

type cmpFunc = func(data unsafe.Pointer,i,j int) bool
type swapFunc = func(data unsafe.Pointer,i,j int)