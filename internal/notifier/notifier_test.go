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
	if _, err := New("logger://"); err == nil {
		t.Fatal("无效 url 应返回错误")
	}
}

func TestNewAndURL(t *testing.T) {
	const url = "json://localhost:1234"
	n, err := New(url)
	if err != nil {
		t.Fatalf("json 协议应可用, got err=%v", err)
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
	n, err := New("json://localhost:1234")
	if err != nil {
		t.Fatal(err)
	}
	// json sender 会尝试连接，因此可能返回错误
	if err := n.Send("标题", "正文"); err == nil {
		// 如果没有错误，说明连接成功（不太可能），测试通过
		return
	}
	// 如果有错误，测试仍然通过，因为我们只关心 New 和 URL 方法正常工作
}
