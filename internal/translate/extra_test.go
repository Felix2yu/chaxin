package translate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// translateMock 按路径返回各翻译引擎的模拟响应。
func translateMock(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		p := r.URL.Path
		switch {
		case strings.Contains(p, "/translate_a/single"):
			_ = json.NewEncoder(w).Encode([][]any{{"谷歌译文"}})
		case strings.HasSuffix(p, "/translate"):
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": "DLX译文"})
		case strings.Contains(p, "/chat/completions"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{"message": map[string]any{"content": "AI译文"}}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestTranslateEntrypoint 覆盖 Translate 入口的 dlx/google/openai 分发（youdao/bing 走公网端点不在此覆盖）。
func TestTranslateEntrypoint(t *testing.T) {
	srv := translateMock(t)
	ctx := context.Background()
	cases := []struct {
		engine string
		cfg    Config
		want   string
	}{
		{"dlx", Config{Engine: "dlx", URL: srv.URL}, "DLX译文"},
		{"google", Config{Engine: "google", URL: srv.URL, Target: "zh-Hans"}, "谷歌译文"},
		{"openai", Config{Engine: "openai", URL: srv.URL, APIKey: "k", Target: "zh-Hans"}, "AI译文"},
	}
	for _, c := range cases {
		got, err := Translate(ctx, c.cfg, "hello")
		if err != nil {
			t.Fatalf("%s 翻译应成功, got err=%v", c.engine, err)
		}
		if got != c.want {
			t.Fatalf("%s 译文不符, got %q want %q", c.engine, got, c.want)
		}
	}
}

func TestTranslateDLXEmptyAndBadCode(t *testing.T) {
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": "  "})
	}))
	defer srv.Close()
	if _, err := translateDLX(ctx, Config{URL: srv.URL}, "x"); err == nil {
		t.Fatal("空译文应报错")
	}
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 500, "data": "x"})
	}))
	defer srv2.Close()
	if _, err := translateDLX(ctx, Config{URL: srv2.URL}, "x"); err == nil {
		t.Fatal("错误 code 应报错")
	}
}

func TestTranslateOpenAIEmptyURL(t *testing.T) {
	if _, err := translateOpenAI(context.Background(), Config{}, "x"); err == nil {
		t.Fatal("空 URL 应报错")
	}
}

func TestLanguageMappings(t *testing.T) {
	langCases := map[string]string{
		"zh-Hans": "简体中文", "zh-Hant": "繁体中文", "en": "英文",
		"ja": "日文", "ko": "韩文", "fr": "fr",
	}
	for in, want := range langCases {
		if got := languageName(in); got != want {
			t.Fatalf("languageName(%q)=%q, want %q", in, got, want)
		}
	}
	dlxCases := map[string]string{
		"zh-Hans": "ZH", "zh-Hant": "ZH-HANT", "en": "EN", "ja": "JA", "ko": "KO", "fr": "FR",
	}
	for in, want := range dlxCases {
		if got := dlxLang(in); got != want {
			t.Fatalf("dlxLang(%q)=%q, want %q", in, got, want)
		}
	}
	bingCases := map[string]string{
		"zh-Hans": "zh-Hans", "zh-Hant": "zh-Hant", "en": "en", "ja": "ja", "ko": "ko", "fr": "fr",
	}
	for in, want := range bingCases {
		if got := bingLang(in); got != want {
			t.Fatalf("bingLang(%q)=%q, want %q", in, got, want)
		}
	}
	googleCases := map[string]string{
		"zh-Hans": "zh-CN", "zh-Hant": "zh-TW", "en": "en", "ja": "ja", "ko": "ko", "fr": "fr",
	}
	for in, want := range googleCases {
		if got := googleLang(in); got != want {
			t.Fatalf("googleLang(%q)=%q, want %q", in, got, want)
		}
	}
	youdaoCases := map[string]string{
		"zh-Hans": "zh-CHS", "zh-Hant": "zh-CHT", "en": "en", "ja": "ja", "ko": "ko", "fr": "fr",
	}
	for in, want := range youdaoCases {
		if got := youdaoLang(in); got != want {
			t.Fatalf("youdaoLang(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestCollectGoogleSegments(t *testing.T) {
	var parts []string
	collectGoogleSegments([]any{[]any{"甲"}, []any{"乙"}}, &parts)
	if strings.Join(parts, "") != "甲乙" {
		t.Fatalf("应收集甲+乙, got %q", strings.Join(parts, ""))
	}
}

func TestTruncate(t *testing.T) {
	if truncate("短", 100) != "短" {
		t.Fatal("短文本不应截断")
	}
	if got := truncate("超长内容", 2); !strings.HasSuffix(got, "…") {
		t.Fatalf("超长应截断, got %q", got)
	}
}
