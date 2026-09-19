package githubx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/google/go-github/v89/github"
)

var (
	ErrNoRelease    = errors.New("no published release found")
	ErrNoTag        = errors.New("no tag found")
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

// tagCommitProbeLimit 为确定 tag 顺序而查询 commit 时间的最大 tag 数量。
// 纯 tag 本身不含发布时间，需逐个查询其指向的 commit；该上限用于约束单次检查的 API 调用量。
const tagCommitProbeLimit = 20

// RecentTags 返回仓库最近的 tag（包含未发布 Release 的纯 tag），按指向 commit 的时间从新到旧排序。
// 纯 tag 没有标题、更新日志与独立发布页，故 Name 与 Body 留空，HTMLURL 指向 releases/tag 页面。
// 该接口作为「无 Release 时」的版本来源回退，供开启 track_tags 的仓库使用。
func (c *Client) RecentTags(ctx context.Context, owner, repo string, max int) ([]*Release, error) {
	tags, _, err := c.gh.Repositories.ListTags(ctx, owner, repo, &github.ListOptions{PerPage: 100})
	if err != nil {
		return nil, classifyErr(err)
	}
	if len(tags) == 0 {
		return nil, ErrNoTag
	}
	var out []*Release
	for i, t := range tags {
		name := t.GetName()
		if name == "" {
			continue
		}
		rel := &Release{
			TagName: name,
			HTMLURL: fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", owner, repo, name),
		}
		// 仅对前若干 tag 查询 commit 时间以确定最新顺序，其余 tag 保持列表原始顺序
		if i < tagCommitProbeLimit && t.GetCommit() != nil {
			if cm, _, e := c.gh.Repositories.GetCommit(ctx, owner, repo, t.GetCommit().GetSHA(), nil); e == nil && cm.Commit != nil {
				if d := cm.Commit.Committer.GetDate(); !d.IsZero() {
					rel.PublishedAt = d.Time
				} else if d := cm.Commit.Author.GetDate(); !d.IsZero() {
					rel.PublishedAt = d.Time
				}
			}
		}
		out = append(out, rel)
		if max > 0 && len(out) >= max {
			break
		}
	}
	if len(out) == 0 {
		return nil, ErrNoTag
	}
	// 按 commit 时间倒序排序，无时间的 tag（超出探测上限者）排在末尾
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].PublishedAt.After(out[j].PublishedAt)
	})
	return out, nil
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
