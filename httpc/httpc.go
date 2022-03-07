package httpc

import (
	"fmt"
	"net/url"
	"strings"
)

func Get(url string, param map[string]interface{}, urlEncode bool,args...interface{}) (body []byte, ok bool) {
	return get(url, param, urlEncode, args...)
}

func GetSSL(url string, param map[string]interface{}, urlEncode bool,args...interface{}) (body []byte, ok bool) {
	args = append(args,WithSSL())
	return get(url, param, urlEncode, args...)
}

func get(url string, param map[string]interface{},urlEncode bool, args ...interface{}) (body []byte, ok bool) {
	if len(param) > 0 {
		paramStr := linkParam(param, urlEncode)
		if strings.Index(url, "?") >= 0 {
			url += "&" + paramStr
		} else {
			url += "?" + paramStr
		}
	}
	return Request(server, "GET", url, args...)
}

func Post(url, data, contentType string,args ...interface{}) (body []byte, ok bool) {
	return post(url, data, contentType,args...)
}

func PostSSL(url, data, contentType string,args ...interface{}) (body []byte, ok bool) {
	args = append(args,WithSSL())
	return post(url, data, contentType,args...)
}

func post(url, data, contentType string,args ...interface{}) (body []byte, ok bool) {
	args = append(args,WithBody(data),WithContentType(contentType))
	return Request(server, "POST", url, args...)
}

func linkParam(param map[string]interface{}, urlEncode bool) string {
	var list []string
	for k, v := range param {
		switch d := v.(type) {
		case string:
			list = append(list, k+"="+encode(d, urlEncode))
		case float32, float64:
			list = append(list, fmt.Sprintf("%s=%f", k, d))
		default:
			list = append(list, fmt.Sprintf("%s=%d", k, d))
		}
	}
	return strings.Join(list, "&")
}

func encode(str string, need bool) string {
	if need {
		return url.QueryEscape(str)
	}
	return str
}

func WithSSL() useSSL {
	return useSSL{}
}

func WithBody(bodyStr string) bodyType {
	return bodyType(bodyStr)
}

func WithContentType(contentT string) contentType {
	return contentType(contentT)
}

// timeout sec,default is 3s
func WithTimeOut(t int32) timeOut {
	return timeOut(t)
}

func WithHeader(key,value string) header {
	return header{key: key,value: value}
}