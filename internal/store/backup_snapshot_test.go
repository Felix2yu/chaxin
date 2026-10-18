package store

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// sqliteMagic 是 SQLite 数据库文件头部的 16 字节魔数。
var sqliteMagic = []byte("SQLite format 3\x00")

func TestWriteSQLiteSnapshotMagicAndRows(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.UpsertRepo(repo("a/b", 1), false); err != nil {
		t.Fatalf("UpsertRepo: %v", err)
	}

	var buf bytes.Buffer
	if err := s.WriteSQLiteSnapshot(&buf); err != nil {
		t.Fatalf("WriteSQLiteSnapshot: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), sqliteMagic) {
		t.Fatalf("快照头部 16 字节应为 SQLite 魔数, got %q", buf.Bytes()[:16])
	}

	// 重新打开快照，确认插入的行可读。
	path := writeSnapshotFile(t, buf.Bytes())
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("打开快照失败: %v", err)
	}
	defer db.Close()

	var full string
	err = db.QueryRow(`SELECT full_name FROM repos WHERE full_name = 'a/b'`).Scan(&full)
	if err != nil {
		t.Fatalf("快照中应能读到插入的行: %v", err)
	}
	if full != "a/b" {
		t.Fatalf("full_name=%q, want a/b", full)
	}
}

func TestOpenForExportDoesNotMigrateOrModify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, dbFileName)

	// 手工造一个只有 fixture 表的库（不经过 Open，也就没有应用 schema）。
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE fixture (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO fixture (name) VALUES ('x')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	before := sha256File(t, path)

	s, err := OpenForExport(dir)
	if err != nil {
		t.Fatalf("OpenForExport: %v", err)
	}
	var buf bytes.Buffer
	if err := s.WriteSQLiteSnapshot(&buf); err != nil {
		t.Fatalf("WriteSQLiteSnapshot: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	// 导出进程不得改动源文件。
	if got := sha256File(t, path); got != before {
		t.Fatalf("导出后源文件发生变化: before=%s after=%s", before, got)
	}

	// 快照里不应出现迁移才会创建的表，fixture 数据应原样保留。
	snap := writeSnapshotFile(t, buf.Bytes())
	sdb, err := sql.Open("sqlite", "file:"+snap+"?_pragma=query_only(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer sdb.Close()

	for _, table := range []string{"settings", "repos", "repo_platforms"} {
		var name string
		err := sdb.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != sql.ErrNoRows {
			t.Fatalf("导出路径不应执行迁移, 快照中却出现了表 %q (err=%v)", table, err)
		}
	}
	var name string
	if err := sdb.QueryRow(`SELECT name FROM fixture WHERE id = 1`).Scan(&name); err != nil {
		t.Fatalf("快照应保留 fixture 行: %v", err)
	}
	if name != "x" {
		t.Fatalf("name=%q, want x", name)
	}
}

// writeSnapshotFile 把快照字节落到独立临时目录里的新文件，返回路径。
func writeSnapshotFile(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "snapshot.db")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func sha256File(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}
