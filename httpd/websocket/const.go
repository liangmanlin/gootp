package websocket

type WsError struct {
	ErrType ErrorType
}

type ErrorType int

const (
	ErrTypeClosed ErrorType = iota + 1
)