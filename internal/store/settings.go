package store

import (
	"database/sql"
	"strconv"
)

const (
	KeyGitHubToken        = "github_token"
	KeyShoutrrrURL        = "shoutrrr_url"
	KeyPollInterval       = "poll_interval"
	KeyNotifyFirstRun     = "notify_on_first_run"
	KeyGitHubAPIBaseURL   = "github_api_base_url"
	KeyMaxNotifications   = "max_notifications"
)

type Settings struct {
	GitHubToken      string `json:"github_token"`
	ShoutrrrURL      string `json:"shoutrrr_url"`
	PollInterval     string `json:"poll_interval"`
	NotifyOnFirstRun bool   `json:"notify_on_first_run"`
	GitHubAPIBaseURL string `json:"github_api_base_url"`
	MaxNotifications int    `json:"max_notifications"` // 0 表示不限制
}

func (s *Store) GetSettings() (Settings, error) {
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return Settings{}, err
	}
	defer rows.Close()

	raw := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return Settings{}, err
		}
		raw[k] = v
	}
	if err := rows.Err(); err != nil {
		return Settings{}, err
	}

	maxN, _ := strconv.Atoi(raw[KeyMaxNotifications])
	if maxN < 0 {
		maxN = 0
	}
	return Settings{
		GitHubToken:      raw[KeyGitHubToken],
		ShoutrrrURL:      raw[KeyShoutrrrURL],
		PollInterval:     raw[KeyPollInterval],
		NotifyOnFirstRun: raw[KeyNotifyFirstRun] == "1" || raw[KeyNotifyFirstRun] == "true",
		GitHubAPIBaseURL: raw[KeyGitHubAPIBaseURL],
		MaxNotifications: maxN,
	}, nil
}

func (s *Store) GetSetting(key string) (string, error) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func (s *Store) SaveSettings(in Settings) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	pairs := map[string]string{
		KeyGitHubToken:      in.GitHubToken,
		KeyShoutrrrURL:      in.ShoutrrrURL,
		KeyPollInterval:     in.PollInterval,
		KeyNotifyFirstRun:   boolStr(in.NotifyOnFirstRun),
		KeyGitHubAPIBaseURL: in.GitHubAPIBaseURL,
		KeyMaxNotifications: strconv.Itoa(in.MaxNotifications),
	}
	for k, v := range pairs {
		if _, err := tx.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
