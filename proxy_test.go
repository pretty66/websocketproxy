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

func TestNewHandler(t *testing.T) {
	tlsc := tls.Config{InsecureSkipVerify: true}
	wp, err := NewProxy("ws://www.baidu.com:80/ajaxchattest", auth, SetTLSConfig(&tlsc))
	if err != nil {
		t.Fatal(err)
	}
	http.ListenAndServe(":9696", wp)
}

func auth(r *http.Request) error {
	// Permission to verify
	r.Header.Set("Cookie", "----")
	// Source of disguise
	r.Header.Set("Origin", "http://82.157.123.54:9010")
	return nil
}
