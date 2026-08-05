package web

import (
	"encoding/xml"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yufei/chaxin/internal/monitor"
	"github.com/yufei/chaxin/internal/store"
)

func newTestServer(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mon := monitor.New(st, logger)
	return NewServer(st, mon, logger), st
}

func TestFeedEmpty(t *testing.T) {
	srv, _ := newTestServer(t)
	req := httptest.NewRequest("GET", "/feed", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/rss+xml") {
		t.Fatalf("content-type = %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<rss version=\"2.0\"") {
		t.Fatalf("缺少 rss 根元素: %s", body)
	}
	if !strings.Contains(body, "察新") {
		t.Fatalf("缺少频道标题")
	}
}

func TestFeedItems(t *testing.T) {
	srv, st := newTestServer(t)
	// 两个被监控仓库，一个有 latest 缓存，一个没有；再加一个未监控的
	st.UpsertRepo(store.Repo{FullName: "a/one", Owner: "a", Name: "one", Stargazers: 1, HTMLURL: "https://github.com/a/one"}, false)
	st.UpsertRepo(store.Repo{FullName: "b/two", Owner: "b", Name: "two", Stargazers: 2, HTMLURL: "https://github.com/b/two"}, false)
	st.UpsertRepo(store.Repo{FullName: "c/unmon", Owner: "c", Name: "unmon", Stargazers: 3, HTMLURL: "https://github.com/c/unmon"}, false)
	repos, _ := st.ListRepos(store.RepoFilter{})
	for _, r := range repos {
		switch r.FullName {
		case "a/one", "b/two":
			if err := st.SetRepoMonitored(r.ID, true); err != nil {
				t.Fatal(err)
			}
		}
	}

	monitored, _ := st.ListRepos(store.RepoFilter{Monitored: boolPtr(true)})
	for _, r := range monitored {
		at := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
		if r.FullName == "b/two" {
			at = time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
		}
		if err := st.SetLatestRelease(r.ID, "v1.0", "https://github.com/"+r.FullName+"/releases/v1.0", "Release "+r.FullName, at); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest("GET", "/feed", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var out struct {
		Channel struct {
			Items []rssItem `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("解析 RSS 失败: %v", err)
	}
	if len(out.Channel.Items) != 2 {
		t.Fatalf("应有 2 个 item（未监控或无缓存的不输出）, got %d", len(out.Channel.Items))
	}
	// 时间较新的 b/two 应排在前
	if out.Channel.Items[0].Title != "b/two 发布 v1.0" {
		t.Fatalf("排序错误, first = %q", out.Channel.Items[0].Title)
	}
}

func TestFeedDescriptionTruncate(t *testing.T) {
	long := strings.Repeat("a", 1500)
	got := feedDescription(long)
	if !strings.Contains(got, "已截断") {
		t.Fatalf("缺少截断提示: %s", got)
	}
	if !strings.Contains(got, "<p>") {
		t.Fatalf("描述应为 markdown 渲染后的 HTML, got: %s", got)
	}
	if feedDescription("") != "暂无更新日志" {
		t.Fatalf("空描述处理错误")
	}
}

func TestFeedDescriptionMarkdownHTML(t *testing.T) {
	got := feedDescription("## 标题\n\n- 甲\n- 乙\n\n**加粗** [链接](https://example.com)")
	for _, want := range []string{"<h2", "<li>甲</li>", "<strong>加粗</strong>", `href="https://example.com"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown 未渲染为 HTML, 缺少 %q, got: %s", want, got)
		}
	}
}

func boolPtr(b bool) *bool {
	return &b
}
