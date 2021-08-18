package websocketproxy

import (
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

const (
	WsScheme  = "ws"
	WssScheme = "wss"
)

var ErrFormatAddr = errors.New("remote websockets addr format error")

type WebsocketProxy struct {
	scheme          string // ws, wss
	remoteAddr      string // 目标地址: host:port
	defaultPath     string // path地址
	tlsc            *tls.Config
	beforeHandshake func(r *http.Request) error // 发送握手之前回调
}

// ex: ws://82.157.123.54:9010/ajaxchattest
func NewWebsocketProxy(addr string, beforeCallback func(r *http.Request) error) (*WebsocketProxy, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, ErrFormatAddr
	}
	_, _, err = net.SplitHostPort(u.Host)
	if err != nil {
		return nil, ErrFormatAddr
	}
	if u.Scheme != WsScheme && u.Scheme != WssScheme {
		return nil, ErrFormatAddr
	}
	wp := &WebsocketProxy{
		scheme:          u.Scheme,
		remoteAddr:      u.Host,
		defaultPath:     u.Path,
		beforeHandshake: beforeCallback,
	}
	if u.Scheme == WssScheme {
		wp.tlsc = &tls.Config{InsecureSkipVerify: true} // 不验证证书
	}
	return wp, nil
}

func (wp *WebsocketProxy) Proxy(writer http.ResponseWriter, request *http.Request) {
	if strings.ToLower(request.Header.Get("Connection")) != "upgrade" ||
		strings.ToLower(request.Header.Get("Upgrade")) != "websocket" {
		_, _ = writer.Write([]byte(`Must be a websocket request`))
		return
	}
	hijacker, ok := writer.(http.Hijacker)
	if !ok {
		return
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer conn.Close()
	req := request.Clone(request.Context())
	req.URL.Path, req.URL.RawPath, req.RequestURI = wp.defaultPath, wp.defaultPath, wp.defaultPath
	req.Host = wp.remoteAddr
	if wp.beforeHandshake != nil {
		// 增加头部，权限认证 + 伪装来源
		err = wp.beforeHandshake(req)
		if err != nil {
			_, _ = writer.Write([]byte(err.Error()))
			return
		}
	}
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
	b, _ := httputil.DumpRequest(req, false)
	remoteConn.Write(b)

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
