package store

import (
	"time"
)

type Notification struct {
	ID         int64     `json:"id"`
	RepoID     int64     `json:"repo_id"`
	FullName   string    `json:"full_name"`
	Tag        string    `json:"tag"`
	ReleaseURL string    `json:"release_url"`
	ReleaseBody string   `json:"release_body"`
	ReleasedAt time.Time `json:"released_at"`
	SentAt     time.Time `json:"sent_at"`
	Status     string    `json:"status"`
	Error      string    `json:"error"`
}

func (s *Store) AddNotification(n Notification) error {
	_, err := s.db.Exec(`INSERT INTO notifications
		(repo_id, full_name, tag, release_url, release_body, released_at, status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		n.RepoID, n.FullName, n.Tag, n.ReleaseURL, n.ReleaseBody,
		n.ReleasedAt.Format("2006-01-02 15:04:05"), n.Status, n.Error)
	return err
}

func (s *Store) ListNotifications(limit int) ([]Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT id, repo_id, full_name, tag, COALESCE(release_url,''),
		COALESCE(release_body,''), COALESCE(released_at,''), sent_at, status, COALESCE(error,'')
		FROM notifications ORDER BY sent_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Notification{}
	for rows.Next() {
		var n Notification
		var released, sent string
		if err := rows.Scan(&n.ID, &n.RepoID, &n.FullName, &n.Tag, &n.ReleaseURL,
			&n.ReleaseBody, &released, &sent, &n.Status, &n.Error); err != nil {
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
