// Package db owns the SQLite connection and the migration runner.
//
// It is the only package that imports the SQLite driver
// (PLAN.md §3); all domain code works against database/sql.
package db

import (
	"database/sql"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"warehouse-server/migrations"

	_ "modernc.org/sqlite"
)

const migrationsDir = "."

// Open creates the SQLite connection pool for path, creating the
// parent directory when needed, and applies the v1 pragmas:
//
//	WAL journal mode, foreign_keys=ON, busy_timeout=5000, synchronous=NORMAL.
//
// The DSN pins _txlock=immediate so every database/sql transaction
// starts as BEGIN IMMEDIATE: check-then-write inventory sequences
// (PLAN.md §8) are atomic against concurrent writers.
//
// MaxOpenConns is pinned to 1: SQLite serializes writers anyway, and
// a single connection keeps lock contention predictable.
func Open(path string) (*sql.DB, error) {
	dsn, err := dsnFor(path)
	if err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
	} {
		if _, err := conn.Exec(pragma); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("apply %q: %w", pragma, err)
		}
	}
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return conn, nil
}

// dsnFor builds the driver DSN for path. File paths become absolute
// file: URIs carrying _txlock=immediate; :memory: is passed through
// in URI form with the same lock mode.
func dsnFor(path string) (string, error) {
	if path == ":memory:" {
		return "file::memory:?_txlock=immediate", nil
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("create database dir: %w", err)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve database path: %w", err)
	}
	return (&url.URL{Scheme: "file", Path: abs, RawQuery: "_txlock=immediate"}).String(), nil
}

// CurrentVersion returns the highest applied migration version,
// or 0 on a fresh database. Read-only: the table itself is created
// by migration 0001, not here.
func CurrentVersion(conn *sql.DB) (int, error) {
	var exists bool
	err := conn.QueryRow(
		`SELECT COUNT(*) > 0 FROM sqlite_master WHERE type='table' AND name='schema_migrations'`,
	).Scan(&exists)
	if err != nil {
		return 0, fmt.Errorf("check schema_migrations: %w", err)
	}
	if !exists {
		return 0, nil
	}
	var v sql.NullInt64
	if err := conn.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&v); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return int(v.Int64), nil
}

// Migrate applies every pending embedded migration in version order
// and returns the resulting schema version.
func Migrate(conn *sql.DB) (int, error) {
	current, err := CurrentVersion(conn)
	if err != nil {
		return 0, err
	}
	pending, err := pendingMigrations(current)
	if err != nil {
		return 0, err
	}
	for _, m := range pending {
		if err := applyMigration(conn, m); err != nil {
			return 0, err
		}
	}
	return CurrentVersion(conn)
}

type migration struct {
	version int
	name    string
	sql     string
}

func pendingMigrations(current int) ([]migration, error) {
	entries, err := fs.ReadDir(migrations.FS, migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	var out []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		ver, err := strconv.Atoi(strings.SplitN(e.Name(), "_", 2)[0])
		if err != nil {
			return nil, fmt.Errorf("bad migration name %q: %w", e.Name(), err)
		}
		if ver <= current {
			continue
		}
		raw, err := fs.ReadFile(migrations.FS, e.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", e.Name(), err)
		}
		out = append(out, migration{version: ver, name: e.Name(), sql: string(raw)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// applyMigration runs one migration file inside a transaction, then
// records its version. Statements are split on ";" after stripping
// "--" line comments; sufficient for DDL-only migrations without
// triggers or multi-line string literals containing semicolons.
func applyMigration(conn *sql.DB, m migration) error {
	tx, err := conn.Begin()
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", m.name, err)
	}
	rollback := func() { _ = tx.Rollback() }

	var sb strings.Builder
	for _, line := range strings.Split(m.sql, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	for _, stmt := range strings.Split(sb.String(), ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			rollback()
			return fmt.Errorf("migration %s: exec %q: %w", m.name, truncate(stmt), err)
		}
	}
	appliedAt := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec(
		`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
		m.version, appliedAt,
	); err != nil {
		rollback()
		return fmt.Errorf("migration %s: record version: %w", m.name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migration %s: commit: %w", m.name, err)
	}
	return nil
}

// Backup writes a consistent snapshot of the open database to dest
// using `VACUUM INTO`. Never plain-cp a live SQLite file: WAL
// checkpoints can interleave with the copy. The destination's parent
// directory is created when needed; an existing file is overwritten
// by SQLite.
func Backup(conn *sql.DB, dest string) error {
	if dir := filepath.Dir(dest); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create backup dir: %w", err)
		}
	}
	abs, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("resolve backup path: %w", err)
	}
	// VACUUM INTO takes a string literal, not a bound parameter.
	literal := "'" + strings.ReplaceAll(abs, "'", "''") + "'"
	if _, err := conn.Exec(`VACUUM INTO ` + literal); err != nil {
		return fmt.Errorf("vacuum into %q: %w", dest, err)
	}
	return nil
}

func truncate(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 80 {
		return s[:80] + "…"
	}
	return s
}
