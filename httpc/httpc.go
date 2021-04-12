package httpc

import (
	"fmt"
	"github.com/liangmanlin/gootp/kernel"
	"net/url"
	"strings"
)

func Get(url string, param map[string]interface{}, timeOut int32, urlEncode bool) (body []byte, ok bool) {
	return get(url, param, timeOut, urlEncode, false)
}

func GetSSL(url string, param map[string]interface{}, timeOut int32, urlEncode bool) (body []byte, ok bool) {
	return get(url, param, timeOut, urlEncode, true)
}

func get(url string, param map[string]interface{}, timeOut int32, urlEncode bool, ssl bool) (body []byte, ok bool) {
	if len(param) > 0 {
		paramStr := linkParam(param, urlEncode)
		if strings.Index(url, "?") >= 0 {
			url += "&" + paramStr
		} else {
			url += "?" + paramStr
		}
	}
	return request(server, "GET", url, "", "", timeOut, ssl)
}

func Post(url, data, contentType string, timeOut int32) (body []byte, ok bool) {
	return post(url, data, contentType, timeOut, false)
}

func PostSSL(url, data, contentType string, timeOut int32) (body []byte, ok bool) {
	return post(url, data, contentType, timeOut, true)
}

func post(url, data, contentType string, timeOut int32, ssl bool) (body []byte, ok bool) {
	return request(server, "POST", url, data, contentType, timeOut, ssl)
}

func Request(pid *kernel.Pid, method, url, body, contentType string, timeOut int32, ssl bool) ([]byte, bool) {
	return request(pid, method, url, body, contentType, timeOut, ssl)
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
