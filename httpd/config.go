package httpd

import "github.com/liangmanlin/gootp/httpd/websocket"

func WithManagerNum(num int) config {
	return func(engine *Engine) {
		engine.managerNum = num
	}
}

func WithMaxWorkerNum(num int) config {
	return func(engine *Engine) {
		engine.maxWorkerNum = num
	}
}

func WithReadLimit(num int) config {
	return func(engine *Engine) {
		engine.readLimit = num
	}
}

func WithTcpBuff(num int) config {
	return func(engine *Engine) {
		engine.tcpReadBuff = num
	}
}

// 修改负载均衡为随机，默认根据fd取模负载
func WithBalancingRand() config {
	return func(engine *Engine) {
		engine.balancing = 1
	}
}

// 如果你仅仅想监听内网ip，或者多个端口，使用这个
// WithAddr("127.0.0.1:8080")
// WithAddr(":8081")
func WithAddr(addr string) config {
	return func(engine *Engine) {
		engine.addr = append(engine.addr,addr)
	}
}

func WithWsConfig(cfg websocket.Config) config {
	return func(engine *Engine) {
		engine.wsConfig = &cfg
	}
}