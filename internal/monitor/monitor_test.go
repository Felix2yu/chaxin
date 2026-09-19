package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v89/github"
	"github.com/yufei/chaxin/internal/githubx"
	"github.com/yufei/chaxin/internal/notifier"
	"github.com/yufei/chaxin/internal/store"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func sampleReleases() []map[string]any {
	return []map[string]any{
		{
			"tag_name":     "v2.0.0",
			"name":         "v2.0.0",
			"body":         "release body",
			"html_url":     "https://github.com/owner/repo/releases/v2.0.0",
			"published_at": "2024-01-02T00:00:00Z",
			"draft":        false, "prerelease": false,
		},
	}
}

func releaseServer(t *testing.T, releases []map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/releases") {
			_ = json.NewEncoder(w).Encode(releases)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func addMonitored(t *testing.T, s *store.Store, full string) int64 {
	t.Helper()
	s.AddRepo(store.Repo{FullName: full, Owner: "owner", Name: "repo"}, store.SourceManual, true)
	list, err := s.ListRepos(store.RepoFilter{})
	if err != nil {
		t.Fatal(err)
	}
	var id int64
	for _, r := range list {
		if r.FullName == full {
			id = r.ID
		}
	}
	s.SetRepoMonitored(id, true)
	return id
}

// checkRepoDirect 构造依赖并直接调用未导出的 checkRepo，避免触发带 ticker 的整轮调度。
func checkRepoDirect(t *testing.T, s *store.Store, repo store.Repo, releases []map[string]any, settings store.Settings) error {
	t.Helper()
	srv := releaseServer(t, releases)
	settings.GitHubToken = "tok"
	settings.GitHubAPIBaseURL = srv.URL + "/"
	_ = s.SaveSettings(settings)
	c, err := githubx.NewClient("tok", srv.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	n, _ := notifier.New("logger://")
	m := New(s, testLogger())
	return m.checkRepo(context.Background(), c, n, repo, settings)
}

func TestCheckAllNoToken(t *testing.T) {
	s := newTestStore(t)
	m := New(s, testLogger())
	if err := m.CheckAll(context.Background()); err != nil {
		t.Fatalf("无 token 应直接返回 nil, got %v", err)
	}
}

func TestCheckAllNoMonitored(t *testing.T) {
	s := newTestStore(t)
	srv := releaseServer(t, sampleReleases())
	s.SaveSettings(store.Settings{
		GitHubToken:      "tok",
		GitHubAPIBaseURL: srv.URL + "/",
	})
	m := New(s, testLogger())
	if err := m.CheckAll(context.Background()); err != nil {
		t.Fatalf("无监控仓库应直接返回 nil, got %v", err)
	}
}

func TestCheckRepoFirstRunNotify(t *testing.T) {
	s := newTestStore(t)
	id := addMonitored(t, s, "owner/repo")
	r, _ := s.GetRepoByID(id)
	if err := checkRepoDirect(t, s, r, sampleReleases(), store.Settings{NotifyOnFirstRun: true}); err != nil {
		t.Fatalf("checkRepo 应成功, got %v", err)
	}
	items, _ := s.ListNotifications(store.NotificationFilter{})
	if len(items) != 1 {
		t.Fatalf("首次运行应通知 1 条, got %d", len(items))
	}
	if items[0].Tag != "v2.0.0" || items[0].Status != "sent" {
		t.Fatalf("通知内容不符, got %+v", items[0])
	}
	tag, found, _ := s.GetPlatformTag(id, "default")
	if !found || tag != "v2.0.0" {
		t.Fatalf("平台基线应为 v2.0.0, got %q found=%v", tag, found)
	}
}

func TestCheckRepoNoRelease(t *testing.T) {
	s := newTestStore(t)
	id := addMonitored(t, s, "owner/repo")
	r, _ := s.GetRepoByID(id)
	if err := checkRepoDirect(t, s, r, nil, store.Settings{}); err != nil {
		t.Fatalf("空发布应成功, got %v", err)
	}
	r2, _ := s.GetRepoByID(id)
	if r2.LastCheckedAt.IsZero() {
		t.Fatal("ErrNoRelease 分支应 TouchCheckedAt")
	}
}

func TestCheckRepoIgnorePattern(t *testing.T) {
	s := newTestStore(t)
	id := addMonitored(t, s, "owner/repo")
	s.SetRepoIgnorePattern(id, `^v2\.`)
	r, _ := s.GetRepoByID(id)
	if err := checkRepoDirect(t, s, r, sampleReleases(), store.Settings{NotifyOnFirstRun: true}); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.CountNotifications(); n != 0 {
		t.Fatalf("命中忽略规则不应通知, got %d", n)
	}
}

func TestCheckRepoNewVersionNotify(t *testing.T) {
	s := newTestStore(t)
	id := addMonitored(t, s, "owner/repo")
	s.SetPlatformTag(id, "default", "v1.0.0", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	r, _ := s.GetRepoByID(id)
	if err := checkRepoDirect(t, s, r, sampleReleases(), store.Settings{}); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.CountNotifications(); n != 1 {
		t.Fatalf("检测到新版本应通知 1 条, got %d", n)
	}
	tag, _, _ := s.GetPlatformTag(id, "default")
	if tag != "v2.0.0" {
		t.Fatalf("平台基线应更新为 v2.0.0, got %q", tag)
	}
}

func TestRunReturnsOnCancel(t *testing.T) {
	s := newTestStore(t)
	m := New(s, testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		m.Run(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run 应在 context 取消后退出")
	}
}

func TestParsePlatform(t *testing.T) {
	cases := map[string]string{
		"v1.2.3":     "default",
		"iOS-7.1.2":  "ios",
		"mac-1.0":    "mac",
		"Random-1.0": "default",
		"cli-2.0":    "cli",
		"1.0":        "default",
	}
	for tag, want := range cases {
		if got := parsePlatform(tag); got != want {
			t.Fatalf("parsePlatform(%q)=%q, want %q", tag, got, want)
		}
	}
}

func TestIsASCIIAlpha(t *testing.T) {
	if !isASCIIAlpha('a') || !isASCIIAlpha('Z') {
		t.Fatal("a/Z 应判定为字母")
	}
	if isASCIIAlpha('1') || isASCIIAlpha('-') {
		t.Fatal("数字/连字符不应判定为字母")
	}
}

func TestMatchesIgnorePattern(t *testing.T) {
	if ok, _ := matchesIgnorePattern("", "v1"); ok {
		t.Fatal("空正则应不匹配")
	}
	if ok, _ := matchesIgnorePattern(`^v1\.`, "v1.2.3"); !ok {
		t.Fatal("应命中忽略正则")
	}
	if _, err := matchesIgnorePattern("[invalid", "v1"); err == nil {
		t.Fatal("非法正则应返回错误")
	}
}

func TestTrimChangelog(t *testing.T) {
	if trimChangelog("") != "" {
		t.Fatal("空文本应返回空")
	}
	short := "short log"
	if trimChangelog(short) != short {
		t.Fatal("短文本不应被截断")
	}
	long := strings.Repeat("x", 900) + "\n尾部"
	if got := trimChangelog(long); !strings.Contains(got, "已截断") {
		t.Fatalf("超长文本应被截断, got %q", got)
	}
}

func TestIsRateLimit(t *testing.T) {
	if isRateLimit(errors.New("普通错误")) {
		t.Fatal("普通错误不应判定为限流")
	}
	if !isRateLimit(&github.RateLimitError{}) {
		t.Fatal("RateLimitError 应判定为限流")
	}
}

func TestCurrentInterval(t *testing.T) {
	s := newTestStore(t)
	m := New(s, testLogger())
	if d := m.currentInterval(context.Background()); d != defaultInterval {
		t.Fatalf("默认应为 %v, got %v", defaultInterval, d)
	}
	s.SetSetting(store.KeyPollInterval, "15m")
	if d := m.currentInterval(context.Background()); d != 15*time.Minute {
		t.Fatalf("应读取 15m, got %v", d)
	}
	s.SetSetting(store.KeyPollInterval, "invalid")
	if d := m.currentInterval(context.Background()); d != defaultInterval {
		t.Fatalf("非法值应回退默认, got %v", d)
	}
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

// TestCheckAllShortCtx 覆盖 checkAll 正常路径的前置逻辑与调度循环（ctx 取消后退出）。
func TestCheckAllShortCtx(t *testing.T) {
	s := newTestStore(t)
	srv := releaseServer(t, sampleReleases())
	s.SaveSettings(store.Settings{
		GitHubToken:      "tok",
		GitHubAPIBaseURL: srv.URL + "/",
		NotifyURL:       "logger://",
	})
	addMonitored(t, s, "owner/repo")
	m := New(s, testLogger())
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	// checkAll 在 ticker 触发前会因 ctx 取消返回
	if err := m.CheckAll(ctx); err == nil {
		t.Fatal("短 ctx 下 checkAll 应返回错误")
	}
}

// TestTranslateBody 覆盖更新日志翻译调用（成功与失败分支）。
func TestTranslateBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/translate" {
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": "中文译文"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := newTestStore(t)
	m := New(s, testLogger())
	cfg := store.Settings{
		TranslateEngine:     "dlx",
		TranslateURL:        srv.URL,
		TranslateAPIKey:     "k",
		TranslateModel:      "m",
		TranslateTargetLang: "zh-Hans",
	}
	if got := m.translateBody(context.Background(), "English release notes", cfg); got != "中文译文" {
		t.Fatalf("translateBody 应返回译文, got %q", got)
	}
	// 失败分支：引擎关闭 → 返回空
	if got := m.translateBody(context.Background(), "x", store.Settings{TranslateEngine: "off"}); got != "" {
		t.Fatalf("engine off 应返回空, got %q", got)
	}
}

// TestCheckRepoNoNewWhenSameTag 覆盖「平台版本与最新 tag 相同则跳过」分支。
func TestCheckRepoNoNewWhenSameTag(t *testing.T) {
	s := newTestStore(t)
	id := addMonitored(t, s, "owner/repo")
	// 已存在与最新发布相同的平台基线版本
	_ = s.SetPlatformTag(id, "default", "v2.0.0", time.Now())
	r, _ := s.GetRepoByID(id)
	if err := checkRepoDirect(t, s, r, sampleReleases(), store.Settings{NotifyOnFirstRun: true}); err != nil {
		t.Fatalf("checkRepo 应成功, got %v", err)
	}
	if n, _ := s.CountNotifications(); n != 0 {
		t.Fatalf("版本相同不应通知, got %d", n)
	}
}

// TestCheckAllNotifierError 覆盖 checkAll 中 notifier.New 失败分支。
func TestCheckAllNotifierError(t *testing.T) {
	s := newTestStore(t)
	srv := releaseServer(t, sampleReleases())
	s.SaveSettings(store.Settings{
		GitHubToken:      "tok",
		GitHubAPIBaseURL: srv.URL + "/",
		NotifyURL:       "invalid://scheme",
	})
	addMonitored(t, s, "owner/repo")
	m := New(s, testLogger())
	if err := m.CheckAll(context.Background()); err == nil {
		t.Fatal("notifier.New 失败应返回错误")
	}
}

// TestCheckRepoTranslate 覆盖通知时更新日志翻译分支。
func TestCheckRepoTranslate(t *testing.T) {
	s := newTestStore(t)
	tl := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": "中文译文"})
	}))
	defer tl.Close()
	id := addMonitored(t, s, "owner/repo")
	r, _ := s.GetRepoByID(id)
	err := checkRepoDirect(t, s, r, sampleReleases(), store.Settings{
		NotifyOnFirstRun:     true,
		TranslateEngine:      "dlx",
		TranslateURL:         tl.URL,
		TranslateTargetLang:  "zh-Hans",
	})
	if err != nil {
		t.Fatalf("checkRepo 应成功, got %v", err)
	}
	items, _ := s.ListNotifications(store.NotificationFilter{})
	if len(items) != 1 {
		t.Fatalf("应通知 1 条, got %d", len(items))
	}
	if items[0].ReleaseBodyTranslated != "中文译文" {
		t.Fatalf("应写入译文, got %q", items[0].ReleaseBodyTranslated)
	}
}

// TestCheckRepoEmptyBody 覆盖 notify 中更新日志为空的回退分支。
func TestCheckRepoEmptyBody(t *testing.T) {
	s := newTestStore(t)
	id := addMonitored(t, s, "owner/repo")
	// 发布 body 为空：translateBody 不应被调用，通知仍应生成
	rels := []map[string]any{{"tag_name": "v3.0.0", "name": "v3.0.0", "body": "", "html_url": "https://x", "published_at": "2024-01-01T00:00:00Z", "draft": false, "prerelease": false}}
	r, _ := s.GetRepoByID(id)
	if err := checkRepoDirect(t, s, r, rels, store.Settings{NotifyOnFirstRun: true}); err != nil {
		t.Fatalf("checkRepo 应成功, got %v", err)
	}
	items, _ := s.ListNotifications(store.NotificationFilter{})
	if len(items) != 1 {
		t.Fatalf("空 body 也应通知 1 条, got %d", len(items))
	}
	if items[0].ReleaseBodyTranslated != "" {
		t.Fatalf("空 body 不应有译文, got %q", items[0].ReleaseBodyTranslated)
	}
}

// TestCheckRepoTagsFallback 验证：当仓库无 Release 且开启 track_tags 时，
// checkRepo 应回退到 tag 作为版本来源，并按 commit 时间选出最新 tag 写入缓存与平台基线。
// 该断言基于数据层（不依赖 notifier 实际发送），可在 notify 不可用的测试环境下稳定验证回退逻辑。
func TestCheckRepoTagsFallback(t *testing.T) {
	s := newTestStore(t)
	id := addMonitored(t, s, "owner/repo")
	if err := s.SetRepoTracksTags(id, true); err != nil {
		t.Fatal(err)
	}
	r, _ := s.GetRepoByID(id)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(req.URL.Path, "/releases"):
			_ = json.NewEncoder(w).Encode([]any{}) // 无 Release
		case strings.HasSuffix(req.URL.Path, "/tags"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"name": "v1.5.0", "commit": map[string]any{"sha": "sha111"}},
				{"name": "v1.4.0", "commit": map[string]any{"sha": "sha222"}},
			})
		case strings.Contains(req.URL.Path, "/commits/"):
			sha := strings.TrimPrefix(req.URL.Path, "/repos/owner/repo/commits/")
			date := "2024-05-01T00:00:00Z"
			if sha == "sha222" {
				date = "2024-03-01T00:00:00Z"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sha":    sha,
				"commit": map[string]any{"committer": map[string]any{"date": date}},
			})
		default:
			http.NotFound(w, req)
		}
	}))
	defer srv.Close()

	settings := store.Settings{NotifyOnFirstRun: true}
	if err := s.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	c, err := githubx.NewClient("tok", srv.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	n, _ := notifier.New("logger://")
	m := New(s, testLogger())
	if err := m.checkRepo(context.Background(), c, n, r, settings); err != nil {
		t.Fatalf("checkRepo 应成功, got %v", err)
	}
	r2, _ := s.GetRepoByID(id)
	if r2.LatestTag != "v1.5.0" {
		t.Fatalf("回退后最新 tag 应为 v1.5.0, got %q", r2.LatestTag)
	}
	if r2.LatestReleaseURL != "https://github.com/owner/repo/releases/tag/v1.5.0" {
		t.Fatalf("回退后最新版本链接应为 tag 页面, got %q", r2.LatestReleaseURL)
	}
	tag, found, _ := s.GetPlatformTag(id, "default")
	if !found || tag != "v1.5.0" {
		t.Fatalf("平台基线应为 v1.5.0, got %q found=%v", tag, found)
	}
}

// TestCheckRepoNoFallbackWhenTagsDisabled 验证：未开启 track_tags 时，
// 即使无 Release，也应静默跳过、不写入任何版本缓存（保持既有行为）。
func TestCheckRepoNoFallbackWhenTagsDisabled(t *testing.T) {
	s := newTestStore(t)
	id := addMonitored(t, s, "owner/repo")
	r, _ := s.GetRepoByID(id)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(req.URL.Path, "/releases") {
			_ = json.NewEncoder(w).Encode([]any{})
		} else {
			http.NotFound(w, req)
		}
	}))
	defer srv.Close()

	settings := store.Settings{}
	if err := s.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	c, err := githubx.NewClient("tok", srv.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	n, _ := notifier.New("logger://")
	m := New(s, testLogger())
	if err := m.checkRepo(context.Background(), c, n, r, settings); err != nil {
		t.Fatalf("checkRepo 应成功, got %v", err)
	}
	r2, _ := s.GetRepoByID(id)
	if r2.LatestTag != "" {
		t.Fatalf("未开启 track_tags 时不应写入版本, got %q", r2.LatestTag)
	}
}
