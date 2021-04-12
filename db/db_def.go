package db

var dbTabDef map[string]*TabDef

type TabDef struct {
	Name       string
	Pkey       []string
	Keys       []string
	DataStruct interface{}
}

type dbVersion struct {
	TabName string
	Version string
}

func initDef(defSlice []*TabDef) {
	dbTabDef = make(map[string]*TabDef,100)
	dbTabDef["db_version"] = &TabDef{Name: "db_version", DataStruct: &dbVersion{}, Pkey: []string{"TabName"}}
	for _,v := range defSlice {
		dbTabDef[v.Name] = v
	}
}

func getDef(tab string) *TabDef {
	return dbTabDef[tab]
}
