package store

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func repo(full string, stars int) Repo {
	return Repo{
		FullName: full,
		Owner:    "owner",
		Name:     full,
		Stargazers: stars,
	}
}

func TestUpsertRepoThreeStates(t *testing.T) {
	s := newTestStore(t)
	r := repo("a/b", 1)

	res, err := s.UpsertRepo(r, false)
	if err != nil || res != UpsertInserted {
		t.Fatalf("首次插入应为 Inserted, got %v err=%v", res, err)
	}

	// 相同值再次写入 → Skipped
	res, err = s.UpsertRepo(r, false)
	if err != nil || res != UpsertSkipped {
		t.Fatalf("相同值应为 Skipped, got %v err=%v", res, err)
	}

	// 变更字段 → Updated
	r.Stargazers = 99
	res, err = s.UpsertRepo(r, false)
	if err != nil || res != UpsertUpdated {
		t.Fatalf("变更值应为 Updated, got %v err=%v", res, err)
	}
}

func TestUpsertRepoSetsSourceStar(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.UpsertRepo(repo("a/b", 1), false); err != nil {
		t.Fatal(err)
	}
	if err := s.AddRepo(repo("m/n", 1), SourceManual, true); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListRepos(RepoFilter{})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, r := range list {
		got[r.FullName] = r.Source
	}
	if got["a/b"] != SourceStar {
		t.Errorf("同步入库应标记为 star, got %q", got["a/b"])
	}
	if got["m/n"] != SourceManual {
		t.Errorf("手动添加应标记为 manual, got %q", got["m/n"])
	}
}

func TestDeleteStarReposNotIn(t *testing.T) {
	s := newTestStore(t)
	s.UpsertRepo(repo("a/b", 1), false)
	s.UpsertRepo(repo("c/d", 1), false)
	s.AddRepo(repo("m/n", 1), SourceManual, true)

	n, err := s.DeleteStarReposNotIn(map[string]struct{}{"a/b": {}})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("应删除 1 个 star 来源仓库(c/d), got %d", n)
	}
	list, _ := s.ListRepos(RepoFilter{})
	if len(list) != 2 {
		t.Fatalf("应剩 a/b 和 m/n, got %d 个", len(list))
	}
}

// 旧库升级遗留：source 被迁移默认值标为 manual、pinned=0 且未监控，取消 Star 后应被同步清理。
func TestDeleteStarReposNotInLegacyManual(t *testing.T) {
	s := newTestStore(t)
	s.AddRepo(repo("a/b", 1), SourceManual, false)

	n, err := s.DeleteStarReposNotIn(map[string]struct{}{"x/y": {}})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("遗留 manual 仓库应被清理, got %d", n)
	}
	list, _ := s.ListRepos(RepoFilter{})
	if len(list) != 0 {
		t.Fatalf("应清空, got %d 个", len(list))
	}
}

// 手动添加（pinned=1）的仓库保留；监控中的旧数据（pinned=0）取消 Star 后仍应被清理。
func TestDeleteStarReposNotInKeepsPinnedOnly(t *testing.T) {
	s := newTestStore(t)
	s.AddRepo(repo("p/q", 1), SourceManual, true)
	s.AddRepo(repo("m/n", 1), SourceManual, false)
	list, _ := s.ListRepos(RepoFilter{})
	for _, r := range list {
		if r.FullName == "m/n" {
			if err := s.SetRepoMonitored(r.ID, true); err != nil {
				t.Fatal(err)
			}
		}
	}

	n, err := s.DeleteStarReposNotIn(map[string]struct{}{"a/b": {}})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("监控中的遗留仓库也应被清理, got %d", n)
	}
	list, _ = s.ListRepos(RepoFilter{})
	if len(list) != 1 {
		t.Fatalf("应仅保留 p/q, got %d 个", len(list))
	}
	if list[0].FullName != "p/q" {
		t.Fatalf("保留项应为 p/q, got %s", list[0].FullName)
	}
}

func TestIgnorePatternPersist(t *testing.T) {
	s := newTestStore(t)
	r := repo("a/b", 1)
	s.UpsertRepo(r, false)
	list, _ := s.ListRepos(RepoFilter{})
	id := list[0].ID
	if err := s.SetRepoIgnorePattern(id, `^v0\.`); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetRepoByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.IgnorePattern != `^v0\.` {
		t.Fatalf("ignore_pattern 未持久化, got %q", got.IgnorePattern)
	}
}

