package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Repo struct {
	ID               int64     `json:"id"`
	FullName         string    `json:"full_name"`
	Owner            string    `json:"owner"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Language         string    `json:"language"`
	Stargazers       int       `json:"stargazers_count"`
	HTMLURL          string    `json:"html_url"`
	Monitored        bool      `json:"monitored"`
	LastKnownTag     string    `json:"last_known_tag"`
	LastCheckedAt    time.Time `json:"last_checked_at"`
	CreatedAt        time.Time `json:"created_at"`
	Source           string    `json:"source"`            // "star"（来自同步）或 "manual"（手动添加）
	IgnorePattern    string    `json:"ignore_pattern"`    // 忽略版本的正则，命中则跳过该仓库
	LatestTag        string    `json:"latest_tag"`        // 最新版本缓存（RSS 用）
	LatestReleaseURL string    `json:"latest_release_url"`
	LatestReleaseAt  time.Time `json:"latest_release_at"`
	LatestReleaseBody string   `json:"latest_release_body"`
}

const (
	SourceStar   = "star"
	SourceManual = "manual"
)

// UpsertResult 描述一次仓库 upsert 的结果。
type UpsertResult int

const (
	UpsertInserted UpsertResult = iota // 新增
	UpsertUpdated                      // 信息有变化
	UpsertSkipped                      // 已存在且无变化
)

type RepoFilter struct {
	Query     string
	Language  string
	Monitored *bool
}

// UpsertRepo 插入或更新仓库，返回本次结果（新增 / 信息更新 / 无变化）。
// 已存在且各字段与传入值一致时返回 UpsertSkipped，避免无谓写入。
// 同步来源的仓库一律标记 source=star 且 pinned=0。
// monitorNewStars 为 true 时，新增的仓库默认设置为监控状态。
func (s *Store) UpsertRepo(r Repo, monitorNewStars bool) (UpsertResult, error) {
	// 1) 尝试插入（仅新仓库生效，已存在则忽略）
	res, err := s.db.Exec(`INSERT OR IGNORE INTO repos
		(full_name, owner, repo, description, language, stargazers_count, html_url, source, pinned, monitored)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.FullName, r.Owner, r.Name, r.Description, r.Language, r.Stargazers, r.HTMLURL, SourceStar, 0, boolInt(monitorNewStars))
	if err != nil {
		return UpsertSkipped, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return UpsertSkipped, err
	}
	if n == 1 {
		return UpsertInserted, nil
	}

	// 2) 已存在：仅当字段有变化时才更新，避免无谓写入
	res, err = s.db.Exec(`UPDATE repos SET
			description = ?, language = ?, stargazers_count = ?, html_url = ?, source = ?, pinned = 0
		WHERE full_name = ? AND (
			description IS NOT ? OR language IS NOT ? OR
			stargazers_count != ? OR html_url IS NOT ? OR source != ? OR pinned != 0)`,
		r.Description, r.Language, r.Stargazers, r.HTMLURL, SourceStar,
		r.FullName, r.Description, r.Language, r.Stargazers, r.HTMLURL, SourceStar)
	if err != nil {
		return UpsertSkipped, err
	}
	n, err = res.RowsAffected()
	if err != nil {
		return UpsertSkipped, err
	}
	if n == 1 {
		return UpsertUpdated, nil
	}
	return UpsertSkipped, nil
}

func (s *Store) RepoExists(fullName string) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM repos WHERE full_name = ?`, fullName).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// AddRepo 手动添加仓库。source 取 SourceManual / SourceStar；
// pinned 表示该仓库由用户显式手动添加且未被确认 Star，同步清理时予以保留。
func (s *Store) AddRepo(r Repo, source string, pinned bool) error {
	_, err := s.db.Exec(`INSERT INTO repos (full_name, owner, repo, description, language, stargazers_count, html_url, source, pinned)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.FullName, r.Owner, r.Name, r.Description, r.Language, r.Stargazers, r.HTMLURL, source, boolInt(pinned))
	return err
}

