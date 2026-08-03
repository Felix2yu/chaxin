package translate

import (
	"strings"
	"testing"
)

func TestCleanHTMLBlock(t *testing.T) {
	body := `<div align="center">
  <img src="https://cdn.tw93.fun/pic/cole.png" alt="Mole Logo" width="120" height="120" style="border-radius:50%" />
  <h1 style="margin: 12px 0 6px;">Mole</h1>
  <p><em>Deep clean and optimize your Mac.</em></p>
</div>
- **远程工作台期间可管理后台任务** — 描述
* f81cb321789aa3df62871248f5e4d361a59e7cc1 feat: Fix multiple security vulns thanks to secur0.com`
	got := Clean(body)
	if strings.Contains(got, "Mole") {
		t.Errorf("HTML 块应被整体移除, got %q", got)
	}
	if strings.Contains(got, "align=\"center\"") || strings.Contains(got, "<div") {
		t.Errorf("HTML 标签应被移除, got %q", got)
	}
}

func TestCleanMarkdownLink(t *testing.T) {
	body := `- **远程工作台期间可管理后台任务** — 远程工作台激活时，现在可以列出和取消本地后台任务，改善多任务处理。 ([#7198](https://github.com/esengine/DeepSeek-Reasonix/pull/7198))`
	got := Clean(body)
	if strings.Contains(got, "https://github.com") {
		t.Errorf("markdown 链接应被整体移除, got %q", got)
	}
	if strings.Contains(got, "#7198") {
		t.Errorf("链接文字也应移除, got %q", got)
	}
	if !strings.Contains(got, "远程工作台期间可管理后台任务") {
		t.Errorf("正文内容应保留, got %q", got)
	}
}

func TestCleanSHA(t *testing.T) {
	body := `* f81cb321789aa3df62871248f5e4d361a59e7cc1 feat: Fix multiple security vulns thanks to secur0.com`
	got := Clean(body)
	if strings.Contains(got, "f81cb321789aa3df62871248f5e4d361a59e7cc1") {
		t.Errorf("SHA 应被移除, got %q", got)
	}
	if !strings.Contains(got, "feat: Fix multiple security vulns") {
		t.Errorf("SHA 后的提交说明应保留, got %q", got)
	}
}

func TestCleanShortSHA(t *testing.T) {
	body := `* f81cb32 feat: Fix multiple security vulns thanks to secur0.com`
	got := Clean(body)
	if strings.Contains(got, "f81cb32") {
		t.Errorf("7 位短 SHA 应被移除, got %q", got)
	}
	if !strings.Contains(got, "feat: Fix multiple security vulns") {
		t.Errorf("提交说明应保留, got %q", got)
	}
}

func TestCleanDoesNotMangleNormalEnglish(t *testing.T) {
	body := "- Fixed a bug that caused the database to deadlock under load\n- Added support for new features"
	got := Clean(body)
	if !strings.Contains(got, "database to deadlock under load") {
		t.Errorf("正常英文不应被误删, got %q", got)
	}
	if !strings.Contains(got, "Added support for new features") {
		t.Errorf("正常英文不应被误删, got %q", got)
	}
}

func TestCleanPlainText(t *testing.T) {
	body := "hello world\n\n第二行内容"
	got := Clean(body)
	if got != body {
		t.Fatalf("纯文本不应被改动, got %q", got)
	}
}

func TestCleanEmpty(t *testing.T) {
	if got := Clean(""); got != "" {
		t.Fatalf("空文本应返回空, got %q", got)
	}
	if got := Clean("<div>only html</div>"); got != "" {
		t.Fatalf("纯 HTML 应被清空, got %q", got)
	}
}
