package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/yufei/chaxin/internal/githubx"
	"github.com/yufei/chaxin/internal/notifier"
	"github.com/yufei/chaxin/internal/store"
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

	for _, r := range repos {
		m.checkRepo(ctx, client, notif, r, settings.NotifyOnFirstRun)
	}
	return nil
}

func (m *Monitor) checkRepo(ctx context.Context, client *githubx.Client, notif *notifier.Notifier, repo store.Repo, notifyOnFirstRun bool) {
	rel, err := client.LatestRelease(ctx, repo.Owner, repo.Name)
	if err != nil {
		if errors.Is(err, githubx.ErrNoRelease) {
			_ = m.store.TouchCheckedAt(repo.ID)
			return
		}
		m.logger.Error("获取版本失败", "repo", repo.FullName, "err", err)
		return
	}
	if rel.TagName == "" {
		_ = m.store.TouchCheckedAt(repo.ID)
		return
	}

	if repo.LastKnownTag == "" {
		// 首次建立基线：可选是否通知既有最新版
		if notifyOnFirstRun && notif != nil {
			m.notify(notif, repo, repo.LastKnownTag, rel)
		}
		if err := m.store.SetLastKnownTag(repo.ID, rel.TagName); err != nil {
			m.logger.Error("更新基线版本失败", "repo", repo.FullName, "err", err)
		}
		m.logger.Info("已建立监控基线", "repo", repo.FullName, "tag", rel.TagName)
		return
	}

	if repo.LastKnownTag == rel.TagName {
		_ = m.store.TouchCheckedAt(repo.ID)
		return
	}

	// 检测到新版本
	if notif != nil {
		m.notify(notif, repo, repo.LastKnownTag, rel)
	}
	if err := m.store.SetLastKnownTag(repo.ID, rel.TagName); err != nil {
		m.logger.Error("更新版本失败", "repo", repo.FullName, "err", err)
	}
	m.logger.Info("检测到新版本", "repo", repo.FullName, "from", repo.LastKnownTag, "to", rel.TagName)
}

func (m *Monitor) notify(notif *notifier.Notifier, repo store.Repo, from string, rel *githubx.Release) {
	title := fmt.Sprintf("%s 发布新版本 %s", repo.FullName, rel.TagName)
	msg := fmt.Sprintf("仓库: %s\n版本: %s -> %s\n发布时间: %s",
		repo.FullName, from, rel.TagName, rel.PublishedAt.Format(timeFormat))
	if body := trimChangelog(rel.Body); body != "" {
		msg += fmt.Sprintf("\n\n更新日志:\n%s", body)
	}
	msg += fmt.Sprintf("\n链接: %s", rel.HTMLURL)

	record := store.Notification{
		RepoID:      repo.ID,
		FullName:    repo.FullName,
		Tag:         rel.TagName,
		ReleaseURL:  rel.HTMLURL,
		ReleaseBody: rel.Body,
		ReleasedAt:  rel.PublishedAt,
		Status:      "sent",
	}
	if err := notif.Send(title, msg); err != nil {
		m.logger.Error("发送通知失败", "repo", repo.FullName, "err", err)
		record.Status = "failed"
		record.Error = err.Error()
	}
	if err := m.store.AddNotification(record); err != nil {
		m.logger.Error("记录通知失败", "err", err)
	}
}

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
	// 尝试在末尾换行处截断，避免切断一行
	lastNL := strings.LastIndex(string(cut), "\n")
	if lastNL > maxChangelogLen/2 {
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
