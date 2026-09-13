package db

import (
	"path/filepath"
	"strings"
	"testing"
)

func openMigrated(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()
	if _, err := Migrate(conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return path
}

func querySingle[T any](t *testing.T, path, pragma string) T {
	t.Helper()
	conn, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer conn.Close()
	var v T
	if err := conn.QueryRow("PRAGMA " + pragma).Scan(&v); err != nil {
		t.Fatalf("PRAGMA %s: %v", pragma, err)
	}
	return v
}

func TestMigrateCreatesSchema(t *testing.T) {
	path := openMigrated(t)

	conn, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer conn.Close()

	for _, table := range []string{
		"users", "api_keys", "vendors", "warehouses", "locations",
		"items", "vendor_items", "stock", "stock_movements", "schema_migrations",
	} {
		var name string
		err := conn.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}

	var version int
	if err := conn.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("read schema_migrations: %v", err)
	}
	if version != 1 {
		t.Fatalf("version = %d, want 1", version)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	path := openMigrated(t)

	conn, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer conn.Close()

	version, err := Migrate(conn)
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if version != 1 {
		t.Fatalf("version = %d, want 1", version)
	}
	var count int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("migration rows = %d, want 1", count)
	}
}

func TestPragmas(t *testing.T) {
	path := openMigrated(t)

	if mode := querySingle[string](t, path, "journal_mode"); !strings.EqualFold(mode, "wal") {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
	if fk := querySingle[int](t, path, "foreign_keys"); fk != 1 {
		t.Fatalf("foreign_keys = %d, want 1", fk)
	}
	if to := querySingle[int](t, path, "busy_timeout"); to != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", to)
	}
	// synchronous=NORMAL reads back as 1.
	if s := querySingle[int](t, path, "synchronous"); s != 1 {
		t.Fatalf("synchronous = %d, want 1 (NORMAL)", s)
	}
}

func TestForeignKeysEnforced(t *testing.T) {
	path := openMigrated(t)

	conn, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer conn.Close()

	_, err = conn.Exec(
		`INSERT INTO locations (warehouse_id, code, name) VALUES (9999, 'A-01', 'bogus')`,
	)
	if err == nil {
		t.Fatal("expected foreign key violation, got nil")
	}
}

func TestOpenCreatesParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deep", "test.db")
	conn, err := Open(path)
	if err != nil {
		t.Fatalf("Open with nested path: %v", err)
	}
	_ = conn.Close()
}

func TestBackup(t *testing.T) {
	path := openMigrated(t)
	conn, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Exec(
		`INSERT INTO users (username, created_at) VALUES ('op', '2026-01-01T00:00:00Z')`,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "nested", "backup.db")
	if err := Backup(conn, dest); err != nil {
		t.Fatalf("Backup: %v", err)
	}
	restored, err := Open(dest)
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer restored.Close()
	var count int
	if err := restored.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if count != 1 {
		t.Fatalf("backup users = %d, want 1", count)
	}
}
