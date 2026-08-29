package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yufei/chaxin/internal/store"
)

// --- 备份 ---

func TestBackup(t *testing.T) {
	e := newEnv(t)
	_ = e.st.AddRepo(store.Repo{FullName: "owner/repo", Language: "Go"}, store.SourceStar, false)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/backup", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var bkp Backup
	if err := json.Unmarshal(rr.Body.Bytes(), &bkp); err != nil {
		t.Fatalf("解码备份失败: %v", err)
	}
	if bkp.Version != 1 {
		t.Fatalf("版本应为 1, got %d", bkp.Version)
	}
	if len(bkp.Repos) != 1 {
		t.Fatalf("应导出 1 个仓库, got %d", len(bkp.Repos))
	}
}

// --- 同步状态（非空）---

func TestSyncStarsStatusWithState(t *testing.T) {
	e := newEnv(t)
	e.s.syncMu.Lock()
	e.s.sync = &SyncState{Running: true, Page: 2, Total: 5, Repos: 3, Added: 1, Removed: 0, Progress: 0.4}
	e.s.syncMu.Unlock()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/repos/sync-stars/status", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d", rr.Code)
	}
	var st SyncState
	if err := json.Unmarshal(rr.Body.Bytes(), &st); err != nil {
		t.Fatalf("解码失败: %v", err)
	}
	if !st.Running || st.Page != 2 || st.Total != 5 {
		t.Fatalf("状态字段不匹配: %+v", st)
	}
}

// --- 删除仓库：异常 fullname ---

