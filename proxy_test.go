package websocketproxy

import (
	"crypto/tls"
	"net/http"
	"testing"
)

func TestNewWebsocketProxy(t *testing.T) {
	tlsc := tls.Config{InsecureSkipVerify: true}
	wp, err := NewProxy("ws://www.baidu.com:80/ajaxchattest", auth, SetTLSConfig(&tlsc))
	if err != nil {
		t.Fatal(err)
	}
	http.HandleFunc("/wsproxy", wp.Proxy)
	http.ListenAndServe(":9696", nil)
}

func auth(r *http.Request) error {
	// 权限验证
	r.Header.Set("Cookie", "----")
	// 伪装来源
	r.Header.Set("Origin", "http://82.157.123.54:9010")
	return nil
}
