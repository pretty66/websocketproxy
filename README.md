# websocket proxy

### 100行代码实现轻量的websocket代理库，不依赖其他三方库，支持ws、wss代理

### 使用示例

#### Install
> go get pretty66/websocketproxy


#### 
```go
import (
    "github.com/pretty66/websocketproxy"
    "net/http"
)

wp, err := websocketproxy.NewWebsocketProxy("ws://82.157.123.54:9010/ajaxchattest", func(r *http.Request) error {
    // 权限验证
    r.Header.Set("Cookie", "----")
    // 伪装来源
    r.Header.Set("Origin", "http://82.157.123.54:9010")
    return nil
})
if err != nil {
    t.Fatal()
}
// 代理路径
http.HandleFunc("/wsproxy", wp.Proxy)
http.ListenAndServe(":9696", nil)
```

### 测试
运行test文件启动后监听`127.0.0.1:9696`端口，使用在线测试工具`http://coolaf.com/tool/chattest` 连接代理测试请求响应

### 示例
![示例](ws_test.png)