package gate

func WithAcceptNum(num int) optFun {
	return func(o *optStruct) {
		o.acceptNum = num
	}
}

func WithClientArgs(args ...interface{}) optFun {
	return func(o *optStruct) {
		o.clientArgs = args
	}
}

// 如果使用了epoll，那么返回的bpool.Buff的前面几个字节是head，需要逻辑代码注意
func WithUseEpoll() optFun {
	return func(o *optStruct) {
		o.isUseNbio = true
	}
}
