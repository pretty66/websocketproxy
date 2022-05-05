/*
 *
 *  * Licensed to the Apache Software Foundation (ASF) under one or more
 *  * contributor license agreements.  See the NOTICE file distributed with
 *  * this work for additional information regarding copyright ownership.
 *  * The ASF licenses this file to You under the Apache License, Version 2.0
 *  * (the "License"); you may not use this file except in compliance with
 *  * the License.  You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 */

package websocketproxy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	WsScheme  = "ws"
	WssScheme = "wss"
	BufSize   = 1024 * 32
)

var ErrFormatAddr = errors.New("remote websockets addr format error")

type WebsocketProxy struct {
	scheme          string // ws, wss
	remoteAddr      string // 目标地址: host:port
	defaultPath     string // path地址
	tlsc            *tls.Config
	logger          *log.Logger
	beforeHandshake func(r *http.Request) error // 发送握手之前回调
}

type Options func(wp *WebsocketProxy)

// 一定需要携带端口号，ws://www:baidu.com:80/ssss, wss://www:baidu.com:443/aaaa
// ex: ws://82.157.123.54:9010/ajaxchattest
func NewProxy(addr string, beforeCallback func(r *http.Request) error, options ...Options) (*WebsocketProxy, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, ErrFormatAddr
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return nil, ErrFormatAddr
	}
	if u.Scheme != WsScheme && u.Scheme != WssScheme {
		return nil, ErrFormatAddr
	}
	wp := &WebsocketProxy{
		scheme:          u.Scheme,
		remoteAddr:      fmt.Sprintf("%s:%s", host, port),
		defaultPath:     u.Path,
		beforeHandshake: beforeCallback,
		logger:          log.New(os.Stderr, "", log.LstdFlags),
	}
	if u.Scheme == WssScheme {
		wp.tlsc = &tls.Config{InsecureSkipVerify: true} // 不验证证书
	}
	for op := range options {
		options[op](wp)
	}
	return wp, nil
}

func (wp *WebsocketProxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	wp.Proxy(writer, request)
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
	err = req.Write(remoteConn)
	if err != nil {
		wp.logger.Println("remote write err:", err)
		return
	}
	errChan := make(chan error, 2)
	copyConn := func(a, b net.Conn) {
		buf := ByteSliceGet(BufSize)
		defer ByteSlicePut(buf)
		_, err := io.CopyBuffer(a, b, buf)
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

func SetTLSConfig(tlsc *tls.Config) Options {
	return func(wp *WebsocketProxy) {
		wp.tlsc = tlsc
	}
}

func SetLogger(l *log.Logger) Options {
	return func(wp *WebsocketProxy) {
		if l != nil {
			wp.logger = l
		}
	}
}
