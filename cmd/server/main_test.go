package main

import (
	"os"
	"testing"

	"github.com/yufei/chaxin/internal/store"
)

func TestEnvOr(t *testing.T) {
	t.Setenv("CHAXIN_TEST_ENV", "value")
	if got := envOr("CHAXIN_TEST_ENV", "def"); got != "value" {
		t.Fatalf("应返回环境变量值, got %q", got)
	}
	if got := envOr("CHAXIN_TEST_MISSING", "def"); got != "def" {
		t.Fatalf("缺失应返回默认值, got %q", got)
	}
}

func TestParseLogLevel(t *testing.T) {
	cases := map[string]struct {
		want string
	}{
		"debug": {"DEBUG"},
		"warn":  {"WARN"},
		"error": {"ERROR"},
		"":      {"INFO"},
		"other": {"INFO"},
	}
	for in, c := range cases {
		if got := parseLogLevel(in).String(); got != c.want {
			t.Fatalf("parseLogLevel(%q)=%q, want %q", in, got, c.want)
		}
	}
}

func TestApplyEnvDefaults(t *testing.T) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	t.Setenv("GITHUB_TOKEN", "env-token")
	if err := applyEnvDefaults(s); err != nil {
		t.Fatalf("applyEnvDefaults 应成功, got %v", err)
	}
	if v, _ := s.GetSetting(store.KeyGitHubToken); v != "env-token" {
		t.Fatalf("应写入 env token, got %q", v)
	}
	// 再次调用不应覆盖已有值（非幂等写入但值为相同，仍应为 env-token）
	if err := applyEnvDefaults(s); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.GetSetting(store.KeyGitHubToken); v != "env-token" {
		t.Fatalf("已有值不应改变, got %q", v)
	}
}

func TestApplyEnvDefaultsEmpty(t *testing.T) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// 所有 env 为空 → 仅遍历跳过，不报错
	os.Unsetenv("GITHUB_TOKEN")
	if err := applyEnvDefaults(s); err != nil {
		t.Fatalf("空 env 应成功, got %v", err)
	}
}
