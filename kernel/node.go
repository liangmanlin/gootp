package kernel

import (
	"sync"
	"sync/atomic"
)

var node = &Node{id: 0}

var nodeMap sync.Map

var nodeId2Node sync.Map

var nodeID int32

var nodeNetWork sync.Map

func SetSelfNodeName(name string) {
	node.name = name
	nodeMap.Store(name, node)
}

func SelfNode() *Node {
	return node
}

func GetNode(nodeName string) *Node {
	if n, ok := nodeMap.Load(nodeName); ok {
		return n.(*Node)
	}
	id := atomic.AddInt32(&nodeID,1)
	n := &Node{id: id, name: nodeName}
	nodeMap.Store(nodeName, n)
	nodeId2Node.Store(id, n)
	return n
}

func SetNodeNetWork(node *Node, pid *Pid) {
	nodeNetWork.Store(node.id, pid)
}

func NodeDisconnect(node *Node) {
	nodeNetWork.Delete(node.id)
}

func GetNodeNetWork(node *Node) (*Pid, bool) {
	if p, ok := nodeNetWork.Load(node.id); ok {
		return p.(*Pid), true
	}
	return nil, false
}

func IsNodeConnect(nodeName string) bool {
	n, ok := nodeMap.Load(nodeName)
	if !ok {
		return false
	}
	id := n.(*Node).id
	if p, ok := nodeNetWork.Load(id); ok {
		return p.(*Pid).IsAlive()
	}
	return false
}

func Nodes() []*Node {
	var rs []*Node
	nodeNetWork.Range(func(key, value interface{}) bool {
		if n, ok := nodeId2Node.Load(key); ok {
			rs = append(rs, n.(*Node))
		}
		return true
	})
	return rs
}

func (n *Node) Name() string {
	return n.name
}

// 判断节点相等
func (n *Node) Equal(node *Node) bool {
	return n.id == node.id
}
