package githubx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v89/github"
)

var (
	ErrNoRelease    = errors.New("no published release found")
	ErrUnauthorized = errors.New("github authentication failed: check token")
)

type StarredRepo struct {
	FullName    string `json:"full_name"`
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Stargazers  int    `json:"stargazers_count"`
	HTMLURL     string `json:"html_url"`
}

type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
}

type Client struct {
	gh *github.Client
}

func NewClient(token, apiBaseURL string) (*Client, error) {
	opts := []github.ClientOptionsFunc{
		github.WithTimeout(30 * time.Second),
		github.WithUserAgent("chaxin"),
		github.WithEnvProxy(),
	}
	if token != "" {
		opts = append(opts, github.WithAuthToken(token))
	}
	if apiBaseURL != "" {
		opts = append(opts, github.WithURLs(&apiBaseURL, nil))
	}
	gh, err := github.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("初始化 GitHub 客户端失败: %w", err)
	}
	return &Client{gh: gh}, nil
}

// VerifyToken 校验 token 有效性并返回认证用户名。
func (c *Client) VerifyToken(ctx context.Context) (string, error) {
	u, _, err := c.gh.Users.Get(ctx, "")
	if err != nil {
		return "", classifyErr(err)
	}
	return u.GetLogin(), nil
}

// ListStarredRepos 拉取认证用户 star 的全部仓库（自动分页）。
func (c *Client) ListStarredRepos(ctx context.Context) ([]StarredRepo, error) {
	opts := &github.ActivityListStarredOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	var out []StarredRepo
	for {
		stars, resp, err := c.gh.Activity.ListStarred(ctx, "", opts)
		if err != nil {
			return nil, classifyErr(err)
		}
		for _, s := range stars {
			r := s.Repository
			if r == nil || r.GetFullName() == "" {
				continue
			}
			out = append(out, StarredRepo{
				FullName:    r.GetFullName(),
				Owner:       r.GetOwner().GetLogin(),
				Name:        r.GetName(),
				Description: r.GetDescription(),
				Language:    r.GetLanguage(),
				Stargazers:  r.GetStargazersCount(),
				HTMLURL:     r.GetHTMLURL(),
			})
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return out, nil
}

// LatestRelease 返回最新已发布（非 draft、非 prerelease）的版本。
func (c *Client) LatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	opts := &github.ListOptions{PerPage: 100}
	// 最多翻 5 页，避免极端情况下全是草稿/预发布
	for page := 1; page <= 5; page++ {
		opts.Page = page
		releases, resp, err := c.gh.Repositories.ListReleases(ctx, owner, repo, opts)
		if err != nil {
			return nil, classifyErr(err)
		}
		for _, r := range releases {
			if r.GetDraft() || r.GetPrerelease() {
				continue
			}
			return &Release{
				TagName:     r.GetTagName(),
				Name:        r.GetName(),
				Body:        r.GetBody(),
				HTMLURL:     r.GetHTMLURL(),
				PublishedAt: r.GetPublishedAt().Time,
			}, nil
		}
		if resp.NextPage == 0 {
			break
		}
	}
	return nil, ErrNoRelease
}

// RepoInfo 获取单个仓库信息（手动添加时校验并补全信息）。
func (c *Client) RepoInfo(ctx context.Context, owner, repo string) (*StarredRepo, error) {
	r, _, err := c.gh.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, classifyErr(err)
	}
	return &StarredRepo{
		FullName:    r.GetFullName(),
		Owner:       r.GetOwner().GetLogin(),
		Name:        r.GetName(),
		Description: r.GetDescription(),
		Language:    r.GetLanguage(),
		Stargazers:  r.GetStargazersCount(),
		HTMLURL:     r.GetHTMLURL(),
	}, nil
}

func classifyErr(err error) error {
	var e *github.ErrorResponse
	if errors.As(err, &e) {
		if e.Response != nil && e.Response.StatusCode == http.StatusUnauthorized {
			return ErrUnauthorized
		}
	}
	return err
}
