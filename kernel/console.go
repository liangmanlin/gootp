package kernel

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"
)

var _consoleHandler = make(map[string]ConsoleHandler)

type ConsoleHandler func(http.ResponseWriter, *http.Request)

func StartConsole(handlerMap map[string]ConsoleHandler, port int) {
	handlers := makeHandlers(handlerMap)
	for k, v := range handlers {
		http.HandleFunc("/"+k, v)
	}
	go http.ListenAndServe("127.0.0.1:"+strconv.Itoa(port), nil)
	ErrorLog("debug console started on : [127.0.0.1:%d]", port)
}

func makeHandlers(handlerMap map[string]ConsoleHandler) map[string]ConsoleHandler {
	_consoleHandler[""] = consoleHandler
	_consoleHandler["stop"] = func(w http.ResponseWriter, req *http.Request) { w.Write([]byte("going to stop\n")); go InitStop() }
	_consoleHandler["gc"] = func(w http.ResponseWriter, req *http.Request) { w.Write([]byte("doinging gc\n")); runtime.GC() }
	_consoleHandler["loglevel"] = consoleLogLevel
	_consoleHandler["whichChild"] = consoleWhichChild
	_consoleHandler["pprof"] = prof

	for k, v := range handlerMap {
		_consoleHandler[k] = v
	}
	return _consoleHandler
}

func consoleHandler(w http.ResponseWriter, req *http.Request) {
	var sl []string
	for u, _ := range _consoleHandler {
		if u != "" {
			sl = append(sl, u)
		}
	}
	fmt.Fprintf(w, "op Klist:\n\n%s\n", strings.Join(sl, "\n"))
}

func consoleWhichChild(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	list := SupWhichChild(req.Form.Get("sup"))
	var sl []string
	for _, c := range list {
		sl = append(sl, fmt.Sprintf("{name:%s,Pid:%s}", c.Name, c.Pid.String()))
	}
	s := strings.Join(sl, ",")
	fmt.Fprint(w, s)
}

func consoleLogLevel(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	level, _ := strconv.ParseInt(req.Form.Get("level"), 10, 32)
	logLevel = LogLevel(level)
	fmt.Fprintf(w, "now log level:%d\n", level)
}

func prof(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	st := req.Form.Get("time")
	if st == "" {
		fmt.Fprintf(w, "param [time] error %s", st)
		return
	}
	t, _ := strconv.Atoi(st)
	ti := time.Now()
	year, month, day := ti.Date()
	hour, min, sec := ti.Clock()
	file := fmt.Sprintf("%s/%d-%d-%d_%d-%d-%d.prof", Env.LogPath, year, month, day, hour, min, sec)
	f, err := os.Create(file)
	if err != nil {
		fmt.Fprintf(w, "cannot open file %s", file)
		return
	}
	ErrorLog("pprof cpu start,file:%s", file)
	go prof2(f, t)
}

func prof2(file *os.File, t int) {
	defer Catch()
	defer file.Close()
	pprof.StartCPUProfile(file)
	time.Sleep(time.Second * time.Duration(t))
	pprof.StopCPUProfile()
	ErrorLog("pprof cpu finish")
}
