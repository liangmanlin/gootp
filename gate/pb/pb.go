package pb

import (
	"fmt"
	"log"
	"reflect"
)

const coderSize = 10

var coderArray [coderSize]*Coder

/*
	利用反射，实现一个定长协议
	（大端字节）
	TODO 理论上应该使用自动生成代码，实现更高效率
*/

func ParseSlice(def []interface{},id int) *Coder {
	if len(def)+1 >= 0xffff {
		log.Panicf("to many proto def,limit:%d",0xffff)
	}
	m := make(map[int]interface{},len(def))
	for i,v := range def{
		m[i+1] = v
	}
	return Parse(m,id)
}

func Parse(def map[int]interface{}, id int) *Coder {
	if id >= coderSize{
		log.Panic(fmt.Errorf("id out of range(0-%d)", coderSize-1))
	}
	m := make(map[reflect.Type]*inDef)
	c := &Coder{def: m, id2def: make(map[int]*inDef)}
	for i, v := range def {
		makeDef(v, i, c, m)
	}
	if id >= 0 {
		coderArray[id] = c
	}
	c.init()
	return c
}

// 可以通过该函数并发获取coder
func GetCoder(id int) *Coder {
	if id >= coderSize || id < 0 {
		return nil
	}
	return coderArray[id]
}

func makeDef(src interface{}, id int, c *Coder, m map[reflect.Type]*inDef) {
	rt := reflect.TypeOf(src)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	minSize := 0
	def := &inDef{id: id, objType: rt}
	m[rt] = def // 预先插入防止下层形成环
	if id > 0 {
		c.id2def[id] = def
	}
	num := rt.NumField()
	ens := make([]*encode, num)
	des := make([]*decode, num)
	for i := 0; i < num; i++ {
		ft := rt.Field(i)
		t := ft.Type
		checkPanic(rt, ft, t, "type")
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		v := reflect.New(t).Elem()
		switch v.Kind() {
		case reflect.Bool:
			minSize++
			ens[i] = &encode{enc:c.encodeBool}
			des[i] = &decode{dec: c.decodeBool}
		case reflect.Int8:
			minSize++
			ens[i] = &encode{enc:c.encodeInt8}
			des[i] = &decode{dec: c.decodeInt8}
		case reflect.Int16:
			minSize += 2
			ens[i] = &encode{enc:c.encodeInt16}
			des[i] = &decode{dec: c.decodeInt16}
		case reflect.Int32:
			minSize += 4
			ens[i] = &encode{enc:c.encodeInt32}
			des[i] = &decode{dec: c.decodeInt32}
		case reflect.Int64:
			minSize += 8
			ens[i] = &encode{enc:c.encodeInt64}
			des[i] = &decode{dec: c.decodeInt64}
		case reflect.Float32:
			minSize += 4
			ens[i] = &encode{enc:c.encodeFloat32}
			des[i] = &decode{dec: c.decodeFloat32}
		case reflect.Float64:
			minSize += 4
			ens[i] = &encode{enc:c.encodeFloat64}
			des[i] = &decode{dec: c.decodeFloat64}
		case reflect.Uint16:
			minSize += 2
			ens[i] = &encode{enc:c.encodeUint16}
			des[i] = &decode{dec: c.decodeUint16}
		case reflect.String:
			minSize += 2
			ens[i] = &encode{enc:c.encodeString}
			des[i] = &decode{dec: c.decodeString}
		case reflect.Slice:
			minSize += 2
			child := v.Type()
			if child.Elem().Kind() == reflect.Uint8 { //理论上属于字节数组
				ens[i] = &encode{enc:c.encodeByteArray}
				des[i] = &decode{dec: c.decodeByteArray}
			} else {
				if child.Elem().String() == "&kernel.Pid"{
					vv := make(chan int)
					child = reflect.TypeOf(&vv)
				}else{
					checkPanic(rt, ft, child.Elem(), "slice")
					checkStruct(rt, ft, v.Type().Elem(), c, m)
				}
				ens[i] = &encode{enc:c.encodeSlice,child: child}
				des[i] = &decode{dec: c.decodeSlice, child: child}
			}
		case reflect.Map:
			minSize += 2
			child := v.Type()
			checkMapKey(rt, ft, child.Key())
			if child.Elem().String() == "&kernel.Pid"{
				vv := make(chan int)
				child = reflect.MapOf(child.Key(),reflect.TypeOf(&vv))
			}else{
				checkPanic(rt, ft, child.Elem(), "map")
			}

			ens[i] = &encode{enc:c.encodeMap,child: child}
			des[i] = &decode{dec: c.decodeMap, child: child}
			checkStruct(rt, ft, v.Type().Elem(), c, m)
		case reflect.Struct:
			minSize += 2
			child := v.Type()
			if child.String() == "kernel.Pid"{
				ens[i] = &encode{enc:c.encodePidCheck,child: child}
				des[i] = &decode{dec: c.decodePidCheck, child: child}
			}else{
				checkStruct(rt, ft, v.Type(), c, m)
				ens[i] = &encode{enc:c.encodeStructCheck,child: child}
				des[i] = &decode{dec: c.decodeStructCheck, child: child}
			}

		default:
			log.Panic(fmt.Errorf("struct [%s.%s] not support type: %s", rt.String(), ft.Name, v.Type()))
		}
	}
	def.minBufSize = minSize
	def.ens = ens
	def.des = des
}

