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

// ListStarredReposPaged 分页拉取认证用户 star 的仓库，每页通过 onPage 回调处理。
// onPage 收到该页仓库、当前页码与总页数（第一页响应后确定）；返回错误可提前终止。
// 返回已处理的仓库总数。
func (c *Client) ListStarredReposPaged(ctx context.Context, onPage func(page []StarredRepo, pageNum, totalPages int) error) (int, error) {
	opts := &github.ActivityListStarredOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	totalPages := 1
	processed := 0
	for {
		stars, resp, err := c.gh.Activity.ListStarred(ctx, "", opts)
		if err != nil {
			return processed, classifyErr(err)
		}
		if opts.Page == 1 && resp.LastPage > 0 {
			totalPages = resp.LastPage
		}
		page := make([]StarredRepo, 0, len(stars))
		for _, s := range stars {
			r := s.Repository
			if r == nil || r.GetFullName() == "" {
				continue
			}
			page = append(page, StarredRepo{
				FullName:    r.GetFullName(),
				Owner:       r.GetOwner().GetLogin(),
				Name:        r.GetName(),
				Description: r.GetDescription(),
				Language:    r.GetLanguage(),
				Stargazers:  r.GetStargazersCount(),
				HTMLURL:     r.GetHTMLURL(),
			})
		}
		processed += len(page)
		if err := onPage(page, opts.Page, totalPages); err != nil {
			return processed, err
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return processed, nil
}

// RecentReleases 返回最近 max 个已发布（非 draft、非 prerelease）的版本，按发布时间从新到旧。
// 多平台仓库（如 iOS/mac/CLI 各自发布 tag）需要同时取多个版本以按平台比较。
func (c *Client) RecentReleases(ctx context.Context, owner, repo string, max int) ([]*Release, error) {
	opts := &github.ListOptions{PerPage: 100}
	// 最多翻 5 页，避免极端情况下全是草稿/预发布
	var out []*Release
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
			out = append(out, &Release{
				TagName:     r.GetTagName(),
				Name:        r.GetName(),
				Body:        r.GetBody(),
				HTMLURL:     r.GetHTMLURL(),
				PublishedAt: r.GetPublishedAt().Time,
			})
			if max > 0 && len(out) >= max {
				return out, nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
	}
	if len(out) == 0 {
		return nil, ErrNoRelease
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

// IsStarred 返回认证用户是否已 Star 指定仓库。
func (c *Client) IsStarred(ctx context.Context, owner, repo string) (bool, error) {
	starred, _, err := c.gh.Activity.IsStarred(ctx, owner, repo)
	if err == nil {
		return starred, nil
	}
	var e *github.ErrorResponse
	if errors.As(err, &e) && e.Response != nil && e.Response.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, classifyErr(err)
}

// Unstar 取消对仓库的星标。若仓库本来未星标（GitHub 返回 404），视为成功返回 nil。
func (c *Client) Unstar(ctx context.Context, owner, repo string) error {
	_, err := c.gh.Activity.Unstar(ctx, owner, repo)
	if err == nil {
		return nil
	}
	var e *github.ErrorResponse
	if errors.As(err, &e) && e.Response != nil && e.Response.StatusCode == http.StatusNotFound {
		return nil
	}
	return classifyErr(err)
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
