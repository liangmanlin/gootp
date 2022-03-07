GOARCH=amd64 go build -o bin/gmpd.x64 main.go
GOARCH=386 go build -o bin/gmpd.x32 main.go