func TestPruneNotifications(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 5; i++ {
		if err := s.AddNotification(Notification{FullName: "a/b", Tag: "v1", Status: "sent"}); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := s.PruneNotifications(2)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 3 {
		t.Fatalf("应删除 3 条, got %d", removed)
	}
	cnt, _ := s.CountNotifications()
	if cnt != 2 {
		t.Fatalf("应剩 2 条, got %d", cnt)
	}
	// keep<=0 不清理
	if n, _ := s.PruneNotifications(0); n != 0 {
		t.Fatalf("keep=0 不应清理, got %d", n)
	}
}

func TestNotificationFilter(t *testing.T) {
	s := newTestStore(t)
	s.AddNotification(Notification{FullName: "alpha/repo", Tag: "v1.0", Status: "sent"})
	s.AddNotification(Notification{FullName: "beta/repo", Tag: "v2.0", Status: "failed"})

	all, _ := s.ListNotifications(NotificationFilter{})
	if len(all) != 2 {
		t.Fatalf("应返回全部 2 条, got %d", len(all))
	}
	failed, _ := s.ListNotifications(NotificationFilter{Status: "failed"})
	if len(failed) != 1 || failed[0].FullName != "beta/repo" {
		t.Fatalf("状态筛选失败, got %+v", failed)
	}
	byQuery, _ := s.ListNotifications(NotificationFilter{Query: "alpha"})
	if len(byQuery) != 1 || byQuery[0].FullName != "alpha/repo" {
		t.Fatalf("关键词筛选失败, got %+v", byQuery)
	}
	byTag, _ := s.ListNotifications(NotificationFilter{Query: "v2.0"})
	if len(byTag) != 1 || byTag[0].FullName != "beta/repo" {
		t.Fatalf("tag 筛选失败, got %+v", byTag)
	}
}

func TestRestore(t *testing.T) {
	s := newTestStore(t)
	if err := s.SaveSettings(Settings{GitHubToken: "tok", MaxNotifications: 42}); err != nil {
		t.Fatal(err)
	}
	s.UpsertRepo(repo("a/b", 1), false)
	s.UpsertRepo(repo("c/d", 2), false)

	// 恢复为空 + 新设置
	if err := s.Restore(Settings{GitHubToken: "newtok", MaxNotifications: 7}, []Repo{repo("x/y", 5)}); err != nil {
		t.Fatal(err)
	}
	list, _ := s.ListRepos(RepoFilter{})
	if len(list) != 1 || list[0].FullName != "x/y" {
		t.Fatalf("restore 后仓库不符, got %+v", list)
	}
	got, _ := s.GetSettings()
	if got.GitHubToken != "newtok" || got.MaxNotifications != 7 {
		t.Fatalf("restore 后设置不符, got %+v", got)
	}
}

func TestCurrentSettingsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	orig := Settings{
		GitHubToken:         "tok",
		ShoutrrrURL:         "telegram://t",
		PollInterval:        "30m",
		NotifyOnFirstRun:    true,
		MonitorNewStars:     true,
		GitHubAPIBaseURL:    "https://example.com",
		MaxNotifications:    123,
		TranslateEngine:     "dlx",
		TranslateTargetLang: "zh-Hans",
		TranslateURL:        "http://localhost:1188",
		TranslateAPIKey:     "sk-test",
		TranslateModel:      "gpt-4o-mini",
	}
	if err := s.SaveSettings(orig); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if got != orig {
		t.Fatalf("设置读写不一致\ngot  %+v\nwant %+v", got, orig)
	}
}

func TestOpenCreatesDbFile(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	if _, err := s.db.Exec(`SELECT count(*) FROM repos`); err != nil {
		t.Fatalf("repos 表不存在: %v", err)
	}
	// 迁移后的列应存在
	rows, err := s.db.Query(`PRAGMA table_info(repos)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatal(err)
		}
		cols[name] = true
	}
	for _, want := range []string{"source", "ignore_pattern", "latest_tag", "latest_release_url", "latest_release_body", "latest_release_at"} {
		if !cols[want] {
			t.Errorf("迁移后缺少列 %s", want)
		}
	}
	// 通知表应包含译文列
	nrows, err := s.db.Query(`PRAGMA table_info(notifications)`)
	if err != nil {
		t.Fatal(err)
	}
	defer nrows.Close()
	hasTranslated := false
	for nrows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt any
		if err := nrows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatal(err)
		}
		if name == "release_body_translated" {
			hasTranslated = true
		}
	}
	if !hasTranslated {
		t.Error("迁移后缺少列 release_body_translated")
	}
	if filepath.Join(dir, "chaxin.db") == "" {
		t.Fatal("unreachable")
	}
}

func TestNotificationTranslatedBodyRoundTrip(t *testing.T) {
	s := newTestStore(t)
	n := Notification{
		FullName:              "a/b",
		Tag:                   "v1.0",
		ReleaseBody:           "This is English",
		ReleaseBodyTranslated: "这是中文译文",
		Status:                "sent",
	}
	if err := s.AddNotification(n); err != nil {
		t.Fatal(err)
	}
	items, err := s.ListNotifications(NotificationFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("应返回 1 条, got %d", len(items))
	}
	if items[0].ReleaseBodyTranslated != "这是中文译文" {
		t.Fatalf("译文未持久化, got %q", items[0].ReleaseBodyTranslated)
	}
	if items[0].ReleaseBody != "This is English" {
		t.Fatalf("原文未持久化, got %q", items[0].ReleaseBody)
	}
}

func TestLatestReleaseRoundTrip(t *testing.T) {
	s := newTestStore(t)
	s.UpsertRepo(repo("a/b", 1), false)
	list, _ := s.ListRepos(RepoFilter{})
	id := list[0].ID

	at := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	if err := s.SetLatestRelease(id, "v1.2.3", "https://github.com/a/b/releases/v1.2.3", "Release body", at); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetRepoByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.LatestTag != "v1.2.3" || got.LatestReleaseURL != "https://github.com/a/b/releases/v1.2.3" {
		t.Fatalf("latest release 未持久化, got %+v", got)
	}
	if got.LatestReleaseBody != "Release body" {
		t.Fatalf("latest body 未持久化, got %q", got.LatestReleaseBody)
	}
	if !got.LatestReleaseAt.Equal(at) {
		t.Fatalf("latest at 未持久化, got %v want %v", got.LatestReleaseAt, at)
	}
}
