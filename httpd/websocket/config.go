package websocket

import "time"

type Config struct {
	ReadLimit              int
	MaxMessageSize         int
	CompressionLevel       int
	HandshakeTimeout       time.Duration
	EnableCompression      bool // 是否开启压缩，并且告知客户端
	EnableWriteCompression bool // 是否对写数据进行压缩，通常不需要压缩
	Origin                 bool // 是否检查域名
}

func DefaultConfig() *Config {
	return &Config{
		ReadLimit:      1024 * 1024,
		MaxMessageSize: 1025 * 1024,
	}
}

func (c *Config) isMessageTooLarge(size int) bool {
	return size > c.MaxMessageSize
}

func (c *Config) Confirm() {
	if c.ReadLimit <= 0 {
		c.ReadLimit = 1024 * 1024
	}
	if c.MaxMessageSize <= 0 {
		c.MaxMessageSize = 1024 * 1024
	}
}
