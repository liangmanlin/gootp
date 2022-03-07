package node

import (
	"fmt"
	"github.com/liangmanlin/gootp/crypto"
	"github.com/liangmanlin/gootp/gate"
	"github.com/liangmanlin/gootp/gate/pb"
	"github.com/liangmanlin/gootp/kernel"
	"reflect"
)

var pingBuf = []byte{0, 0, 0, 1, M_TYPE_PING}
var pongBuf = []byte{0, 0, 0, 1, M_TYPE_PONG}

type client struct {
	conn           gate.Conn
	node           *kernel.Node
	callID2Channel map[int64]chan interface{}
	pingFail       int
	buf            []byte
}

var nodeClient = &kernel.Actor{
	Init: func(context *kernel.Context, pid *kernel.Pid, args ...interface{}) interface{} {
		c := client{}
		c.conn = args[0].(gate.Conn)
		c.callID2Channel = make(map[int64]chan interface{})
		return &c
	},
	HandleCast: func(context *kernel.Context, msg interface{}) {
		state := context.State.(*client)
		buf := state.buf[0:0]
		switch m := msg.(type) {
		case *kernel.NodeMsg:
			if state.conn != nil {
				switch tm := m.Msg.(type) {
				case *kernel.KMsg:
					// 特殊结构
					buf = makeBuf(buf, 9, M_TYPE_CAST_KMSG)
					buf = m.Dest.ToBytes(buf)
					pb.WriteIn32(buf, tm.ModID, 5)
					buf = coder.EncodeBuff(tm.Msg, 0, buf)
				default:
					buf = makeBuf(buf, 5, M_TYPE_CAST)
					buf = m.Dest.ToBytes(buf)
					buf = coder.EncodeBuff(m.Msg, 0, buf)
				}
				gate.WriteSize(buf, 4, len(buf)-4)
				state.conn.SendBufHead(buf)
			}
		case *kernel.NodeCall:
			if state.conn != nil {
				state.callID2Channel[m.CallID] = m.Ch
				buf = makeBuf(buf, 13, M_TYPE_CAST_KMSG)
				buf = m.Dest.ToBytes(buf) // head+type+callID
				pb.WriteInt64(buf, m.CallID, 5)
				buf = coder.EncodeBuff(m.Req, 0, buf)
				gate.WriteSize(buf, 4, len(buf)-4)
				state.conn.SendBufHead(buf)
			}
		case *kernel.NodeCallName:
			if state.conn != nil {
				state.callID2Channel[m.CallID] = m.Ch
				const size = 15
				buf = makeBuf(buf, size, M_TYPE_CALL_NAME)
				pb.WriteInt64(buf, m.CallID, 5)
				buf = pb.WriteString(buf, m.Dest, 13)
				buf = coder.EncodeBuff(m.Req, 0, buf)
				gate.WriteSize(buf, 4, len(buf)-4)
				state.conn.SendBufHead(buf)
			}
		case *kernel.CallResult:
			if state.conn != nil {
				buf = makeBuf(buf, 13, M_TYPE_CALL_RESULT)
				pb.WriteInt64(buf, m.ID, 5)
				if !reflect.ValueOf(m.Result).IsNil() {
					buf = coder.EncodeBuff(m.Result, 0, buf)
				}
				gate.WriteSize(buf, 4, len(buf)-4)
				state.conn.SendBufHead(buf)
			}
		case *call:
			if ch, ok := state.callID2Channel[m.callID]; ok {
				defer func() {recover()}() // channel maybe closed
				delete(state.callID2Channel, m.callID)
				ch <- &kernel.CallResult{ID: m.callID, Result: m.reply}
			}
		case *kernel.NodeMsgName:
			if state.conn != nil {
				switch tm := m.Msg.(type) {
				case *kernel.KMsg:
					const size = 11
					buf = makeBuf(buf, size, M_TYPE_CAST_NAME_KMSG)
					pb.WriteIn32(buf, tm.ModID, 5)
					buf = pb.WriteString(buf, m.Dest, 9)
					buf = coder.EncodeBuff(tm.Msg, 0, buf)
				default:
					const size = 7
					buf = makeBuf(buf, size, M_TYPE_CAST_NAME)
					buf = pb.WriteString(buf, m.Dest, 5)
					buf = coder.EncodeBuff(m.Msg, 0, buf)
				}
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
			state.conn.SendBufHead(pongBuf)
		case pong:
			state.pingFail = 0
		case int:
			if state.pingFail >= 2 {
				// 断开连接
				kernel.ErrorLog("Node %s ping fail", context.Name())
				context.Exit("normal")
				break
			}
			state.pingFail++
			// 发送ping包
			state.conn.SendBufHead(pingBuf)
		default:
			kernel.ErrorLog("un handle msg :%#v", m)
		}
		state.buf = buf
	},
	HandleCall: func(context *kernel.Context, request interface{}) interface{} {
		state := context.State.(*client)
		switch request.(type) {
		case bool:
			return startWorkerConnect(state, context)
		}
		return nil
	},
	Terminate: func(context *kernel.Context, reason *kernel.Terminate) {
		state := context.State.(*client)
		if state.conn != nil {
			state.conn.Close()
		}
		// 报告节点退出
		if state.node != nil {
			kernel.NodeDisconnect(state.node)
			kernel.Cast(monitorPid, &NodeOP{Name: state.node.Name(), OP: OPDisConnect})
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
	buf, err := state.conn.Recv(0, 0)
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
		kernel.Cast(monitorPid, &NodeOP{Name: m.Name, OP: OPConnect})

		kernel.ErrorLog("Node [%s] connected", m.Name)

		buf = coder.Encode(&ConnectSucc{Name: Env.nodeName, Self: context.Self()}, 4)
		state.conn.SendBufHead(buf)
		kernel.SendAfterForever(context.Self(), Env.PingTick, 1)
		// 启动接收进程
		go recPacket(state.conn, context.Self())
	}
}

func startWorkerConnect(state *client, context *kernel.Context) bool {
	state.conn.SetHead(4)
	time := kernel.Now()
	sign := crypto.Md5(([]byte)(fmt.Sprintf("%s.%d", Env.cookie, time)))
	info := &Connect{Time: time, Sign: sign, Name: Env.nodeName, DestName: context.Name()}
	buf := coder.Encode(info, 4)
	state.conn.SendBufHead(buf)
	buf, err := state.conn.Recv(0, 0)
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
		kernel.Cast(monitorPid, &NodeOP{Name: m.Name, OP: OPConnect})
		kernel.SendAfterForever(context.Self(), Env.PingTick, 1)
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

func makeBuf(buf []byte, size int, v byte) []byte {
	if cap(buf) < size {
		buf = make([]byte, size, 2*size)
		buf[4] = v
	} else {
		buf = buf[0:size]
		buf[4] = v
	}
	return buf
}
