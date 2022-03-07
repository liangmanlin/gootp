package httpd

import (
	"github.com/liangmanlin/gootp/kernel"
)

type eapp struct {
	*Engine
}

var _ = kernel.Application(&eapp{})
// application
func (e *eapp)Name() string {
	return e.name
}
func (e *eapp) Start(bootType kernel.AppBootType) *kernel.Pid {
	sup := kernel.SupStart(e.name + "_sup")
	e.start(sup)
	return sup
}

func (e *eapp)Stop(stopType kernel.AppStopType) {
	kernel.ErrorLog("httpd [%s] stop",e.name)
	e.engine.Stop()
}

func (e *eapp)SetEnv(Key string, value interface{}) {

}

func (e *eapp)GetEnv(key string) interface{} {
	return nil
}
