package httpd

import (
	"encoding/json"
	"github.com/liangmanlin/gootp/bpool"
	"github.com/liangmanlin/gootp/httpd/ejson"
	"github.com/liangmanlin/gootp/kernel"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	requestPool = sync.Pool{
		New: func() interface{} {
			return &http.Request{}
		},
	}

	responsePool = sync.Pool{
		New: func() interface{} {
			return &http.Response{}
		},
	}

	rPool = sync.Pool{
		New: func() interface{} {
			return &Request{}
		},
	}
)

func (r *Request) AddCookie(name, value string) {
	cookie := http.Cookie{
		Name:  name,
		Value: value,
	}
	r.response.Header.Add("Set-Cookie", cookie.String())
}

func (r *Request) AddCookieExpire(name, value string, Second int) {
	cookie := http.Cookie{
		Name:    name,
		Value:   value,
		Expires: time.Now().Add(time.Duration(Second) * time.Second),
	}
	r.response.Header.Add("Set-Cookie", cookie.String())
}

func (r *Request) SetCookie(name, value string) {
	cookie := http.Cookie{
		Name:  name,
		Value: value,
	}
	r.response.Header.Set("Set-Cookie", cookie.String())
}

func (r *Request) SetCookieExpire(name, value string, Second int) {
	cookie := http.Cookie{
		Name:    name,
		Value:   value,
		Expires: time.Now().Add(time.Duration(Second) * time.Second),
	}
	r.response.Header.Set("Set-Cookie", cookie.String())
}

func (r *Request) AddHead(name, value string) {
	r.response.Header.Add(name, value)
}

func (r *Request) SetHead(name, value string) {
	r.response.Header.Set(name, value)
}

func (r *Request) AddBody(body []byte) {
	if len(body) == 0 {
		return
	}
	if r.responseBody == nil {
		r.responseBody = bpool.NewBuf(body)
	} else {
		r.responseBody = r.responseBody.Append(body...)
	}
}

func (r *Request) AddJsonBody(rs interface{}) {
	buf, err := json.Marshal(rs)
	if err != nil {
		kernel.ErrorLog("ejson %#v error:%s", rs, err)
		return
	}
	r.AddBody(buf)
}

func (r *Request) RemoteIP() string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	addr := r.Conn.RemoteAddr().String()
	return strings.Split(addr, ":")[0]
}

// 返回查询字符串的值
func (r *Request) Lookup(key string) (value string) {
	if r.Form == nil {
		r.Form = r.URL.Query()
	}
	return r.Form.Get(key)
}

func (r *Request) addHeader(key, value string) {
	r.Header.Set(key, value)
}

// 返回表单多个values
func (r *Request) FormValues(key string) (values []string) {
	if r.Method == "POST" {
		if r.PostForm == nil {
			if r.ParseForm() != nil {
				return
			}
		}
		values = r.PostForm[key]
	}
	return
}

// 返回表单单个value
func (r *Request) FormValue(key string) (value string) {
	if vl := r.FormValues(key); len(vl) > 0 {
		value = vl[0]
	}
	return
}

// 把整个body解析为json
func (r *Request) Json() ejson.Json {
	if r.Body == nil {
		return nil
	}
	if t, ok := r.Body.(*BodyReader); ok {
		return ejson.Decode(t.RawBody())
	}
	size := int(r.ContentLength)
	if size <= 0 {
		size = 4 * 1024
	}
	bp, _ := bpool.ReadAll(r.Body, size)
	return ejson.Decode(bp.ToBytes())
}

func (r *Request) CacheTime(time int) {
	r.SetHead("Cache-Control", "max-age="+strconv.Itoa(time))
}

func (r *Request) ResponseWriter() http.ResponseWriter {
	return &responseWriter{r}
}

// 可以提前根据自己的需要，提前返回
func (r *Request) Reply(statusCode int, body []byte) error {
	r.has()
	r.response.StatusCode = statusCode
	r.AddBody(body)
	return r.replyNormal()
}

func (r *Request) Reply304(ETag string) {
	r.has()
	r.response.StatusCode = http.StatusNotModified
	r.SetHead("ETag", ETag)
	r.replyNormal()
}

func (r *Request) replyNormal() error {
	if r.isReply {
		return nil
	}
	r.has()
	err := r.reply()
	if r.Header.Get("Connection") == "close" {
		r.Conn.Close()
	} else if r.ProtoMajor == 1 && r.ProtoMinor == 0 {
		r.Conn.Close()
	}
	r.release()
	return err
}

func (r *Request) reply404() {
	r.has()
	r.SetHead("Content-Type", "text/plan")
	body := []byte("404 page not found")
	r.AddBody(body)
	r.response.StatusCode = http.StatusNotFound
	r.reply()
	r.Conn.Close()
	r.release()
}

func (r *Request) reply() error {
	response := r.response
	response.Header.Set("Server", "gootp-web")
	if r.responseBody != nil {
		if response.Header.Get("Content-Type") == "" {
			r.SetHead("Content-Type", "text/html")
		}
		response.ContentLength = int64(r.responseBody.Size())
		response.Body = NewBodyReader(r.responseBody)
	}
	w := bpool.NewWriter(r.Conn)
	err := response.Write(w)
	if err == nil {
		err = w.Flush()
	}
	w.Free()
	r.isReply = true
	return err
}

func (r *Request) release() {
	// 释放资源
	if r.Body != nil {
		r.Body.Close()
		r.Body = nil
	}
	requestPool.Put(r.Request)
	if r.response != nil {
		responsePool.Put(r.response)
	}
	r.Conn = nil
	r.response = nil
	r.Request = nil
	r.responseBody = nil
	rPool.Put(r)
}

func (r *Request) has() {
	if r.response == nil {
		r.response = responsePool.Get().(*http.Response)
		*r.response = http.Response{
			StatusCode: http.StatusOK,
			Proto:      r.Proto,
			ProtoMajor: r.ProtoMajor,
			ProtoMinor: r.ProtoMinor,
			Header:     http.Header{},
			Request:    r.Request,
		}
	}
}

func newRequest(req *http.Request, conn net.Conn) *Request {
	r := rPool.Get().(*Request)
	r.Request = req
	r.Conn = conn
	r.isReply = false
	r.f = nil
	return r
}

type responseWriter struct {
	*Request
}

func (r *responseWriter) Header() http.Header {
	r.has()
	return r.response.Header
}

func (r *responseWriter) Write(b []byte) (int, error) {
	n := len(b)
	err := r.Reply(r.response.StatusCode, b)
	return n, err
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.has()
	r.response.StatusCode = statusCode
}
