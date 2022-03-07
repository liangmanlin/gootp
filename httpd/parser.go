// Copyright 2020 lesismal. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package httpd

import (
	"fmt"
	"github.com/lesismal/nbio"
	"github.com/liangmanlin/gootp/bpool"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
)

const (
	transferEncodingHeader = "Transfer-Encoding"
	trailerHeader          = "Trailer"
	contentLengthHeader    = "Content-Length"

	// MaxUint .
	MaxUint = ^uint(0)
	// MaxInt .
	MaxInt = int64(int(MaxUint >> 1))
)

// Parser .
type Parser struct {
	cache *bpool.Buff

	request *http.Request

	proto string

	statusCode int
	status     string

	headerKey   string
	headerValue string

	header  http.Header
	trailer http.Header

	chunkSize     int
	chunked       bool
	headerExists  bool

	state    int8

	readLimit int

	Conn *nbio.Conn

	onRequest func(conn *nbio.Conn,r *http.Request)
}

// NewParser .
func NewParser(onRequest func(conn *nbio.Conn,r *http.Request),conn *nbio.Conn,readLimit int) *Parser {
	state := stateMethodBefore
	if readLimit <= 0 {
		readLimit = DefaultHTTPReadLimit
	}
	p := &Parser{
		cache: bpool.New(4*1024), // 缺省4k
		state:     state,
		readLimit: readLimit,
		onRequest: onRequest,
		Conn: conn,
	}
	return p
}

func (p *Parser) nextState(state int8) {
	switch p.state {
	case stateClose:
	default:
		p.state = state
	}
}

// Close .
func (p *Parser) Close(err error) {
	if p.state == stateClose {
		return
	}

	p.state = stateClose

	if p.cache != nil {
		p.cache.Free()
		p.cache = nil
	}
}

func parseAndValidateChunkSize(originalStr string) (int, error) {
	chunkSize, err := strconv.ParseInt(originalStr, 16, 63)
	if err != nil {
		return -1, fmt.Errorf("chunk size parse error %v: %w", originalStr, err)
	}
	if chunkSize < 0 {
		return -1, fmt.Errorf("chunk size zero")
	}
	if chunkSize > MaxInt {
		return -1, fmt.Errorf("chunk size greater than max int %d", chunkSize)
	}
	return int(chunkSize), nil
}

