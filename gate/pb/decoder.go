package pb

import (
	"encoding/binary"
	"fmt"
	"github.com/liangmanlin/gootp/kernel"
	"log"
	"reflect"
	"unsafe"
)

func (c *Coder) Decode(buf []byte) (int, interface{}) {
	// 先取出协议号
	index := 0
	var id int
	index, id = c.decodeUint16V(buf, index)
	def := c.id2def[id]
	_,vPtr := c.decodeStructDef(buf,index,def.objType,def)
	return id, vPtr.Interface()
}

func (c *Coder) decodeUint16V(buf []byte, index int) (int, int) {
	var v int
	v = int(buf[index]) << 8
	index++
	v += int(buf[index])
	index++
	return index, v
}

func (c *Coder) decodeBool(buf []byte, index int,_ reflect.Type) (int, reflect.Value) {
	var v reflect.Value
	if buf[index] == 1{
		v = reflect.ValueOf(true)
	}else{
		v = reflect.ValueOf(false)
	}
	index++
	return index,v
}

func (c *Coder) decodeInt8(buf []byte, index int,_ reflect.Type) (int, reflect.Value) {
	v := reflect.ValueOf(int8(buf[index]))
	index++
	return index,v
}

func (c *Coder) decodeUint8(buf []byte, index int,_ reflect.Type) (int, reflect.Value) {
	v := reflect.ValueOf(buf[index])
	index++
	return index,v
}

func (c *Coder) decodeInt16(buf []byte, index int,_ reflect.Type) (int, reflect.Value) {
	v := int16(buf[index]) << 8 + int16(buf[index+1])
	return index+2,reflect.ValueOf(v)
}

func (c *Coder) decodeInt32(buf []byte, index int,_ reflect.Type) (int, reflect.Value) {
	v := int32(buf[index]) << 24 + int32(buf[index+1]) << 16 + int32(buf[index+2]) << 8 + int32(buf[index+3])
	return index+4,reflect.ValueOf(v)
}

func (c *Coder) decodeInt64(buf []byte, index int,_ reflect.Type) (int, reflect.Value) {
	v := int64(buf[index]) << 56 + int64(buf[index+1]) << 48 + int64(buf[index+2]) << 40 + int64(buf[index+3]) << 32+
		int64(buf[index+4]) << 24 + int64(buf[index+5]) << 16 + int64(buf[index+6]) << 8 + int64(buf[index+7])
	return index+8,reflect.ValueOf(v)
}

func (c *Coder) decodeFloat32(buf []byte, index int,_ reflect.Type) (int, reflect.Value) {
	bits := binary.BigEndian.Uint32(buf[index:])
	v := *(*float32)(unsafe.Pointer(&bits))
	return index+4,reflect.ValueOf(v)
}

func (c *Coder) decodeFloat64(buf []byte, index int,_ reflect.Type) (int, reflect.Value) {
	bits := binary.BigEndian.Uint64(buf[index:])
	v := *(*float64)(unsafe.Pointer(&bits))
	return index+8,reflect.ValueOf(v)
}

func (c *Coder) decodeUint16(buf []byte, index int,_ reflect.Type)(int, reflect.Value) {
	v := uint16(buf[index]) << 8 + uint16(buf[index])
	return index+2, reflect.ValueOf(v)
}

func (c *Coder) decodeString(buf []byte, index int,_ reflect.Type)(int, reflect.Value) {
	var size int
	index,size = c.decodeUint16V(buf,index)
	end := index+size
	v := string(buf[index:end])
	return end, reflect.ValueOf(v)
}

func (c *Coder)decodeByteArray(buf []byte, index int,_ reflect.Type)(int, reflect.Value) {
	var size int
	index,size = c.decodeUint16V(buf,index)
	end := index+size
	v := buf[index:end] // 这里暂时不拷贝，因为即使是整个buf块，也不会很大，通常用于传输语音，图片等文件流
	return end,reflect.ValueOf(v)
}

func (c *Coder) decodeSlice(buf []byte, index int,child reflect.Type)(int, reflect.Value) {
	var size int
	index,size = c.decodeUint16V(buf,index)
	slice := reflect.MakeSlice(child,0,2)
	if size == 0 {
		return index,slice
	}
	end := index + size
	decFunc,child2 := c.getDecodeFunc(child.Elem())
	var v reflect.Value
	for index < end {
		index ,v = decFunc(buf,index,child2)
		slice = reflect.Append(slice,v)
	}
	return end,slice
}

