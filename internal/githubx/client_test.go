package githubx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockGitHub 返回一个模拟 GitHub REST API 的测试服务器。
// 约定：token 含 "bad" 时 /user 返回 401；starred 检查中 owner 为 "unstarred" 返回 404；
// releases 查询中 owner 为 "norelease" 返回空列表。
func mockGitHub(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		p := r.URL.Path
		switch {
		case p == "/user":
			if strings.Contains(r.Header.Get("Authorization"), "bad") {
				http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}
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
			rest := strings.TrimPrefix(p, "/user/starred/")
			owner := strings.Split(rest, "/")[0]
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent) // Unstar 成功
				return
			}
			// IsStarred
			if owner == "unstarred" {
				http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case strings.HasSuffix(p, "/releases"):
			if strings.HasPrefix(p, "/repos/norelease/") {
				_ = json.NewEncoder(w).Encode([]any{})
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"tag_name": "v2.0.0", "name": "v2.0.0", "body": "release body",
					"html_url":    "https://github.com/o/r/releases/v2.0.0",
					"published_at": "2024-01-02T00:00:00Z",
					"draft":       false, "prerelease": false,
				},
				{
					"tag_name": "v1.0.0", "name": "v1.0.0", "body": "old",
					"html_url":    "https://github.com/o/r/releases/v1.0.0",
					"published_at": "2024-01-01T00:00:00Z",
					"draft":       false, "prerelease": false,
				},
			})
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

func TestVerifyToken(t *testing.T) {
	srv := mockGitHub(t)
	c, err := NewClient("good-token", srv.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	login, err := c.VerifyToken(context.Background())
	if err != nil {
		t.Fatalf("VerifyToken 应成功, got err=%v", err)
	}
	if login != "octocat" {
		t.Fatalf("login 应为 octocat, got %q", login)
	}
}

func TestVerifyTokenUnauthorized(t *testing.T) {
	srv := mockGitHub(t)
	c, err := NewClient("bad-token", srv.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.VerifyToken(context.Background())
	if err == nil {
		t.Fatal("bad token 应返回错误")
	}
	if !strings.Contains(err.Error(), "authentication") {
		t.Fatalf("错误应提示认证失败, got %v", err)
	}
}

func TestListStarredReposPaged(t *testing.T) {
	srv := mockGitHub(t)
	c, _ := NewClient("tok", srv.URL+"/")
	total := 0
	_, err := c.ListStarredReposPaged(context.Background(), func(page []StarredRepo, pageNum, totalPages int) error {
		total += len(page)
		if len(page) == 1 && page[0].FullName != "owner/repo" {
			t.Fatalf("仓库名不符, got %q", page[0].FullName)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("应成功, got err=%v", err)
	}
	if total != 1 {
		t.Fatalf("应处理 1 个仓库, got %d", total)
	}
}

func TestRecentReleases(t *testing.T) {
	srv := mockGitHub(t)
	c, _ := NewClient("tok", srv.URL+"/")
	rels, err := c.RecentReleases(context.Background(), "owner", "repo", 0)
	if err != nil {
		t.Fatalf("应成功, got err=%v", err)
	}
	if len(rels) != 2 {
		t.Fatalf("应返回 2 个版本, got %d", len(rels))
	}
	if rels[0].TagName != "v2.0.0" {
		t.Fatalf("首个应为 v2.0.0, got %q", rels[0].TagName)
	}
}

func TestRecentReleasesNone(t *testing.T) {
	srv := mockGitHub(t)
	c, _ := NewClient("tok", srv.URL+"/")
	_, err := c.RecentReleases(context.Background(), "norelease", "repo", 0)
	if err != ErrNoRelease {
		t.Fatalf("空仓库应返回 ErrNoRelease, got %v", err)
	}
}

func TestLatestRelease(t *testing.T) {
	srv := mockGitHub(t)
	c, _ := NewClient("tok", srv.URL+"/")
	rel, err := c.LatestRelease(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("应成功, got err=%v", err)
	}
	if rel.TagName != "v2.0.0" {
		t.Fatalf("最新应为 v2.0.0, got %q", rel.TagName)
	}
}

func TestLatestReleaseNone(t *testing.T) {
	srv := mockGitHub(t)
	c, _ := NewClient("tok", srv.URL+"/")
	_, err := c.LatestRelease(context.Background(), "norelease", "repo")
	if err != ErrNoRelease {
		t.Fatalf("空仓库应返回 ErrNoRelease, got %v", err)
	}
}

func TestRepoInfo(t *testing.T) {
	srv := mockGitHub(t)
	c, _ := NewClient("tok", srv.URL+"/")
	info, err := c.RepoInfo(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("应成功, got err=%v", err)
	}
	if info.FullName != "owner/repo" || info.Stargazers != 10 {
		t.Fatalf("RepoInfo 字段不符, got %+v", info)
	}
}

func TestIsStarred(t *testing.T) {
	srv := mockGitHub(t)
	c, _ := NewClient("tok", srv.URL+"/")
	starred, err := c.IsStarred(context.Background(), "owner", "repo")
	if err != nil || !starred {
		t.Fatalf("应已 star, got starred=%v err=%v", starred, err)
	}
	notStarred, err := c.IsStarred(context.Background(), "unstarred", "repo")
	if err != nil || notStarred {
		t.Fatalf("应未 star, got starred=%v err=%v", notStarred, err)
	}
}

func TestUnstar(t *testing.T) {
	srv := mockGitHub(t)
	c, _ := NewClient("tok", srv.URL+"/")
	if err := c.Unstar(context.Background(), "owner", "repo"); err != nil {
		t.Fatalf("Unstar 应成功, got err=%v", err)
	}
}

func TestNewClientError(t *testing.T) {
	// 非法 baseURL 应导致 NewClient 失败
	if _, err := NewClient("tok", "://not a url"); err == nil {
		t.Fatal("非法 apiBaseURL 应返回错误")
	}
}
