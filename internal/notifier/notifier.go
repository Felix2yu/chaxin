package notifier

import (
	"fmt"
	"log"
	"strings"

	apprise "github.com/unraid/apprise-go"
)

type Notifier struct {
	url  string
	send func(title, message string) error
}

// New 创建通知器；url 为空时返回 nil, nil。
//
// url 为 "logger://" 时返回日志模式通知器：仅把通知写入标准日志、不真正发送，
// Send 始终成功。该模式用于本地调试与单元测试（相当于 dry-run），不影响生产行为。
// 其余 url 经 apprise 校验，无法识别的 scheme 返回错误。
func New(url string) (*Notifier, error) {
	if url == "" {
		return nil, nil
	}
	if url == "logger://" {
		return &Notifier{
			url: url,
			send: func(title, message string) error {
				log.Printf("[notify] %s\n%s", title, message)
				return nil
			},
		}, nil
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

// Send 通过 apprise 发送通知。日志模式下直接调用内置 send 函数（仅记录）。
func (n *Notifier) Send(title, message string) error {
	if n == nil || n.url == "" {
		return fmt.Errorf("通知 url 未配置")
	}
	if n.send != nil {
		return n.send(title, message)
	}
	body := strings.TrimSpace(message)
	if body == "" {
		body = title
	}
	return apprise.Send([]string{n.url}, body, apprise.WithTitle(title), apprise.WithInputFormat("markdown"))
}