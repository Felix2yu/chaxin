package monitor

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

// 引擎关闭时不应翻译（返回空，调用方回退原文）。
func TestTranslateBodyDisabled(t *testing.T) {
	m, _ := newTestMonitor(t)
	ctx := context.Background()
	st := store.Settings{TranslateEngine: "off", TranslateTargetLang: "zh-Hans"}
	if got := m.translateBody(ctx, "Some English text", st); got != "" {
		t.Fatalf("引擎关闭应返回空, got %q", got)
	}
	// 未设置引擎
	if got := m.translateBody(ctx, "Some English text", store.Settings{}); got != "" {
		t.Fatalf("未配置引擎应返回空, got %q", got)
	}
}

// 日志已是目标语言时不翻译（返回空，调用方直接用原文，避免存冗余译文）。
func TestTranslateBodyAlreadyTarget(t *testing.T) {
	m, _ := newTestMonitor(t)
	ctx := context.Background()
	st := store.Settings{TranslateEngine: "dlx", TranslateTargetLang: "zh-Hans"}
	body := "本次更新修复了一个问题。"
	if got := m.translateBody(ctx, body, st); got != "" {
		t.Fatalf("已是目标语言应返回空, got %q", got)
	}
}

// 翻译成功时返回译文。
func TestTranslateBodySuccess(t *testing.T) {
	m, _ := newTestMonitor(t)
	ctx := context.Background()
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = w.Write([]byte(`{"code":200,"data":"这是翻译后的内容"}`))
	}))
	defer srv.Close()

	st := store.Settings{TranslateEngine: "dlx", TranslateTargetLang: "zh-Hans", TranslateURL: srv.URL}
	got := m.translateBody(ctx, "Some English text here", st)
	if !called {
		t.Fatal("应调用翻译引擎")
	}
	if got != "这是翻译后的内容" {
		t.Fatalf("翻译结果不符, got %q", got)
	}
}

// 翻译失败时返回空（调用方降级使用原文）。
func TestTranslateBodyFailure(t *testing.T) {
	m, _ := newTestMonitor(t)
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	st := store.Settings{TranslateEngine: "dlx", TranslateTargetLang: "zh-Hans", TranslateURL: srv.URL}
	if got := m.translateBody(ctx, "Some English text here", st); got != "" {
		t.Fatalf("翻译失败应返回空以回退原文, got %q", got)
	}
}
