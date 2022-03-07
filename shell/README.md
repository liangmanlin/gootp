# 一个用于`kernel`的交互式命令行工具

你只需要拷贝`bin`目录下的文件到你的`path`目录，或者当前目录

```bash
gshell -node game@127.0.0.1 -cookie 6d27544c07937e4a7fab8123291cc4df
```

即可启动一个交互式命令行窗口

`kernel`原有的web控制台已经被这个替代。

如过你只想执行命令

```bash
## 停止服务
gshell -node game@127.0.0.1 -cookie 6d27544c07937e4a7fab8123291cc4df -cmd stop
```