package kernel

import (
	"strconv"
	"sync/atomic"
)

var deadPid = &Pid{isAlive: 0}

type Pid struct {
	isAlive    int32 // 放在第一个位置，有利于cpu快速定位
	id         int64
	c          chan interface{}
	callResult chan interface{}
	node       *Node // 添加分布式支持
}

func (p *Pid) GetChannel() chan interface{} {
	return p.c
}

func (p *Pid) GetID() int64 {
	return p.id
}

func (p *Pid) String() string {
	if p.node == nil {
		return "<pid:" + strconv.FormatInt(p.id, 10) + ">"
	}
	return "<pid:" + strconv.FormatInt(p.id, 10) + ":" + p.node.name + ">"
}

func (p *Pid) IsAlive() bool {
	if p == nil || p.node != nil {
		return false
	}
	return atomic.LoadInt32(&p.isAlive) == 1
}

// 在一些阻塞的进程里，可以设置为消亡，这样停服就不会卡主
func (p *Pid) SetDie() {
	atomic.StoreInt32(&p.isAlive,0)
}

func (p *Pid) Node() *Node {
	if p.node == nil {
		return node
	}
	return p.node
}

func (p *Pid)Cast(msg interface{})  {
	Cast(p,msg)
}

func (p *Pid) ToBytes(buf []byte) []byte {
	v := p.id
	buf = append(buf, uint8(v>>56), uint8(v>>48), uint8(v>>40), uint8(v>>32), uint8(v>>24), uint8(v>>16), uint8(v>>8), uint8(v))
	var nodeName string
	if p.node != nil {
		nodeName = p.node.name
	} else {
		nodeName = node.name
	}
	lens := len(nodeName)
	buf = append(buf, uint8(lens>>8), uint8(lens))
	buf = append(buf, nodeName...)
	return buf
}

func DecodePid(buf []byte, index int) (int, *Pid) {
	id := int64(buf[index])<<56 + int64(buf[index+1])<<48 + int64(buf[index+2])<<40 + int64(buf[index+3])<<32 +
		int64(buf[index+4])<<24 + int64(buf[index+5])<<16 + int64(buf[index+6])<<8 + int64(buf[index+7])
	index += 8
	var size int
	size = int(buf[index]) << 8
	size += int(buf[index+1])
	index += 2
	end := index + size
	nodeName := string(buf[index:end])
	// 判断是否本地pid
	if node.name == nodeName {
		// 判断pid存活
		if pid, ok := kernelAliveMap.Load(id); ok {
			return end, pid.(*Pid)
		}
		return end, deadPid
	}
	n := GetNode(nodeName)
	return end, &Pid{id: id, c: nil, node: n}
}

func LocalPid(id int64) *Pid {
	if pid, ok := kernelAliveMap.Load(id); ok {
		return pid.(*Pid)
	}
	return nil
}
