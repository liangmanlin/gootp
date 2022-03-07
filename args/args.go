package args

import (
	"github.com/liangmanlin/gootp/pfun"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

/*
	提供一个比较方便获取命令行参数的方法，支持等号赋值和shell模式赋值

	-key=value or -key value

	支持多次赋值

	-key value1 -key value2

	func FillEvn(env interface{})

	支持自动填入命令行参数到对象如
	type Env struct{
		k        int
		v        int `command:"v"`
	}
	var env = &Env{}
	FillEvn(env)

	./main -k 2 -v 1

	系统只会填入tag:command的字段
*/

var _args = make(map[string][]string)
var _other []string

func init() {
	l := os.Args[1:]
	for i := 0; i < len(l); {
		v := l[i]
		if v[0] == '-' {
			// 读取下一个
			i = readValue(v[1:], l, i)
		} else {
			// 把余下的放到一个大列表中
			_other = append(_other, v)
			i++
		}
	}
}

func readValue(key string, l []string, i int) int {
	if len(key) == 0 {
		return i + 1
	}
	// 支持golang 等号（=）赋值
	spl := strings.Split(key, "=")
	if len(spl) > 1 {
		key = spl[0]
		appendValue(spl[0], spl[1])
		return i + 1
	}
	// 最后一个
	if len(l) == i+1 {
		appendValue(key, "")
		return i + 1
	}
	v := l[i+1]
	if v[0] == '-' {
		appendValue(key, "")
		return i + 1
	} else {
		appendValue(key, v)
		return i + 2
	}
}

func GetInt(key string) (v int, ok bool) {
	if vl, ok := _args[key]; ok && len(vl) > 0 {
		// 获取最后一个
		if i, err := strconv.Atoi(vl[len(vl)-1]); err == nil {
			return i, ok
		}
	}
	return
}

func GetString(key string) (v string, ok bool) {
	if vl, ok := _args[key]; ok && len(vl) > 0 {
		return vl[len(vl)-1], ok
	}
	return
}
func GetIntDefault(key string,df int) int{
	if vl, ok := _args[key]; ok && len(vl) > 0 {
		// 获取最后一个
		if i, err := strconv.Atoi(vl[len(vl)-1]); err == nil {
			return i
		}
	}
	return df
}

func GetStringDefault(key string,df string) string {
	if vl, ok := _args[key]; ok && len(vl) > 0 {
		return vl[len(vl)-1]
	}
	return df
}

func GetValues(key string) []string {
	if v, ok := _args[key]; ok {
		return v
	}
	return nil
}

func GetOther() []string {
	return _other
}

func appendValue(key, value string) {
	vl := _args[key]
	vl = append(vl, value)
	_args[key] = vl
}

// 根据命令行参数，自动填充，目前仅仅支持 整数，bool，字符串
// env 需要是指针
func FillEvn(env interface{}) {
	vt := reflect.ValueOf(env)
	if vt.Kind() != reflect.Ptr {
		return
	}
	vt = vt.Elem()
	ft := vt.Type()
	fieldNum := ft.NumField()
	// 利用指针，不导出的字段也可以更新
	ptr := pfun.Ptr(env)
	for i := 0; i < fieldNum; i++ {
		fieldType := ft.Field(i)
		if name, ok := fieldType.Tag.Lookup("command"); ok {
			setValue(fieldType.Type, name, ptr, fieldType.Offset)
		}
	}
}

func setValue(fieldType reflect.Type, name string, ptr unsafe.Pointer, offset uintptr) {
	var (
		v  int
		vs string
		ok bool
	)
	switch fieldType.Kind() {
	case reflect.Int8:
		if v, ok = GetInt(name); ok {
			*(*int8)(unsafe.Pointer(uintptr(ptr) + offset)) = int8(v)
		}
	case reflect.Uint8:
		if v, ok = GetInt(name); ok {
			*(*uint8)(unsafe.Pointer(uintptr(ptr) + offset)) = uint8(v)
		}
	case reflect.Int16:
		if v, ok = GetInt(name); ok {
			*(*int16)(unsafe.Pointer(uintptr(ptr) + offset)) = int16(v)
		}
	case reflect.Uint16:
		if v, ok = GetInt(name); ok {
			*(*uint16)(unsafe.Pointer(uintptr(ptr) + offset)) = uint16(v)
		}
	case reflect.Int32:
		if v, ok = GetInt(name); ok {
			*(*int32)(unsafe.Pointer(uintptr(ptr) + offset)) = int32(v)
		}
	case reflect.Uint32:
		if v, ok = GetInt(name); ok {
			*(*uint32)(unsafe.Pointer(uintptr(ptr) + offset)) = uint32(v)
		}
	case reflect.Int64:
		if v, ok = GetInt(name); ok {
			*(*int64)(unsafe.Pointer(uintptr(ptr) + offset)) = int64(v)
		}
	case reflect.Uint64:
		if v, ok = GetInt(name); ok {
			*(*uint64)(unsafe.Pointer(uintptr(ptr) + offset)) = uint64(v)
		}
	case reflect.Int:
		if v, ok = GetInt(name); ok {
			*(*int)(unsafe.Pointer(uintptr(ptr) + offset)) = v
		}
	case reflect.Uint:
		if v, ok = GetInt(name); ok {
			*(*uint)(unsafe.Pointer(uintptr(ptr) + offset)) = uint(v)
		}
	case reflect.String:
		if vs, ok = GetString(name); ok {
			*(*string)(unsafe.Pointer(uintptr(ptr) + offset)) = vs
		}
	case reflect.Bool:
		if vs, ok = GetString(name); ok {
			*(*bool)(unsafe.Pointer(uintptr(ptr) + offset)) = vs == "true"
		}
	}
}