// Read .
func (p *Parser) Read(data []byte) error {
	if p.state == stateClose {
		return ErrClosed
	}

	if len(data) == 0 {
		return nil
	}

	var c byte
	var start = 0
	var offset int
	if p.cache != nil{
		offset = p.cache.Size()
	}
	if offset > 0 {
		if offset+len(data) > p.readLimit {
			return ErrTooLong
		}
		p.cache = p.cache.Append(data...)
		data = p.cache.ToBytes()
	}

	for i := offset; i < len(data); i++ {
		c = data[i]
		switch p.state {
		case stateClose:
			return ErrClosed
		case stateMethodBefore:
			if isValidMethodChar(c) {
				start = i
				p.nextState(stateMethod)
				continue
			}
			return ErrInvalidMethod
		case stateMethod:
			if c == ' ' {
				var method = strings.ToUpper(string(data[start:i]))
				if !isValidMethod(method) {
					return ErrInvalidMethod
				}
				p.onMethod(method)
				start = i + 1
				p.nextState(statePathBefore)
				continue
			}
			if !isAlpha(c) {
				return ErrInvalidMethod
			}
		case statePathBefore:
			switch c {
			case '/', '*':
				start = i
				p.nextState(statePath)
				continue
			}
			switch c {
			case ' ':
			default:
				return ErrInvalidRequestURI
			}
		case statePath:
			if c == ' ' {
				var uri = string(data[start:i])
				if err := p.onURL(uri); err != nil {
					return err
				}
				start = i + 1
				p.nextState(stateProtoBefore)
			}
		case stateProtoBefore:
			if c != ' ' {
				start = i
				p.nextState(stateProto)
			}
		case stateProto:
			switch c {
			case ' ':
				if p.proto == "" {
					p.proto = string(data[start:i])
				}
			case '\r':
				if p.proto == "" {
					p.proto = string(data[start:i])
				}
				if err := p.onProto(p.proto); err != nil {
					p.proto = ""
					return err
				}
				p.proto = ""
				p.nextState(stateProtoLF)
			}
		case stateClientProtoBefore:
			if c == 'H' {
				start = i
				p.nextState(stateClientProto)
				continue
			}
			return ErrInvalidMethod
		case stateClientProto:
			switch c {
			case ' ':
				if p.proto == "" {
					p.proto = string(data[start:i])
				}
				if err := p.onProto(p.proto); err != nil {
					p.proto = ""
					return err
				}
				p.proto = ""
				p.nextState(stateStatusCodeBefore)
			}
		case stateStatusCodeBefore:
			switch c {
			case ' ':
			default:
				if isNum(c) {
					start = i
					p.nextState(stateStatusCode)
				}
				continue
			}
			return ErrInvalidHTTPStatusCode
		case stateStatusCode:
			if c == ' ' {
				cs := string(data[start:i])
				code, err := strconv.Atoi(cs)
				if err != nil {
					return err
				}
				p.statusCode = code
				p.nextState(stateStatusBefore)
				continue
			}
			if !isNum(c) {
				return ErrInvalidHTTPStatusCode
			}
		case stateStatusBefore:
			switch c {
			case ' ':
			default:
				if isAlpha(c) {
					start = i
					p.nextState(stateStatus)
				}
				continue
			}
			return ErrInvalidHTTPStatus
		case stateStatus:
			switch c {
			case ' ':
				if p.status == "" {
					p.status = string(data[start:i])
				}
			case '\r':
				if p.status == "" {
					p.status = string(data[start:i])
				}
				p.onStatus(p.statusCode, p.status)
				p.statusCode = 0
				p.status = ""
				p.nextState(stateStatusLF)
			}
		case stateStatusLF:
			if c == '\n' {
				p.nextState(stateHeaderKeyBefore)
				continue
			}
			return ErrLFExpected
		case stateProtoLF:
			if c == '\n' {
				start = i + 1
				p.nextState(stateHeaderKeyBefore)
				continue
			}
			return ErrLFExpected
		case stateHeaderValueLF:
			if c == '\n' {
				start = i + 1
				p.nextState(stateHeaderKeyBefore)
				continue
			}
			return ErrLFExpected
		case stateHeaderKeyBefore:
			switch c {
			case ' ':
				if !p.headerExists {
					return ErrInvalidCharInHeader
				}
			case '\r':
				err := p.parseTransferEncoding()
				if err != nil {
					return err
				}
				var contentLength int64
				err = p.parseContentLength(&contentLength)
				if err != nil {
					return err
				}
				p.onContentLength(contentLength)
				err = p.parseTrailer()
				if err != nil {
					return err
				}
				start = i + 1
				p.nextState(stateHeaderOverLF)
			case '\n':
				return ErrInvalidCharInHeader
			default:
				if isAlpha(c) {
					start = i
					p.nextState(stateHeaderKey)
					p.headerExists = true
					continue
				}
				return ErrInvalidCharInHeader
			}
		case stateHeaderKey:
			switch c {
			case ' ':
				if p.headerKey == "" {
					p.headerKey = http.CanonicalHeaderKey(string(data[start:i]))
				}
			case ':':
				if p.headerKey == "" {
					p.headerKey = http.CanonicalHeaderKey(string(data[start:i]))
				}
				start = i + 1
				p.nextState(stateHeaderValueBefore)
			case '\r', '\n':
				return ErrInvalidCharInHeader
			default:
				if !isToken(c) {
					return ErrInvalidCharInHeader
				}
			}
		case stateHeaderValueBefore:
			switch c {
			case ' ':
			case '\r':
				if p.headerValue == "" {
					p.headerValue = string(data[start:i])
				}
				switch p.headerKey {
				case transferEncodingHeader, trailerHeader, contentLengthHeader:
					if p.header == nil {
						p.header = http.Header{}
					}
					p.header.Add(p.headerKey, p.headerValue)
				default:
				}

				p.onHeader(p.headerKey, p.headerValue)
				p.headerKey = ""
				p.headerValue = ""

				start = i + 1
				p.nextState(stateHeaderValueLF)
			case '\n':
				return ErrInvalidCharInHeader
			default:
				// if !isToken(c) {
				// 	return ErrInvalidCharInHeader
				// }
				start = i
				p.nextState(stateHeaderValue)
			}
		case stateHeaderValue:
			switch c {
			case '\r':
				if p.headerValue == "" {
					p.headerValue = string(data[start:i])
				}
				switch p.headerKey {
				case transferEncodingHeader, trailerHeader, contentLengthHeader:
					if p.header == nil {
						p.header = http.Header{}
					}
					p.header.Add(p.headerKey, p.headerValue)
				default:
				}

				p.onHeader(p.headerKey, p.headerValue)
				p.headerKey = ""
				p.headerValue = ""

				start = i + 1
				p.nextState(stateHeaderValueLF)
			case '\n':
				return ErrInvalidCharInHeader
			default:
			}
		case stateHeaderOverLF:
			if c == '\n' {
				p.headerExists = false
				if p.chunked {
					start = i + 1
					p.nextState(stateBodyChunkSizeBefore)
				} else {
					start = i + 1
					if p.request.ContentLength > 0 {
						p.nextState(stateBodyContentLength)
					} else {
						p.oneMessage()
					}
				}
				continue
			}
			return ErrLFExpected
		case stateBodyContentLength:
			cl := int(p.request.ContentLength)
			left := len(data) - start
			if left >= cl {
				p.onBody(data[start : start+cl])
				p.oneMessage()
				start += cl
				i = start - 1
			} else {
				goto Exit
			}
		case stateBodyChunkSizeBefore:
			if isHex(c) {
				p.chunkSize = -1
				start = i
				p.nextState(stateBodyChunkSize)
				continue
			}
			return ErrInvalidChunkSize
		case stateBodyChunkSize:
			switch c {
			case ' ':
				if p.chunkSize < 0 {
					chunkSize, err := parseAndValidateChunkSize(string(data[start:i]))
					if err != nil {
						return err
					}
					p.chunkSize = chunkSize
				}
			case '\r':
				if p.chunkSize < 0 {
					chunkSize, err := parseAndValidateChunkSize(string(data[start:i]))
					if err != nil {
						return err
					}
					p.chunkSize = chunkSize
				}
				start = i + 1
				p.nextState(stateBodyChunkSizeLF)
			default:
				if !isHex(c) && p.chunkSize < 0 {
					chunkSize, err := parseAndValidateChunkSize(string(data[start:i]))
					if err != nil {
						return err
					}
					p.chunkSize = chunkSize
				}
			}
		case stateBodyChunkSizeLF:
			if c == '\n' {
				start = i + 1
				if p.chunkSize > 0 {
					p.nextState(stateBodyChunkData)
				} else {
					// chunk size is 0

					if len(p.trailer) > 0 {
						// read trailer headers
						p.nextState(stateBodyTrailerHeaderKeyBefore)
					} else {
						// read tail cr lf
						p.nextState(stateTailCR)
					}
				}
				continue
			}
			return ErrLFExpected
		case stateBodyChunkData:
			cl := p.chunkSize
			left := len(data) - start
			if left >= cl {
				p.onBody(data[start : start+cl])
				start += cl
				i = start - 1
				p.nextState(stateBodyChunkDataCR)
			} else {
				goto Exit
			}
		case stateBodyChunkDataCR:
			if c == '\r' {
				p.nextState(stateBodyChunkDataLF)
				continue
			}
			return ErrCRExpected
		case stateBodyChunkDataLF:
			if c == '\n' {
				p.nextState(stateBodyChunkSizeBefore)
				continue
			}
			return ErrLFExpected
		case stateBodyTrailerHeaderValueLF:
			if c == '\n' {
				start = i
				p.nextState(stateBodyTrailerHeaderKeyBefore)
				continue
			}
			return ErrLFExpected
		case stateBodyTrailerHeaderKeyBefore:
			if isAlpha(c) {
				start = i
				p.nextState(stateBodyTrailerHeaderKey)
				continue
			}

			// all trailer header readed
			if c == '\r' {
				if len(p.trailer) > 0 {
					return ErrTrailerExpected
				}
				start = i + 1
				p.nextState(stateTailLF)
				continue
			}
		case stateBodyTrailerHeaderKey:
			switch c {
			case ' ':
				if p.headerKey == "" {
					p.headerKey = http.CanonicalHeaderKey(string(data[start:i]))
				}
				continue
			case ':':
				if p.headerKey == "" {
					p.headerKey = http.CanonicalHeaderKey(string(data[start:i]))
				}
				start = i + 1
				p.nextState(stateBodyTrailerHeaderValueBefore)
				continue
			}
			if !isToken(c) {
				return ErrInvalidCharInHeader
			}
		case stateBodyTrailerHeaderValueBefore:
			switch c {
			case ' ':
			case '\r':
				if p.headerValue == "" {
					p.headerValue = string(data[start:i])
				}
				p.onTrailerHeader(p.headerKey, p.headerValue)
				p.headerKey = ""
				p.headerValue = ""

				start = i + 1
				p.nextState(stateBodyTrailerHeaderValueLF)
			default:
				// if !isToken(c) {
				// 	return ErrInvalidCharInHeader
				// }
				start = i
				p.nextState(stateBodyTrailerHeaderValue)
			}
		case stateBodyTrailerHeaderValue:
			switch c {
			case ' ':
				if p.headerValue == "" {
					p.headerValue = string(data[start:i])
				}
			case '\r':
				if p.headerValue == "" {
					p.headerValue = string(data[start:i])
				}
				if len(p.trailer) == 0 {
					return fmt.Errorf("invalid trailer '%v'", p.headerKey)
				}
				delete(p.trailer, p.headerKey)

				p.onTrailerHeader(p.headerKey, p.headerValue)
				start = i + 1
				p.headerKey = ""
				p.headerValue = ""
				p.nextState(stateBodyTrailerHeaderValueLF)
			default:
				// if !isToken(c) {
				// 	return ErrInvalidCharInHeader
				// }
			}
		case stateTailCR:
			if c == '\r' {
				p.nextState(stateTailLF)
				continue
			}
			return ErrCRExpected
		case stateTailLF:
			if c == '\n' {
				start = i + 1
				p.oneMessage()
				continue
			}
			return ErrLFExpected
		default:
		}
	}

Exit:
	left := len(data) - start
	if left > 0 {
		if p.cache == nil {
			p.cache = bpool.NewBuf(data[start:])
		} else if start > 0 {
			oldCache := p.cache
			p.cache = bpool.NewBuf(data[start:]) // TODO 这里是一个需要考虑的点，可能不应该从新分配
			oldCache.Free()
		}
	} else if p.cache != nil {
		p.cache.Free()
		p.cache = nil
	}

	return nil
}

