package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Repo struct {
	ID             int64     `json:"id"`
	FullName       string    `json:"full_name"`
	Owner          string    `json:"owner"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Language       string    `json:"language"`
	Stargazers     int       `json:"stargazers_count"`
	HTMLURL        string    `json:"html_url"`
	Monitored      bool      `json:"monitored"`
	LastKnownTag   string    `json:"last_known_tag"`
	LastCheckedAt  time.Time `json:"last_checked_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type RepoFilter struct {
	Query     string
	Language  string
	Monitored *bool
}

func (s *Store) UpsertRepo(r Repo) (bool, error) {
	res, err := s.db.Exec(`INSERT INTO repos (full_name, owner, repo, description, language, stargazers_count, html_url)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(full_name) DO UPDATE SET
			description = excluded.description,
			language = excluded.language,
			stargazers_count = excluded.stargazers_count,
			html_url = excluded.html_url`,
		r.FullName, r.Owner, r.Name, r.Description, r.Language, r.Stargazers, r.HTMLURL)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	// 影响行数为 1 表示 INSERT 新行，2 表示既有行被更新
	return n == 1, nil
}

func (s *Store) RepoExists(fullName string) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM repos WHERE full_name = ?`, fullName).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) AddRepo(r Repo) error {
	_, err := s.db.Exec(`INSERT INTO repos (full_name, owner, repo, description, language, stargazers_count, html_url)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.FullName, r.Owner, r.Name, r.Description, r.Language, r.Stargazers, r.HTMLURL)
	return err
}

func (s *Store) GetRepoByID(id int64) (Repo, error) {
	return scanRepo(s.db.QueryRow(
		`SELECT id, full_name, owner, repo, COALESCE(description,''), COALESCE(language,''), stargazers_count,
		        COALESCE(html_url,''), monitored, COALESCE(last_known_tag,''), last_checked_at, created_at
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
	        COALESCE(html_url,''), monitored, COALESCE(last_known_tag,''), last_checked_at, created_at
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
		        COALESCE(html_url,''), monitored, COALESCE(last_known_tag,''), last_checked_at, created_at
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

func (s *Store) TouchCheckedAt(id int64) error {
	_, err := s.db.Exec(`UPDATE repos SET last_checked_at = datetime('now') WHERE id = ?`, id)
	return err
}

func (s *Store) DeleteRepo(id int64) error {
	_, err := s.db.Exec(`DELETE FROM repos WHERE id = ?`, id)
	return err
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
	var monitored int
	err := row.Scan(&r.ID, &r.FullName, &r.Owner, &r.Name, &r.Description, &r.Language,
		&r.Stargazers, &r.HTMLURL, &monitored, &r.LastKnownTag, &lastChecked, &createdAt)
	if err != nil {
		return Repo{}, err
	}
	r.Monitored = monitored == 1
	r.LastCheckedAt = parseTime(lastChecked.String)
	r.CreatedAt = parseTime(createdAt)
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
