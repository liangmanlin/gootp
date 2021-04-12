package httpc

import (
	"container/list"
	"github.com/liangmanlin/gootp/kernel"
	"net/http"
)

type manager struct {
	idle        []*kernel.Pid
	queue       *list.List
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
}

type maxPoolSize int32
