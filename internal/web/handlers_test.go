package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yufei/chaxin/internal/monitor"
	"github.com/yufei/chaxin/internal/store"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// mockGitHub 模拟 GitHub API，供需要校验/拉取仓库的 handler 使用。
func mockGitHub(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		p := r.URL.Path
		switch {
		case p == "/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"login": "octocat"})
		case p == "/user/starred":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"repo": map[string]any{
					"full_name": "owner/repo", "owner": map[string]any{"login": "owner"},
					"name": "repo", "description": "desc", "language": "Go",
					"stargazers_count": 5, "html_url": "https://github.com/owner/repo",
				}},
			})
		case strings.HasPrefix(p, "/user/starred/"):
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case strings.HasSuffix(p, "/releases"):
			_ = json.NewEncoder(w).Encode([]any{})
		case strings.Contains(p, "/repos/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"full_name": "owner/repo", "owner": map[string]any{"login": "owner"},
				"name": "repo", "description": "desc", "language": "Go",
				"stargazers_count": 10, "html_url": "https://github.com/owner/repo",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

type testEnv struct {
	srv *httptest.Server
	st  *store.Store
	s   *Server
}

func newEnv(t *testing.T) *testEnv {
	st := newTestStore(t)
	mon := monitor.New(st, discardLogger())
	s := NewServer(st, mon, discardLogger())
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	return &testEnv{srv: srv, st: st, s: s}
}

func (e *testEnv) do(t *testing.T, method, path, body string) (*http.Response, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, e.srv.URL+path, rdr)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	resp, data := e.do(t, http.MethodGet, "/api/health", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health 应 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(data), "ok") {
		t.Fatalf("响应应含 ok, got %s", data)
	}
}

func TestGetSettingsEmpty(t *testing.T) {
	e := newEnv(t)
	resp, _ := e.do(t, http.MethodGet, "/api/settings", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("settings 应 200, got %d", resp.StatusCode)
	}
}