// DeleteStarReposNotIn 删除不在 keep（当前 GitHub Star 列表）中的仓库，返回删除数量。
// 仅保留用户显式手动添加（source=manual 且 pinned=1）的仓库；
// 其余一律视为已取消 Star 的来源仓库清理掉，包括旧库中 source 被迁移默认值误标为 manual 的记录。
func (s *Store) DeleteStarReposNotIn(keep map[string]struct{}) (int64, error) {
	rows, err := s.db.Query(`SELECT id, full_name, source, pinned FROM repos`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		var fullName, source string
		var pinned int
		if err := rows.Scan(&id, &fullName, &source, &pinned); err != nil {
			return 0, err
		}
		if _, ok := keep[fullName]; ok {
			continue
		}
		if source == SourceManual && pinned == 1 {
			continue
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	res, err := s.db.Exec(`DELETE FROM repos WHERE id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return 0, err
	}
	for _, id := range ids {
		if _, err := s.db.Exec(`DELETE FROM repo_platforms WHERE repo_id = ?`, id); err != nil {
			return 0, err
		}
	}
	return res.RowsAffected()
}

func (s *Store) GetRepoByID(id int64) (Repo, error) {
	return scanRepo(s.db.QueryRow(
		`SELECT id, full_name, owner, repo, COALESCE(description,''), COALESCE(language,''), stargazers_count,
		        COALESCE(html_url,''), monitored, COALESCE(last_known_tag,''), last_checked_at, created_at, COALESCE(source,''), COALESCE(ignore_pattern,''),
		        COALESCE(latest_tag,''), COALESCE(latest_release_url,''), latest_release_at, COALESCE(latest_release_body,'')
		 FROM repos WHERE id = ?`, id))
}

func (s *Store) ListRepos(f RepoFilter) ([]Repo, error) {
	where := []string{}
	args := []any{}
	if f.Query != "" {
		where = append(where, "(full_name LIKE ? OR description LIKE ?)")
		q := "%" + f.Query + "%"
		args = append(args, q, q)
	}
	if f.Language != "" {
		where = append(where, "language = ?")
		args = append(args, f.Language)
	}
	if f.Monitored != nil {
		if *f.Monitored {
			where = append(where, "monitored = 1")
		} else {
			where = append(where, "monitored = 0")
		}
	}
	query := `SELECT id, full_name, owner, repo, COALESCE(description,''), COALESCE(language,''), stargazers_count,
	        COALESCE(html_url,''), monitored, COALESCE(last_known_tag,''), last_checked_at, created_at, COALESCE(source,''), COALESCE(ignore_pattern,''),
	        COALESCE(latest_tag,''), COALESCE(latest_release_url,''), latest_release_at, COALESCE(latest_release_body,'')
		FROM repos`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY monitored DESC, stargazers_count DESC, full_name ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	repos := []Repo{}
	for rows.Next() {
		r, err := scanRepo(rows)
		if err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (s *Store) ListMonitoredRepos() ([]Repo, error) {
	rows, err := s.db.Query(
		`SELECT id, full_name, owner, repo, COALESCE(description,''), COALESCE(language,''), stargazers_count,
		        COALESCE(html_url,''), monitored, COALESCE(last_known_tag,''), last_checked_at, created_at, COALESCE(source,''), COALESCE(ignore_pattern,''),
		        COALESCE(latest_tag,''), COALESCE(latest_release_url,''), latest_release_at, COALESCE(latest_release_body,'')
		 FROM repos WHERE monitored = 1 ORDER BY full_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	repos := []Repo{}
	for rows.Next() {
		r, err := scanRepo(rows)
		if err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (s *Store) SetRepoMonitored(id int64, monitored bool) error {
	_, err := s.db.Exec(`UPDATE repos SET monitored = ? WHERE id = ?`, boolInt(monitored), id)
	return err
}

func (s *Store) SetRepoIgnorePattern(id int64, pattern string) error {
	_, err := s.db.Exec(`UPDATE repos SET ignore_pattern = ? WHERE id = ?`, pattern, id)
	return err
}

// SetReposMonitored 批量设置监控状态，返回受影响的仓库数。
func (s *Store) SetReposMonitored(ids []int64, monitored bool) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	args := make([]any, 0, len(ids)+1)
	args = append(args, boolInt(monitored))
	placeholders := make([]string, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	res, err := s.db.Exec(`UPDATE repos SET monitored = ? WHERE id IN (`+strings.Join(placeholders, ",")+`)`,
		args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) SetLastKnownTag(id int64, tag string) error {
	_, err := s.db.Exec(`UPDATE repos SET last_known_tag = ?, last_checked_at = datetime('now') WHERE id = ?`, tag, id)
	return err
}

// SetLatestRelease 更新仓库最新版本缓存（RSS 订阅数据源）。
func (s *Store) SetLatestRelease(id int64, tag, url, body string, at time.Time) error {
	var atStr string
	if !at.IsZero() {
		atStr = at.Format("2006-01-02 15:04:05")
	}
	_, err := s.db.Exec(`UPDATE repos SET latest_tag = ?, latest_release_url = ?, latest_release_body = ?, latest_release_at = ?
		WHERE id = ?`, tag, url, body, atStr, id)
	return err
}

func (s *Store) TouchCheckedAt(id int64) error {
	_, err := s.db.Exec(`UPDATE repos SET last_checked_at = datetime('now') WHERE id = ?`, id)
	return err
}

func (s *Store) DeleteRepo(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM repo_platforms WHERE repo_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM repos WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// Restore 用备份数据覆盖 settings 并替换全部仓库列表（事务内完成）。
func (s *Store) Restore(settings Settings, repos []Repo) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	pairs := map[string]string{
		KeyGitHubToken:      settings.GitHubToken,
		KeyShoutrrrURL:      settings.ShoutrrrURL,
		KeyPollInterval:     settings.PollInterval,
		KeyNotifyFirstRun:   boolStr(settings.NotifyOnFirstRun),
		KeyMonitorNewStars:  boolStr(settings.MonitorNewStars),
		KeyGitHubAPIBaseURL: settings.GitHubAPIBaseURL,
		KeyMaxNotifications: fmt.Sprintf("%d", settings.MaxNotifications),
	}
	for k, v := range pairs {
		if _, err := tx.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM repos`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM repo_platforms`); err != nil {
		return err
	}
	for _, r := range repos {
		source := r.Source
		if source != SourceStar && source != SourceManual {
			source = SourceManual
		}
		pinned := source == SourceManual
		if _, err := tx.Exec(`INSERT INTO repos (full_name, owner, repo, description, language,
			stargazers_count, html_url, monitored, last_known_tag, last_checked_at, created_at, source, ignore_pattern, pinned)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.FullName, r.Owner, r.Name, r.Description, r.Language, r.Stargazers, r.HTMLURL,
			boolInt(r.Monitored), r.LastKnownTag, r.LastCheckedAt.Format("2006-01-02 15:04:05"),
			r.CreatedAt.Format("2006-01-02 15:04:05"), source, r.IgnorePattern, boolInt(pinned)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) CountRepos() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM repos`).Scan(&n)
	return n, err
}

func (s *Store) CountMonitored() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM repos WHERE monitored = 1`).Scan(&n)
	return n, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRepo(row rowScanner) (Repo, error) {
	var r Repo
	var lastChecked sql.NullString
	var createdAt string
	var latestReleaseAt sql.NullString
	var monitored int
	err := row.Scan(&r.ID, &r.FullName, &r.Owner, &r.Name, &r.Description, &r.Language,
		&r.Stargazers, &r.HTMLURL, &monitored, &r.LastKnownTag, &lastChecked, &createdAt, &r.Source, &r.IgnorePattern,
		&r.LatestTag, &r.LatestReleaseURL, &latestReleaseAt, &r.LatestReleaseBody)
	if err != nil {
		return Repo{}, err
	}
	r.Monitored = monitored == 1
	r.LastCheckedAt = parseTime(lastChecked.String)
	r.CreatedAt = parseTime(createdAt)
	r.LatestReleaseAt = parseTime(latestReleaseAt.String)
	return r, nil
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (r Repo) String() string {
	return fmt.Sprintf("%s", r.FullName)
}
