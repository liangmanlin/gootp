package httpc

import (
	"github.com/liangmanlin/gootp/kernel"
	"github.com/liangmanlin/gootp/ringbuffer"
	"net/http"
)

type manager struct {
	idle        []*kernel.Pid
	queue       *ringbuffer.SingleRingBuffer
	poolSize    int32
	maxPoolSize int32
}

type worker struct {
	father    *kernel.Pid
	client    *http.Client
	sslClient *http.Client
}

type response struct {
	ok  bool
	rsp []byte
}

type requestData struct {
	method      string
	url         string
	body        string
	contentType string
	ssl         bool
	close       int32
	ch          chan response
	timeOut     int32
	headers		[]header
}

type maxPoolSize int32

type useSSL struct {
}

type contentType string

type bodyType string

type timeOut int32

type header struct {
	key, value string
}