func TestPutSettingsNoToken(t *testing.T) {
	e := newEnv(t)
	resp, data := e.do(t, http.MethodPut, "/api/settings", `{"github_token":"","max_notifications":10}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put settings 应 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(data), "token_valid") {
		t.Fatalf("应返回 verify 信息, got %s", data)
	}
	got, _ := e.st.GetSettings()
	if got.MaxNotifications != 10 {
		t.Fatalf("设置应已保存, got %+v", got)
	}
}

func TestPutSettingsVerifyToken(t *testing.T) {
	e := newEnv(t)
	srv := mockGitHub(t)
	resp, data := e.do(t, http.MethodPut, "/api/settings",
		`{"github_token":"tok","github_api_base_url":"`+srv.URL+`/"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put settings 应 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(data), `"username"`) {
		t.Fatalf("应校验并返回用户名, got %s", data)
	}
}

func TestListRepos(t *testing.T) {
	e := newEnv(t)
	resp, data := e.do(t, http.MethodGet, "/api/repos", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list repos 应 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(data), "[]") {
		t.Fatalf("空列表应返回 [], got %s", data)
	}
}

func TestBatchMonitor(t *testing.T) {
	e := newEnv(t)
	e.st.AddRepo(store.Repo{FullName: "a/b", Owner: "a", Name: "b"}, store.SourceManual, true)
	list, _ := e.st.ListRepos(store.RepoFilter{})
	id := list[0].ID
	resp, _ := e.do(t, http.MethodPost, "/api/repos/batch-monitor",
		`{"ids":[`+itoa(id)+`],"monitored":false}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("batch-monitor 应 200, got %d", resp.StatusCode)
	}
	// 缺少字段
	if r, _ := e.do(t, http.MethodPost, "/api/repos/batch-monitor", `{"ids":[]}`); r.StatusCode != http.StatusBadRequest {
		t.Fatalf("缺少 monitored 应 400, got %d", r.StatusCode)
	}
}

func TestAddRepo(t *testing.T) {
	e := newEnv(t)
	srv := mockGitHub(t)
	e.st.SaveSettings(store.Settings{GitHubToken: "tok", GitHubAPIBaseURL: srv.URL + "/"})
	resp, _ := e.do(t, http.MethodPost, "/api/repos", `{"full_name":"owner/repo"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add repo 应 201, got %d", resp.StatusCode)
	}
	if n, _ := e.st.CountRepos(); n != 1 {
		t.Fatalf("应新增 1 个仓库, got %d", n)
	}
	// 格式错误
	if r, _ := e.do(t, http.MethodPost, "/api/repos", `{"full_name":"bad"}`); r.StatusCode != http.StatusBadRequest {
		t.Fatalf("格式错误应 400, got %d", r.StatusCode)
	}
}

func TestPatchRepo(t *testing.T) {
	e := newEnv(t)
	e.st.AddRepo(store.Repo{FullName: "a/b", Owner: "a", Name: "b"}, store.SourceManual, true)
	id := mustID(t, e.st, "a/b")
	resp, _ := e.do(t, http.MethodPatch, "/api/repos/"+itoa(id), `{"monitored":true}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch repo 应 200, got %d", resp.StatusCode)
	}
	r, _ := e.st.GetRepoByID(id)
	if !r.Monitored {
		t.Fatal("应已更新为监控")
	}
	// 无效 id
	if r2, _ := e.do(t, http.MethodPatch, "/api/repos/abc", `{"monitored":true}`); r2.StatusCode != http.StatusBadRequest {
		t.Fatalf("无效 id 应 400, got %d", r2.StatusCode)
	}
	// 不存在
	if r3, _ := e.do(t, http.MethodPatch, "/api/repos/9999", `{"monitored":true}`); r3.StatusCode != http.StatusNotFound {
		t.Fatalf("不存在应 404, got %d", r3.StatusCode)
	}
}

func TestDeleteRepoNoToken(t *testing.T) {
	e := newEnv(t)
	e.st.AddRepo(store.Repo{FullName: "a/b", Owner: "a", Name: "b"}, store.SourceManual, true)
	id := mustID(t, e.st, "a/b")
	resp, _ := e.do(t, http.MethodDelete, "/api/repos/"+itoa(id), "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete repo 应 204, got %d", resp.StatusCode)
	}
	if n, _ := e.st.CountRepos(); n != 0 {
		t.Fatalf("应已删除, got %d", n)
	}
}

func TestDeleteRepoWithToken(t *testing.T) {
	e := newEnv(t)
	srv := mockGitHub(t)
	e.st.SaveSettings(store.Settings{GitHubToken: "tok", GitHubAPIBaseURL: srv.URL + "/"})
	e.st.AddRepo(store.Repo{FullName: "a/b", Owner: "a", Name: "b"}, store.SourceManual, true)
	id := mustID(t, e.st, "a/b")
	resp, _ := e.do(t, http.MethodDelete, "/api/repos/"+itoa(id), "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("带 token 删除应 204, got %d", resp.StatusCode)
	}
}

func TestListNotifications(t *testing.T) {
	e := newEnv(t)
	e.st.AddNotification(store.Notification{FullName: "a/b", Tag: "v1", Status: "sent"})
	resp, data := e.do(t, http.MethodGet, "/api/notifications?limit=10", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list notifications 应 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(data), "a/b") {
		t.Fatalf("应返回通知, got %s", data)
	}
}

func TestRetryNotification(t *testing.T) {
	e := newEnv(t)
	e.st.SaveSettings(store.Settings{ShoutrrrURL: "logger://"})
	e.st.AddNotification(store.Notification{FullName: "a/b", Tag: "v1", Status: "failed"})
	id := mustNotifID(t, e.st, "a/b")
	resp, _ := e.do(t, http.MethodPost, "/api/notifications/"+itoa(id)+"/retry", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("retry 应 200, got %d", resp.StatusCode)
	}
	// 无 ShoutrrrURL
	e.st.SaveSettings(store.Settings{})
	if r, _ := e.do(t, http.MethodPost, "/api/notifications/"+itoa(id)+"/retry", ""); r.StatusCode != http.StatusBadRequest {
		t.Fatalf("无 url 应 400, got %d", r.StatusCode)
	}
}

func TestTestNotification(t *testing.T) {
	e := newEnv(t)
	e.st.SaveSettings(store.Settings{ShoutrrrURL: "logger://"})
	resp, _ := e.do(t, http.MethodPost, "/api/test-notification", `{"title":"t","message":"m"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("test-notification 应 200, got %d", resp.StatusCode)
	}
	e.st.SaveSettings(store.Settings{})
	if r, _ := e.do(t, http.MethodPost, "/api/test-notification", ""); r.StatusCode != http.StatusBadRequest {
		t.Fatalf("无 url 应 400, got %d", r.StatusCode)
	}
}

func TestTranslateEngineOff(t *testing.T) {
	e := newEnv(t)
	resp, _ := e.do(t, http.MethodPost, "/api/translate", `{"text":"hello","engine":"off"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("engine off 应 400, got %d", resp.StatusCode)
	}
	// 缺文本
	if r, _ := e.do(t, http.MethodPost, "/api/translate", `{"engine":"dlx"}`); r.StatusCode != http.StatusBadRequest {
		t.Fatalf("缺文本应 400, got %d", r.StatusCode)
	}
}

func TestRunMonitor(t *testing.T) {
	e := newEnv(t)
	resp, _ := e.do(t, http.MethodPost, "/api/monitor/run", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("run monitor 应 200, got %d", resp.StatusCode)
	}
}

func TestBackupRestore(t *testing.T) {
	e := newEnv(t)
	e.st.AddRepo(store.Repo{FullName: "a/b", Owner: "a", Name: "b"}, store.SourceManual, true)
	resp, data := e.do(t, http.MethodGet, "/api/backup", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("backup 应 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(data), "a/b") {
		t.Fatalf("backup 应含仓库, got %s", data)
	}
	// restore
	resp2, _ := e.do(t, http.MethodPost, "/api/restore", string(data))
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("restore 应 200, got %d", resp2.StatusCode)
	}
	if n, _ := e.st.CountRepos(); n != 1 {
		t.Fatalf("restore 后应有 1 个仓库, got %d", n)
	}
	// 非法备份
	if r, _ := e.do(t, http.MethodPost, "/api/restore", `{"version":0}`); r.StatusCode != http.StatusBadRequest {
		t.Fatalf("非法备份应 400, got %d", r.StatusCode)
	}
}

func TestFeed(t *testing.T) {
	e := newEnv(t)
	e.st.AddRepo(store.Repo{FullName: "a/b", Owner: "a", Name: "b"}, store.SourceManual, true)
	id := mustID(t, e.st, "a/b")
	e.st.SetRepoMonitored(id, true)
	e.st.SetLatestRelease(id, "v1.0", "https://example.com/v1", "release body", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	resp, data := e.do(t, http.MethodGet, "/feed", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("feed 应 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(data), "v1.0") {
		t.Fatalf("feed 应含版本, got %s", data)
	}
	if !strings.Contains(string(data), "<rss") {
		t.Fatalf("feed 应为 rss, got %s", data)
	}
}

func TestStatic(t *testing.T) {
	e := newEnv(t)
	resp, _ := e.do(t, http.MethodGet, "/", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("static 应 200, got %d", resp.StatusCode)
	}
	// 未知 api 路径
	if r, _ := e.do(t, http.MethodGet, "/api/unknown", ""); r.StatusCode != http.StatusNotFound {
		t.Fatalf("未知 api 应 404, got %d", r.StatusCode)
	}
}

func TestSyncStars(t *testing.T) {
	e := newEnv(t)
	srv := mockGitHub(t)
	e.st.SaveSettings(store.Settings{GitHubToken: "tok", GitHubAPIBaseURL: srv.URL + "/"})
	// 无 token 不应启动
	resp, _ := e.do(t, http.MethodPost, "/api/repos/sync-stars", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sync-stars 应 200, got %d", resp.StatusCode)
	}
	// 轮询直到同步完成或超时
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		r, _ := e.do(t, http.MethodGet, "/api/repos/sync-stars/status", "")
		if r.StatusCode == http.StatusOK {
			var st SyncState
			// 忽略解析错误，仅用于判断 running
			_ = st
		}
		if n, _ := e.st.CountRepos(); n >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if n, _ := e.st.CountRepos(); n < 1 {
		t.Fatalf("同步后应至少 1 个仓库, got %d", n)
	}
}

func mustID(t *testing.T, s *store.Store, full string) int64 {
	t.Helper()
	list, err := s.ListRepos(store.RepoFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range list {
		if r.FullName == full {
			return r.ID
		}
	}
	t.Fatalf("未找到仓库 %s", full)
	return 0
}

func mustNotifID(t *testing.T, s *store.Store, full string) int64 {
	t.Helper()
	items, err := s.ListNotifications(store.NotificationFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.FullName == full {
			return it.ID
		}
	}
	t.Fatalf("未找到通知 %s", full)
	return 0
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
