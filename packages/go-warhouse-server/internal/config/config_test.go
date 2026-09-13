package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("WAREHOUSE_LISTEN", "")
	t.Setenv("WAREHOUSE_DATABASE", "")
	t.Setenv("WAREHOUSE_LOG_LEVEL", "")

	cfg := Load()
	if cfg.Listen != DefaultListen {
		t.Fatalf("Listen = %q, want %q", cfg.Listen, DefaultListen)
	}
	if cfg.Database != DefaultDatabase {
		t.Fatalf("Database = %q, want %q", cfg.Database, DefaultDatabase)
	}
	if cfg.LogLevel != DefaultLogLevel {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, DefaultLogLevel)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("WAREHOUSE_LISTEN", "127.0.0.1:9090")
	t.Setenv("WAREHOUSE_DATABASE", "/tmp/test.db")
	t.Setenv("WAREHOUSE_LOG_LEVEL", "debug")

	cfg := Load()
	if cfg.Listen != "127.0.0.1:9090" {
		t.Fatalf("Listen = %q", cfg.Listen)
	}
	if cfg.Database != "/tmp/test.db" {
		t.Fatalf("Database = %q", cfg.Database)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q", cfg.LogLevel)
	}
}
