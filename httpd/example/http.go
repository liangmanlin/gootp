package main

import (
	"github.com/liangmanlin/gootp/args"
	"github.com/liangmanlin/gootp/httpd"
	"github.com/liangmanlin/gootp/kernel"
)

func main() {
	kernel.Env.LogPath = ""
	kernel.KernelStart(func() {
		port ,_:= args.GetInt("port")
		if port == 0 {
			port = 8080
		}
		const managerNum = 8
		eg := httpd.New("test", port,
			httpd.WithManagerNum(managerNum),
			httpd.WithMaxWorkerNum(managerNum*2048),
			httpd.WithBalancingRand(),
		)
		g := eg.GetGroup("/2")
		g2 := g.Group("/e")
		{
			g2.Get("/e", func(ctx *kernel.Context,request *httpd.Request) {
				request.AddBody([]byte("hello e"))
			})
		}
		g = g.Group("/2")
		{
			g.Get("/2", func(ctx *kernel.Context,request *httpd.Request) {
				request.AddBody([]byte("hello "+request.Lookup("name")))
			})
			g.Get("/1", func(ctx *kernel.Context,request *httpd.Request) {
				request.AddBody([]byte("hello "+request.Lookup("name")))
			})
		}
		eg.Run()
	}, nil)
}