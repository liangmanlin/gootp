package node

import (
	"fmt"
	"github.com/liangmanlin/gootp/args"
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
	nodeStart(nodeName, cookie, defs, true)
}

func StartHidden(nodeName string, cookie string, defs []interface{}) {
	nodeStart(nodeName, cookie, defs, false)
}

func StartFromCommandLine(defs []interface{})  {
	var nodeName,cookie string
	var ok bool
	if nodeName,ok = args.GetString("name");!ok{
		log.Panic("node miss -name arg")
	}
	if cookie,ok = args.GetString("cookie");!ok{
		log.Panic("node miss -cookie arg")
	}
	nodeStart(nodeName, cookie, defs, true)
}

func nodeStart(nodeName string, cookie string, defs []interface{}, register bool) {
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
		&kernel.ConsoleCommand{},
	}
	def = append(def, defs...)
	coder = pb.ParseSlice(def, -1)
	Env.nodeName = nodeName
	if err = kernel.AppStart(&app{register: register}); err != nil {
		panic(err)
	}
}

func start(nodeName string, register bool) {
	// 首先验证当前节点是否启动
	if ok, c := tryConnect(nodeName, true); ok {
		c.Close()
		log.Panicf("node:[%s] aready started", nodeName)
	}
	gate.Start("Node", nodeClient, Env.Port, gate.WithAcceptNum(5))
	addr := gate.GetAddr("Node")
	kernel.ErrorLog("Node: [%s] listen on: %s", nodeName, addr)
	if !register {
		started = true
		return
	}
	// 分析出使用的端口
	exp := regexp.MustCompile(`.+:(\d+)`)
	m := exp.FindStringSubmatch(addr)
	port, _ := strconv.Atoi(m[1])
	var c gate.Conn
	var ok bool
	if ok, c = connectGmpd("127.0.0.1", true); !ok {
		return
	}
	defer c.Close()
	c.SetHead(2)
	buf := []byte(fmt.Sprintf("1:%s:%d", nodeName, port))
	if _,err := c.Send(buf);err != nil {
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
	// 判断是否已经连接上
	if kernel.IsNodeConnect(destNode) {
		return true
	}
	ok, c := tryConnect(destNode, false)
	if !ok {
		return false
	}
	pid, _ := kernel.StartName(destNode, nodeClient, c)
	rs, e := kernel.Call(pid, false)
	if !rs {
		kernel.ErrorLog("connect to Node error:%#v", e)
	}
	kernel.ErrorLog("connect Node [%s] succ:%v", destNode, e)
	return e.(bool)
}

func tryConnect(destNode string, abort bool) (ok bool,cn gate.Conn) {
	exp := regexp.MustCompile(`\w+@([\w.]+)`)
	if !exp.MatchString(destNode) {
		kernel.ErrorLog("Node name [%s] not allow", destNode)
		return false, nil
	}
	fl := exp.FindStringSubmatch(destNode)
	ip := fl[1]
	var c gate.Conn
	if ok, c = connectGmpd(ip, abort); !ok {
		return false, nil
	}
	c.SetHead(2)
	defer c.Close()
	_,err := c.Send([]byte(fmt.Sprintf("2:%s", destNode)))
	if err != nil {
		if abort {
			log.Panicf("query port error: %s", err.Error())
		} else {
			return false, nil
		}
	}
	var buf []byte
	buf,err = c.Recv(0, 0)
	if err != nil {
		if abort {
			log.Panicf("query port error: %s", err.Error())
		} else {
			return false, nil
		}
	}
	// 获取真实的端口
	port := int(buf[0])<<8 + int(buf[1])
	if port == 0 {
		return false, nil
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		if !abort {
			kernel.ErrorLog("can not connect to Node: %s", destNode)
		}
		return false, nil
	}
	return true, gate.NewConn(conn)
}

func connectGmpd(ip string, abort bool) (bool, gate.Conn) {
	failCount := 0
reg:
	// 向本地注册名字
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, gmpdPort))
	if err != nil {
		failCount++
		if failCount >= 10 {
			if abort {
				log.Panicf("can not connect to gmpd %s port: %d", ip,gmpdPort)
			} else {
				return false, nil
			}
		} else {
			time.Sleep(100 * time.Millisecond)
			goto reg
		}
	}
	return true, gate.NewConn(conn)
}

func GetCookie() string {
	return Env.cookie
}

func IsProtoDef(rType reflect.Type) bool {
	return coder.IsDef(rType)
}
