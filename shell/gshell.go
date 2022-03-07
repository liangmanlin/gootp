package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/liangmanlin/readline"
	"github.com/liangmanlin/gootp/gutil"
	"github.com/liangmanlin/gootp/kernel"
	"github.com/liangmanlin/gootp/kernel/kct"
	"github.com/liangmanlin/gootp/node"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

func main() {
	kernel.Env.WriteLogStd = false
	kernel.Env.LogPath = ""
	var nodeName, cookie, cmd string
	flag.StringVar(&nodeName, "node", "", "请输入节点名")
	flag.StringVar(&cookie, "cookie", "", "请输入cookie")
	flag.StringVar(&cmd, "cmd", "", "可选，脱离交互界面直接执行")
	flag.Parse()
	if nodeName == "" || cookie == "" {
		flag.Usage()
		os.Exit(1)
	}
	var live = true
	if cmd != "" {
		live = false
	}
	kernel.KernelStart(func() {
		node.StartHidden("debug-"+strconv.Itoa(os.Getpid())+"-"+nodeName, cookie, nil)
		if !node.ConnectNode(nodeName) {
			println("cannot connect " + nodeName + " with cookie " + cookie)
			os.Exit(1)
		}
		var l *readline.Instance
		// 启动一个专门用来显示的进程
		echoActor := kernel.NewActor(
			kernel.HandleCastFunc(func(ctx *kernel.Context, msg interface{}) {
				switch m := msg.(type) {
				case *kernel.ConsoleCommand:
					println(m.Command)
				case *node.NodeOP:
					if m.OP == node.OPDisConnect {
						println("node: " + nodeName + " disconnect")
						l.Close()
					}
				}
			}),
		)
		recPid, _ := kernel.Start(echoActor)
		if !live {
			_, rs := kernel.CallNameNode("console", nodeName, &kernel.ConsoleCommand{RecvPid: recPid, Command: cmd})
			switch r := rs.(type) {
			case *kernel.ConsoleCommand:
				fmt.Print(r.Command)
			}
			os.Exit(0)
		}
		node.Monitor(nodeName, recPid)
		// 获取所有命令
		_, rs := kernel.CallNameNode("console", nodeName, &kernel.ConsoleCommand{CType: 3})
		help, pl, needConfirm := buildHelp(rs.(*kernel.ConsoleCommand).Command)
		go func() {
			defer kernel.InitStop()
			var completer = readline.NewPrefixCompleter(pl...)
			var err error
			l, err = readline.NewEx(&readline.Config{
				Prompt:              "(" + nodeName + ")\033[31m>\033[0m ",
				AutoComplete:        completer,
				InterruptPrompt:     "^C",
				EOFPrompt:           "exit",
				HistorySearchFold:   true,
				FuncFilterInputRune: filterInput,
			})
			if err != nil {
				panic(err)
			}
			defer l.Close()
			log.SetOutput(l.Stderr())
			println("\nwelcome to use gshell\n")
			println("command: help for more information\n")
			println("To exit this debug: \u001B[31mCtrl-C\u001B[0m\n")
			for {
				line, err := l.Readline()
				if err == readline.ErrInterrupt {
					if len(line) == 0 {
						break
					} else {
						continue
					}
				} else if err == io.EOF {
					break
				}
			start:
				line = strings.TrimSpace(line)
				switch line {
				case "":
				case "help":
					println("commands:\n" + strings.Join(help, "\n"))
				default:
					C := kct.CutWith(line,' ')
					if confirmCommit, ok := needConfirm[C[0]]; ok {
						println("ensure to " + confirmCommit + ": " + line + "? [y/n]")
						read, err := l.ReadlineWithDefault("n")
						if err != nil || read != "y" {
							println("cancel " + confirmCommit)
							break
						}
					}
					if ok, result := kernel.CallNameNode("console", nodeName, &kernel.ConsoleCommand{RecvPid: recPid, Command: line}); ok {
						switch c := result.(type) {
						case *kernel.ConsoleCommand:
							if c.CType == 1 {
								line = c.Command
								goto start
							} else {
								println(c.Command)
							}
						default:
							fmt.Fprintf(l.Stderr(), "%#v", result)
						}
					} else {
						fmt.Fprintf(l.Stderr(), "%#v", result)
					}
				}
			}
		}()
	}, nil)
}

func filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

func buildHelp(jsonStr string) ([]string, []readline.PrefixCompleterInterface, map[string]string) {
	var commands map[string]map[string]string
	_ = json.Unmarshal([]byte(jsonStr), &commands)
	// 生成
	pl := []readline.PrefixCompleterInterface{
		readline.PcItem("help"),
	}
	var help []string
	var list []string
	var maxSize, maxArgs int32
	for k, v := range commands {
		list = append(list, k)
		maxSize = gutil.MaxInt32(maxSize, int32(len(k)))
		maxArgs = gutil.MaxInt32(maxArgs, int32(len(v["args"])))
	}
	sort.Strings(list)
	var needConfirm = make(map[string]string)
	for _, k := range list {
		pl = append(pl, readline.PcItem(k))
		cm := commands[k]
		help = append(help, fmt.Sprintf("    %-"+strconv.Itoa(int(maxSize))+"s  %-"+strconv.Itoa(int(maxArgs))+"s  \u001B[35m# %s\033[0m",
			k, cm["args"], cm["commit"]))
		if confirm, ok := cm["confirm"]; ok {
			needConfirm[k] = confirm
		}
	}
	return help, pl, needConfirm
}