func (p *Parser) parseTransferEncoding() error {
	raw, present := p.header[transferEncodingHeader]
	if !present {
		return nil
	}
	delete(p.header, transferEncodingHeader)

	if len(raw) != 1 {
		return fmt.Errorf("too many transfer encodings: %q", raw)
	}
	if strings.ToLower(textproto.TrimString(raw[0])) != "chunked" {
		return fmt.Errorf("unsupported transfer encoding: %q", raw[0])
	}
	delete(p.header, contentLengthHeader)
	p.chunked = true

	return nil
}

func (p *Parser) parseContentLength(contentLength *int64) (err error) {
	if cl := p.header.Get(contentLengthHeader); cl != "" {
		if p.chunked {
			return ErrUnexpectedContentLength
		}
		end := len(cl) - 1
		for i := end; i >= 0; i-- {
			if cl[i] != ' ' {
				if i != end {
					cl = cl[:i+1]
				}
				break
			}
		}
		l, err := strconv.ParseInt(cl, 10, 63)
		if err != nil {
			return fmt.Errorf("%s %q", "bad Content-Length", cl)
		}
		if l < 0 {
			return fmt.Errorf("length less than zero (%d): %w", l, ErrInvalidContentLength)
		}
		if l > MaxInt {
			return fmt.Errorf("length greater than maxint (%d): %w", l, ErrInvalidContentLength)
		}
		*contentLength = l
	} else {
		*contentLength = -1
	}
	return nil
}

