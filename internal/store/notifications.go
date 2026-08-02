package store

import (
	"strings"
	"time"
)

type Notification struct {
	ID                    int64     `json:"id"`
	RepoID                int64     `json:"repo_id"`
	FullName              string    `json:"full_name"`
	Tag                   string    `json:"tag"`
	ReleaseURL            string    `json:"release_url"`
	ReleaseBody           string    `json:"release_body"`
	ReleaseBodyTranslated string    `json:"release_body_translated"`
	ReleasedAt            time.Time `json:"released_at"`
	SentAt                time.Time `json:"sent_at"`
	Status                string    `json:"status"`
	Error                 string    `json:"error"`
}

func (s *Store) AddNotification(n Notification) error {
	_, err := s.db.Exec(`INSERT INTO notifications
		(repo_id, full_name, tag, release_url, release_body, release_body_translated, released_at, status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		n.RepoID, n.FullName, n.Tag, n.ReleaseURL, n.ReleaseBody, n.ReleaseBodyTranslated,
		n.ReleasedAt.Format("2006-01-02 15:04:05"), n.Status, n.Error)
	return err
}

// NotificationFilter 通知记录筛选条件。
type NotificationFilter struct {
	Limit  int
	Query  string // 匹配 full_name 或 tag
	Status string // "sent" / "failed"，空表示全部
}

func (s *Store) ListNotifications(f NotificationFilter) ([]Notification, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	where := []string{}
	args := []any{}
	if f.Query != "" {
		where = append(where, "(full_name LIKE ? OR tag LIKE ?)")
		q := "%" + f.Query + "%"
		args = append(args, q, q)
	}
	if f.Status != "" {
		where = append(where, "status = ?")
		args = append(args, f.Status)
	}
	query := `SELECT id, repo_id, full_name, tag, COALESCE(release_url,''),
		COALESCE(release_body,''), COALESCE(release_body_translated,''), COALESCE(released_at,''), sent_at, status, COALESCE(error,'')
		FROM notifications`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY sent_at DESC, id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Notification{}
	for rows.Next() {
		var n Notification
		var released, sent string
		if err := rows.Scan(&n.ID, &n.RepoID, &n.FullName, &n.Tag, &n.ReleaseURL,
			&n.ReleaseBody, &n.ReleaseBodyTranslated, &released, &sent, &n.Status, &n.Error); err != nil {
			return nil, err
		}
		n.ReleasedAt = parseTime(released)
		n.SentAt = parseTime(sent)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) CountNotifications() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM notifications`).Scan(&n)
	return n, err
}

// MarkNotificationSent 将通知标记为已发送并清除错误。
func (s *Store) MarkNotificationSent(id int64) error {
	_, err := s.db.Exec(`UPDATE notifications SET status = 'sent', error = '' WHERE id = ?`, id)
	return err
}

// PruneNotifications 仅保留最新的 keep 条通知，删除更早的记录；keep<=0 时不清理。返回删除条数。
func (s *Store) PruneNotifications(keep int) (int64, error) {
	if keep <= 0 {
		return 0, nil
	}
	res, err := s.db.Exec(`DELETE FROM notifications WHERE id NOT IN (
		SELECT id FROM notifications ORDER BY id DESC LIMIT ?)`, keep)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
