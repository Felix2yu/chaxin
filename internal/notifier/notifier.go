package notifier

import (
	"fmt"
	"strings"

	apprise "github.com/unraid/apprise-go"
)

type Notifier struct {
	url string
}

// New 创建通知器；url 为空时返回 nil, nil。
func New(url string) (*Notifier, error) {
	if url == "" {
		return nil, nil
	}
	// 验证 URL 是否有效
	client := apprise.New()
	if err := client.Add(url); err != nil {
		return nil, fmt.Errorf("解析通知 url 失败: %w", err)
	}
	return &Notifier{url: url}, nil
}

// URL 返回当前通知目标地址。
func (n *Notifier) URL() string {
	if n == nil {
		return ""
	}
	return n.url
}

// Send 通过 apprise 发送通知。
func (n *Notifier) Send(title, message string) error {
	if n == nil || n.url == "" {
		return fmt.Errorf("通知 url 未配置")
	}
	body := strings.TrimSpace(message)
	if body == "" {
		body = title
	}
	return apprise.Send([]string{n.url}, body, apprise.WithTitle(title))
}