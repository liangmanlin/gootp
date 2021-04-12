package pb

import (
	"reflect"
	"unsafe"
)

type Coder struct {
	def    map[reflect.Type]*inDef
	id2def map[int]*inDef
	enMap  []fieldEncodeFunc
}

type inDef struct {
	id         int
	minBufSize int
	objType    reflect.Type
	ens        []*encode
	des        []*decode
}

type encode struct {
	enc   fieldEncodeFunc
	child reflect.Type
}

type decode struct {
	dec   fieldDecodeFunc
	child reflect.Type
}

type fieldEncodeFunc func(buf []byte, ptr unsafe.Pointer,vType reflect.Type) []byte

type fieldDecodeFunc func(buf []byte, index int, vType reflect.Type) (int, reflect.Value)
