package db

import "github.com/liangmanlin/gootp/args"

var Env = &env{
	ConnNum: 8,
	IsOpenCache: true,
}

func init()  {
	args.FillEvn(Env)
}

