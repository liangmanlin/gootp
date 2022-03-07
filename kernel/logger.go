package kernel

import (
	"fmt"
	"github.com/liangmanlin/gootp/bpool"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const loggerName = "logger"

var loggerServerPid *Pid = nil

type makeFile struct{}

type loggerState struct {
	out      io.Writer
	redirect bool
}

type LogLevel int

var logLevel LogLevel = 2

type logData struct {
	pkg    string
	module string
	line   int
	log    string
}

func SetLogLevel(lv LogLevel) {
	logLevel = lv
}

func SetLoggerOut(writer io.Writer) {
	Cast(loggerServerPid, writer)
}

func DebugLog(format string, args ...interface{}) {
	if logLevel < 2 {
		var pkg string
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			file = "???"
			pkg = "???"
			line = 0
		} else {
			pkg, file = filepath.Split(file)
			pkg = filepath.Base(pkg)
		}
		sendLog(pkg, file, line, format, args...)
	}
}

func ErrorLog(format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	var pkg string
	if !ok {
		file = "???"
		pkg = "???"
		line = 0
	} else {
		pkg, file = filepath.Split(file)
		pkg = filepath.Base(pkg)
	}
	sendLog(pkg, file, line, format, args...)
}

func UnHandleMsg(msg interface{}) {
	var pkg string
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		pkg = "???"
		line = 0
	} else {
		pkg, file = filepath.Split(file)
		pkg = filepath.Base(pkg)
	}
	sendLog(pkg, file, line, "un handle msg : %#v", msg)
}

func startLogger() {
	StartNameOpt(loggerName, loggerActor, ActorOpt(ActorChanCacheSize(10000)))
}

type logWrite struct {
	logger *Pid
}

func (l *logWrite) Write(p []byte) (n int, err error) {
	n = len(p)
	// 考虑到上层逻辑可能会重用，这里copy
	Cast(l.logger, bpool.NewBuf(p))
	return
}

func LoggerWriter() io.Writer {
	return &logWrite{logger: loggerServerPid}
}

var loggerActor = &Actor{
	Init: func(context *Context, pid *Pid, args ...interface{}) interface{} {
		ErrorLog("%s %s started", loggerName, pid)
		addToKernelMap(pid)
		startLogTimer(pid)
		// 捕获go内置log信息
		log.SetOutput(&logWrite{logger: pid})
		state := initFile(&loggerState{out: nil})
		if Env.LogPath != "" && (*loggerState)(state).out == nil {
			os.Exit(789)
		}
		loggerServerPid = pid
		return state
	},
	HandleCast: func(context *Context, msg interface{}) {
		state := context.State.(*loggerState)
		switch m := msg.(type) {
		case *logData:
			if state.out != nil {
				writeLog(state.out, m)
			}
		case *bpool.Buff:
			if state.out != nil {
				_, _ = state.out.Write(m.ToBytes())
			}
			if Env.WriteLogStd {
				_, _ = os.Stdout.Write(m.ToBytes())
			}
			m.Free()
		case io.Writer:
			state.out = m
			state.redirect = true
		case makeFile:
			if !state.redirect {
				context.State = initFile(state)
				startLogTimer(context.Self())
			}
		}
	},
	HandleCall: func(context *Context, request interface{}) interface{} {
		return nil
	},
	Terminate: func(context *Context, reason *Terminate) {

	},
	ErrorHandler: func(context *Context, err interface{}) bool {
		return true
	},
}

func initFile(logger *loggerState) *loggerState {
	if logger.out != nil {
		logger.out.(*os.File).Close()
	}
	logger.out = defaultWriter()
	return logger
}

func startLogTimer(self *Pid) {
	// 计算整点时间，用来更改文件名
	t := time.Now()
	_, min, sec := t.Clock()
	less := hourMillisecond - (int64(min)*minMillisecond + int64(sec)*Millisecond)
	SendAfter(TimerTypeOnce, self, less, makeFile{})
}

func sendLog(pkg, module string, line int, format string, args ...interface{}) () {
	msg := &logData{pkg: pkg, module: module, line: line, log: fmt.Sprintf(format, args...)}
	if loggerServerPid != nil {
		Cast(loggerServerPid, msg)
	} else if Env.LogPath != "" {
		// if the logger not Start yet,write to the file native
		logger := defaultWriter()
		defer logger.Close()
		writeLog(logger, msg)
	}
	if Env.WriteLogStd {
		writeLog(os.Stdout, msg)
	}
}

func writeLog(w io.Writer, data *logData) {
	t := time.Now()
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	_, _ = fmt.Fprintf(w, "\n%d-%d-%d %d:%02d:%02d [%s.%s:%d] %s\n",
		year, month, day, hour, min, sec, data.pkg, data.module, data.line, data.log)
}

func defaultWriter() *os.File {
	if Env.LogPath == "" {
		return nil
	}
	t := time.Now()
	year, month, day := t.Date()
	hour, _, _ := t.Clock()
	path := Env.LogPath + fmt.Sprintf("/%d_%d_%d", year, month, day)
	file := path + fmt.Sprintf("/sy_%d_%d_%d___%02d.log", year, month, day, hour)
	if _, err := os.Stat(path); err != nil {
		_ = os.MkdirAll(path, os.ModeDir)
	}
	ioFile, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		ErrorLog("cannot open file:%s", err.Error())
		return nil
	}
	return ioFile
}
