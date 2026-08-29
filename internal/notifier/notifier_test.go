package notifier

import "testing"

func TestNewEmpty(t *testing.T) {
	n, err := New("")
	if err != nil {
		t.Fatalf("空 url 应返回 nil,nil, got err=%v", err)
	}
	if n != nil {
		t.Fatalf("空 url 应返回 nil notifier, got %+v", n)
	}
}

func TestNewInvalid(t *testing.T) {
	if _, err := New("this is not a valid shoutrrr url"); err == nil {
		t.Fatal("无效 url 应返回错误")
	}
}

func TestNewAndURL(t *testing.T) {
	const url = "logger://"
	n, err := New(url)
	if err != nil {
		t.Fatalf("logger 协议应可用, got err=%v", err)
	}
	if n.URL() != url {
		t.Fatalf("URL() 应返回 %q, got %q", url, n.URL())
	}
}

func TestURLNil(t *testing.T) {
	var n *Notifier
	if n.URL() != "" {
		t.Fatalf("nil notifier 的 URL() 应返回空串, got %q", n.URL())
	}
}

func TestSendNil(t *testing.T) {
	var n *Notifier
	if err := n.Send("title", "msg"); err == nil {
		t.Fatal("nil notifier 发送应返回错误")
	}
}

func TestSendOK(t *testing.T) {
	n, err := New("logger://")
	if err != nil {
		t.Fatal(err)
	}
	// logger sender 不实际发送，不返回错误
	if err := n.Send("标题", "正文"); err != nil {
		t.Fatalf("logger 发送应成功, got err=%v", err)
	}
}
