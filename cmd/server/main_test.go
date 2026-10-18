package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
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

func TestRunBackupWritesSnapshotToWriter(t *testing.T) {
	dataDir := t.TempDir()
	s, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpsertRepo(store.Repo{FullName: "a/b", Owner: "a", Name: "b"}, false); err != nil {
		t.Fatal(err)
	}
	s.Close()

	t.Setenv("DATA_DIR", dataDir)
	var buf bytes.Buffer
	if err := runBackup(&buf); err != nil {
		t.Fatalf("runBackup 应成功, got %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("SQLite format 3\x00")) {
		t.Fatalf("输出应以 SQLite 魔数开头, got %q", buf.Bytes()[:min(16, buf.Len())])
	}
}

func TestRunBackupMissingDBFailsAndDoesNotCreate(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)

	var buf bytes.Buffer
	if err := runBackup(&buf); err == nil {
		t.Fatal("数据文件不存在时 runBackup 应报错")
	}
	if buf.Len() != 0 {
		t.Fatalf("失败时不应有输出, got %d 字节", buf.Len())
	}
	if _, err := os.Stat(filepath.Join(dataDir, "chaxin.db")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("backup 不应创建数据库文件, stat err=%v", err)
	}
	// 数据目录本身也不应被新建
	missing := filepath.Join(t.TempDir(), "no-such-dir")
	t.Setenv("DATA_DIR", missing)
	if err := runBackup(&buf); err == nil {
		t.Fatal("目录不存在时 runBackup 应报错")
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("backup 不应创建数据目录, stat err=%v", err)
	}
}
