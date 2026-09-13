package auth

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"warehouse-server/internal/db"
)

func openMigrated(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := db.Migrate(conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return conn
}

func seedUser(t *testing.T, conn *sql.DB, username string, disabled bool) int64 {
	t.Helper()
	disabledAt := any(nil)
	if disabled {
		disabledAt = "2026-09-12T00:00:00Z"
	}
	res, err := conn.Exec(
		`INSERT INTO users (username, created_at, disabled_at) VALUES (?, ?, ?)`,
		username, "2026-09-12T00:00:00Z", disabledAt,
	)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func seedKey(t *testing.T, conn *sql.DB, userID int64, plaintext, perm string, revoked bool) {
	t.Helper()
	revokedAt := any(nil)
	if revoked {
		revokedAt = "2026-09-12T00:00:00Z"
	}
	if _, err := conn.Exec(
		`INSERT INTO api_keys (user_id, key_hash, name, permission, created_at, revoked_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		userID, HashKey(plaintext), "test", perm, "2026-09-12T00:00:00Z", revokedAt,
	); err != nil {
		t.Fatalf("seed key: %v", err)
	}
}

func TestParsePermission(t *testing.T) {
	for _, p := range []string{"read", "read-movements", "write", "admin"} {
		if _, err := ParsePermission(p); err != nil {
			t.Fatalf("ParsePermission(%q): %v", p, err)
		}
	}
	if _, err := ParsePermission("root"); err == nil {
		t.Fatal("expected error for unknown permission")
	}
}

func TestGrantsMatrix(t *testing.T) {
	cases := []struct {
		have Permission
		need Permission
		want bool
	}{
		{Read, Read, true},
		{Read, ReadMovements, false},
		{Read, Write, false},
		{Read, Admin, false},
		{ReadMovements, Read, true},
		{ReadMovements, ReadMovements, true},
		{ReadMovements, Write, false},
		{Write, Read, true},
		{Write, ReadMovements, true},
		{Write, Write, true},
		{Write, Admin, false},
		{Admin, Read, true},
		{Admin, ReadMovements, true},
		{Admin, Write, true},
		{Admin, Admin, true},
	}
	for _, c := range cases {
		if got := c.have.Grants(c.need); got != c.want {
			t.Fatalf("%q.Grants(%q) = %v, want %v", c.have, c.need, got, c.want)
		}
	}
}

func TestGenerateKeyFormat(t *testing.T) {
	plain, hash, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if !strings.HasPrefix(plain, KeyPrefix) {
		t.Fatalf("plaintext %q missing prefix %q", plain, KeyPrefix)
	}
	if hash != HashKey(plain) {
		t.Fatal("hash mismatch")
	}
	other, _, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if plain == other {
		t.Fatal("two generated keys are identical")
	}
}

func TestAuthenticate(t *testing.T) {
	conn := openMigrated(t)
	uid := seedUser(t, conn, "alice", false)
	plaintext, _, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	seedKey(t, conn, uid, plaintext, "write", false)

	id, err := Authenticate(conn, plaintext)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if id.UserID != uid || id.Permission != Write || id.Username != "alice" {
		t.Fatalf("unexpected identity: %+v", id)
	}
	var lastUsed sql.NullString
	if err := conn.QueryRow(`SELECT last_used_at FROM api_keys WHERE id = ?`, id.KeyID).Scan(&lastUsed); err != nil {
		t.Fatalf("read last_used_at: %v", err)
	}
	if !lastUsed.Valid {
		t.Fatal("last_used_at was not recorded")
	}

	if _, err := Authenticate(conn, "wks_bogus"); err != ErrUnauthenticated {
		t.Fatalf("bogus key err = %v, want ErrUnauthenticated", err)
	}
}

func TestAuthenticateDenied(t *testing.T) {
	conn := openMigrated(t)

	// Revoked key.
	uid := seedUser(t, conn, "bob", false)
	revokedPlain, _, _ := GenerateKey()
	seedKey(t, conn, uid, revokedPlain, "read", true)
	if _, err := Authenticate(conn, revokedPlain); err != ErrUnauthenticated {
		t.Fatalf("revoked key err = %v, want ErrUnauthenticated", err)
	}

	// Disabled owner's key.
	duid := seedUser(t, conn, "mallory", true)
	disabledPlain, _, _ := GenerateKey()
	seedKey(t, conn, duid, disabledPlain, "admin", false)
	if _, err := Authenticate(conn, disabledPlain); err != ErrUnauthenticated {
		t.Fatalf("disabled owner err = %v, want ErrUnauthenticated", err)
	}
}

func TestAdminExists(t *testing.T) {
	conn := openMigrated(t)
	exists, err := AdminExists(conn)
	if err != nil {
		t.Fatalf("AdminExists: %v", err)
	}
	if exists {
		t.Fatal("fresh DB reports an admin")
	}

	uid := seedUser(t, conn, "admin", false)
	plain, _, _ := GenerateKey()
	seedKey(t, conn, uid, plain, "admin", false)
	exists, err = AdminExists(conn)
	if err != nil {
		t.Fatalf("AdminExists: %v", err)
	}
	if !exists {
		t.Fatal("DB with active admin reports none")
	}

	// A read key alone must not count.
	ruid := seedUser(t, conn, "reader", false)
	rplain, _, _ := GenerateKey()
	seedKey(t, conn, ruid, rplain, "read", false)

	if _, err := conn.Exec(`UPDATE users SET disabled_at = ? WHERE id = ?`, "2026-09-12T00:00:00Z", uid); err != nil {
		t.Fatalf("disable admin: %v", err)
	}
	exists, err = AdminExists(conn)
	if err != nil {
		t.Fatalf("AdminExists: %v", err)
	}
	if exists {
		t.Fatal("disabled admin still counts (lockout risk)")
	}
}
