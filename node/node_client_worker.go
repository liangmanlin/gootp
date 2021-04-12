package node

import (
	"fmt"
	"github.com/liangmanlin/gootp/gate"
	"github.com/liangmanlin/gootp/gate/pb"
	"github.com/liangmanlin/gootp/kernel"
	"io"
	"runtime/debug"
	"time"
)

func recPacket(conn *gate.Conn, father *kernel.Pid) {
	if err := conn.Conn.SetReadDeadline(time.Time{}); err != nil { // 规避超时
		kernel.Cast(father, &gate.TcpError{Err: err})
		return
	}
	recv := true
	for {
		rec(conn,father,&recv)
		if !recv {
			goto end
		}
	}
end:
}

func rec(conn *gate.Conn,father *kernel.Pid,recv *bool) {
	defer kernel.Catch()
	c := conn.Conn
	head := conn.GetHead()
	headBuf := make([]byte, head)
	var packSize int
	var pack []byte
	for {
		_, err := io.ReadAtLeast(c, headBuf, head)
		if err != nil {
			kernel.Cast(father, &gate.TcpError{Err: err})
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
				kernel.Cast(father, &gate.TcpError{Err: err})
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
	case 0: // ping
		kernel.Cast(father, ping(1))
	case 1: //目标消息
		index, pid := kernel.DecodePid(pack, 1)
		if pid != nil {
			_, msg := coder.Decode(pack[index:])
			kernel.Cast(pid, msg)
		}
	case 2: // call
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
	case 3: // call result
		index, callID := pb.DecodeInt64(pack, 1)
		var reply interface{}
		if len(pack) > 9 {
			_, reply = coder.Decode(pack[index:])
		}else{
			reply = nil
		}
		r := &call{callID: callID, reply: reply}
		kernel.Cast(father, r)
	case 4:
		index, name := pb.DecodeString(pack, 1)
		if pid := kernel.WhereIs(name); pid != nil {
			_, msg := coder.Decode(pack[index:])
			kernel.Cast(pid, msg)
		}
	case 5: // call name
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
	}
}

func replyNil(callID int64,father *kernel.Pid)  {
	buf := []byte{0,0,0,9,3,0,0,0,0,0,0,0,0}
	pb.WriteInt64(buf, callID, 5)
	kernel.Cast(father,buf)
}

func catch(father *kernel.Pid) {
	err := recover()
	if err != nil {
		kernel.ErrorLog("catch error reason: %s,Stack: %s", err, debug.Stack())
	}
	kernel.Cast(father, &gate.TcpError{Err: fmt.Errorf("catch error")})
}
