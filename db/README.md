# Db

## go mysql service

## useage

### insert

```go
SyncInsert(tab string, indexKey int64, data interface{})
ModInsert(db *sql.DB, tab string, data interface{}) (sql.Result, error)
```

### select
```go
SyncSelectRow(context *kernel.Context, tab string, indexKey int64, key ...interface{}) interface{}
SyncSelect(context *kernel.Context, tab string, indexKey int64, key ...interface{}) []interface{}
ModSelectRow(db *sql.DB, tab string, key []interface{}) interface{}
ModSelectAll(db *sql.DB, tab string, key []interface{}) []interface{}
```

### update

```go
SyncUpdate(tab string, indexKey int64, data interface{})
ModUpdate(db *sql.DB, tab string, data interface{}) (sql.Result, error)
```

### delete

```go
SyncDelete(tab string, indexKey int64, data interface{})
SyncDeletePKey(tab string, indexKey int64, pkey ...interface{})
ModDelete(db *sql.DB, tab string, data interface{}) (sql.Result, error)
ModDeletePKey(db *sql.DB, tab string, pkey []interface{}) (sql.Result, error)
```

## Test see: [db_test](db_test.go)
