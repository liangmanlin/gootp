// Copyright 2020 lesismal. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package websocket

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/liangmanlin/gootp/bpool"
	"github.com/liangmanlin/gootp/kernel"
	"net"
	"runtime"
	"sync"
	"unicode/utf8"
)

const (
	maxControlFramePayloadSize = 125
)

// MessageType .
type MessageType int8

// The message types are defined in RFC 6455, section 11.8.t .
const (
	// FragmentMessage .
	FragmentMessage MessageType = 0 // Must be preceded by Text or Binary message
	// TextMessage .
	TextMessage MessageType = 1
	// BinaryMessage .
	BinaryMessage MessageType = 2
	// CloseMessage .
	CloseMessage MessageType = 8
	// PingMessage .
	PingMessage MessageType = 9
	// PongMessage .
	PongMessage MessageType = 10
)

var (
	MaxWebsocketFramePayloadSize = 16 * 1024 // 16k
)

// Conn .
type Conn struct {
	net.Conn

	mux sync.Mutex

	config *Config

	remoteCompressionEnabled bool
	enableWriteCompression   bool
	compressionLevel         int

	expectingFragments bool
	compress           bool
	opcode             MessageType

	buffer  *bpool.Buff
	message *bpool.Buff

	handler MessageHandler
}

type MessageHandler interface {
	Cast(msg interface{})
}

func validCloseCode(code int) bool {
	switch code {
	case 1000:
		return true //| Normal Closure  | hybi@ietf.org | RFC 6455  |
	case 1001:
		return true //      | Going Away      | hybi@ietf.org | RFC 6455  |
	case 1002:
		return true //   | Protocol error  | hybi@ietf.org | RFC 6455  |
	case 1003:
		return true //     | Unsupported Data| hybi@ietf.org | RFC 6455  |
	case 1004:
		return false //     | ---Reserved---- | hybi@ietf.org | RFC 6455  |
	case 1005:
		return false //      | No Status Rcvd  | hybi@ietf.org | RFC 6455  |
	case 1006:
		return false //      | Abnormal Closure| hybi@ietf.org | RFC 6455  |
	case 1007:
		return true //      | Invalid frame   | hybi@ietf.org | RFC 6455  |
		//      |            | payload data    |               |           |
	case 1008:
		return true //     | Policy Violation| hybi@ietf.org | RFC 6455  |
	case 1009:
		return true //       | Message Too Big | hybi@ietf.org | RFC 6455  |
	case 1010:
		return true //       | Mandatory Ext.  | hybi@ietf.org | RFC 6455  |
	case 1011:
		return true //       | Internal Server | hybi@ietf.org | RFC 6455  |
		//     |            | Error           |               |           |
	case 1015:
		return true //  | TLS handshake   | hybi@ietf.org | RFC 6455
	default:
	}
	// IANA registration policy and should be granted in the range 3000-3999.
	// The range of status codes from 4000-4999 is designated for Private
	if code >= 3000 && code < 5000 {
		return true
	}
	return false
}

// OnClose .
func (c *Conn) OnClose() {
	if c.handler != nil {
		c.handler.Cast(&WsError{ErrType: ErrTypeClosed})
	}
	// 当客户端主动发送一次关闭消息，又同时关闭tcp连接的时候，会有概率Onclose先执行，所以修改为不进行归还池
	// 顶多就是浪费一点点性能，考虑到websocket不可能频繁连接关闭，可以忽略不计
	//c.buffer.Free()
	//c.buffer = nil
	//if c.message != nil {
	//	c.message.Free()
	//	c.message = nil
	//}
}

func (c *Conn) SetHandler(h MessageHandler) {
	c.handler = h
}

type writeBuffer struct {
	*bpool.Buff
}

func (w *writeBuffer) Write(b []byte) (n int, err error) {
	n = len(b)
	w.Buff = w.Append(b...)
	return
}

// Close .
func (w *writeBuffer) Close() error {
	if w.Buff != nil {
		w.Free()
		w.Buff = nil
	}
	return nil
}

// WriteMessage .
func (c *Conn) WriteMessage(messageType MessageType, data []byte) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	switch messageType {
	case TextMessage:
	case BinaryMessage:
	case PingMessage, PongMessage, CloseMessage:
		if len(data) > maxControlFramePayloadSize {
			return ErrInvalidControlFrame
		}
	case FragmentMessage:
	default:
	}

	compress := c.enableWriteCompression && (messageType == TextMessage || messageType == BinaryMessage)
	if compress {
		w := &writeBuffer{
			Buff: bpool.New(len(data)),
		}
		defer w.Close()
		w.Reset()
		cw := compressWriter(w, c.compressionLevel)
		_, err := cw.Write(data)
		if err != nil {
			compress = false
		} else {
			cw.Close()
			data = w.ToBytes()
		}
	}
	if len(data) > 0 {
		sendOpcode := true
		for len(data) > 0 {
			n := len(data)
			if n > MaxWebsocketFramePayloadSize {
				n = MaxWebsocketFramePayloadSize
			}
			err := c.writeFrame(messageType, sendOpcode, n == len(data), data[:n], compress)
			if err != nil {
				return err
			}
			sendOpcode = false
			data = data[n:]
		}
		return nil
	}

	return c.writeFrame(messageType, true, true, []byte{}, compress)
}

