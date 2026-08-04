package store

import (
	"testing"
	"time"
)

func TestPlatformTagRoundTrip(t *testing.T) {
	s := newTestStore(t)
	s.UpsertRepo(repo("a/b", 1))
	list, _ := s.ListRepos(RepoFilter{})
	id := list[0].ID

	if has, _ := s.RepoHasPlatformRecord(id); has {
		t.Fatal("初始不应有平台记录")
	}
	if _, found, _ := s.GetPlatformTag(id, "ios"); found {
		t.Fatal("ios 平台初始不应有记录")
	}

	at := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	if err := s.SetPlatformTag(id, "ios", "iOS-1.0", at); err != nil {
		t.Fatal(err)
	}
	if has, _ := s.RepoHasPlatformRecord(id); !has {
		t.Fatal("设置后应有平台记录")
	}
	tag, found, err := s.GetPlatformTag(id, "ios")
	if err != nil || !found || tag != "iOS-1.0" {
		t.Fatalf("读取 ios 平台不符, tag=%q found=%v err=%v", tag, found, err)
	}

	// 更新同一平台
	if err := s.SetPlatformTag(id, "ios", "iOS-2.0", at); err != nil {
		t.Fatal(err)
	}
	tag, _, _ = s.GetPlatformTag(id, "ios")
	if tag != "iOS-2.0" {
		t.Fatalf("平台 tag 未更新, got %q", tag)
	}

	// 其他平台记录互相独立
	if _, found, _ := s.GetPlatformTag(id, "mac"); found {
		t.Fatal("mac 平台不应受 ios 写入影响")
	}

	// 删除仓库时连带清理平台记录
	if err := s.DeleteRepo(id); err != nil {
		t.Fatal(err)
	}
	if has, _ := s.RepoHasPlatformRecord(id); has {
		t.Fatal("删除仓库后平台记录应被清理")
	}
}

func TestMigrateLastKnownTagToDefaultPlatform(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.db.Exec(`INSERT INTO repos (full_name, owner, repo, last_known_tag)
		VALUES ('a/b', 'a', 'b', 'v1.0.0')`); err != nil {
		t.Fatal(err)
	}

	// 触发迁移（Open 时也会执行）
	if err := s.migrate(); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM repo_platforms WHERE platform = 'default' AND tag = 'v1.0.0'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("last_known_tag 应迁移到 default 平台, got %d 条", n)
	}

	// 幂等：重复 migrate 不应重复迁移
	if err := s.migrate(); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM repo_platforms WHERE platform = 'default'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("重复 migrate 不应重复迁移, got %d 条", n)
	}
}