func (p *Parser) parseTrailer() error {
	if !p.chunked {
		return nil
	}
	header := p.header

	trailers, ok := header[trailerHeader]
	if !ok {
		return nil
	}

	header.Del(trailerHeader)

	trailer := http.Header{}
	for _, key := range trailers {
		key = textproto.TrimString(key)
		if key == "" {
			continue
		}
		if !strings.Contains(key, ",") {
			key = http.CanonicalHeaderKey(key)
			switch key {
			case transferEncodingHeader, trailerHeader, contentLengthHeader:
				return fmt.Errorf("%s %q", "bad trailer key", key)
			default:
				trailer[key] = nil
			}
			continue
		}
		for _, k := range strings.Split(key, ",") {
			if k = textproto.TrimString(k); k != "" {
				k = http.CanonicalHeaderKey(k)
				switch k {
				case transferEncodingHeader, trailerHeader, contentLengthHeader:
					return fmt.Errorf("%s %q", "bad trailer key", k)
				default:
					trailer[k] = nil
				}
			}
		}
	}
	if len(trailer) > 0 {
		p.trailer = trailer
	}
	return nil
}

// 一次完整的http请求协议
func (p *Parser) oneMessage() {
	p.onRequest(p.Conn,p.request)
	p.request = nil
	p.header = nil

	p.nextState(stateMethodBefore)
}

