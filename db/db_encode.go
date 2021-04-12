package db

import (
	"github.com/liangmanlin/gootp/kernel"
	"reflect"
	"strconv"
)

func Encode(v interface{}) string {
	switch v2 := v.(type) {
	case int:
		return strconv.Itoa(v2)
	case int32:
		return strconv.FormatInt(int64(v2),10)
	case int64:
		return strconv.FormatInt(v2,10)
	case uint:
		return strconv.FormatUint(uint64(v2),10)
	case uint32:
		return strconv.FormatUint(uint64(v2),10)
	case uint64:
		return strconv.FormatUint(v2,10)
	case string:
		return quote([]byte(v2))
	case []byte:
		return quote(v2)
	default:
		kernel.ErrorLog("db encode error:%#v",v2)
		return "1"
	}
}

func quote(bin []byte)string{
	sl := make([]byte,1,1)
	sl[0] = '\''
	//sl = append(sl,bin...)
	for _,b := range bin{
		switch b {
		case 0:
			sl = append(sl,92,48) // \\\0
		//case 10:
		//	sl = append(sl,92,110) // \\n
		//case 13:
		//	sl = append(sl,92,114) // \\r
		//case 26:
		//	sl = append(sl,92,90) // \\Z
		case 34:
			sl = append(sl,92,34) // \\\"
		case 39:
			sl = append(sl,92,39) // \\\'
		case 92:
			sl = append(sl,92,92) // \\\\
		default:
			sl = append(sl,b)
		}
	}
	sl = append(sl,'\'')
	return string(sl)
}

func encodeValue(f *reflect.Value) string {
	switch f.Kind() {
	case reflect.Bool:
		var v = int32(0)
		if f.Bool() {
			v = 1
		}
		return Encode(v)
	case reflect.Slice,reflect.Map,reflect.Struct,reflect.Ptr:
		return Encode(toBinary(f))
	default:
		return Encode(f.Interface())
	}
}