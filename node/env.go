package node

import "github.com/liangmanlin/gootp/args"

var Env = &NodeEnv{
	PingTick: 60000,
}

const gmpdPort = 3333

func init()  {
	args.FillEvn(Env)
}