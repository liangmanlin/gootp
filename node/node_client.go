package node

import (
	"fmt"
	"github.com/liangmanlin/gootp/gate"
	"github.com/liangmanlin/gootp/gate/pb"
	"github.com/liangmanlin/gootp/kernel"
	"github.com/liangmanlin/gootp/kernel/crypto"
	"unsafe"
)

var pingBuf = []byte{0,0,0,1,0}

type client struct {
	conn           *gate.Conn
	node           *kernel.Node
	callID2Channel map[int64]chan interface{}
	pingFail       int
}

var nodeClient = &kernel.Actor{
	Init: func(context *kernel.Context,pid *kernel.Pid, args ...interface{}) unsafe.Pointer {
		c := client{}
		c.conn = args[0].(*gate.Conn)
		c.callID2Channel = make(map[int64]chan interface{})
		return unsafe.Pointer(&c)
	},
	HandleCast: func(context *kernel.Context, msg interface{}) {
		state := (*client)(context.State)
		switch m := msg.(type) {
		case *kernel.NodeMsg:
			if state.conn != nil {
				buf := m.Dest.ToBytes([]byte{0, 0, 0, 0, 1})
				b := coder.Encode(m.Msg, 0)
				buf = append(buf, b...)
				gate.WriteSize(buf, 4, len(buf)-4)
				state.conn.SendBufHead(buf)
			}
		case *kernel.NodeCall:
			if state.conn != nil {
				state.callID2Channel[m.CallID] = m.Ch
				buf := m.Dest.ToBytes([]byte{0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0}) // head+type+callID
				pb.WriteInt64(buf, m.CallID, 5)
				b := coder.Encode(m.Req, 0)
				buf = append(buf, b...)
				gate.WriteSize(buf, 4, len(buf)-4)
				state.conn.SendBufHead(buf)
			}
		case *kernel.NodeCallName:
			if state.conn != nil {
				state.callID2Channel[m.CallID] = m.Ch
				size := 15
				buf := make([]byte, size)
				buf[4] = 5
				pb.WriteInt64(buf, m.CallID, 5)
				buf = pb.WriteString(buf, m.Dest, 13)
				b := coder.Encode(m.Req, 0)
				buf = append(buf, b...)
				gate.WriteSize(buf, 4, len(buf)-4)
				state.conn.SendBufHead(buf)
			}
		case *kernel.CallResult:
			if state.conn != nil {
				buf := []byte{0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0} // head+type+callID
				pb.WriteInt64(buf, m.ID, 5)
				if m.Result != nil {
					b := coder.Encode(m.Result, 0)
					buf = append(buf, b...)
				}
				gate.WriteSize(buf, 4, len(buf)-4)
				state.conn.SendBufHead(buf)
			}
		case *call:
			if ch, ok := state.callID2Channel[m.callID]; ok {
				defer kernel.CatchNoPrint() // channel maybe closed
				delete(state.callID2Channel, m.callID)
				ch <- &kernel.CallResult{ID: m.callID, Result: m.reply}
			}
		case *kernel.NodeMsgName:
			if state.conn != nil {
				size := 7
				buf := make([]byte, size)
				buf[4] = 4
				buf = pb.WriteString(buf, m.Dest, 5)
				b := coder.Encode(m.Msg, 0)
				buf = append(buf, b...)
				gate.WriteSize(buf, 4, len(buf)-4)
				state.conn.SendBufHead(buf)
			}
		case bool:
			startWorker(state, context)
		case []byte:
			state.conn.SendBufHead(m)
		case *gate.TcpError:
			context.Exit("normal")
		case ping:
			// ping
			state.pingFail = 0
		case int:
			if state.pingFail >=2 {
				// 断开连接
				kernel.ErrorLog("Node ping fail")
				context.Exit("normal")
				break
			}
			state.pingFail++
			// 发送ping包
			state.conn.SendBufHead(pingBuf)
		default:
			kernel.ErrorLog("un handle msg :%#v", m)
		}
	},
	HandleCall: func(context *kernel.Context, request interface{}) interface{} {
		state := (*client)(context.State)
		switch request.(type) {
		case bool:
			return startWorkerConnect(state, context)
		}
		return nil
	},
	Terminate: func(context *kernel.Context, reason *kernel.Terminate) {
		state := (*client)(context.State)
		if state.conn != nil {
			state.conn.Close()
		}
		// 报告节点退出
		if state.node != nil {
			kernel.NodeDisconnect(state.node)
			kernel.Cast(monitorPid,&NodeOP{Name: state.node.Name(),OP: OPDisConnect})
			kernel.ErrorLog("Node [%s] disconnect", state.node.Name())
		}
	},
	ErrorHandler: func(context *kernel.Context, err interface{}) bool {
		kernel.ErrorLog("handle error:%#v", err)
		return false
	},
}

func startWorker(state *client, context *kernel.Context) {
	state.conn.SetHead(4)
	err, buf := state.conn.Recv(0, 0)
	if err != nil {
		context.Exit("normal")
		return
	}
	_, msg := coder.Decode(buf)
	switch m := msg.(type) {
	case *Connect:
		if !checkCookie(m) {
			context.Exit("normal")
			kernel.ErrorLog("Node:%s connect error,time:%d,sign:%s", m.Name, m.Time, m.Sign)
			return
		}
		// 分配本地id
		state.node = kernel.GetNode(m.Name)
		kernel.SetNodeNetWork(state.node, context.Self())
		kernel.Cast(monitorPid,&NodeOP{Name: m.Name,OP: OPConnect})

		kernel.ErrorLog("Node [%s] connected", m.Name)

		buf = coder.Encode(&ConnectSucc{Name: Env.nodeName, Self: context.Self()}, 4)
		state.conn.SendBufHead(buf)
		kernel.TimerStart(kernel.TimerTypeForever, context.Self(), Env.PingTick, 1)
		// 启动接收进程
		go recPacket(state.conn, context.Self())
	}
}

func startWorkerConnect(state *client, context *kernel.Context) bool {
	state.conn.SetHead(4)
	time := kernel.Now()
	sign := crypto.Md5(([]byte)(fmt.Sprintf("%s.%d", Env.cookie, time)))
	info := &Connect{Time: time, Sign: sign, Name: Env.nodeName,DestName: context.Name()}
	buf := coder.Encode(info, 4)
	state.conn.SendBufHead(buf)
	err, buf := state.conn.Recv(0, 0)
	if err != nil {
		context.Exit("normal")
		return false
	}
	_, msg := coder.Decode(buf)
	switch m := msg.(type) {
	case *ConnectSucc:
		// 分配本地id
		state.node = kernel.GetNode(m.Name)
		kernel.SetNodeNetWork(state.node, context.Self())
		kernel.Cast(monitorPid,&NodeOP{Name: m.Name,OP: OPConnect})
		kernel.TimerStart(kernel.TimerTypeForever, context.Self(),Env.PingTick, 1)
		// 启动接收进程
		go recPacket(state.conn, context.Self())
		return true
	}
	context.Exit("normal")
	return false
}

func checkCookie(m *Connect) bool {
	if m.Time+60 < kernel.Now() {
		return false
	}
	// 有可能端口复用了，但是不是这个节点了
	if m.DestName != Env.nodeName {
		return false
	}
	return m.Sign == crypto.Md5(([]byte)(fmt.Sprintf("%s.%d", Env.cookie, m.Time)))
}