func (p *Parser) onMethod(method string) {
	if p.request == nil {
		p.request = requestPool.Get().(*http.Request)
		*p.request = http.Request{}
		p.request.Method = method
		p.request.Header = http.Header{}
	} else {
		p.request.Method = method
	}
}

func (p *Parser) onURL(uri string) error {
	u, err := url.ParseRequestURI(uri)
	if err != nil {
		return err
	}
	p.request.URL = u
	p.request.RequestURI = uri
	return nil
}

func (p *Parser) onProto(proto string) error {
	protoMajor, protoMinor, ok := http.ParseHTTPVersion(proto)
	if !ok {
		return fmt.Errorf("%s %q", "malformed HTTP version", proto)
	}
	p.request.Proto = proto
	p.request.ProtoMajor = protoMajor
	p.request.ProtoMinor = protoMinor
	return nil
}

func (p *Parser) onStatus(code int, status string) {
}

func (p *Parser) onHeader(key, value string) {
	values := p.request.Header[key]
	values = append(values, value)
	p.request.Header[key] = values
	// p.isUpgrade = (key == "Connection" && value == "upgrade")
}

// OnContentLength .
func (p *Parser) onContentLength(contentLength int64) {
	p.request.ContentLength = contentLength
}

func (p *Parser) onBody(data []byte) {
	if len(data) == 0 {
		return
	}
	if p.request.Body == nil {
		p.request.Body = NewBodyReader(bpool.NewBuf(data))
	} else {
		p.request.Body.(*BodyReader).Append(data)
	}
}

func (p *Parser) onTrailerHeader(key, value string) {
	if p.request.Trailer == nil {
		p.request.Trailer = http.Header{}
	}
	p.request.Trailer.Add(key, value)
}
