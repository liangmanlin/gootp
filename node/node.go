package node

import (
	"fmt"
	"github.com/liangmanlin/gootp/gate"
	"github.com/liangmanlin/gootp/gate/pb"
	"github.com/liangmanlin/gootp/kernel"
	"log"
	"net"
	"os/exec"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

var started bool
var coder *pb.Coder

/*
	需要引入gate模块以及pb模块，
*/

func Start(nodeName string, cookie string, defs []interface{}) {
	if runtime.GOOS == "windows" {
		log.Panicf("not support node on %s", runtime.GOOS)
	}
	// 先启动本地服务
	cmd := exec.Command("gmpd.sh", fmt.Sprintf("%d", gmpdPort))
	err := cmd.Run()
	if err != nil {
		log.Panic(err.Error())
	}
	time.Sleep(100 * time.Millisecond)
	checkNodeName(nodeName)
	Env.cookie = cookie
	// 加载pb
	def := []interface{}{
		&Connect{},
		&ConnectSucc{},
		&RpcCallArgs{},
	}
	def = append(def, defs...)
	coder = pb.ParseSlice(def, -1)
	Env.nodeName = nodeName
	if err = kernel.AppStart(&app{});err != nil {
		log.Panic(err)
	}
}

func start(nodeName string) {
	gate.Start("Node", nodeClient, Env.Port, gate.AcceptNum(5))
	addr := gate.GetAddr("Node")
	kernel.ErrorLog("Node: [%s] listen on: %s", nodeName, addr)
	// 分析出使用的端口
	exp := regexp.MustCompile(`.+:(\d+)`)
	m := exp.FindStringSubmatch(addr)
	port, _ := strconv.Atoi(m[1])
	failCount := 0
reg:
	// 向本地注册名字
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", gmpdPort))
	if err != nil {
		failCount++
		if failCount >= 10 {
			log.Panicf("can not connect to gmpd port: %d", gmpdPort)
			conn.Close()
		} else {
			time.Sleep(100 * time.Millisecond)
			goto reg
		}
	}
	c := gate.NewConn(conn)
	defer c.Close()
	c.SetHead(2)
	buf := []byte(fmt.Sprintf("1:%s:%d", nodeName, port))
	if c.Send(buf) != nil {
		log.Panicf("can not connect to gmpd port: %d", gmpdPort)
	}
	started = true
}

func checkNodeName(nodeName string) {
	exp := regexp.MustCompile(`\w+@[\w.]+`)
	if exp.MatchString(nodeName) {
		return
	}
	log.Panicf("Node name [%s] not allow", nodeName)
}

func ConnectNode(destNode string) bool {
	if !started {
		log.Panicf("node is not start")
	}
	exp := regexp.MustCompile(`\w+@([\w.]+)`)
	if !exp.MatchString(destNode) {
		kernel.ErrorLog("Node name [%s] not allow", destNode)
		return false
	}
	// 判断是否已经连接上
	if kernel.IsNodeConnect(destNode) {
		return true
	}

	fl := exp.FindStringSubmatch(destNode)
	ip := fl[1]
	port := gmpdPort
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		kernel.ErrorLog("can not connect to dest ip: %s", ip)
		return false
	}
	c := gate.NewConn(conn)
	c.SetHead(2)
	err = c.Send([]byte(fmt.Sprintf("2:%s", destNode)))
	if err != nil {
		kernel.ErrorLog("query port error: %s", err.Error())
		c.Close()
		return false
	}
	var buf []byte
	err, buf = c.Recv(0, 0)
	if err != nil {
		kernel.ErrorLog("query port error: %s", err.Error())
		c.Close()
		return false
	}
	c.Close()
	// 获取真实的端口
	port = int(buf[0])<<8 + int(buf[1])
	if port == 0 {
		return false
	}
	conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		kernel.ErrorLog("can not connect to Node: %s", destNode)
		c.Close()
		return false
	}
	c = gate.NewConn(conn)
	pid, _ := kernel.StartName(destNode, nodeClient, c)
	rs, e := kernel.Call(pid, false)
	if !rs {
		kernel.ErrorLog("connect to Node error:%#v", e)
	}
	kernel.ErrorLog("connect Node [%s] succ:%v", destNode, e)
	return e.(bool)
}

func GetCookie() string {
	return Env.cookie
}

func IsProtoDef(rType reflect.Type) bool {
	return coder.IsDef(rType)
}