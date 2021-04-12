package kernel

import (
	"strconv"
	"sync/atomic"
)

type Pid struct {
	id      int64
	c       chan interface{}
	call    chan interface{}
	node    *Node // 添加分布式支持
	isAlive int32
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

func (p *Pid) Node() *Node {
	if p.node == nil {
		return node
	}
	return p.node
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

// 进程退出时调用，使得判断进程是否存活不需要获取一个全局锁
func (p *Pid) exit() {
	atomic.StoreInt32(&p.isAlive, 0)
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
		return end, nil
	}
	n := GetNode(nodeName)
	return end, &Pid{id: id, c: nil, node: n}
}
