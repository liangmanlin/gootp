package kernel

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"unsafe"
)

const loggerName = "logger"

var loggerServerPid *Pid = nil

type makeFile int

type loggerState struct {
	file *os.File
}

type LogLevel int

var logLevel LogLevel = 2

type logData struct {
	module string
	line   int
	format string
	args   []interface{}
}

var logWriter io.Writer

func Touch(writer io.Writer) {
	Env.LogPath = ""
	logWriter = writer
}

func DebugLog(format string, args ...interface{}) {
	if logLevel < 2 {
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			file = "???"
			line = 0
		} else {
			file = filepath.Base(file)
		}
		sendLog(file, line, format, args...)
	}
}

func ErrorLog(format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	} else {
		file = filepath.Base(file)
	}
	sendLog(file, line, format, args...)
}

func startLogger() {
	StartName(loggerName, loggerActor)
}

type logWrite struct {
	logger *Pid
}

func (l *logWrite) Write(p []byte) (n int, err error) {
	Cast(l.logger, p)
	n = len(p)
	return
}

var loggerActor = &Actor{
	Init: func(context *Context, pid *Pid, args ...interface{}) unsafe.Pointer {
		ErrorLog("%s %s started", loggerName, pid)
		loggerServerPid = pid
		addToKernelMap(pid)
		startLogTimer(pid)
		// 捕获go内置log信息
		log.SetOutput(&logWrite{logger: pid})
		state := initFile(&loggerState{file: nil})
		if Env.LogPath != "" && (*loggerState)(state).file == nil {
			os.Exit(789)
		}
		return state
	},
	HandleCast: func(context *Context, msg interface{}) {
		state := (*loggerState)(context.State)
		switch m := msg.(type) {
		case *logData:
			if state.file != nil {
				writeLog(state.file, m)
			} else if logWriter != nil {
				writeLog(logWriter, m)
			}
		case []byte:
			if state.file != nil {
				_, _ = state.file.Write(m)
			} else if logWriter != nil {
				_, _ = logWriter.Write(m)
			}
			if Env.WriteLogStd {
				_, _ = os.Stdout.Write(m)
			}
		case makeFile:
			context.State = initFile(state)
			startLogTimer(context.Self())
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

func initFile(logger *loggerState) unsafe.Pointer {
	if logger.file != nil {
		_ = logger.file.Close()
	}
	logger.file = makeLogFile()
	return unsafe.Pointer(logger)
}

func startLogTimer(self *Pid) {
	// 计算整点时间，用来更改文件名
	t := time.Now()
	_, min, sec := t.Clock()
	less := hourMillisecond - (int64(min)*minMillisecond + int64(sec)*Millisecond)
	TimerStart(TimerTypeOnce, self, less, makeFile(1))
}

func sendLog(module string, line int, format string, args ...interface{}) () {
	msg := &logData{module: module, line: line, format: format, args: args}
	if loggerServerPid != nil {
		Cast(loggerServerPid, msg)
	} else if Env.LogPath != "" {
		// if the logger not start yet,write to the file native
		file := makeLogFile()
		writeLog(file, msg)
		file.Close()
	} else if logWriter != nil {
		writeLog(logWriter, msg)
	}
	if Env.WriteLogStd {
		writeLog(os.Stdout, msg)
	}
}

func writeLog(w io.Writer, data *logData) {
	t := time.Now()
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	format := fmt.Sprintf("\n%d-%d-%d %d:%02d:%02d [%s:%d] %s\n",
		year, month, day, hour, min, sec, data.module, data.line, data.format)
	_, _ = fmt.Fprintf(w, format, data.args...)
}

func makeLogFile() *os.File {
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