func (c *Conn) writeFrame(messageType MessageType, sendOpcode, fin bool, data []byte, compress bool) error {
	var (
		bp       *bpool.Buff
		buf      []byte
		maskLen  int
		headLen  int
		totalLen int
		bodyLen  = len(data)
	)
	if bodyLen < 126 {
		headLen = 2 + maskLen
		totalLen = bodyLen + headLen
		bp = bpool.New(totalLen)
		buf = bp.ToBytes()[0:totalLen]
		buf[1] = byte(bodyLen)
	} else if bodyLen <= 65535 {
		headLen = 4 + maskLen
		totalLen = bodyLen + headLen
		bp = bpool.New(totalLen)
		buf = bp.ToBytes()[0:totalLen]
		buf[1] = 126
		binary.BigEndian.PutUint16(buf[2:4], uint16(bodyLen))
	} else {
		headLen = 10 + maskLen
		totalLen = bodyLen + headLen
		bp = bpool.New(totalLen)
		buf = bp.ToBytes()[0:totalLen]
		buf[1] = 127
		binary.BigEndian.PutUint64(buf[2:10], uint64(bodyLen))
	}
	copy(buf[headLen:], data)

	// opcode
	if sendOpcode {
		buf[0] = byte(messageType)
	} else {
		buf[0] = 0
	}

	if compress {
		buf[0] |= 0x40
	}

	// fin
	if fin {
		buf[0] |= byte(0x80)
	}

	_, err := c.Conn.Write(buf)
	bp.Free()
	return err
}

// Write overwrites nbio.Conn.Write.
func (c *Conn) Write(data []byte) (int, error) {
	return -1, ErrInvalidWriteCalling
}

// EnableWriteCompression .
func (c *Conn) EnableWriteCompression(enable bool) {
	if enable {
		if c.remoteCompressionEnabled {
			c.enableWriteCompression = enable
		}
	} else {
		c.enableWriteCompression = enable
	}
}

// Read .
func (c *Conn) Read(data []byte) error {
	bufLen := c.buffer.Size()
	if c.config.ReadLimit > 0 && bufLen+len(data) > c.config.ReadLimit || bufLen+c.messageSize() > c.config.ReadLimit {
		return ErrMessageTooLarge
	}
	c.buffer = c.buffer.Append(data...)

	var err error
	for i := 0; true; i++ {
		opcode, body, ok, fin, res1, res2, res3 := c.nextFrame()
		if !ok {
			break
		}
		if err = c.validFrame(opcode, fin, res1, res2, res3, c.expectingFragments); err != nil {
			break
		}
		if opcode == FragmentMessage || opcode == TextMessage || opcode == BinaryMessage {
			if c.opcode == 0 {
				c.opcode = opcode
				c.compress = res1
			}
			bl := len(body)
			if bl > 0 {
				if c.message == nil {
					c.message = bpool.NewBuf(body)
					if c.config.isMessageTooLarge(bl) {
						err = ErrMessageTooLarge
						break
					}
				} else {
					if c.config.isMessageTooLarge(c.message.Size() + bl) {
						err = ErrMessageTooLarge
						break
					}
					c.message = c.message.Append(body...)
				}
			}
			if fin {
				if c.compress {
					tmp := c.message
					c.message, err = Decompress(c.message.ToBytes())
					tmp.Free()
					if err != nil {
						break
					}
				}
				c.handleMessage(c.opcode, c.message)
				c.compress = false
				c.expectingFragments = false
				c.message = nil
				c.opcode = 0
			} else {
				c.expectingFragments = true
			}
		} else {
			frame := bpool.New(2)
			if len(body) > 0 {
				if c.config.isMessageTooLarge(len(body)) {
					err = ErrMessageTooLarge
					break
				}
				frame = frame.Append(body...)
			}
			c.handleProtocolMessage(opcode, frame)
		}

		if c.buffer.Size() == 0 {
			break
		}
	}

	return err
}

func (c *Conn) messageSize() int {
	if c.message == nil {
		return 0
	}
	return c.message.Size()
}

func (c *Conn) validFrame(opcode MessageType, fin, res1, res2, res3, expectingFragments bool) error {
	if res1 && !c.config.EnableCompression {
		return ErrReserveBitSet
	}
	if res2 || res3 {
		return ErrReserveBitSet
	}
	if opcode > BinaryMessage && opcode < CloseMessage {
		return fmt.Errorf("%w: opcode=%d", ErrReservedOpcodeSet, opcode)
	}
	if !fin && (opcode != FragmentMessage && opcode != TextMessage && opcode != BinaryMessage) {
		return fmt.Errorf("%w: opcode=%d", ErrControlMessageFragmented, opcode)
	}
	if expectingFragments && (opcode == TextMessage || opcode == BinaryMessage) {
		return ErrFragmentsShouldNotHaveBinaryOrTextOpcode
	}
	return nil
}

