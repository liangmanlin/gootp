# 基于 [nbio](https://github.com/lesismal/nbio) 的webserver

## 系统构建于`kernel`之上

- 基于 [nbio](https://github.com/lesismal/nbio) 支持百万长链接
  
- 你可以通过配置，控制固定的处理线程

- 高性能的路由器

- 支持`websocket`

- 支持拦截器

    - 如果使用了`websocket`，不建议在拦截器中执行阻塞代码，这样会使响应效率降低
    - 可以单独启动一个`httpd`作为websocket使用

- 由于基于`kernel`，你可以使用所有相关技术

- 仅仅支持http,如果有https需求，建议通过nginx做反向代理

## websocket可以作为一个单独库使用

## 使用方法参考 [example](example)

## parser,websocket 均修改自 [nbio](https://github.com/lesismal/nbio) 的http部分

