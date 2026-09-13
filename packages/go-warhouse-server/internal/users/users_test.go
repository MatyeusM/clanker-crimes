package users

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"warehouse-server/internal/auth"
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

func TestCreateAndGetUser(t *testing.T) {
	conn := openMigrated(t)
	u, err := CreateUser(conn, "alice", "Alice A")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.Username != "alice" || u.DisplayName == nil || *u.DisplayName != "Alice A" {
		t.Fatalf("unexpected user: %+v", u)
	}
	if u.DisabledAt != nil {
		t.Fatal("new user is disabled")
	}
	got, err := GetUser(conn, u.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.Username != "alice" {
		t.Fatalf("got %q", got.Username)
	}
	if _, err := GetUser(conn, 9999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing user err = %v, want ErrNotFound", err)
	}
}

func TestCreateUserValidation(t *testing.T) {
	conn := openMigrated(t)
	if _, err := CreateUser(conn, "  ", ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty username err = %v, want ErrValidation", err)
	}
	if _, err := CreateUser(conn, "bob", ""); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, err := CreateUser(conn, "bob", ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate err = %v, want ErrConflict", err)
	}
}

func TestDisableUser(t *testing.T) {
	conn := openMigrated(t)
	u, _ := CreateUser(conn, "carol", "")
	d, err := DisableUser(conn, u.ID)
	if err != nil {
		t.Fatalf("DisableUser: %v", err)
	}
	if d.DisabledAt == nil {
		t.Fatal("disabled_at not set")
	}
	// Idempotent.
	if _, err := DisableUser(conn, u.ID); err != nil {
		t.Fatalf("second DisableUser: %v", err)
	}
	if _, err := DisableUser(conn, 9999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing user err = %v, want ErrNotFound", err)
	}
}

func TestKeyLifecycle(t *testing.T) {
	conn := openMigrated(t)
	u, _ := CreateUser(conn, "dave", "")

	plain, k, err := CreateAPIKey(conn, u.ID, "ci", auth.Write)
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if plain == "" || k.KeyHash == plain {
		t.Fatal("plaintext must be returned and differ from the stored hash")
	}
	if k.Permission != auth.Write {
		t.Fatalf("permission = %q", k.Permission)
	}
	if k.KeyHash != auth.HashKey(plain) {
		t.Fatal("stored hash does not match plaintext")
	}

	// Plaintext authenticates.
	if _, err := auth.Authenticate(conn, plain); err != nil {
		t.Fatalf("Authenticate fresh key: %v", err)
	}

	keys, err := ListKeys(conn, u.ID)
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(keys) != 1 || keys[0].KeyHash != k.KeyHash {
		t.Fatalf("unexpected keys: %+v", keys)
	}

	revoked, err := RevokeKey(conn, u.ID, k.ID)
	if err != nil {
		t.Fatalf("RevokeKey: %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("revoked_at not set")
	}
	// Idempotent second revoke.
	if _, err := RevokeKey(conn, u.ID, k.ID); err != nil {
		t.Fatalf("second RevokeKey: %v", err)
	}
	if _, err := auth.Authenticate(conn, plain); !errors.Is(err, auth.ErrUnauthenticated) {
		t.Fatalf("revoked key err = %v, want ErrUnauthenticated", err)
	}

	// Wrong owner scope.
	other, _ := CreateUser(conn, "erin", "")
	if _, err := RevokeKey(conn, other.ID, k.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-owner revoke err = %v, want ErrNotFound", err)
	}
}

func TestCreateKeyValidation(t *testing.T) {
	conn := openMigrated(t)
	u, _ := CreateUser(conn, "frank", "")
	if _, _, err := CreateAPIKey(conn, u.ID, "  ", auth.Read); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty name err = %v, want ErrValidation", err)
	}
	if _, _, err := CreateAPIKey(conn, 9999, "x", auth.Read); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing user err = %v, want ErrNotFound", err)
	}
	if _, err := DisableUser(conn, u.ID); err != nil {
		t.Fatalf("DisableUser: %v", err)
	}
	if _, _, err := CreateAPIKey(conn, u.ID, "x", auth.Read); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled user err = %v, want ErrDisabled", err)
	}
}