func (c *Conn) nextFrame() (opcode MessageType, body []byte, ok, fin, res1, res2, res3 bool) {
	l := int64(c.buffer.Size())
	buf := c.buffer.ToBytes()
	headLen := int64(2)
	if l >= 2 {
		opcode = MessageType(buf[0] & 0xF)
		res1 = int8(buf[0]&0x40) != 0
		res2 = int8(buf[0]&0x20) != 0
		res3 = int8(buf[0]&0x10) != 0
		fin = (buf[0] & 0x80) != 0
		payloadLen := buf[1] & 0x7F
		bodyLen := int64(-1)

		switch payloadLen {
		case 126:
			if l >= 4 {
				bodyLen = int64(binary.BigEndian.Uint16(buf[2:4]))
				headLen = 4
			}
		case 127:
			if c.buffer.Size() >= 10 {
				bodyLen = int64(binary.BigEndian.Uint64(buf[2:10]))
				headLen = 10
			}
		default:
			bodyLen = int64(payloadLen)
		}
		if bodyLen >= 0 {
			masked := (buf[1] & 0x80) != 0
			if masked {
				headLen += 4
			}
			total := headLen + bodyLen
			if l >= total {
				body = buf[headLen:total]
				if masked {
					maskKey := buf[headLen-4 : headLen]
					for i := 0; i < len(body); i++ {
						body[i] ^= maskKey[i%4]
					}
				}

				ok = true

				if l == total {
					c.buffer.Reset()
				} else {
					tmp := c.buffer
					c.buffer = bpool.NewBuf(buf[total:l])
					tmp.Free()
				}
			}
		}
	}

	return opcode, body, ok, fin, res1, res2, res3
}

func (c *Conn) handleMessage(opcode MessageType, body *bpool.Buff) {
	if opcode == TextMessage && !utf8.Valid(c.message.ToBytes()) {
		c.Close()
		return
	}
	// 通常都是约定的，所以没必要通知opcode
	c.handler.Cast(body)
}

func (c *Conn) handleProtocolMessage(opcode MessageType, body *bpool.Buff) {
	c.handleWsMessage(opcode, body)
}

func (c *Conn) handleWsMessage(opcode MessageType, body *bpool.Buff) {
	data := body.ToBytes()
	switch opcode {
	case TextMessage, BinaryMessage:
		c.handleMessage(opcode, body)
	case CloseMessage:
		if len(data) >= 2 {
			code := int(binary.BigEndian.Uint16(data[:2]))
			if !validCloseCode(code) || !utf8.Valid(data[2:]) {
				protoErrorCode := make([]byte, 2)
				binary.BigEndian.PutUint16(protoErrorCode, 1002)
				c.WriteMessage(CloseMessage, protoErrorCode)
			} else {
				c.closeMessage(code, data[2:])
			}
		} else {
			c.WriteMessage(CloseMessage, nil)
		}
		// close immediately, no need to wait for data flushed on a blocked conn
		c.Close()
	case PingMessage:
		c.pingMessage(data)
	case PongMessage:
		// 不用处理
	case FragmentMessage:
		kernel.DebugLog("invalid fragment message")
		c.Close()
	default:
		c.Close()
	}
}

func (c *Conn) closeMessage(code int, text []byte) {
	if len(text)+2 > maxControlFramePayloadSize {
		return //ErrInvalidControlFrame
	}
	buf := bpool.New(len(text) + 2)
	tmp := make([]byte, 2)
	binary.BigEndian.PutUint16(tmp, uint16(code))
	buf.Append(tmp...).Append(text...)
	c.WriteMessage(CloseMessage, buf.ToBytes())
	buf.Free()
}

func (c *Conn) pingMessage(data []byte) {
	if len(data) > 125 {
		c.Close()
		return
	}
	err := c.WriteMessage(PongMessage, data)
	if err != nil {
		kernel.DebugLog("failed to send pong %v", err)
		c.Close()
		return
	}
}

// SetCompressionLevel .
func (c *Conn) SetCompressionLevel(level int) error {
	if !isValidCompressionLevel(level) {
		return errors.New("websocket: invalid compression level")
	}
	c.compressionLevel = level
	return nil
}

func newConn(c net.Conn, cfg *Config, remoteCompressionEnabled bool) *Conn {
	conn := &Conn{
		Conn:                     c,
		config:                   cfg,
		remoteCompressionEnabled: remoteCompressionEnabled,
		buffer:                   bpool.New(4 * 1024),
		compressionLevel:         defaultCompressionLevel,
	}
	conn.EnableWriteCompression(cfg.EnableWriteCompression)
	conn.SetCompressionLevel(cfg.CompressionLevel)
	runtime.SetFinalizer(conn, freeConn)
	return conn
}

func freeConn(c *Conn) {
	c.buffer.Free()
	c.buffer = nil
	if c.message != nil {
		c.message.Free()
		c.message = nil
	}
}
