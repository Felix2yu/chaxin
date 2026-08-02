package translate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTranslateDLX(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/translate" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("bad request body: %v", err)
		}
		if body["target_lang"] != "ZH" {
			t.Errorf("target_lang = %q", body["target_lang"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"data":"你好，世界","source_lang":"EN","target_lang":"ZH"}`))
	}))
	defer srv.Close()

	got, err := Translate(context.Background(), Config{Engine: "dlx", URL: srv.URL, Target: "zh-Hans"}, "Hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "你好，世界" {
		t.Fatalf("got %q", got)
	}
}

func TestTranslateDLXTraditionalChinese(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["target_lang"] != "ZH-HANT" {
			t.Errorf("target_lang = %q, want ZH-HANT", body["target_lang"])
		}
		_, _ = w.Write([]byte(`{"code":200,"data":"你好，世界"}`))
	}))
	defer srv.Close()
	_, err := Translate(context.Background(), Config{Engine: "dlx", URL: srv.URL, Target: "zh-Hant"}, "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTranslateDLXError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	_, err := Translate(context.Background(), Config{Engine: "dlx", URL: srv.URL, Target: "zh-Hans"}, "Hello")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTranslateBing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("to"); got != "zh-Hans" {
			t.Errorf("to = %q", got)
		}
		var body []map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("bad body: %v", err)
		}
		if len(body) != 1 || body[0]["Text"] != "Hello" {
			t.Fatalf("bad body content: %+v", body)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"translations":[{"text":"你好","to":"zh-Hans"}]}]`))
	}))
	defer srv.Close()

	// bing 引擎固定使用公网 endpoint，无法指向 httptest；此处只验证请求解析逻辑通过私有翻译。
	got, err := translateBingEndpoint(context.Background(), Config{Target: "zh-Hans"}, "Hello", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "你好" {
		t.Fatalf("got %q", got)
	}
}

func TestTranslateOpenAI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("bad body: %v", err)
		}
		if body["model"] != "gpt-4o-mini" {
			t.Errorf("model = %v", body["model"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"这是译文"}}]}`))
	}))
	defer srv.Close()

	got, err := Translate(context.Background(), Config{
		Engine: "openai", URL: srv.URL, APIKey: "test-key", Model: "gpt-4o-mini", Target: "zh-Hans",
	}, "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "这是译文" {
		t.Fatalf("got %q", got)
	}
}

func TestTranslateOpenAIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer srv.Close()
	_, err := Translate(context.Background(), Config{Engine: "openai", URL: srv.URL, Target: "zh-Hans"}, "Hello")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTranslateUnknownEngine(t *testing.T) {
	_, err := Translate(context.Background(), Config{Engine: "nope", Target: "zh-Hans"}, "Hello")
	if err == nil {
		t.Fatal("expected error")
	}
}
