package translate

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func decodeInto(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func TestPrepareNoEngineAlreadyTarget(t *testing.T) {
	// 未配置引擎，但日志已是目标语言：直接返回原文，不报错
	body := "本次更新包含多项改进。\n修复了一个崩溃问题。"
	res, err := Prepare(context.Background(), Config{Target: "zh-Hans"}, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Text != body {
		t.Fatalf("expected original body, got %q", res.Text)
	}
	if res.Translated || res.Extracted {
		t.Fatalf("expected not translated/extracted, got %+v", res)
	}
}

func TestPrepareNoEngineForeignNeedsEngine(t *testing.T) {
	// 未配置引擎，日志为英文：应返回 ErrNotConfigured
	body := "This release brings several improvements.\nFixed a crash."
	_, err := Prepare(context.Background(), Config{Target: "zh-Hans"}, body)
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestPrepareExtractTargetLangFromBilingual(t *testing.T) {
	// 中英对照：目标中文应直接提取中文部分，不翻译
	body := strings.Join([]string{
		"### English",
		"This release adds new features.",
		"### 中文",
		"本次更新新增了若干功能。",
		"修复了一些问题。",
	}, "\n")
	res, err := Prepare(context.Background(), Config{Target: "zh-Hans"}, body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Extracted {
		t.Fatalf("expected extracted, got %+v", res)
	}
	if res.Translated {
		t.Fatalf("expected not translated")
	}
	if !strings.Contains(res.Text, "本次更新新增了若干功能。") || strings.Contains(res.Text, "English") {
		t.Fatalf("unexpected extraction: %q", res.Text)
	}
}

func TestPrepareTranslateForeignBody(t *testing.T) {
	// 纯英文日志 + DLX 引擎：应翻译
	var gotURL string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.Path
		_ = decodeInto(r, &gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"data":"本次更新包含新功能。","source_lang":"EN","target_lang":"ZH"}`))
	}))
	defer srv.Close()

	res, err := Prepare(context.Background(), Config{
		Engine: "dlx",
		URL:    srv.URL,
		Target: "zh-Hans",
	}, "This release includes new features.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Translated {
		t.Fatalf("expected translated")
	}
	if res.Text != "本次更新包含新功能。" {
		t.Fatalf("unexpected translation: %q", res.Text)
	}
	if gotURL != "/translate" {
		t.Fatalf("expected /translate, got %q", gotURL)
	}
	if gotBody["target_lang"] != "ZH" {
		t.Fatalf("expected target_lang ZH, got %q", gotBody["target_lang"])
	}
	if gotBody["source_lang"] != "auto" {
		t.Fatalf("expected source_lang auto, got %q", gotBody["source_lang"])
	}
}
