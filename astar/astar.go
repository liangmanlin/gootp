package astar

import (
	"github.com/liangmanlin/gootp/gutil"
	"sync"
)

type ReturnType int32

const (
	ASTAR_SAME_POS ReturnType = iota + 1
	ASTAR_FOUNDED
	ASTAR_UNREACHABLE
)

type AStar struct {
	idx     uint8
	nodeIdx []int32
	nodes   []node
}

type node struct {
	X, Y   int16
	idx    uint8
	g, f   int32
	parent *node
}

type aStarOpenNode struct {
	node *node
	next *aStarOpenNode
}

type grid struct {
	x, y int16
}

type MapStatus interface {
	GetAStarCache() *AStar
	SetAStarCache(star *AStar)
	GetGridConfig() GridConfig
}

type GridConfig interface {
	XYI32WalkAble(x, y int32) bool
	XYI32WalkAbleBorder(x, y int32, c int) int
	GetWidth() int32
	GetHeight() int32
	GridType(gridIdx int32) (gridType int16, g int16) // g = 1 or 3
}

var dirNormal = [8]grid{
	{0, 1}, {0, -1}, {-1, 0}, {1, 0}, //直线
	{-1, 1}, {1, 1}, {-1, -1}, {1, -1}, //斜线
}

/*
简单A*变种，并不是最优路径
*/

func Search(maps MapStatus, x, y, tx, ty float32) (ReturnType, []int16) {
	sx := gutil.Round(x)
	sy := gutil.Round(y)
	dx := gutil.Round(tx)
	dy := gutil.Round(ty)
	if sx == dx && sy == dy {
		return ASTAR_SAME_POS, nil
	}
	config := maps.GetGridConfig()
	if !config.XYI32WalkAble(dx, dy) {
		return ASTAR_UNREACHABLE, nil
	}
	// 判断直线可达
	if IsThrough(sx, sy, dx, dy, config.XYI32WalkAble) {
		return ASTAR_FOUNDED, nil
	}
	if maps.GetAStarCache() == nil {
		maps.SetAStarCache(newAStar(config))
	}
	astar := maps.GetAStarCache()
	width := config.GetWidth()
	astar.idx++
	ci := astar.idx
	cellLength := (width + 1) * (config.GetHeight() + 1)
	openNode := newOpenNode()
	childIdx := gridIdx(sx, sy, width)
	endIdx := gridIdx(dx, dy, width)
	var curNode, childNode *node
	var ok, found bool
	curNode, ok = astar.getNode(childIdx)
	if !ok {
		curNode.X = int16(sx)
		curNode.Y = int16(sy)
	}
	curNode.f = gutil.Abs(dx-sx) + gutil.Abs(dy-sy)
	curNode.g = 0
	curNode.parent = curNode
	curNode.idx = ci
	// 设置开放节点
	openNode.next = nil
	openNode.node = curNode

	var g, h int32
	var i int
	var dirPos grid
	var gridG, gridType int16
	for openNode != nil {
		openNode, curNode = minOpenNode(openNode)
		for i = 0; i < 8; i++ {
			dirPos = dirNormal[i]
			childIdx = gridIdx(int32(curNode.X+dirPos.x), int32(curNode.Y+dirPos.y), width)
			if childIdx < 0 || childIdx >= cellLength {
				continue
			}
			gridType, gridG = config.GridType(childIdx)
			if gridType == 0 {
				continue
			}
			childNode, ok = astar.getNode(childIdx)
			if !ok {
				childNode.X = curNode.X + dirPos.x
				childNode.Y = curNode.Y + dirPos.y
			} else if childNode.idx == ci {
				continue
			}
			h = gutil.Abs(dx-int32(curNode.X)) + gutil.Abs(dy-int32(curNode.Y))
			g = curNode.g + int32(gridG)
			childNode.parent = curNode
			childNode.g = g
			childNode.f = g + h
			childNode.idx = ci
			openNode = insertOpen(openNode, childNode)
			if childIdx == endIdx {
				found = true
				goto founded
			}
		}
	}
founded:
	var tmpO *aStarOpenNode
	// 释放回池
	for openNode != nil {
		tmpO = openNode.next
		_openNodePool.Put(openNode)
		openNode = tmpO
	}
	if found {
		pathNode, _ := astar.getNode(endIdx)
		endNode := pathNode
		start := pathNode
		// 移除最后一个点,因为一开始就把起点赋值为自己
		if pathNode.parent != pathNode {
			pathNode = pathNode.parent
		}
		var count int
		for {
			if pathNode.parent == pathNode {
				pathNode.parent = start
				break
			}
			if !IsThroughBorder(int32(start.X), int32(start.Y), int32(pathNode.parent.X), int32(pathNode.parent.Y), config.XYI32WalkAbleBorder) {
				tmp := pathNode.parent
				// 通过反向构造,同时统计出拐点数量，减少分配对象的时间
				pathNode.parent = start
				count++
				start = pathNode
				pathNode = tmp
			} else {
				pathNode = pathNode.parent
			}
		}
		pathNode.parent = start
		gridList := make([]int16, 0, count*2)
		for pathNode.parent != endNode {
			gridList = append(gridList, pathNode.parent.X, pathNode.parent.Y)
			pathNode = pathNode.parent
		}
		return ASTAR_FOUNDED, gridList
	}
	return ASTAR_UNREACHABLE, nil
}

