# Golang/OTP

## go actor framework & go simple otp

### `kernel`提供`application` `supervisor` `gen_server`类似的行为

### `node`提供多节点功能

- 需要自行拷贝`node/gmpd/bin`目录下的文件到环境变量`PATH`目录下，例如：`/usr/bin`

- 目前暂时没有支持windows的想法

### 一个简单的例子

```golang

import (
  "github.com/liangmanlin/gootp/kernel"
  "unsafe"
)

func main(){
  kernel.KernelStart(func(){
    actor := kernel.DefaultActor()
    actor.Init = func(ctx *kernel.Context,pid *kernel.Pid,args ...interface{})unsafe.Pointer{
      kernel.SendAfter(kernel.TimerTypeForever,pid,2000,true)
      return nil
    }
    actor.HandleCast = func(ctx *kernel.Context,msg interface{}){
      switch msg.(type) {
      case bool:
        kernel.ErrorLog("loop")
      }
    }
    kernel.Start(actor)
  },nil)
}

```

### 完整使用例子 [go-game-server](https://github.com/liangmanlin/go-game-server)

## 更多详情请查阅 [WIKI](https://github.com/liangmanlin/gootp/wiki)
