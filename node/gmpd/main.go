package main

import (
	"fmt"
	"github.com/liangmanlin/gootp/gate"
	"net"
	"os"
	"regexp"
	"strconv"
	"sync"
)

var nodeMap sync.Map

func main() {
	port, _ := strconv.Atoi(os.Args[1])
	ls,err := net.Listen("tcp",fmt.Sprintf("0.0.0.0:%d",port))
	if err != nil {
		os.Exit(1)
	}
	fmt.Printf("start on port :%d\n",port)
	for{
		conn,err := ls.Accept()
		if err != nil {
			return
		}
		go handle(conn)
	}
}

func handle(conn net.Conn)  {
	defer func() {recover()}()
	c := gate.NewConn(conn)
	c.SetHead(2)
	err,buf := c.Recv(0,0)
	if err != nil {
		return
	}
	switch buf[0] {
	case 49:// "1" 注册本地端口
		exp := regexp.MustCompile(`(\w+@[^:]+):(\d+)`)
		sl := exp.FindStringSubmatch(string(buf[2:]))
		nodeName := sl[1]
		port,_ := strconv.Atoi(sl[2])
		fmt.Printf("node register:%s,port:%d\n",nodeName,port)
		nodeMap.Store(nodeName,port)
	case 50:// "2" 查询端口
		nodeName := string(buf[2:])
		fmt.Printf("query node:%s\n",nodeName)
		if p,ok := nodeMap.Load(nodeName);ok{
			port := p.(int)
			buf = []byte{uint8(port >> 8),uint8(port)}
			c.Send(buf)
		}else{
			buf = []byte{0,0}
			c.Send(buf)
		}
	}
	c.Close()
}