func (c *Coder)decodeMap(buf []byte,index int,child reflect.Type)(int,reflect.Value){
	var size int
	index,size = c.decodeUint16V(buf,index)
	m := reflect.MakeMapWithSize(child,2)
	if size == 0 {
		return index,m
	}
	end := index + size
	kt := child.Key()
	vt := child.Elem()
	keyFunc,_ := c.getDecodeFunc(kt)
	var vkRs func(v reflect.Value)reflect.Value
	if kt.Kind() == reflect.Struct {
		vkRs = vtReturnElem
	} else {
		vkRs = vrReturn
	}
	decFunc,child2 := c.getDecodeFunc(vt)
	var vk,vv reflect.Value
	for index < end{
		index ,vk = keyFunc(buf,index,kt)
		index ,vv = decFunc(buf,index,child2)
		m.SetMapIndex(vkRs(vk),vv)
	}
	return end,m
}

func (c *Coder)decodePid(buf []byte,index int,_ reflect.Type)(int,reflect.Value){
	var v *kernel.Pid
	index,v = kernel.DecodePid(buf,index)
	if v == nil {
		return index,reflect.Zero(reflect.TypeOf(&kernel.Pid{}))
	}
	return index,reflect.ValueOf(v)
}

func (c *Coder)decodePidCheck(buf []byte,index int,child reflect.Type)(int,reflect.Value){
	if buf[index] == 1 {
		index++
		return c.decodePid(buf,index,child)
	}
	index++
	return index,reflect.Zero(reflect.TypeOf(&kernel.Pid{}))
}

func (c *Coder)decodeStruct(buf []byte,index int,child reflect.Type)(int,reflect.Value){
	def := c.getDefTF(child)
	return c.decodeStructDef(buf,index,child,def)
}

func (c *Coder)decodeStructCheck(buf []byte,index int,child reflect.Type)(int,reflect.Value){
	if buf[index] == 1 {
		index++
		def := c.getDefTF(child)
		return c.decodeStructDef(buf,index,child,def)
	}
	index++
	return index,reflect.Zero(reflect.PtrTo(child))
}

func (c *Coder)decodeStructDef(buf []byte,index int,child reflect.Type,def *inDef)(int,reflect.Value){
	vPtr := reflect.New(child)
	v := vPtr.Elem()
	des := def.des
	num := len(des)
	var value reflect.Value
	for i := 0; i < num; i++ {
		de := des[i]
		f := v.Field(i)
		index, value = de.dec(buf, index,de.child)
		f.Set(value)
	}
	return index, vPtr
}


func (c *Coder)getDecodeFunc(t reflect.Type) (fieldDecodeFunc,reflect.Type) {
	kind := t.Kind()
	if kind == reflect.Ptr {
		kind = t.Elem().Kind()
		t = t.Elem()
	}
	switch kind {
	case reflect.Bool:
		return c.decodeBool,nil
	case reflect.Int8:
		return c.decodeInt8,nil
	case reflect.Int16:
		return c.decodeInt16,nil
	case reflect.Int32:
		return c.decodeInt32,nil
	case reflect.Int64:
		return c.decodeInt64,nil
	case reflect.Uint8:
		return c.decodeUint8,nil
	case reflect.Uint16:
		return c.decodeUint16,nil
	case reflect.String:
		return c.decodeString,nil
	case reflect.Slice:
		return c.decodeSlice,t
	case reflect.Map:
		return c.decodeMap,t
	case reflect.Struct:
		return c.decodeStruct,t
	case reflect.Chan:
		return c.decodePid,nil
	default:
		log.Panic(fmt.Errorf("unknow type:%s",t.String()))
		return nil,nil
	}
}

func DecodeInt64(buf []byte, index int) (int, int64) {
	v := int64(buf[index]) << 56 + int64(buf[index+1]) << 48 + int64(buf[index+2]) << 40 + int64(buf[index+3]) << 32+
		int64(buf[index+4]) << 24 + int64(buf[index+5]) << 16 + int64(buf[index+6]) << 8 + int64(buf[index+7])
	return index+8,v
}

func DecodeInt32(buf []byte, index int) (int, int32) {
	v := int32(buf[index]) << 24 + int32(buf[index+1]) << 16 + int32(buf[index+2]) << 8 + int32(buf[index+3])
	return index+8,v
}

func DecodeString(buf []byte, index int)(int, string) {
	var size int
	size = int(buf[index]) << 8
	size += int(buf[index+1])
	index += 2
	end := index+size
	v := string(buf[index:end])
	return end, v
}

func vtReturnElem(v reflect.Value) reflect.Value {
	return v.Elem()
}

func vrReturn(v reflect.Value) reflect.Value {
	return v
}