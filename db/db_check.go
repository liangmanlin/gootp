package db

import (
	"fmt"
	"github.com/liangmanlin/gootp/crypto"
	"github.com/liangmanlin/gootp/kernel"
	"log"
	"reflect"
	"strings"
)

func tableCheck(group *Group) {
	rows, err := group.db.Query("show tables;")
	if err != nil {
		if rows != nil {
			rows.Close()
		}
		kernel.ErrorLog(err.Error())
		log.Panic(err.Error())
	}
	tabs := make(map[string]string, len(group.dbTabDef))
	for rows.Next() {
		var tabName string
		err = rows.Scan(&tabName)
		ifExit(err)
		tabs[tabName] = "1"
	}
	_ = rows.Close()
	if _, ok := tabs["db_version"]; !ok {
		tabs["db_version"] = createTable(group, "db_version")
	}
	vers := group.SyncSelect(kernel.Call, "db_version", 1, true)
	for _, v := range vers {
		v2 := v.(*dbVersion)
		tabs[v2.TabName] = v2.Version
	}
	for tabName, def := range group.dbTabDef {
		md5, ok := tabs[tabName]
		if !ok {
			tabs[tabName] = createTable(group, tabName)
			continue
		}
		md52 := tabMd5(def.DataStruct)
		if md5 != md52 {
			kernel.ErrorLog("tab:[%s] check fields,ver: %s,old ver: %s", tabName, md5, md52)
			tabs[tabName] = md52
			checkField(group, def, md52)
		}
	}
}

func createTable(group *Group, tab string) string {
	def := group.GetDef(tab)
	md5Str := tabMd5(def.DataStruct)
	sqlStr := genCreateSql(def)
	_, err := group.db.Exec(sqlStr)
	ifExit(err)
	group.SyncInsert("db_version", 1, &dbVersion{TabName: tab, Version: md5Str})
	return md5Str
}

func genCreateSql(def *TabDef) string {
	vf := reflect.ValueOf(def.DataStruct)
	vt := reflect.TypeOf(def.DataStruct)
	if vf.Kind() == reflect.Ptr {
		vf = vf.Elem()
		vt = vt.Elem()
	}
	fNum := vf.NumField()
	fsl := make([]string, fNum, fNum)
	for i := 0; i < fNum; i++ {
		f := vf.Field(i)
		t := vt.Field(i)
		fsl[i] = getFileDef(&f, t.Name)
	}
	if len(def.Pkey) > 0 {
		psl := make([]string, 0, 2)
		for _, pk := range def.Pkey {
			psl = append(psl, pk)
		}
		fsl = append(fsl, fmt.Sprintf("PRIMARY KEY(%s)", strings.Join(psl, ",")))
	}
	for _, k := range def.Keys {
		fsl = append(fsl, fmt.Sprintf("KEY `%s` (`%s`)", k, k))
	}
	return fmt.Sprintf("create table if not exists `%s` (%s) ENGINE=InnoDB DEFAULT CHARSET=utf8;", def.Name, strings.Join(fsl, ",\n"))
}

func checkField(group *Group, def *TabDef, md5 string) {
	rows, err := group.db.Query(fmt.Sprintf("desc %s;", def.Name))
	ifExit(err)
	fmap := make(map[string]bool)
	for rows.Next() {
		var field, a, b, c, d, e string
		rows.Scan(&field, &a, &b, &c, &d, &e)
		fmap[field] = true
	}
	rows.Close()
	rt := reflect.TypeOf(def.DataStruct)
	rf := reflect.ValueOf(def.DataStruct)
	if rf.Kind() == reflect.Ptr {
		rt = rt.Elem()
		rf = rf.Elem()
	}
	fNum := rt.NumField()
	add := make([]string, 0, 0)
	for i := 0; i < fNum; i++ {
		t := rt.Field(i)
		f := rf.Field(i)
		if _, ok := fmap[t.Name]; !ok {
			add = append(add, fmt.Sprintf("Add %s AFTER `%s`", getFileDef(&f, t.Name), rt.Field(i-1).Name))
		} else {
			delete(fmap, t.Name)
		}
	}
	if len(add) > 0 {
		sqlAdd := fmt.Sprintf("ALTER TABLE `%s` %s;", def.Name, strings.Join(add, ","))
		_, err = group.db.Exec(sqlAdd)
		ifExit(err)
	}
	if len(fmap) > 0 {
		del := make([]string, 0, 0)
		for fn, _ := range fmap {
			del = append(del, fmt.Sprintf("DROP `%s`", fn))
		}
		sqlDel := fmt.Sprintf("ALTER TABLE `%s` %s;", def.Name, strings.Join(del, ","))
		_, err = group.db.Exec(sqlDel)
		ifExit(err)
	}
	group.SyncUpdate("db_version", 1, &dbVersion{TabName: def.Name, Version: md5})

}

func getFileDef(f *reflect.Value, fieldName string) string {
	var fs string
	switch f.Kind() {
	case reflect.Bool,reflect.Int,reflect.Uint,reflect.Int32:
		fs = fmt.Sprintf("`%s` INT( 11 ) NOT NULL", fieldName)
	case reflect.Int64,reflect.Uint64:
		fs = fmt.Sprintf("`%s` BIGINT( 23 ) NOT NULL", fieldName)
	case reflect.Ptr,reflect.Slice,reflect.Map,reflect.Struct:
		fs = fmt.Sprintf("`%s` mediumblob", fieldName)
	case reflect.String:
		fs = fmt.Sprintf("`%s` VARCHAR( 250 ) CHARACTER SET utf8 COLLATE utf8_general_ci NOT NULL", fieldName)
	default:
		log.Panic(fmt.Errorf("%s not support type,%s", fieldName, f.Type()))
	}
	return fs
}

func ifExit(err error) {
	if err != nil {
		kernel.ErrorLog(err.Error())
		log.Panic(err)
	}
}

func tabMd5(src interface{}) string {
	rt := reflect.TypeOf(src)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	n := rt.NumField()
	sl := make([]string,0,n)
	for i := 0; i < n; i++ {
		t := rt.Field(i)
		sl = append(sl, t.Name)
	}
	return crypto.Md5([]byte(strings.Join(sl, "_")))
}
