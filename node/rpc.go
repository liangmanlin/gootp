package node

import (
	"github.com/liangmanlin/gootp/kernel"
	"runtime/debug"
)

var callBack = make(map[string]*func(interface{})interface{})


// args 是一个协议struct，对应的函数会接收到这个参数
func RpcCall(node interface{}, fun string, argStruct interface{}) (succ bool, result interface{}) {
	var dn *kernel.Node
	switch n := node.(type) {
	case string:
		dn = kernel.GetNode(n)
	case *kernel.Node:
		dn = n
	default:
		return
	}
	if dn.Equal(kernel.SelfNode()) {
		succ = true
		return succ, callFunc(fun, argStruct)
	}
	var buf []byte
	if argStruct != nil {
		buf = coder.Encode(argStruct,0)
	}
	a := &RpcCallArgs{Fun: fun,Args: buf}
	return kernel.CallNameNode("RPC",dn,a)
}

// 注册rpc回调函数
// 不建议运行时动态注册，因为没有加锁，建议项目启动前注册函数
func RpcRegister(fun string,function RpcFunc)  {
	callBack[fun] = function
}

func callFunc(fun string, argStruct interface{}) interface{} {
	defer func() {
		p := recover()
		if p != nil {
			kernel.ErrorLog("catch error:%s,Stack:%s", p, debug.Stack())
		}
	}()
	if f,ok:= callBack[fun];ok{
		return (*f)(argStruct)
	}
	return nil
}
