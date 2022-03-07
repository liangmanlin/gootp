package pb

import (
	"github.com/liangmanlin/gootp/kernel"
	"log"
	"reflect"
	"unsafe"
)

var _i = int(0)

const cpuSize = unsafe.Sizeof(&_i) // 预先计算出int指针的大小

func (c *Coder) EncodeBuff(proto interface{}, head int, buf []byte) (minBuf []byte) {
	rt := getRType(proto)
	def := c.getDefTF(rt)
	if def == nil {
		log.Panicf("proto :%s not define",rt.Name())
	}
	// 前期就可以计算出最小buff，减少多余的内存申请
	sizeBuf := len(buf)
	totalSize := def.minBufSize + head + 2 + sizeBuf
	if cap(buf) >= totalSize {
		minBuf = buf[0 : 2+head+sizeBuf : totalSize]
	} else {
		minBuf = make([]byte, 2+head+sizeBuf, totalSize)
		copy(minBuf, buf)
	}
	// 压入协议编号 大端
	WriteSize(minBuf[sizeBuf:], head, def.id)
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(uintptr(unsafe.Pointer(&proto)) + cpuSize)) // TODO 这里是根据interface 的内部实现写死的地址
	minBuf = c.encodeDef(minBuf, ptr, rt, def)
	if head > 0 {
		size := len(minBuf) - head
		writeSizeHead(minBuf, head, size)
	}
	return minBuf
}

// 打包的数据可以直接发送，无需添加头部
func (c *Coder) Encode(proto interface{}, head int) []byte {
	rt := getRType(proto)
	def := c.getDefTF(rt)
	if def == nil {
		log.Panicf("proto :%s not define",rt.Name())
	}
	// 前期就可以计算出最小buff，减少多余的内存申请
	minBuf := make([]byte, 2+head, def.minBufSize+head+2)
	// 压入协议编号 大端
	WriteSize(minBuf, head, def.id)
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(uintptr(unsafe.Pointer(&proto)) + cpuSize)) // TODO 这里是根据interface 的内部实现写死的地址
	minBuf = c.encodeDef(minBuf, ptr, rt, def)
	if head > 0 {
		size := len(minBuf) - head
		writeSizeHead(minBuf, head, size)
	}
	return minBuf
}

func (c *Coder) encodeStruct(buf []byte, ptr unsafe.Pointer, child reflect.Type) []byte {
	if *(*unsafe.Pointer)(ptr) == nil {
		return buf
	}
	ptr = *(*unsafe.Pointer)(ptr)
	def := c.getDefTF(child)
	buf = c.encodeDef(buf, ptr, child, def)
	return buf
}

func (c *Coder) encodeStructCheck(buf []byte, ptr unsafe.Pointer, child reflect.Type) []byte {
	buf = append(buf, 0)
	start := len(buf)
	buf = c.encodeStruct(buf, ptr, child)
	if start != len(buf) {
		buf[start-1] = 1
	}
	return buf
}

func (c *Coder) encodeDef(buf []byte, ptr unsafe.Pointer, rt reflect.Type, def *inDef) []byte {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	ens := def.ens
	for i, en := range ens {
		f := rt.Field(i)
		buf = en.enc(buf, unsafe.Pointer(uintptr(ptr)+f.Offset), en.child)
	}
	return buf
}

func (c *Coder) getDef(proto interface{}) *inDef {
	t := reflect.TypeOf(proto)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return c.def[t]
}

func (c *Coder) getDefTF(t reflect.Type) *inDef {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return c.def[t]
}

func (c *Coder) getDefVF(value *reflect.Value) *inDef {
	t := value.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return c.def[t]
}

func (c *Coder) encodeUint16V(buf []byte, v int) []byte {
	buf = append(buf, uint8(v>>8), uint8(v))
	return buf
}

func (c *Coder) encodeBool(buf []byte, ptr unsafe.Pointer, _ reflect.Type) []byte {
	v := *(*bool)(ptr)
	var i uint8
	if v {
		i = 1
	}
	buf = append(buf, i)
	return buf
}

