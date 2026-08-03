package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(dataDir string) (*Store, error) {
	if err := ensureDir(dataDir); err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)",
		filepath.Join(dataDir, "chaxin.db"))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS repos (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			full_name        TEXT NOT NULL UNIQUE,
			owner            TEXT NOT NULL,
			repo             TEXT NOT NULL,
			description      TEXT,
			language         TEXT,
			stargazers_count INTEGER NOT NULL DEFAULT 0,
			html_url         TEXT,
			monitored        INTEGER NOT NULL DEFAULT 0,
			last_known_tag   TEXT,
			last_checked_at  DATETIME,
			created_at       DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,		`CREATE TABLE IF NOT EXISTS notifications (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id      INTEGER NOT NULL,
			full_name    TEXT NOT NULL,
			tag          TEXT NOT NULL,
			release_url  TEXT,
			released_at  DATETIME,
			sent_at      DATETIME NOT NULL DEFAULT (datetime('now')),
			status       TEXT NOT NULL,
			error        TEXT,
			release_body TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_repo ON notifications(repo_id)`,
		`CREATE INDEX IF NOT EXISTS idx_repos_monitored ON repos(monitored)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	// 兼容旧数据库：为已存在的表补充新列
	if err := ensureColumn(s.db, "notifications", "release_body", "TEXT"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := ensureColumn(s.db, "notifications", "release_body_translated", "TEXT"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := ensureColumn(s.db, "repos", "source", "TEXT NOT NULL DEFAULT 'manual'"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	// pinned 标记用户显式手动添加（未被 Star 确认）的仓库，同步清理时保留。
	// 旧库中 source 列由迁移默认值补充为 manual，pinned 为 0，可被同步清理修正。
	if err := ensureColumn(s.db, "repos", "pinned", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := ensureColumn(s.db, "repos", "ignore_pattern", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := ensureColumn(s.db, "repos", "latest_tag", "TEXT"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := ensureColumn(s.db, "repos", "latest_release_url", "TEXT"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := ensureColumn(s.db, "repos", "latest_release_body", "TEXT"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := ensureColumn(s.db, "repos", "latest_release_at", "DATETIME"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// ensureColumn 检查表中是否存在指定列，不存在则补充（用于对已建表做轻量迁移）。
func ensureColumn(db *sql.DB, table, column, decl string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, decl))
	return err
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}
