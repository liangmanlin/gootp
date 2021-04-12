package kernel

import "reflect"

func DeepCopy(src interface{}) interface{} {
	rt := reflect.TypeOf(src)
	vf := reflect.ValueOf(src)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		vf = vf.Elem()
	}
	destPtr := reflect.New(rt)
	dest := destPtr.Elem()
	fNum := vf.NumField()
	for i := 0; i < fNum; i++ {
		f := vf.Field(i)
		df := dest.Field(i)
		dv := copyValue(&f)
		df.Set(dv)
	}
	return destPtr.Interface()
}

func copySlice(src *reflect.Value) reflect.Value {
	lens := src.Len()
	sl := reflect.MakeSlice(src.Type(), lens, lens)
	for i := 0; i < lens; i++ {
		v := src.Index(i)
		v = copyValue(&v)
		sl.Index(i).Set(v)
	}
	return sl
}

func copyMap(src *reflect.Value) reflect.Value {
	m := reflect.MakeMapWithSize(src.Type(), src.Len())
	keys := src.MapKeys()
	for _, k := range keys {
		v := src.MapIndex(k)
		v = copyValue(&v)
		m.SetMapIndex(k, v)
	}
	return m
}

func copyValue(src *reflect.Value) reflect.Value {
	switch src.Kind() {
	case reflect.Ptr:
		if src.IsNil() {
			return *src
		}
		return reflect.ValueOf(DeepCopy(src.Interface()))
	case reflect.Struct:
		return reflect.ValueOf(DeepCopy(src.Interface())).Elem()
	case reflect.Map:
		if src.IsNil() {
			return *src
		}
		return copyMap(src)
	case reflect.Slice:
		if src.IsNil() {
			return *src
		}
		return copySlice(src)
	default:
		return *src
	}
}
