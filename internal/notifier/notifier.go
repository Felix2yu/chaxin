package notifier

import (
	"fmt"
	"strings"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/router"
	"github.com/containrrr/shoutrrr/pkg/types"
)

type Notifier struct {
	url    string
	sender *router.ServiceRouter
}

// New 创建通知器；url 为空时返回 nil, nil。
func New(url string) (*Notifier, error) {
	if url == "" {
		return nil, nil
	}
	sender, err := shoutrrr.CreateSender(url)
	if err != nil {
		return nil, fmt.Errorf("解析 shoutrrr url 失败: %w", err)
	}
	return &Notifier{url: url, sender: sender}, nil
}

// URL 返回当前通知目标地址。
func (n *Notifier) URL() string {
	if n == nil {
		return ""
	}
	return n.url
}

// Send 通过 shoutrrr 发送通知。
func (n *Notifier) Send(title, message string) error {
	if n == nil || n.sender == nil {
		return fmt.Errorf("shoutrrr url 未配置")
	}
	body := strings.TrimSpace(message)
	if body == "" {
		body = title
	}
	params := &types.Params{"title": title}
	errs := n.sender.Send(body, params)
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
