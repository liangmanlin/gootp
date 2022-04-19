package ejson

/*
	easy json
*/

import (
	"encoding/json"
	"unsafe"
)

type Json map[string]interface{}

func Decode(buf []byte) Json {
	var m Json
	if json.Unmarshal(buf, &m) != nil {
		return nil
	}
	return m
}

func DecodeString(jsonStr string) Json {
	buf := *(*[]byte)(unsafe.Pointer(&jsonStr))
	return Decode(buf)
}

func (j Json) Int(key string) int {
	if j == nil {
		return 0
	}
	if v, ok := j[key]; ok {
		return intValue(v)
	}
	return 0
}

func (j Json) Float(key string) float64 {
	if j == nil {
		return 0
	}
	if v, ok := j[key]; ok {
		return floatValue(v)
	}
	return 0
}

func (j Json) String(key string) string {
	if j == nil {
		return ""
	}
	if v, ok := j[key]; ok {
		return stringValue(v)
	}
	return ""
}

func (j Json) Json(key string) Json {
	if j == nil {
		return nil
	}
	if v, ok := j[key]; ok {
		if m, ok := v.(Json); ok {
			return m
		}
	}
	return nil
}

func (j Json) List(key string) []Json {
	if j == nil {
		return nil
	}
	if v, ok := j[key]; ok {
		if m, ok := v.([]Json); ok {
			return m
		}
	}
	return nil
}

func (j Json) Encode() []byte {
	bin, _ := json.Marshal(j)
	return bin
}

func intValue(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func floatValue(value interface{}) float64 {
	if v, ok := value.(float64); ok {
		return v
	}
	return 0
}

func stringValue(value interface{}) string {
	if v, ok := value.(string); ok {
		return v
	}
	return ""
}
