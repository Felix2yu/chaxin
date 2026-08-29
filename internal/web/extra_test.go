package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yufei/chaxin/internal/store"
)

// --- 设置/校验 ---

func TestPutSettingsDecodeError(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader("not-json"))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

func TestPutSettingsVerifyFail(t *testing.T) {
	m := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer m.Close()
	e := newEnv(t)
	body, _ := json.Marshal(map[string]any{
		"github_token":         "tok",
		"github_api_base_url":  m.URL + "/",
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(string(body)))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var out struct {
		Verify struct {
			TokenValid bool `json:"token_valid"`
		} `json:"verify"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("解码失败: %v", err)
	}
	if out.Verify.TokenValid {
		t.Fatal("token 校验失败应 token_valid=false")
	}
}

// --- 批量监控 ---

func TestBatchMonitorDecodeError(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/repos/batch-monitor", strings.NewReader("{"))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

func TestBatchMonitorMissingMonitored(t *testing.T) {
	e := newEnv(t)
	body, _ := json.Marshal(map[string]any{"ids": []int{1}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/repos/batch-monitor", strings.NewReader(string(body)))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

func TestBatchMonitorEmptyIDs(t *testing.T) {
	e := newEnv(t)
	mon := true
	body, _ := json.Marshal(map[string]any{"monitored": &mon})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/repos/batch-monitor", strings.NewReader(string(body)))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

func TestListReposMonitoredFilter(t *testing.T) {
	e := newEnv(t)
	_ = e.st.AddRepo(store.Repo{FullName: "owner/repo"}, store.SourceManual, true)
	repos, _ := e.st.ListRepos(store.RepoFilter{})
	_ = e.st.SetRepoMonitored(repos[0].ID, true)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/repos?monitored=1", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200, got %d", rr.Code)
	}
	var got []store.Repo
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("解码失败: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("过滤后应 1 个仓库, got %d", len(got))
	}
}

// --- 添加仓库错误分支 ---

func TestAddRepoInvalidFormat(t *testing.T) {
	e := newEnv(t)
	body, _ := json.Marshal(map[string]any{"full_name": "bad"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/repos", strings.NewReader(string(body)))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

func TestAddRepoInfo404(t *testing.T) {
	m := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer m.Close()
	e := newEnv(t)
	_ = e.st.SaveSettings(store.Settings{GitHubAPIBaseURL: m.URL + "/"})
	body, _ := json.Marshal(map[string]any{"full_name": "owner/repo"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/repos", strings.NewReader(string(body)))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAddRepoAlreadyExists(t *testing.T) {
	e := newEnv(t)
	_ = e.st.AddRepo(store.Repo{FullName: "owner/repo"}, store.SourceStar, false)
	body, _ := json.Marshal(map[string]any{"full_name": "owner/repo"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/repos", strings.NewReader(string(body)))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("应 409, got %d", rr.Code)
	}
}

// --- 修改/删除仓库错误分支 ---

func TestPatchRepoInvalidID(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/repos/abc", strings.NewReader("{}"))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

func TestPatchRepoDecodeError(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/repos/1", strings.NewReader("{"))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

func TestPatchRepoMissingFields(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/repos/1", strings.NewReader(`{"foo":1}`))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

func TestDeleteRepoInvalidID(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/repos/abc", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

func TestDeleteRepoUnstarFail(t *testing.T) {
	m := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/user/starred/") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer m.Close()
	e := newEnv(t)
	_ = e.st.SaveSettings(store.Settings{GitHubToken: "tok", GitHubAPIBaseURL: m.URL + "/"})
	_ = e.st.AddRepo(store.Repo{FullName: "owner/repo"}, store.SourceStar, false)
	repos, _ := e.st.ListRepos(store.RepoFilter{})
	if len(repos) == 0 {
		t.Fatal("应存在仓库")
	}
	id := repos[0].ID
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/repos/"+itoa(id), nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("应 502, got %d body=%s", rr.Code, rr.Body.String())
	}
}

// --- 通知错误分支 ---

func addNotificationID(t *testing.T, e *testEnv, n store.Notification) int64 {
	t.Helper()
	if err := e.st.AddNotification(n); err != nil {
		t.Fatalf("AddNotification: %v", err)
	}
	items, err := e.st.ListNotifications(store.NotificationFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	for _, it := range items {
		if it.Tag == n.Tag && it.FullName == n.FullName {
			return it.ID
		}
	}
	t.Fatal("未找到刚插入的通知")
	return 0
}

func TestRetryNotificationNotFound(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/99999/retry", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("应 404, got %d", rr.Code)
	}
}

func TestRetryNotificationAlreadySent(t *testing.T) {
	e := newEnv(t)
	id := addNotificationID(t, e, store.Notification{FullName: "o/r", Tag: "v1", Status: "sent"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/"+itoa(id)+"/retry", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

func TestRetryNotificationEmptyURL(t *testing.T) {
	e := newEnv(t)
	id := addNotificationID(t, e, store.Notification{FullName: "o/r", Tag: "v2", Status: "failed"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/"+itoa(id)+"/retry", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

func TestTestNotificationEmptyURL(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/test-notification", strings.NewReader("{}"))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

// --- 翻译错误分支 ---

func TestTranslateEmptyText(t *testing.T) {
	e := newEnv(t)
	body, _ := json.Marshal(map[string]any{"text": "   "})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/translate", strings.NewReader(string(body)))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

func TestTranslateFail(t *testing.T) {
	e := newEnv(t)
	_ = e.st.SaveSettings(store.Settings{TranslateEngine: "dlx", TranslateURL: "http://127.0.0.1:1/"})
	body, _ := json.Marshal(map[string]any{"text": "hello", "engine": "dlx"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/translate", strings.NewReader(string(body)))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("应 502, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestTranslateDecodeError(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/translate", strings.NewReader("{"))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

// --- 备份/恢复错误分支 ---

func TestRestoreDecodeError(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/restore", strings.NewReader("{"))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

func TestRestoreVersionTooLow(t *testing.T) {
	e := newEnv(t)
	body, _ := json.Marshal(map[string]any{"version": 0})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/restore", strings.NewReader(string(body)))
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("应 400, got %d", rr.Code)
	}
}

// --- 同步冲突 ---

func TestSyncStarsConflict(t *testing.T) {
	e := newEnv(t)
	_ = e.st.SaveSettings(store.Settings{GitHubToken: "tok"})
	e.s.syncMu.Lock()
	e.s.sync = &SyncState{Running: true}
	e.s.syncMu.Unlock()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/repos/sync-stars", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("应 409, got %d", rr.Code)
	}
}

// --- 静态资源 ---

func TestStaticSPAFallback(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/some/spa/route", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("SPA fallback 应 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "<!doctype html>") {
		t.Fatalf("应返回 index.html 内容")
	}
}

func TestStaticAPI404(t *testing.T) {
	e := newEnv(t)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
	e.s.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("应 404, got %d", rr.Code)
	}
}