func (c *Coder) encodeInt8(buf []byte, ptr unsafe.Pointer, _ reflect.Type) []byte {
	v := *(*int8)(ptr)
	buf = append(buf, uint8(v))
	return buf
}

func (c *Coder) encodeUint8(buf []byte, ptr unsafe.Pointer, _ reflect.Type) []byte {
	v := *(*uint8)(ptr)
	buf = append(buf, v)
	return buf
}

func (c *Coder) encodeUint16(buf []byte, ptr unsafe.Pointer, _ reflect.Type) []byte {
	v := *(*uint16)(ptr)
	buf = append(buf, uint8(v>>8), uint8(v))
	return buf
}

func (c *Coder) encodeInt16(buf []byte, ptr unsafe.Pointer, _ reflect.Type) []byte {
	v := *(*int16)(ptr)
	buf = append(buf, uint8(v>>8), uint8(v))
	return buf
}

func (c *Coder) encodeInt32(buf []byte, ptr unsafe.Pointer, _ reflect.Type) []byte {
	v := *((*int32)(ptr))
	buf = append(buf, uint8(v>>24), uint8(v>>16), uint8(v>>8), uint8(v))
	return buf
}

func (c *Coder) encodeInt64(buf []byte, ptr unsafe.Pointer, _ reflect.Type) []byte {
	v := *(*int64)(ptr)
	buf = append(buf, uint8(v>>56), uint8(v>>48), uint8(v>>40), uint8(v>>32), uint8(v>>24), uint8(v>>16), uint8(v>>8), uint8(v))
	return buf
}

func (c *Coder) encodeFloat32(buf []byte, ptr unsafe.Pointer, _ reflect.Type) []byte {
	bits := *(*uint32)(ptr)
	buf = append(buf, byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits))
	return buf
}

func (c *Coder) encodeFloat64(buf []byte, ptr unsafe.Pointer, _ reflect.Type) []byte {
	bits := *(*uint64)(ptr)
	buf = append(buf, byte(bits>>56), byte(bits>>48), byte(bits>>40), byte(bits>>32), byte(bits>>24), byte(bits>>16), byte(bits>>8), byte(bits))
	return buf
}

func (c *Coder) encodeString(buf []byte, ptr unsafe.Pointer, _ reflect.Type) []byte {
	v := *(*[]byte)(ptr)
	size := len(v)
	buf = append(buf, uint8(size>>8), uint8(size))
	buf = append(buf, v...)
	return buf
}

func (c *Coder) encodeByteArray(buf []byte, ptr unsafe.Pointer, _ reflect.Type) []byte {
	v := *(*[]byte)(ptr)
	size := len(v)
	buf = append(buf, uint8(size>>8), uint8(size))
	buf = append(buf, v...)
	return buf
}

func (c *Coder) encodeSlice(buf []byte, ptr unsafe.Pointer, child reflect.Type) []byte {
	lens := *(*int)(unsafe.Pointer(uintptr(ptr) + cpuSize))
	buf = c.encodeUint16V(buf, 0)
	if lens == 0 {
		return buf
	}
	tf := child.Elem()
	inc := tf.Size()
	slicePtr := *(*unsafe.Pointer)(ptr)
	if tf.Kind() == reflect.Ptr {
		tf = tf.Elem()
	}
	start := len(buf)
	enValueFunc := c.getEncodeFunc(tf)
	offset := uintptr(0)
	for i := 0; i < lens; i++ {
		p := getPtr(slicePtr, offset)
		offset += inc
		buf = enValueFunc(buf, p, tf)
	}
	end := len(buf)
	size := end - start
	WriteSize(buf, start-2, size)
	return buf
}