func TestDeleteRepoBadFullName(t *testing.T) {
	e := newEnv(t)
	_ = e.st.AddRepo(store.Repo{FullName: "badname"}, store.SourceManual, false)
	repos, _ := e.st.ListRepos(store.RepoFilter{})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/repos/"+itoa(repos[0].ID), nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("应 500, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// --- 修改仓库：同时设置两个字段 ---

func TestPatchRepoBothFields(t *testing.T) {
	e := newEnv(t)
	_ = e.st.AddRepo(store.Repo{FullName: "owner/repo"}, store.SourceManual, false)
	repos, _ := e.st.ListRepos(store.RepoFilter{})
	body, _ := json.Marshal(map[string]any{
		"monitored":       false,
		"ignore_pattern":  "v[0-9]+",
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/repos/"+itoa(repos[0].ID), strings.NewReader(string(body)))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// --- 列表过滤：语言 ---

func TestListReposLanguageFilter(t *testing.T) {
	e := newEnv(t)
	_ = e.st.AddRepo(store.Repo{FullName: "owner/go", Language: "Go"}, store.SourceStar, false)
	_ = e.st.AddRepo(store.Repo{FullName: "owner/rust", Language: "Rust"}, store.SourceStar, false)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/repos?language=Go", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d", rr.Code)
	}
	var repos []store.Repo
	_ = json.Unmarshal(rr.Body.Bytes(), &repos)
	if len(repos) != 1 || repos[0].Language != "Go" {
		t.Fatalf("应按语言过滤为 1 个 Go 仓库, got %d", len(repos))
	}
}

// --- 测试通知发送成功 ---

func TestTestNotificationSendSuccess(t *testing.T) {
	e := newEnv(t)
	_ = e.st.SaveSettings(store.Settings{ShoutrrrURL: "logger://"})
	body, _ := json.Marshal(map[string]any{"title": "t", "message": "m"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test-notification", strings.NewReader(string(body)))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// --- 重试通知发送成功 ---

func TestRetryNotificationSendSuccess(t *testing.T) {
	e := newEnv(t)
	_ = e.st.SaveSettings(store.Settings{ShoutrrrURL: "logger://"})
	id := addNotificationID(t, e, store.Notification{FullName: "o/r", Tag: "v9", Status: "failed"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/"+itoa(id)+"/retry", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// --- renderMarkdown 各分支 ---

func TestRenderMarkdown(t *testing.T) {
	cases := []string{
		"# 标题",
		"## 二级",
		"普通段落文字",
		"- 列表项一\n- 列表项二",
		"1. 有序一\n2. 有序二",
		"[链接](https://example.com)",
		"行内 `code` 文本",
		"```\n代码块\n```",
		"> 引用文本",
		"**加粗** 与 *斜体*",
		"",
	}
	for _, c := range cases {
		if got := renderMarkdown(c); got == "" && c != "" {
			t.Fatalf("renderMarkdown(%q) 不应返回空", c)
		}
	}
	// feedDescription / feedBaseURL
	if feedDescription("正文") == "" {
		t.Fatal("feedDescription 不应为空")
	}
}

func TestFeedDescriptionStrip(t *testing.T) {
	// 含 markdown 标记的正文被清理为纯文本摘要
	got := feedDescription("# 标题\n\n这是 **重点** 说明")
	if strings.Contains(got, "#") || strings.Contains(got, "**") {
		t.Fatalf("摘要应去除标记, got %q", got)
	}
}

// --- 手动触发监控（成功路径）---

func TestRunMonitorSuccess(t *testing.T) {
	m := mockGitHub(t)
	e := newEnv(t)
	_ = e.st.SaveSettings(store.Settings{GitHubToken: "tok", GitHubAPIBaseURL: m.URL + "/"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/monitor/run", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// --- 列表查询过滤（handler 层）---

func TestListReposQueryParam(t *testing.T) {
	e := newEnv(t)
	_ = e.st.AddRepo(store.Repo{FullName: "alpha/one"}, store.SourceStar, false)
	_ = e.st.AddRepo(store.Repo{FullName: "beta/two"}, store.SourceStar, false)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/repos?query=alpha", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d", rr.Code)
	}
	var repos []store.Repo
	_ = json.Unmarshal(rr.Body.Bytes(), &repos)
	if len(repos) != 1 || repos[0].FullName != "alpha/one" {
		t.Fatalf("应按 query 过滤为 1 个, got %d", len(repos))
	}
}

// --- 删除不存在的仓库（404 分支）---

func TestDeleteRepoNotFound(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/repos/99999", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("应 404, got %d", rr.Code)
	}
}

// --- 修改不存在的仓库（404 分支）---

func TestPatchRepoNotFound(t *testing.T) {
	e := newEnv(t)
	body, _ := json.Marshal(map[string]any{"monitored": false})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/repos/99999", strings.NewReader(string(body)))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("应 404, got %d", rr.Code)
	}
}

// --- 添加仓库：非法 JSON（解码错误分支）---

func TestAddRepoDecodeError(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/repos", strings.NewReader("{"))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

// --- 通知列表：非法 limit 回退默认 ---

func TestListNotificationsInvalidLimit(t *testing.T) {
	e := newEnv(t)
	_ = e.st.AddNotification(store.Notification{FullName: "o/r", Tag: "v1", Status: "sent"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/notifications?limit=abc", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d", rr.Code)
	}
	var items []map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 1 {
		t.Fatalf("非法 limit 应回退默认并仍返回通知, got %d", len(items))
	}
}

// --- 获取设置（成功路径）---

func TestGetSettingsSuccess(t *testing.T) {
	e := newEnv(t)
	_ = e.st.SaveSettings(store.Settings{GitHubToken: "tok"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d", rr.Code)
	}
}

// --- 通知列表过滤 ---

func TestListNotificationsFilter(t *testing.T) {
	e := newEnv(t)
	_ = e.st.AddNotification(store.Notification{FullName: "o/r", Tag: "v1", Status: "failed"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/notifications?limit=1&status=failed", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d", rr.Code)
	}
	var items []map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 1 {
		t.Fatalf("应返回 1 条失败通知, got %d", len(items))
	}
}

// --- RSS feed ---

func TestFeedRenders(t *testing.T) {
	e := newEnv(t)
	_ = e.st.AddRepo(store.Repo{FullName: "owner/repo", HTMLURL: "https://github.com/owner/repo", Description: "desc"}, store.SourceStar, true)
	repos, _ := e.st.ListRepos(store.RepoFilter{})
	id := repos[0].ID
	_ = e.st.SetRepoMonitored(id, true)
	_ = e.st.SetLatestRelease(id, "v1.0.0", "https://x/releases/v1.0.0", "# 更新", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feed", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Content-Type"), "rss") {
		t.Fatalf("Content-Type 应为 rss, got %q", rr.Header().Get("Content-Type"))
	}
	if !strings.Contains(rr.Body.String(), "owner/repo") {
		t.Fatalf("feed 应含仓库名")
	}
}
