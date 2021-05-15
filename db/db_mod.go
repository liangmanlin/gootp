package db

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"github.com/liangmanlin/gootp/kernel"
	"reflect"
	"strings"
)

type scanFunc func(...interface{}) error

func ModSelectRow(db *sql.DB, tab string, key ...interface{}) interface{} {
	def := getDef(tab)
	row := db.QueryRow(fmt.Sprintf("select * from %s where %s", def.Name, GetWhere(def, key...)))
	return toRow(def, row.Scan)
}

func ModSelectAll(db *sql.DB, tab string, key ...interface{}) []interface{} {
	def := getDef(tab)
	rows, err := db.Query(fmt.Sprintf("select * from %s where %s", def.Name, GetWhere(def, key...)))
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	if err != nil {
		kernel.ErrorLog("%s", err.Error())
		return nil
	}
	sl := make([]interface{}, 0, 2)
	for rows.Next() {
		rs := toRow(def, rows.Scan)
		sl = append(sl, rs)
	}
	return sl
}

func ModSelectAllWhere(db *sql.DB, tab string, where string) []interface{} {
	def := getDef(tab)
	rows, err := db.Query(fmt.Sprintf("select * from %s where %s", def.Name, where))
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	if err != nil {
		kernel.ErrorLog("%s", err.Error())
		return nil
	}
	sl := make([]interface{}, 0, 2)
	for rows.Next() {
		rs := toRow(def, rows.Scan)
		sl = append(sl, rs)
	}
	return sl
}

func ModSelect(db *sql.DB, tab string, fields []string, where string) [][]interface{} {
	rows, err := db.Query(fmt.Sprintf("select %s from %s where %s", strings.Join(fields, ","), tab, where))
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	if err != nil {
		kernel.ErrorLog("%s", err.Error())
		return nil
	}
	def := getDef(tab)
	sl := make([][]interface{}, 0, 2)
	fileNum := len(fields)
	for rows.Next() {
		scan := make([]interface{}, fileNum, fileNum)
		for i := 0; i < fileNum; i++ {
			f := def.nameType[fields[i]]
			switch f.Kind() {
			case reflect.Bool:
				var v int
				scan[i] = &v
			case reflect.Slice, reflect.Map, reflect.Struct, reflect.Ptr:
				var buf []byte
				scan[i] = &buf
			default:
				scan[i] = reflect.New(f).Interface()
			}
		}
		err = rows.Scan(scan...)
		if err != nil {
			if err != sql.ErrNoRows {
				kernel.ErrorLog("%s", err.Error())
			}
		} else {
			rs := make([]interface{}, fileNum, fileNum)
			for i := 0; i < fileNum; i++ {
				f := def.nameType[fields[i]]
				switch f.Kind() {
				case reflect.Bool:
					rs[i] = *scan[i].(*int) == 1
				case reflect.Slice, reflect.Map, reflect.Struct:
					v := reflect.New(f)
					de := gob.NewDecoder(bytes.NewReader(*scan[i].(*[]byte)))
					de.DecodeValue(v)
					rs[i] = v.Elem().Interface()
				case reflect.Ptr:
					v := reflect.New(f.Elem())
					de := gob.NewDecoder(bytes.NewReader(*scan[i].(*[]byte)))
					de.DecodeValue(v)
					rs[i] = v.Interface()
				default:
					rs[i] = reflect.ValueOf(scan[i]).Elem().Interface()
				}
			}
			sl = append(sl, scan)
		}
	}
	return sl
}

func ModUpdate(db *sql.DB, tab string, data interface{}) (sql.Result, error) {
	def := getDef(tab)
	sqlStr := fmt.Sprintf("update %s set %s where %s", def.Name, updateValue(def, data), GetWherePKey(def, data))
	ret, err := db.Exec(sqlStr)
	if err != nil {
		kernel.ErrorLog("%s", err.Error())
	}
	return ret, err
}

func ModUpdateFields(db *sql.DB, tab string,fields []string, data []interface{},where string) (sql.Result, error) {
	sqlStr := fmt.Sprintf("update %s set %s where %s", tab, updateFieldValue(fields, data), where)
	ret, err := db.Exec(sqlStr)
	if err != nil {
		kernel.ErrorLog("%s", err.Error())
	}
	return ret, err
}

func ModInsert(db *sql.DB, tab string, data interface{}) (sql.Result, error) {
	def := getDef(tab)
	sqlStr := fmt.Sprintf("insert into %s values (%s);", def.Name, insertValue(def, data))
	ret, err := db.Exec(sqlStr)
	if err != nil {
		kernel.ErrorLog("%s", err.Error())
	}
	return ret, err
}

