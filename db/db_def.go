package db

import "reflect"

var dbTabDef map[string]*TabDef

type TabDef struct {
	Name       string
	Pkey       []string
	Keys       []string
	DataStruct interface{}
	nameType   map[string]reflect.Type
}

type dbVersion struct {
	TabName string
	Version string
}

func initDef(defSlice []*TabDef) {
	dbTabDef = make(map[string]*TabDef, 100)
	version := &TabDef{Name: "db_version", DataStruct: &dbVersion{}, Pkey: []string{"TabName"}}
	version.buildMap()
	dbTabDef["db_version"] = version
	for _, v := range defSlice {
		v.buildMap()
		dbTabDef[v.Name] = v
	}
}

func getDef(tab string) *TabDef {
	return dbTabDef[tab]
}

func (t *TabDef) buildMap() {
	vf := reflect.TypeOf(t.DataStruct).Elem()
	t.nameType = make(map[string]reflect.Type)
	num := vf.NumField()
	for i := 0; i < num; i++ {
		f := vf.Field(i)
		t.nameType[f.Name] = f.Type
	}
}
