package store

import (
	"testing"
	"time"
)

func TestRepoExists(t *testing.T) {
	s := newTestStore(t)
	if ok, err := s.RepoExists("a/b"); err != nil || ok {
		t.Fatalf("空库应不存在, got ok=%v err=%v", ok, err)
	}
	s.AddRepo(repo("a/b", 1), SourceManual, true)
	if ok, err := s.RepoExists("a/b"); err != nil || !ok {
		t.Fatalf("已添加应存在, got ok=%v err=%v", ok, err)
	}
}

func TestListMonitoredRepos(t *testing.T) {
	s := newTestStore(t)
	if rs, err := s.ListMonitoredRepos(); err != nil || len(rs) != 0 {
		t.Fatalf("应无监控仓库, got %d err=%v", len(rs), err)
	}
	s.AddRepo(repo("a/b", 1), SourceManual, true)
	s.AddRepo(repo("c/d", 2), SourceManual, false)
	id := mustID(t, s, "a/b")
	s.SetRepoMonitored(id, true)
	list, err := s.ListMonitoredRepos()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].FullName != "a/b" {
		t.Fatalf("监控列表不符, got %+v", list)
	}
}

func TestSetAndCountRepos(t *testing.T) {
	s := newTestStore(t)
	s.AddRepo(repo("a/b", 1), SourceManual, true)
	s.AddRepo(repo("c/d", 2), SourceManual, false)
	s.SetRepoMonitored(mustID(t, s, "a/b"), true)
	if n, err := s.CountRepos(); err != nil || n != 2 {
		t.Fatalf("CountRepos 应返回 2, got %d err=%v", n, err)
	}
	if n, err := s.CountMonitored(); err != nil || n != 1 {
		t.Fatalf("CountMonitored 应返回 1, got %d err=%v", n, err)
	}
}

func TestSetRepoMonitored(t *testing.T) {
	s := newTestStore(t)
	s.AddRepo(repo("a/b", 1), SourceManual, false)
	id := mustID(t, s, "a/b")
	if err := s.SetRepoMonitored(id, true); err != nil {
		t.Fatal(err)
	}
	r, _ := s.GetRepoByID(id)
	if !r.Monitored {
		t.Fatal("应已设为监控")
	}
}

func TestSetReposMonitored(t *testing.T) {
	s := newTestStore(t)
	s.AddRepo(repo("a/b", 1), SourceManual, false)
	s.AddRepo(repo("c/d", 1), SourceManual, false)
	ids := []int64{mustID(t, s, "a/b"), mustID(t, s, "c/d")}
	n, err := s.SetReposMonitored(ids, true)
	if err != nil || n != 2 {
		t.Fatalf("应更新 2 个, got %d err=%v", n, err)
	}
	if m, _ := s.CountMonitored(); m != 2 {
		t.Fatalf("应 2 个监控, got %d", m)
	}
	if _, err := s.SetReposMonitored(nil, true); err != nil {
		t.Fatalf("空 ids 不应报错, got %v", err)
	}
}

