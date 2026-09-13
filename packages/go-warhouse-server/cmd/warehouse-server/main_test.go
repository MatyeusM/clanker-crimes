package main

import (
	"path/filepath"
	"testing"
)

// Route coverage (including /healthz) lives in internal/httpapi;
// here we pin the CLI subcommand contracts (PLAN.md §9).
func TestRunVersion(t *testing.T) {
	if code := run([]string{"warehouse-server", "version"}); code != 0 {
		t.Fatalf("run version = %d, want 0", code)
	}
}

func TestRunMigrateAndBackup(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	dest := filepath.Join(dir, "backup.db")
	t.Setenv("WAREHOUSE_DATABASE", dbPath)

	if code := run([]string{"warehouse-server", "migrate"}); code != 0 {
		t.Fatalf("run migrate = %d, want 0", code)
	}
	if code := run([]string{"warehouse-server", "backup", dest}); code != 0 {
		t.Fatalf("run backup = %d, want 0", code)
	}
	if code := run([]string{"warehouse-server", "backup"}); code != 2 {
		t.Fatalf("run backup without dest = %d, want 2", code)
	}
	if code := run([]string{"warehouse-server", "bogus"}); code != 2 {
		t.Fatalf("run bogus subcommand = %d, want 2", code)
	}
}

func TestSplitSub(t *testing.T) {
	sub, rest := splitSub([]string{"-db", "x.db", "serve"})
	if sub != "serve" || len(rest) != 2 {
		t.Fatalf("splitSub = %q %v", sub, rest)
	}
	sub, rest = splitSub([]string{"backup", "out.db"})
	if sub != "backup" || len(rest) != 1 || rest[0] != "out.db" {
		t.Fatalf("splitSub backup = %q %v", sub, rest)
	}
	sub, rest = splitSub([]string{"-listen", ":0"})
	if sub != "" || len(rest) != 2 {
		t.Fatalf("splitSub flags-only = %q %v", sub, rest)
	}
}