func checkStruct(st reflect.Type, ft reflect.StructField, rt reflect.Type, c *Coder, m map[reflect.Type]*inDef) {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	switch rt.Kind() {
	case reflect.Slice:
		child := rt.Elem()
		checkPanic(st, ft, child, "slice")
		checkStruct(st, ft, child, c, m)
	case reflect.Map:
		checkMapKey(st, ft, rt.Key())
		child := rt.Elem()
		checkPanic(st, ft, child, "map")
		checkStruct(st, ft, child, c, m)
	case reflect.Struct:
		if _, ok := m[rt]; !ok {
			v := reflect.New(rt)
			makeDef(v.Interface(), 0, c, m)
		}
	}

}

func checkPanic(st reflect.Type, ft reflect.StructField, child reflect.Type, flag string) {
	kind := child.Kind()
	if kind == reflect.Struct {
		log.Panic(fmt.Errorf("struct [%s.%s] %s: [%s] is not pfun", st.String(), ft.Name, flag, child.String()))
	} else if kind == reflect.Ptr && child.Elem().Kind() != reflect.Struct {
		log.Panic(fmt.Errorf("struct [%s.%s] %s: [%s] pfun not allow", st.String(), ft.Name, flag, child.String()))
	}
}

func checkMapKey(st reflect.Type, ft reflect.StructField, key reflect.Type) {
	switch key.Kind() {
	case reflect.Int8:
	case reflect.Int16:
	case reflect.Int32:
	case reflect.Int64:
	case reflect.Uint16:
	case reflect.String:
	default:
		log.Panic(fmt.Errorf("struct [%s.%s] map key [%s] is not allow", st.String(), ft.Name, key.String()))
	}
}

func (c *Coder) init() {
	c.enMap = make([]fieldEncodeFunc, 30)
	c.enMap[reflect.Bool] = c.encodeBool
	c.enMap[reflect.Int8] = c.encodeInt8
	c.enMap[reflect.Int16] = c.encodeInt16
	c.enMap[reflect.Int32] = c.encodeInt32
	c.enMap[reflect.Int64] = c.encodeInt64
	c.enMap[reflect.Float32] = c.encodeFloat32
	c.enMap[reflect.Float64] = c.encodeFloat64
	c.enMap[reflect.Uint8] = c.encodeUint8
	c.enMap[reflect.Uint16] = c.encodeUint16
	c.enMap[reflect.String] = c.encodeString
	c.enMap[reflect.Slice] = c.encodeSlice
	c.enMap[reflect.Map] = c.encodeMap
	c.enMap[reflect.Struct] = c.encodeStruct
	c.enMap[reflect.Chan] = c.encodePid
}