func TestSetLastKnownTagAndTouch(t *testing.T) {
	s := newTestStore(t)
	s.AddRepo(repo("a/b", 1), SourceManual, true)
	id := mustID(t, s, "a/b")
	if err := s.SetLastKnownTag(id, "v1.0"); err != nil {
		t.Fatal(err)
	}
	r, _ := s.GetRepoByID(id)
	if r.LastKnownTag != "v1.0" {
		t.Fatalf("last_known_tag 不符, got %q", r.LastKnownTag)
	}
	if err := s.TouchCheckedAt(id); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteRepo(t *testing.T) {
	s := newTestStore(t)
	s.AddRepo(repo("a/b", 1), SourceManual, true)
	id := mustID(t, s, "a/b")
	if err := s.DeleteRepo(id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetRepoByID(id); err == nil {
		t.Fatal("删除后应查不到")
	}
}

func TestGetSetSetting(t *testing.T) {
	s := newTestStore(t)
	if v, err := s.GetSetting("missing"); err != nil || v != "" {
		t.Fatalf("缺失键应返回空, got %q err=%v", v, err)
	}
	if err := s.SetSetting(KeyPollInterval, "15m"); err != nil {
		t.Fatal(err)
	}
	if v, err := s.GetSetting(KeyPollInterval); err != nil || v != "15m" {
		t.Fatalf("应返回 15m, got %q err=%v", v, err)
	}
}

func TestMarkNotificationSent(t *testing.T) {
	s := newTestStore(t)
	n := Notification{FullName: "a/b", Tag: "v1", Status: "failed"}
	s.AddNotification(n)
	id := mustNotifID(t, s, "a/b")
	if err := s.MarkNotificationSent(id); err != nil {
		t.Fatal(err)
	}
	items, _ := s.ListNotifications(NotificationFilter{})
	if items[0].Status != "sent" || items[0].Error != "" {
		t.Fatalf("应标记为已发送且清空错误, got %+v", items[0])
	}
}

func TestPlatforms(t *testing.T) {
	s := newTestStore(t)
	s.AddRepo(repo("a/b", 1), SourceManual, true)
	id := mustID(t, s, "a/b")

	has, err := s.RepoHasPlatformRecord(id)
	if err != nil || has {
		t.Fatalf("初始应无平台记录, got %v err=%v", has, err)
	}
	if tag, found, err := s.GetPlatformTag(id, "default"); err != nil || found || tag != "" {
		t.Fatalf("未记录时应 found=false, got tag=%q found=%v err=%v", tag, found, err)
	}
	if err := s.SetPlatformTag(id, "default", "v1.0", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if has, _ = s.RepoHasPlatformRecord(id); !has {
		t.Fatal("写入后应有平台记录")
	}
	tag, found, err := s.GetPlatformTag(id, "default")
	if err != nil || !found || tag != "v1.0" {
		t.Fatalf("应读到 v1.0, got tag=%q found=%v err=%v", tag, found, err)
	}
	if err := s.DeleteRepoPlatforms(id); err != nil {
		t.Fatal(err)
	}
	if has, _ = s.RepoHasPlatformRecord(id); has {
		t.Fatal("删除后应无平台记录")
	}
}

func TestRepoString(t *testing.T) {
	r := Repo{FullName: "owner/repo"}
	if r.String() != "owner/repo" {
		t.Fatalf("String 应返回 full_name, got %q", r.String())
	}
}

func mustID(t *testing.T, s *Store, full string) int64 {
	t.Helper()
	list, err := s.ListRepos(RepoFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range list {
		if r.FullName == full {
			return r.ID
		}
	}
	t.Fatalf("未找到仓库 %s", full)
	return 0
}

func mustNotifID(t *testing.T, s *Store, full string) int64 {
	t.Helper()
	items, err := s.ListNotifications(NotificationFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.FullName == full {
			return it.ID
		}
	}
	t.Fatalf("未找到通知 %s", full)
	return 0
}

func TestListReposFilters(t *testing.T) {
	s := newTestStore(t)
	r1 := repo("alpha/one", 1)
	r1.Language = "Go"
	s.AddRepo(r1, SourceStar, false)
	r2 := repo("beta/two", 2)
	r2.Language = "Rust"
	s.AddRepo(r2, SourceStar, false)
	s.SetRepoMonitored(mustID(t, s, "beta/two"), true)

	byQuery, err := s.ListRepos(RepoFilter{Query: "alpha"})
	if err != nil || len(byQuery) != 1 || byQuery[0].FullName != "alpha/one" {
		t.Fatalf("Query 过滤失败, got %+v err=%v", byQuery, err)
	}
	byLang, err := s.ListRepos(RepoFilter{Language: "Rust"})
	if err != nil || len(byLang) != 1 || byLang[0].Language != "Rust" {
		t.Fatalf("Language 过滤失败, got %+v err=%v", byLang, err)
	}
	byMon, err := s.ListRepos(RepoFilter{Monitored: boolPtr(true)})
	if err != nil || len(byMon) != 1 || byMon[0].FullName != "beta/two" {
		t.Fatalf("Monitored 过滤失败, got %+v err=%v", byMon, err)
	}
}

func TestDeleteStarReposNotInKeep(t *testing.T) {
	s := newTestStore(t)
	s.AddRepo(repo("keep/me", 1), SourceStar, false)
	s.AddRepo(repo("drop/this", 2), SourceStar, false)
	s.AddRepo(repo("manual/pinned", 3), SourceManual, true)

	removed, err := s.DeleteStarReposNotIn(map[string]struct{}{"keep/me": {}})
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("应删除 1 个（manual/pinned 保留）, got %d", removed)
	}
	list, _ := s.ListRepos(RepoFilter{})
	if len(list) != 2 {
		t.Fatalf("应剩 2 个仓库, got %d", len(list))
	}

	// 全部保留时走 len(ids)==0 分支
	all := map[string]struct{}{}
	for _, r := range list {
		all[r.FullName] = struct{}{}
	}
	if n, err := s.DeleteStarReposNotIn(all); err != nil || n != 0 {
		t.Fatalf("全保留应返回 0, got %d err=%v", n, err)
	}
}

func TestRestoreReplaces(t *testing.T) {
	s := newTestStore(t)
	s.AddRepo(repo("old/repo", 1), SourceStar, false)
	settings := Settings{GitHubToken: "tok", ShoutrrrURL: "logger://", PollInterval: "15m"}
	repos := []Repo{repo("a/b", 5), repo("c/d", 9)}
	if err := s.Restore(settings, repos); err != nil {
		t.Fatalf("Restore 失败: %v", err)
	}
	if v, _ := s.GetSetting(KeyGitHubToken); v != "tok" {
		t.Fatalf("Restore 应恢复设置, got %q", v)
	}
	list, _ := s.ListRepos(RepoFilter{})
	if len(list) != 2 {
		t.Fatalf("Restore 应重建 2 个仓库, got %d", len(list))
	}
}

func TestParseTime(t *testing.T) {
	if !parseTime("").IsZero() {
		t.Fatal("空串应返回零值")
	}
	if parseTime("2024-01-01 00:00:00").IsZero() {
		t.Fatal("标准格式应解析成功")
	}
	if parseTime("2024-01-01T00:00:00Z").IsZero() {
		t.Fatal("RFC3339 应解析成功")
	}
	if !parseTime("not-a-time").IsZero() {
		t.Fatal("非法时间应返回零值")
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestPruneNotificationsKeep(t *testing.T) {
	s := newTestStore(t)
	if n, err := s.PruneNotifications(0); err != nil || n != 0 {
		t.Fatalf("keep<=0 不应删除, got %d err=%v", n, err)
	}
	for i := 0; i < 3; i++ {
		s.AddNotification(Notification{FullName: "a/b", Tag: string(rune('0' + i)), Status: "sent"})
	}
	removed, err := s.PruneNotifications(1)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 2 {
		t.Fatalf("应删除 2 条, got %d", removed)
	}
	if n, _ := s.CountNotifications(); n != 1 {
		t.Fatalf("应剩 1 条, got %d", n)
	}
}
