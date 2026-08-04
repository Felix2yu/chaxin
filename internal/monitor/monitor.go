package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v89/github"
	"github.com/yufei/chaxin/internal/githubx"
	"github.com/yufei/chaxin/internal/notifier"
	"github.com/yufei/chaxin/internal/store"
	"github.com/yufei/chaxin/internal/translate"
)

const (
	defaultInterval = 30 * time.Minute
	timeFormat      = "2006-01-02 15:04:05"
)

type Monitor struct {
	store  *store.Store
	logger *slog.Logger
	mu     sync.Mutex
}

func New(st *store.Store, logger *slog.Logger) *Monitor {
	return &Monitor{store: st, logger: logger}
}

// Run 循环执行检查，每次循环前重新读取配置（轮询间隔与凭据变更即时生效）。
func (m *Monitor) Run(ctx context.Context) {
	m.logger.Info("监控调度器启动")
	for {
		m.CheckAll(ctx)
		interval := m.currentInterval(ctx)
		m.logger.Debug("下一轮检查", "interval", interval)
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

// CheckAll 执行一轮全量检查。可被定时任务或 Web API 手动触发。
func (m *Monitor) CheckAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.checkAll(ctx)
}

// maxConcurrent 单轮检查的最大并发请求数。
const maxConcurrent = 8

func (m *Monitor) checkAll(ctx context.Context) error {
	settings, err := m.store.GetSettings()
	if err != nil {
		m.logger.Error("读取配置失败", "err", err)
		return err
	}
	if settings.GitHubToken == "" {
		m.logger.Warn("未配置 GitHub token，跳过本轮监控")
		return nil
	}

	client, err := githubx.NewClient(settings.GitHubToken, settings.GitHubAPIBaseURL)
	if err != nil {
		m.logger.Error("创建 GitHub 客户端失败", "err", err)
		return err
	}
	notif, err := notifier.New(settings.ShoutrrrURL)
	if err != nil {
		m.logger.Error("创建通知器失败", "err", err)
		return err
	}

	repos, err := m.store.ListMonitoredRepos()
	if err != nil {
		m.logger.Error("读取监控仓库失败", "err", err)
		return err
	}
	if len(repos) == 0 {
		return nil
	}

	// 检查结束后按保留条数清理旧通知
	defer func() {
		if removed, err := m.store.PruneNotifications(settings.MaxNotifications); err != nil {
			m.logger.Error("清理通知记录失败", "err", err)
		} else if removed > 0 {
			m.logger.Info("已清理旧通知", "removed", removed)
		}
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, maxConcurrent)
	errCh := make(chan error, len(repos))
	var wg sync.WaitGroup

	for _, r := range repos {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		default:
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(repo store.Repo) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := m.checkRepo(ctx, client, notif, repo, settings); err != nil {
				// rate limit 命中：取消剩余任务
				errCh <- err
				cancel()
			}
		}(r)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if isRateLimit(err) {
			m.logger.Error("GitHub API 限流，本轮剩余仓库已跳过", "err", err)
			return err
		}
	}
	return nil
}

// isRateLimit 判断错误是否为 GitHub API 限流。
func isRateLimit(err error) bool {
	var e *github.RateLimitError
	return errors.As(err, &e)
}

// matchesIgnorePattern 判定 tag 是否命中忽略正则；pattern 为空返回 (false, nil)。
func matchesIgnorePattern(pattern, tag string) (bool, error) {
	if pattern == "" {
		return false, nil
	}
	return regexp.MatchString(pattern, tag)
}

