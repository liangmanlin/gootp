package db

import "reflect"

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

func initDef(g *Group,defSlice []*TabDef) {
	g.dbTabDef = make(map[string]*TabDef, 100)
	version := &TabDef{Name: "db_version", DataStruct: &dbVersion{}, Pkey: []string{"TabName"}}
	version.buildMap()
	g.dbTabDef["db_version"] = version
	for _, v := range defSlice {
		v.buildMap()
		g.dbTabDef[v.Name] = v
	}
}

func (g *Group)GetDef(tab string) *TabDef {
	return g.dbTabDef[tab]
}

func (g *Group)GetAllDef() []*TabDef {
	l := make([]*TabDef,0,len(g.dbTabDef))
	for _,v := range g.dbTabDef {
		l = append(l,v)
	}
	return l
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