func ModDelete(db *sql.DB, tab string, data interface{}) (sql.Result, error) {
	def := getDef(tab)
	sqlStr := fmt.Sprintf("delete from %s where (%s);", def.Name, GetWherePKey(def, data))
	ret, err := db.Exec(sqlStr)
	if err != nil {
		kernel.ErrorLog("%s", err.Error())
	}
	return ret, err
}

func ModDeletePKey(db *sql.DB, tab string, pkey ...interface{}) (sql.Result, error) {
	def := getDef(tab)
	sqlStr := fmt.Sprintf("delete from %s where (%s);", def.Name, GetWhere(def, pkey...))
	ret, err := db.Exec(sqlStr)
	if err != nil {
		kernel.ErrorLog("%s", err.Error())
	}
	return ret, err
}

func GetWhere(def *TabDef, key ...interface{}) string {
	if len(key) == 0 {
		return " 1 "
	} else if len(key) == 1 {
		switch key[0].(type) {
		case bool:
			return " 1 "
		}
		return fmt.Sprintf("%s = %s", def.Pkey[0], Encode(key[0]))
	}
	var sl []string
	for i, v := range key {
		sl = append(sl, fmt.Sprintf("%s = %s", def.Pkey[i], Encode(v)))
	}
	return strings.Join(sl, " and ")
}

func GetWherePKey(def *TabDef, data interface{}) string {
	var sl []string
	vf := reflect.ValueOf(data)
	if vf.Kind() == reflect.Ptr {
		vf = vf.Elem()
	}
	for _, fName := range def.Pkey {
		v := vf.FieldByName(fName).Interface()
		sl = append(sl, fmt.Sprintf("%s = %s", fName, Encode(v)))
	}
	return strings.Join(sl, " and ")
}

func updateValue(def *TabDef, data interface{}) string {
	vf := reflect.ValueOf(data)
	vt := reflect.TypeOf(data)
	if vf.Kind() == reflect.Ptr {
		vf = vf.Elem()
		vt = vt.Elem()
	}
	fieldNum := vf.NumField()
	sl := make([]string, fieldNum, fieldNum)
	for i := 0; i < fieldNum; i++ {
		t := vt.Field(i)
		f := vf.Field(i)
		sl[i] = fmt.Sprintf("%s = %s", t.Name, encodeValue(&f))
	}
	return strings.Join(sl, ",")
}

func updateFieldValue(fields []string, data []interface{}) string {
	sl := make([]string, len(fields))
	for i,v := range fields {
		sl[i] = fmt.Sprintf("%s = %s",v,data[i])
	}
	return strings.Join(sl, ",")
}

func insertValue(def *TabDef, data interface{}) string {
	vf := reflect.ValueOf(data)
	if vf.Kind() == reflect.Ptr {
		vf = vf.Elem()
	}
	fileNum := vf.NumField()
	sl := make([]string, fileNum, fileNum)
	for i := 0; i < fileNum; i++ {
		f := vf.Field(i)
		sl[i] = encodeValue(&f)
	}
	return strings.Join(sl, ",")
}

func toRow(def *TabDef, scanF scanFunc) interface{} {
	dataPtr := reflect.New(reflect.TypeOf(def.DataStruct).Elem()) //定义的时候要求是指针
	data := dataPtr.Elem()
	vf := reflect.ValueOf(data.Interface())
	fileNum := vf.NumField()
	scan := make([]interface{}, fileNum, fileNum)
	for i := 0; i < fileNum; i++ {
		f := vf.Field(i)
		switch f.Kind() {
		case reflect.Bool:
			var v int
			scan[i] = &v
		case reflect.Slice, reflect.Map, reflect.Struct, reflect.Ptr:
			var buf []byte
			scan[i] = &buf
		default:
			scan[i] = reflect.New(f.Type()).Interface()
		}
	}
	err := scanF(scan...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		kernel.ErrorLog("%s", err.Error())
		return nil
	}
	for i := 0; i < fileNum; i++ {
		f := data.Field(i)
		switch f.Kind() {
		case reflect.Bool:
			f.SetBool(*scan[i].(*int) == 1)
		case reflect.Slice, reflect.Map, reflect.Struct:
			v := reflect.New(f.Type())
			de := gob.NewDecoder(bytes.NewReader(*scan[i].(*[]byte)))
			de.DecodeValue(v)
			f.Set(v.Elem())
		case reflect.Ptr:
			v := reflect.New(f.Type().Elem())
			de := gob.NewDecoder(bytes.NewReader(*scan[i].(*[]byte)))
			de.DecodeValue(v)
			f.Set(v)
		default:
			f.Set(reflect.ValueOf(scan[i]).Elem())
		}
	}
	return dataPtr.Interface()
}

func toBinary(f *reflect.Value) []byte {
	if f.IsNil() {
		return nil
	} else {
		var buf = new(bytes.Buffer)
		encoder := gob.NewEncoder(buf)
		encoder.EncodeValue(*f)
		return buf.Bytes()
	}
}
