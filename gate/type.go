package gate

type TcpError struct {
	Err error
}

type AcceptNum int

type ClientArgs []interface{}

type Pack struct {
	ProtoID int
	Proto   interface{}
}
