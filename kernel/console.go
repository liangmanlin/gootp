package kernel

import (
	"encoding/json"
	"fmt"
	"github.com/liangmanlin/gootp/kernel/kct"
	"os"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"time"
)

type chandler struct {
	name          string
	args          string
	commit        string
	argNum        int
	confirm       bool
	confirmCommit string
	handler       consoleHandler
}

type consoleCommit string

type consoleArg string

type consoleConfirm string

var _consoleHandler = make(map[string]*chandler)

type consoleHandler = func(echo func(string), commands []string) string

func StartConsole(handlers ...*chandler) {
	makeHandlers(handlers)
	StartName("console", consoleActor)
}

func ConsoleHandler(command string, handler consoleHandler, opt ...interface{}) *chandler {
	var argNum int
	var confirm bool
	var commit, args, confirmCommit string
	for _, op := range opt {
		switch o := op.(type) {
		case consoleArg:
			argNum++
			args += " " + string(o)
		case consoleCommit:
			commit = string(o)
		case consoleConfirm:
			confirm = true
			confirmCommit = string(o)
		}
	}
	return &chandler{
		name:          command,
		argNum:        argNum,
		args:          args,
		commit:        commit,
		confirm:       confirm,
		confirmCommit: confirmCommit,
		handler:       handler,
	}
}

func ConsoleArg(example string) consoleArg {
	return consoleArg(example)
}

func ConsoleCommit(commit string) consoleCommit {
	return consoleCommit(commit)
}

func ConsoleConfirm(confirm string) consoleConfirm {
	return consoleConfirm(confirm)
}

var consoleActor = NewActor(
	InitFunc(func(ctx *Context, pid *Pid, args ...interface{}) interface{} {
		ErrorLog("console started %s", pid)
		return nil
	}),
	HandleCallFunc(func(ctx *Context, request interface{}) interface{} {
		switch m := request.(type) {
		case *ConsoleCommand:
			if m.CType == 3 {
				var cl = make(map[string]map[string]string)
				for k, v := range _consoleHandler {
					cm := make(map[string]string)
					cm["args"] = v.args
					cm["commit"] = v.commit
					if v.confirm {
						cm["confirm"] = v.confirmCommit
					}
					cl[k] = cm
				}
				rs, _ := json.Marshal(cl)
				return &ConsoleCommand{Command: string(rs)}
			}
			return handleCommand(m)
		}
		return nil
	}),
)

func handleCommand(m *ConsoleCommand) *ConsoleCommand {
	defer func() {
		p := recover()
		if p != nil {
			ErrorLog("catch error:%s,Stack:%s", p, debug.Stack())
		}
	}()
	commands := kct.CutWith(m.Command,' ')
	size := len(commands)
	if size > 0 {
		if f, ok := _consoleHandler[commands[0]]; ok && size > f.argNum {
			ErrorLog("recv command: %s", m.Command)
			if rs := f.handler(m.echo, commands[1:]); rs != "" {
				return &ConsoleCommand{CType: 2, Command: rs}
			}
			return nil
		}
	}
	return &ConsoleCommand{CType: 1, Command: "help"}
}

func (c *ConsoleCommand)echo(s string)  {
	Cast(c.RecvPid, &ConsoleCommand{CType: 2, Command: s})
}

func makeHandlers(handlers []*chandler) {
	_consoleHandler["stop"] = ConsoleHandler("stop",
		func(echo func(string), commands []string) string { go InitStop(); return "going to stop" },
		ConsoleCommit("stop the game"),
		ConsoleConfirm(fmt.Sprintf("stop the game: %s", SelfNode().name)))
	_consoleHandler["gc"] = ConsoleHandler("gc",
		func(echo func(string), commands []string) string { runtime.GC(); return "gc done" },
		ConsoleCommit("global gc"))
	_consoleHandler["loglevel"] = ConsoleHandler("loglevel", consoleLogLevel,
		ConsoleArg("1|2"),
		ConsoleCommit("change the logger level"))
	_consoleHandler["whichChild"] = ConsoleHandler("whichChild", consoleWhichChild,
		ConsoleArg("SupName"),
		ConsoleCommit("get supervisor all child"))
	_consoleHandler["pprof"] = ConsoleHandler("pprof", prof,
		ConsoleArg("time(second)"),
		ConsoleCommit("Start a prof for cpu pprof int time"))
	_consoleHandler["register"] = ConsoleHandler("register", consoleRegisterList,
		ConsoleCommit("all registered pids"))
	for _, v := range handlers {
		_consoleHandler[v.name] = v
	}
}

func consoleRegisterList(echo func(string), commands []string) string {
	var l []string
	type nv struct {
		name string
		pid  *Pid
	}
	var vl []nv
	var maxLen int
	nameMap.Range(func(key, value interface{}) bool {
		v := nv{name: key.(string),pid: value.(*Pid)}
		vl = append(vl,v)
		if size :=len(v.name);size > maxLen{
			maxLen = size
		}
		return true
	})
	sort.Slice(vl, func(i, j int) bool {
		return vl[i].pid.id < vl[j].pid.id
	})
	format := "%-"+strconv.Itoa(maxLen)+"s %s"
	for _,v := range vl{
		l = append(l,fmt.Sprintf(format,v.name,v.pid))
	}
	return strings.Join(l, "\n")
}

func consoleWhichChild(echo func(string), commands []string) string {
	list := SupWhichChild(commands[0])
	var sl []string
	for _, c := range list {
		sl = append(sl, fmt.Sprintf("{name:%s,Pid:%s}", c.Name, c.Pid.String()))
	}
	return strings.Join(sl, "\n")
}

func consoleLogLevel(echo func(string), commands []string) string {
	level, _ := strconv.ParseInt(commands[0], 10, 32)
	logLevel = LogLevel(level)
	return fmt.Sprintf("now log level:%d\n", level)
}

func prof(echo func(string), commands []string) string {
	if len(commands) < 1 {
		return "param [time] error"

	}
	t, _ := strconv.Atoi(commands[0])
	ti := time.Now()
	year, month, day := ti.Date()
	hour, min, sec := ti.Clock()
	file := fmt.Sprintf("%s/%d-%d-%d_%d-%d-%d.prof", Env.LogPath, year, month, day, hour, min, sec)
	f, err := os.Create(file)
	if err != nil {
		return fmt.Sprintf("cannot create file %s,%s", file,err)
	}
	finishTime := TimeString(ti.Add(time.Duration(t)*time.Second))
	ErrorLog("pprof cpu Start,file:%s,finish time :%s", file,finishTime)
	go prof2(f, t)
	return fmt.Sprintf("pprof cpu Start,file:%s\nfinish time :%s please wait", file,finishTime)
}

func prof2(file *os.File, t int) {
	defer func() {
		p := recover()
		if p != nil {
			ErrorLog("catch error:%s,Stack:%s", p, debug.Stack())
		}
	}()
	defer file.Close()
	pprof.StartCPUProfile(file)
	time.Sleep(time.Second * time.Duration(t))
	pprof.StopCPUProfile()
	ErrorLog("pprof cpu finish")
}