func (m *Monitor) checkRepo(ctx context.Context, client *githubx.Client, notif *notifier.Notifier, repo store.Repo, settings store.Settings) error {
	releases, err := client.RecentReleases(ctx, repo.Owner, repo.Name, recentReleaseLimit)
	if err != nil {
		if errors.Is(err, githubx.ErrNoRelease) {
			_ = m.store.TouchCheckedAt(repo.ID)
			return nil
		}
		if isRateLimit(err) {
			return err
		}
		m.logger.Error("获取版本失败", "repo", repo.FullName, "err", err)
		return nil
	}

	// 清洗更新日志并过滤空 tag
	clean := make([]*githubx.Release, 0, len(releases))
	for _, rel := range releases {
		if rel.TagName == "" {
			continue
		}
		rel.Body = translate.Clean(rel.Body)
		clean = append(clean, rel)
	}
	if len(clean) == 0 {
		_ = m.store.TouchCheckedAt(repo.ID)
		return nil
	}
	latest := clean[0] // 发布时间最新的版本

	// 更新最新版本缓存（RSS 订阅数据源）与前端展示用 tag，无论是否通知
	if err := m.store.SetLatestRelease(repo.ID, latest.TagName, latest.HTMLURL, latest.Body, latest.PublishedAt); err != nil {
		m.logger.Warn("更新最新版本缓存失败", "repo", repo.FullName, "err", err)
	}
	if err := m.store.SetLastKnownTag(repo.ID, latest.TagName); err != nil {
		m.logger.Warn("更新最新 tag 失败", "repo", repo.FullName, "err", err)
	}

	// 忽略匹配正则的版本（命中最新 tag 则跳过整个仓库）
	if matched, err := matchesIgnorePattern(repo.IgnorePattern, latest.TagName); err != nil {
		m.logger.Warn("忽略版本正则无效", "repo", repo.FullName, "pattern", repo.IgnorePattern, "err", err)
	} else if matched {
		_ = m.store.TouchCheckedAt(repo.ID)
		m.logger.Debug("命中忽略规则，跳过", "repo", repo.FullName, "tag", latest.TagName, "pattern", repo.IgnorePattern)
		return nil
	}

	// 按平台独立比较版本，不同平台的 tag 不再互相触发
	handled := map[string]bool{}
	for _, rel := range clean {
		p := parsePlatform(rel.TagName)
		if handled[p] {
			continue
		}
		handled[p] = true

		last, found, err := m.store.GetPlatformTag(repo.ID, p)
		if err != nil {
			m.logger.Warn("读取平台版本失败", "repo", repo.FullName, "platform", p, "err", err)
			continue
		}
		if !found {
			// 仓库首次出现任何平台记录时，可选通知既有最新版；后续新平台仅建立基线
			hasAny, _ := m.store.RepoHasPlatformRecord(repo.ID)
			if !hasAny && settings.NotifyOnFirstRun && notif != nil {
				m.notify(ctx, notif, repo, "", rel, settings)
			}
			if err := m.store.SetPlatformTag(repo.ID, p, rel.TagName, rel.PublishedAt); err != nil {
				m.logger.Warn("建立平台基线失败", "repo", repo.FullName, "platform", p, "err", err)
			}
			m.logger.Debug("已建立平台基线", "repo", repo.FullName, "platform", p, "tag", rel.TagName)
			continue
		}
		if last == rel.TagName {
			continue
		}

		// 同平台检测到新版本
		if notif != nil {
			m.notify(ctx, notif, repo, last, rel, settings)
		}
		if err := m.store.SetPlatformTag(repo.ID, p, rel.TagName, rel.PublishedAt); err != nil {
			m.logger.Warn("更新平台版本失败", "repo", repo.FullName, "platform", p, "err", err)
		}
		m.logger.Info("检测到新版本", "repo", repo.FullName, "platform", p, "from", last, "to", rel.TagName)
	}
	_ = m.store.TouchCheckedAt(repo.ID)
	return nil
}

// recentReleaseLimit 单次检查拉取的版本数量上限。多平台仓库按发布时间取最新一批，
// 100 个已足以覆盖各平台的最新版本。
const recentReleaseLimit = 100

// knownPlatforms 已识别的版本 tag 平台前缀词表。
var knownPlatforms = []string{
	"ios", "iphone", "ipad", "mac", "macos", "osx",
	"cli", "desktop", "android", "windows", "win", "linux",
	"server", "web", "wasm", "sdk", "api", "freebsd",
}

// parsePlatform 从 tag 中提取平台前缀。仅当 tag 形如 "<平台>-<版本>" 且前缀命中已知
// 词表时返回该平台（小写），否则返回 "default"。例如 "iOS-7.1.2-7112" -> "ios"，
// "v1.2.3" -> "default"。
func parsePlatform(tag string) string {
	i := 0
	for i < len(tag) && isASCIIAlpha(tag[i]) {
		i++
	}
	if i > 0 && i < len(tag) && tag[i] == '-' {
		p := strings.ToLower(tag[:i])
		for _, kp := range knownPlatforms {
			if p == kp {
				return kp
			}
		}
	}
	return "default"
}

func isASCIIAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func (m *Monitor) notify(ctx context.Context, notif *notifier.Notifier, repo store.Repo, from string, rel *githubx.Release, settings store.Settings) {
	title := fmt.Sprintf("%s 发布新版本 %s", repo.FullName, rel.TagName)
	msg := fmt.Sprintf("**仓库**: %s\n**版本**: %s → %s\n**发布时间**: %s",
		repo.FullName, from, rel.TagName, rel.PublishedAt.Format(timeFormat))

	// 更新日志：检测语言，必要时翻译
	translatedBody := ""
	if body := strings.TrimSpace(rel.Body); body != "" {
		translatedBody = m.translateBody(ctx, rel.Body, settings)
	}
	if translatedBody != "" {
		if body := trimChangelog(translatedBody); body != "" {
			msg += fmt.Sprintf("\n\n## 更新日志\n%s", body)
		}
	} else if body := trimChangelog(rel.Body); body != "" {
		msg += fmt.Sprintf("\n\n## 更新日志\n%s", body)
	}
	if rel.HTMLURL != "" {
		msg += fmt.Sprintf("\n\n**链接**: [%s](%s)", rel.HTMLURL, rel.HTMLURL)
	}

	record := store.Notification{
		RepoID:                repo.ID,
		FullName:              repo.FullName,
		Tag:                   rel.TagName,
		ReleaseURL:            rel.HTMLURL,
		ReleaseBody:           rel.Body,
		ReleaseBodyTranslated: translatedBody,
		ReleasedAt:            rel.PublishedAt,
		Status:                "sent",
	}

	var sendErr error
	for attempt, delay := range retryDelays {
		if attempt > 0 {
			m.logger.Warn("发送通知失败，准备重试", "repo", repo.FullName, "attempt", attempt, "delay", delay)
			time.Sleep(delay)
		}
		if sendErr = notif.Send(title, msg); sendErr == nil {
			break
		}
	}
	if sendErr != nil {
		m.logger.Error("发送通知失败", "repo", repo.FullName, "err", sendErr)
		record.Status = "failed"
		record.Error = sendErr.Error()
	}
	if err := m.store.AddNotification(record); err != nil {
		m.logger.Error("记录通知失败", "err", err)
	}
}

// translateBody 检测更新日志语言：已是目标语言则直接提取/原样返回，否则翻译。
// 失败时返回空字符串（调用方回退使用原文）。
func (m *Monitor) translateBody(ctx context.Context, body string, settings store.Settings) string {
	if settings.TranslateEngine == "" || settings.TranslateEngine == "off" {
		return ""
	}
	cfg := translate.Config{
		Engine: settings.TranslateEngine,
		URL:    settings.TranslateURL,
		APIKey: settings.TranslateAPIKey,
		Model:  settings.TranslateModel,
		Target: settings.TranslateTargetLang,
	}
	tctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	res, err := translate.Prepare(tctx, cfg, body)
	if err != nil {
		m.logger.Warn("更新日志翻译失败，使用原文", "engine", settings.TranslateEngine, "err", err)
		return ""
	}
	if res.Text == "" || res.Text == body {
		return ""
	}
	return res.Text
}

// retryDelays 发送通知失败时的退避重试间隔（最多重试 len-1 次）。
var retryDelays = []time.Duration{0, 2 * time.Second, 8 * time.Second, 30 * time.Second}

// maxChangelogLen 通知消息中更新日志的最大字符数。
const maxChangelogLen = 800

// trimChangelog 截断更新日志，超长时在最近的换行处截断并追加提示。
func trimChangelog(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxChangelogLen {
		return s
	}
	cut := runes[:maxChangelogLen]
	// 尝试在末尾换行处截断，避免切断一行（用 rune 索引避免中文错位）
	lastNL := -1
	for i := len(cut) - 1; i > maxChangelogLen/2; i-- {
		if cut[i] == '\n' {
			lastNL = i
			break
		}
	}
	if lastNL > 0 {
		cut = cut[:lastNL]
	}
	return string(cut) + "\n…（日志过长已截断，完整内容见链接）"
}

func (m *Monitor) currentInterval(ctx context.Context) time.Duration {
	s, err := m.store.GetSetting(store.KeyPollInterval)
	if err != nil || s == "" {
		return defaultInterval
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < time.Minute {
		return defaultInterval
	}
	return d
}
