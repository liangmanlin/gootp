package db

type env struct {
	dbConfig    Config
	ConnNum     int
	SyncNum     int64
	IsOpenCache bool
}

type Config struct {
	Host		string
	Port		int
	User		string
	PWD			string
}