func minOpenNode(openNode *aStarOpenNode) (*aStarOpenNode, *node) {
	min := openNode
	openNode = openNode.next
	n := min.node
	_openNodePool.Put(min)
	return openNode, n
}

func insertOpen(openNode *aStarOpenNode, childNode *node) *aStarOpenNode {
	tmp := newOpenNode()
	tmp.next = openNode
	tmp.node = childNode
	// 使用插入排序，通常直接插入就可以了
	openNode = tmp
	for tmp.next != nil && tmp.node.f > tmp.next.node.f {
		childNode = tmp.node
		tmp.node = tmp.next.node
		tmp.next.node = childNode
		tmp = tmp.next
	}
	return openNode
}

// 完全是为了gc友好，垃圾gc逻辑
func newAStar(config GridConfig) *AStar {
	nl := make([]int32, (config.GetWidth()+1)*(config.GetHeight()+1))
	var cache []node
	cache = make([]node, 1, 101)
	return &AStar{nodeIdx: nl, nodes: cache}
}

var _openNodePool = sync.Pool{
	New: func() interface{} {
		return &aStarOpenNode{}
	},
}

func newOpenNode() *aStarOpenNode {
	return _openNodePool.Get().(*aStarOpenNode)
}

func (a *AStar) getNode(idx int32) (*node, bool) {
	i := a.nodeIdx[idx]
	if i > 0 {
		return &a.nodes[i], true
	}
	capSize := cap(a.nodes)
	size := len(a.nodes)
	if size < capSize {
		a.nodes = a.nodes[0 : size+1]
	} else {
		a.nodes = append(a.nodes, node{})
	}
	a.nodeIdx[idx] = int32(size)
	return &a.nodes[size], false
}

func gridIdx(x, y, width int32) int32 {
	return y*width + x
}

func IsThrough(sx, sy, tx, ty int32, walkAble func(x, y int32) bool) bool {
	dx := tx - sx
	stepX := int32(-1)
	if dx < 0 {
		dx = -dx
	} else {
		stepX = 1
	}
	dy := ty - sy
	stepY := int32(-1)
	if dy < 0 {
		dy = -dy
	} else {
		stepY = 1
	}

	x := sx
	y := sy
	if dy < dx {
		n2dy := dy << 1 // * 2
		n2dydx := (dy - dx) << 1
		d := (dy << 1) - dx
		for {
			if !walkAble(x, y) {
				return false
			}

			if d < 0 {
				d += n2dy
			} else {
				y += stepY
				d += n2dydx
			}
			x += stepX
			if x == tx {
				break
			}
		}
	} else {
		n2dy := dx << 1 // * 2
		n2dydx := (dx - dy) << 1
		d := (dx << 1) - dy
		for {
			if !walkAble(x, y) {
				return false
			}

			if d < 0 {
				d += n2dy
			} else {
				x += stepX
				d += n2dydx
			}
			y += stepY
			if y == ty {
				break
			}
		}
	}
	return true
}

func IsThroughBorder(sx, sy, tx, ty int32, walkAbleBorder func(x, y int32, c int) int) bool {
	dx := tx - sx
	stepX := int32(-1)
	if dx < 0 {
		dx = -dx
	} else {
		stepX = 1
	}
	dy := ty - sy
	stepY := int32(-1)
	if dy < 0 {
		dy = -dy
	} else {
		stepY = 1
	}

	x := sx
	y := sy
	var c int
	if dy < dx {
		n2dy := dy << 1 // * 2
		n2dydx := (dy - dx) << 1
		d := (dy << 1) - dx
		for {
			c = walkAbleBorder(x, y, c)
			if c > 3 {
				return false
			}

			if d < 0 {
				d += n2dy
			} else {
				y += stepY
				d += n2dydx
			}
			x += stepX
			if x == tx {
				break
			}
		}
	} else {
		n2dy := dx << 1 // * 2
		n2dydx := (dx - dy) << 1
		d := (dx << 1) - dy
		for {
			c = walkAbleBorder(x, y, c)
			if c > 3 {
				return false
			}

			if d < 0 {
				d += n2dy
			} else {
				x += stepX
				d += n2dydx
			}
			y += stepY
			if y == ty {
				break
			}
		}
	}
	return true
}
