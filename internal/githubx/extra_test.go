package githubx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// pagingMock 支持多页 star 列表与 releases 列表，并对错误路径返回 500。
func pagingMock(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		p := r.URL.Path
		page := r.URL.Query().Get("page")
		switch {
		case p == "/user":
			jsonEnc(w, map[string]any{"login": "octocat"})
		case p == "/user/starred":
			if page == "" || page == "1" {
				w.Header().Set("Link", `<`+"http://"+r.Host+`/user/starred?page=2>; rel="next", `+
					`<`+"http://"+r.Host+`/user/starred?page=3>; rel="last"`)
				jsonEnc(w, []map[string]any{{"repo": repoJSON("owner/a", 1)}})
			} else {
				jsonEnc(w, []map[string]any{{"repo": repoJSON("owner/b", 2)}})
			}
		case strings.HasPrefix(p, "/user/starred/"):
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case strings.HasSuffix(p, "/releases"):
			if page == "" || page == "1" {
				w.Header().Set("Link", `<`+"http://"+r.Host+`/repos/o/r/releases?page=2>; rel="next", `+
					`<`+"http://"+r.Host+`/repos/o/r/releases?page=2>; rel="last"`)
				jsonEnc(w, releaseJSON("v1.0.0"))
			} else {
				jsonEnc(w, releaseJSON("v0.9.0"))
			}
		case strings.Contains(p, "/repos/"):
			jsonEnc(w, repoJSON("owner/repo", 10))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func errorMock(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func jsonEnc(w http.ResponseWriter, v any) {
	_ = encodeJSON(w, v)
}

func encodeJSON(w http.ResponseWriter, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func repoJSON(full string, stars int) map[string]any {
	parts := strings.SplitN(full, "/", 2)
	owner := parts[0]
	name := full
	if len(parts) == 2 {
		name = parts[1]
	}
	return map[string]any{
		"full_name": full, "owner": map[string]any{"login": owner},
		"name": name, "description": "desc", "language": "Go",
		"stargazers_count": stars, "html_url": "https://github.com/" + full,
	}
}

func releaseJSON(tag string) []map[string]any {
	return []map[string]any{{
		"tag_name": tag, "name": tag, "body": "body",
		"html_url":    "https://github.com/o/r/releases/" + tag,
		"published_at": "2024-01-01T00:00:00Z",
		"draft":        false, "prerelease": false,
	}}
}

func TestListStarredReposPagedMulti(t *testing.T) {
	srv := pagingMock(t)
	c, _ := NewClient("tok", srv.URL+"/")
	count := 0
	_, err := c.ListStarredReposPaged(context.Background(), func(page []StarredRepo, pageNum, totalPages int) error {
		count += len(page)
		return nil
	})
	if err != nil {
		t.Fatalf("应成功, got %v", err)
	}
	if count != 2 {
		t.Fatalf("应处理 2 个仓库（两页各 1）, got %d", count)
	}
}

func TestRecentReleasesMultiPage(t *testing.T) {
	srv := pagingMock(t)
	c, _ := NewClient("tok", srv.URL+"/")
	rels, err := c.RecentReleases(context.Background(), "o", "r", 0)
	if err != nil {
		t.Fatalf("应成功, got %v", err)
	}
	if len(rels) != 2 {
		t.Fatalf("应返回 2 页共 2 个版本, got %d", len(rels))
	}
}

func TestRecentReleasesServerError(t *testing.T) {
	srv := errorMock(t)
	c, _ := NewClient("tok", srv.URL+"/")
	_, err := c.RecentReleases(context.Background(), "o", "r", 0)
	if err == nil {
		t.Fatal("500 应返回错误")
	}
	if err == ErrNoRelease {
		t.Fatal("500 不应是 ErrNoRelease")
	}
}

func TestIsStarredServerError(t *testing.T) {
	srv := errorMock(t)
	c, _ := NewClient("tok", srv.URL+"/")
	_, err := c.IsStarred(context.Background(), "owner", "repo")
	if err == nil {
		t.Fatal("500 应返回错误")
	}
}

func TestUnstarNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c, _ := NewClient("tok", srv.URL+"/")
	if err := c.Unstar(context.Background(), "owner", "repo"); err != nil {
		t.Fatalf("404 应视为成功, got %v", err)
	}
}

func TestLatestReleaseMultiPage(t *testing.T) {
	srv := pagingMock(t)
	c, _ := NewClient("tok", srv.URL+"/")
	rel, err := c.LatestRelease(context.Background(), "o", "r")
	if err != nil {
		t.Fatalf("应成功, got %v", err)
	}
	if rel.TagName != "v1.0.0" {
		t.Fatalf("最新应为 v1.0.0, got %q", rel.TagName)
	}
}