func (c *Coder) encodeMap(buf []byte, ptr unsafe.Pointer, child reflect.Type) []byte {
	value := reflect.NewAt(child, ptr).Elem()
	lens := value.Len()
	buf = c.encodeUint16V(buf, 0)
	if lens == 0 {
		return buf
	}
	start := len(buf)
	keyType := value.Type().Key()
	valueType := value.Type().Elem()
	var keyPtrFun func(ptr unsafe.Pointer) unsafe.Pointer
	if keyType.Kind() == reflect.Struct {
		keyPtrFun = getStructPtr
	} else {
		keyPtrFun = getValuePtr
	}
	enKeyFunc := c.getEncodeFunc(keyType)
	enValueFunc := c.getEncodeFunc(valueType)
	ranges := value.MapRange()
	for ranges.Next() {
		kv := ranges.Key()
		// TODO 这里是根据reflect.Value 的内部实现写死的地址
		buf = enKeyFunc(buf, keyPtrFun(unsafe.Pointer(&kv)), keyType)
		vv := ranges.Value()
		buf = enValueFunc(buf, *(*unsafe.Pointer)(unsafe.Pointer(uintptr(unsafe.Pointer(&vv)) + cpuSize)), valueType)
	}
	end := len(buf)
	size := end - start
	WriteSize(buf, start-2, size)
	return buf
}

func (c *Coder) encodePid(buf []byte, ptr unsafe.Pointer, _ reflect.Type) []byte {
	if *(*unsafe.Pointer)(ptr) == nil {
		return buf
	}
	buf = (*(**kernel.Pid)(ptr)).ToBytes(buf)
	return buf
}

func (c *Coder) encodePidCheck(buf []byte, ptr unsafe.Pointer, child reflect.Type) []byte {
	buf = append(buf, 0)
	start := len(buf)
	buf = c.encodePid(buf, ptr, child)
	if start != len(buf) {
		buf[start-1] = 1
	}
	return buf
}

func (c *Coder) getEncodeFunc(t reflect.Type) fieldEncodeFunc {
	kind := t.Kind()
	if kind == reflect.Ptr {
		kind = t.Elem().Kind()
	}
	return c.enMap[kind]
}

func WriteSize(buf []byte, index int, size int) {
	buf[index] = uint8(size >> 8)
	buf[index+1] = uint8(size)
}

func writeSizeHead(buf []byte, head, size int) {
	if head == 2 {
		buf[0] = uint8(size >> 8)
		buf[1] = uint8(size)
		return
	}
	buf[0] = uint8(size >> 24)
	buf[1] = uint8(size >> 16)
	buf[2] = uint8(size >> 8)
	buf[3] = uint8(size)
}

func getRType(proto interface{}) reflect.Type {
	rt := reflect.TypeOf(proto)
	if rt.Kind() == reflect.Ptr {
		return rt.Elem()
	}
	return rt
}

func getPtr(pointer unsafe.Pointer, offset uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(pointer) + offset)
}

func getPtrPtr(pointer unsafe.Pointer, offset uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(uintptr(pointer) + offset))
}

func getStructPtr(pointer unsafe.Pointer) unsafe.Pointer {
	return unsafe.Pointer(uintptr(pointer) + cpuSize)
}

func getValuePtr(pointer unsafe.Pointer) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(uintptr(pointer) + cpuSize))
}

func WriteInt64(buf []byte, v int64, index int) {
	buf[index] = uint8(v >> 56)
	buf[index+1] = uint8(v >> 48)
	buf[index+2] = uint8(v >> 40)
	buf[index+3] = uint8(v >> 32)
	buf[index+4] = uint8(v >> 24)
	buf[index+5] = uint8(v >> 16)
	buf[index+6] = uint8(v >> 8)
	buf[index+7] = uint8(v)
}

func WriteIn32(buf []byte, v int32, index int) {
	buf[index] = uint8(v >> 24)
	buf[index+1] = uint8(v >> 16)
	buf[index+2] = uint8(v >> 8)
	buf[index+3] = uint8(v)
}

func WriteString(buf []byte, str string, index int) []byte {
	v := *(*[]byte)(unsafe.Pointer(&str))
	size := len(v)
	buf[index] = uint8(size >> 8)
	buf[index+1] = uint8(size)
	return append(buf, v...)
}
