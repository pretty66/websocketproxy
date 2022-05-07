# websocket proxy
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/pretty66/websocketproxy)](https://github.com/pretty66/websocketproxy/blob/master/go.mod)

## [中文](./README_ZH.md)

### Lightweight websocket proxy library
100 lines of code implements a lightweight websocket proxy library, does not depend on other third-party libraries, and supports ws and wss proxies; if you only need a simple websocket traffic proxy function without any modification to the forwarded content, using this library will be very useful.

## Content
- [Features](#Features)
- [Install](#Install)
- [Use](#Use)
- [Test](#Test)
- [Core code](#Core-code)
- [License](#License)

### Features
- Extreme performance, almost no performance loss, very low consumption of cpu and memory
- Support websocket handshake phase for management and control
- Supports setting header headers (cookie, origin, etc.) in the handshake phase
- Support ws, wss proxy

### Install
> go get github.com/pretty66/websocketproxy

### Use
```go
import (
    "github.com/pretty66/websocketproxy"
    "net/http"
)

wp, err := websocketproxy.NewProxy("ws://82.157.123.54:9010/ajaxchattest", func(r *http.Request) error {
    // Permission to verify
    r.Header.Set("Cookie", "----")
    // Source of disguise
    r.Header.Set("Origin", "http://82.157.123.54:9010")
    return nil
})
if err != nil {
    t.Fatal()
}
// proxy path
http.HandleFunc("/wsproxy", wp.Proxy)
http.ListenAndServe(":9696", nil)
```

### Test
Run the test file and start listening on the `127.0.0.1:9696` port, and use the online testing tool `http://coolaf.com/tool/chattest` to connect to the proxy to test the request response

#### example
![example](ws_test.png)



### Core-code
```go
func (wp *WebsocketProxy) Proxy(writer http.ResponseWriter, request *http.Request) {
    // Check whether it is a Websocket request
	if strings.ToLower(request.Header.Get("Connection")) != "upgrade" ||
		strings.ToLower(request.Header.Get("Upgrade")) != "websocket" {
		_, _ = writer.Write([]byte(`Must be a websocket request`))
		return
	}
    // Hijack connections
	hijacker, ok := writer.(http.Hijacker)
	if !ok {
		return
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer conn.Close()
    // Clone request, set destination address path
	req := request.Clone(context.TODO())
	req.URL.Path, req.URL.RawPath, req.RequestURI = wp.defaultPath, wp.defaultPath, wp.defaultPath
	req.Host = wp.remoteAddr
    // Handshake before callback
	if wp.beforeHandshake != nil {
		// Add headers, permission authentication + masquerade sources
		err = wp.beforeHandshake(req)
		if err != nil {
			_, _ = writer.Write([]byte(err.Error()))
			return
		}
	}
    // Determine the protocol and select the dialing process
	var remoteConn net.Conn
	switch wp.scheme {
	case WsScheme:
		remoteConn, err = net.Dial("tcp", wp.remoteAddr)
	case WssScheme:
		remoteConn, err = tls.Dial("tcp", wp.remoteAddr, wp.tlsc)
	}
	if err != nil {
		_, _ = writer.Write([]byte(err.Error()))
		return
	}
	defer remoteConn.Close()
	// Sends a handshake packet to the target WebSocket service
	err = req.Write(remoteConn)
	if err != nil {
		wp.logger.Println("remote write err:", err)
		return
	}
    // Traffic transparent transmission
	errChan := make(chan error, 2)
	copyConn := func(a, b net.Conn) {
		_, err := io.Copy(a, b)
		errChan <- err
	}
	go copyConn(conn, remoteConn) // response
	go copyConn(remoteConn, conn) // request
	select {
	case err = <-errChan:
		if err != nil {
			log.Println(err)
		}
	}
}
```

### Thanks for free JetBrains Open Source license
<a href="https://www.jetbrains.com/?from=github.com/pretty66/websocketproxy" target="_blank">
<img src="https://user-images.githubusercontent.com/21053373/167233149-c91fb02f-6052-4b65-a0bb-c38df84383cb.png" height="300"/></a>

### License
websocketproxy is under the Apache 2.0 license. See the [LICENSE](./LICENSE) directory for details.
