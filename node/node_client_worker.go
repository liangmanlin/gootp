package node

import (
	"github.com/liangmanlin/gootp/gate"
	"github.com/liangmanlin/gootp/gate/pb"
	"github.com/liangmanlin/gootp/kernel"
	"io"
	"log"
	"runtime/debug"
	"time"
)

func recPacket(conn gate.Conn, father *kernel.Pid) {
	if err := conn.SetReadDeadline(time.Time{}); err != nil { // 规避超时
		kernel.Cast(father, &gate.TcpError{ErrType:gate.ErrDeadLine,Err: err})
		return
	}
	// 节点连接毕竟少，这里考虑只使用原生库实现
	var connNet *gate.ConnNet
	var ok bool
	if connNet,ok = conn.(*gate.ConnNet);!ok {
		log.Panic("node not support other conn")
	}
	recv := true
	for {
		rec(connNet,father,&recv)
		if !recv {
			break // close
		}
	}
}

func rec(conn *gate.ConnNet,father *kernel.Pid,recv *bool) {
	defer func() {
		p := recover()
		if p != nil {
			kernel.ErrorLog("catch error:%s,Stack:%s", p, debug.Stack())
		}
	}()
	c := conn.Conn
	head := conn.GetHead()
	headBuf := make([]byte, head)
	var packSize int
	var pack []byte
	for {
		_, err := io.ReadAtLeast(c, headBuf, head)
		if err != nil {
			kernel.Cast(father, &gate.TcpError{ErrType:gate.ErrReadErr,Err: err})
			*recv = false
			return
		}
		packSize = gate.ReadHead(head, headBuf)
		if packSize > 0 {
			if packSize > cap(pack) {
				pack = make([]byte, packSize, packSize) // TODO 使用一个单次分配的缓冲区,后续应该慢慢缩小
			} else {
				pack = pack[:packSize]
			}
			_, err = io.ReadAtLeast(c, pack, packSize)
			if err != nil {
				kernel.Cast(father, &gate.TcpError{ErrType:gate.ErrReadErr,Err: err})
				kernel.ErrorLog("%#v",err)
				*recv = false
				return
			}
			router(pack, father)
		}
	}
}

func router(pack []byte, father *kernel.Pid) {
	switch pack[0] {
	case M_TYPE_PING: // ping
		kernel.Cast(father, ping{})
	case M_TYPE_CAST: //目标消息
		index, pid := kernel.DecodePid(pack, 1)
		if pid != nil {
			_, msg := coder.Decode(pack[index:])
			kernel.Cast(pid, msg)
		}
	case M_TYPE_CALL: // call
		_, callID := pb.DecodeInt64(pack, 1)
		index, pid := kernel.DecodePid(pack, 9)
		if pid != nil {
			_, msg := coder.Decode(pack[index:])
			ch := father.GetChannel()
			ci := &kernel.CallInfo{RecCh: ch, CallID: callID, Request: msg}
			kernel.Cast(pid, ci)
		}else{
			replyNil(callID,father)
		}
	case M_TYPE_CALL_RESULT: // call result
		index, callID := pb.DecodeInt64(pack, 1)
		var reply interface{}
		if len(pack) > 9 {
			_, reply = coder.Decode(pack[index:])
		}else{
			reply = nil
		}
		r := &call{callID: callID, reply: reply}
		kernel.Cast(father, r)
	case M_TYPE_CAST_NAME: // cast name
		index, name := pb.DecodeString(pack, 1)
		if pid := kernel.WhereIs(name); pid != nil {
			_, msg := coder.Decode(pack[index:])
			kernel.Cast(pid, msg)
		}
	case M_TYPE_CALL_NAME: // call name
		_, callID := pb.DecodeInt64(pack, 1)
		index, name := pb.DecodeString(pack, 9)
		if pid := kernel.WhereIs(name); pid != nil {
			_, msg := coder.Decode(pack[index:])
			ch := father.GetChannel()
			ci := &kernel.CallInfo{RecCh: ch, CallID: callID, Request: msg}
			kernel.Cast(pid, ci)
		}else{
			replyNil(callID,father)
		}
	case M_TYPE_CAST_KMSG:
		index, pid := kernel.DecodePid(pack, 1)
		if pid != nil {
			var modID int32
			index,modID = pb.DecodeInt32(pack,index)
			_, msg := coder.Decode(pack[index:])
			kernel.Cast(pid, &kernel.KMsg{ModID: modID,Msg: msg})
		}
	case M_TYPE_CAST_NAME_KMSG:
		index, name := pb.DecodeString(pack, 5)
		if pid := kernel.WhereIs(name); pid != nil {
			_,modID := pb.DecodeInt32(pack,1)
			_, msg := coder.Decode(pack[index:])
			kernel.Cast(pid, &kernel.KMsg{ModID: modID,Msg: msg})
		}
	case M_TYPE_PONG:
		kernel.Cast(father, pong{})
	}
}

func replyNil(callID int64,father *kernel.Pid)  {
	buf := []byte{0,0,0,9,3,0,0,0,0,0,0,0,0}
	pb.WriteInt64(buf, callID, 5)
	kernel.Cast(father,buf)
}
