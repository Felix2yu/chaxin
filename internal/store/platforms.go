package store

import (
	"database/sql"
	"time"
)

// 平台维度上次已知版本的跟踪表。一个仓库可含多个平台（如 iOS/mac/CLI/Desktop），
// 各平台独立记录 last tag，避免不同平台版本轮流触发通知。
const schemaRepoPlatforms = `
CREATE TABLE IF NOT EXISTS repo_platforms (
	repo_id     INTEGER NOT NULL,
	platform    TEXT NOT NULL,
	tag         TEXT NOT NULL,
	released_at DATETIME,
	updated_at  DATETIME NOT NULL DEFAULT (datetime('now')),
	PRIMARY KEY (repo_id, platform)
)`

// GetPlatformTag 返回仓库指定平台上次已知的版本 tag。
// 第二个返回值指示该平台是否已有记录。
func (s *Store) GetPlatformTag(repoID int64, platform string) (string, bool, error) {
	var tag string
	err := s.db.QueryRow(`SELECT tag FROM repo_platforms WHERE repo_id = ? AND platform = ?`,
		repoID, platform).Scan(&tag)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return tag, true, nil
}

// SetPlatformTag 记录（或更新）仓库指定平台的版本 tag。
func (s *Store) SetPlatformTag(repoID int64, platform, tag string, releasedAt time.Time) error {
	var rel string
	if !releasedAt.IsZero() {
		rel = releasedAt.Format("2006-01-02 15:04:05")
	}
	_, err := s.db.Exec(`INSERT INTO repo_platforms (repo_id, platform, tag, released_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(repo_id, platform) DO UPDATE SET
			tag = excluded.tag, released_at = excluded.released_at, updated_at = datetime('now')`,
		repoID, platform, tag, rel)
	return err
}

// RepoHasPlatformRecord 判断仓库是否已有任何平台记录。
func (s *Store) RepoHasPlatformRecord(repoID int64) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM repo_platforms WHERE repo_id = ? LIMIT 1`, repoID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// DeleteRepoPlatforms 删除仓库的全部平台记录。
func (s *Store) DeleteRepoPlatforms(repoID int64) error {
	_, err := s.db.Exec(`DELETE FROM repo_platforms WHERE repo_id = ?`, repoID)
	return err
}
