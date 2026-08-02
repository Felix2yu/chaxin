package monitor

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/go-github/v89/github"
	"github.com/yufei/chaxin/internal/store"
)

func newTestMonitor(t *testing.T) (*Monitor, *store.Store) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(st, logger), st
}

func TestTrimChangelogShort(t *testing.T) {
	m, _ := newTestMonitor(t)
	_ = m
	if got := trimChangelog("hello world"); got != "hello world" {
		t.Fatalf("短文本不应被截断, got %q", got)
	}
	if got := trimChangelog("   \n"); got != "" {
		t.Fatalf("空白应返回空, got %q", got)
	}
}

func TestTrimChangelogLong(t *testing.T) {
	m, _ := newTestMonitor(t)
	_ = m
	long := strings.Repeat("行内容很长很长很长很长很长很长\n", 100)
	got := trimChangelog(long)
	if n := utf8.RuneCountInString(got); n > maxChangelogLen+30 {
		t.Fatalf("超长日志应被截断, runes=%d", n)
	}
	if !strings.Contains(got, "已截断") {
		t.Fatalf("截断提示缺失: %q", got[len(got)-40:])
	}
}

func TestCurrentInterval(t *testing.T) {
	m, st := newTestMonitor(t)
	ctx := context.Background()

	if got := m.currentInterval(ctx); got != defaultInterval {
		t.Fatalf("未配置时应为默认间隔, got %v", got)
	}
	if err := st.SetSetting(store.KeyPollInterval, "5m"); err != nil {
		t.Fatal(err)
	}
	if got := m.currentInterval(ctx); got != 5*time.Minute {
		t.Fatalf("应读取 5m, got %v", got)
	}
	// 非法值回退默认
	if err := st.SetSetting(store.KeyPollInterval, "abc"); err != nil {
		t.Fatal(err)
	}
	if got := m.currentInterval(ctx); got != defaultInterval {
		t.Fatalf("非法间隔应回退默认, got %v", got)
	}
}

func TestIsRateLimit(t *testing.T) {
	if !isRateLimit(&github.RateLimitError{}) {
		t.Fatal("RateLimitError 应被识别")
	}
	if isRateLimit(errors.New("boom")) {
		t.Fatal("普通错误不应被识别为限流")
	}
}

// 验证忽略正则逻辑与 checkRepo 中的判定一致（独立抽出便于测试）。
func TestIgnorePatternMatchesLatest(t *testing.T) {
	cases := []struct {
		pattern string
		tag     string
		want    bool
	}{
		{`^v0\.`, "v0.1.0", true},
		{`^v0\.`, "v1.2.0", false},
		{`beta|preview`, "v2.0.0-beta.1", true},
		{`^v1\.\d+\.\d+$`, "v1.2.3", true},
		{`^v1\.\d+\.\d+$`, "v1.2.3-rc.1", false},
		{``, "v1.0.0", false},
	}
	for _, c := range cases {
		got, err := matchesIgnorePattern(c.pattern, c.tag)
		if err != nil {
			t.Fatalf("pattern=%q: %v", c.pattern, err)
		}
		if got != c.want {
			t.Errorf("pattern=%q tag=%q: got %v want %v", c.pattern, c.tag, got, c.want)
		}
	}
}
