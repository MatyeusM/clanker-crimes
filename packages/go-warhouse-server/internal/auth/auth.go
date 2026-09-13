// Package auth implements API-key authentication and the staggered
// permission model (PLAN.md §6):
//
//	read < read-movements < write, and admin grants everything.
//
// Keys are opaque `wks_...` tokens; only the SHA-256 hex hash is
// persisted. No passwords exist in v1.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Permission is an API-key capability level.
type Permission string

const (
	// Read sees masters and current stock only.
	Read Permission = "read"
	// ReadMovements additionally sees movement history and planned movements.
	ReadMovements Permission = "read-movements"
	// Write additionally mutates inventory and masters.
	Write Permission = "write"
	// Admin grants everything, including user and key management.
	Admin Permission = "admin"
)

// KeyPrefix marks warehouse-server API keys.
const KeyPrefix = "wks_"

var (
	// ErrUnauthenticated covers missing, unknown, revoked, and
	// disabled-owner keys. One error on purpose: callers must not
	// leak which of those applied.
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrForbidden covers authenticated identities lacking the
	// required permission.
	ErrForbidden = errors.New("forbidden")
	// ErrUnknownPermission covers permission strings outside the
	// four known levels.
	ErrUnknownPermission = errors.New("unknown permission")
)

// ParsePermission validates a permission string.
func ParsePermission(s string) (Permission, error) {
	switch Permission(s) {
	case Read, ReadMovements, Write, Admin:
		return Permission(s), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnknownPermission, s)
	}
}

func rank(p Permission) int {
	switch p {
	case Read:
		return 1
	case ReadMovements:
		return 2
	case Write:
		return 3
	case Admin:
		return 4
	default:
		return 0
	}
}

// Grants reports whether permission p satisfies the required level.
// Write implies both read levels; admin implies everything.
func (p Permission) Grants(need Permission) bool {
	return rank(p) >= rank(need) && rank(p) > 0 && rank(need) > 0
}

// HashKey returns the persisted SHA-256 hex digest of a plaintext key.
func HashKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return fmt.Sprintf("%x", sum)
}

// GenerateKey creates a new `wks_...` plaintext key and its hash.
// The plaintext is returned once; callers must not persist it.
func GenerateKey() (plaintext, hash string, err error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", "", fmt.Errorf("generate key: %w", err)
	}
	plaintext = KeyPrefix + base64.RawURLEncoding.EncodeToString(raw[:])
	return plaintext, HashKey(plaintext), nil
}

// Identity is the authenticated caller attached to a request context.
type Identity struct {
	UserID     int64
	KeyID      int64
	Username   string
	Permission Permission
}

// AdminExists reports whether at least one active admin key exists:
// permission='admin', not revoked, owner not disabled. Bootstrap
// (unauthenticated user creation) is open exactly when this is false,
// which also recovers from a lockout where every admin was disabled.
func AdminExists(conn *sql.DB) (bool, error) {
	var exists bool
	err := conn.QueryRow(`
		SELECT COUNT(*) > 0
		FROM api_keys k
		JOIN users u ON u.id = k.user_id
		WHERE k.permission = 'admin'
		  AND k.revoked_at IS NULL
		  AND u.disabled_at IS NULL`,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check admin exists: %w", err)
	}
	return exists, nil
}

// FromRequest authenticates the Bearer key on r and records its use.
// It performs no permission check; use Require for that.
func FromRequest(conn *sql.DB, r *http.Request) (Identity, error) {
	plaintext, ok := bearerKey(r)
	if !ok {
		return Identity{}, ErrUnauthenticated
	}
	return Authenticate(conn, plaintext)
}

// Authenticate validates a plaintext key against the database.
func Authenticate(conn *sql.DB, plaintext string) (Identity, error) {
	hash := HashKey(plaintext)
	var id Identity
	var storedHash string
	var revoked, disabled sql.NullString
	err := conn.QueryRow(`
		SELECT k.id, k.user_id, k.permission, k.key_hash, u.username,
		       k.revoked_at, u.disabled_at
		FROM api_keys k
		JOIN users u ON u.id = k.user_id
		WHERE k.key_hash = ?`,
		hash,
	).Scan(&id.KeyID, &id.UserID, &id.Permission, &storedHash, &id.Username, &revoked, &disabled)
	if err == sql.ErrNoRows {
		return Identity{}, ErrUnauthenticated
	}
	if err != nil {
		return Identity{}, fmt.Errorf("lookup api key: %w", err)
	}
	// Indexed lookup already matched, but compare in constant time
	// as hygiene against timing oracles on the hash column.
	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(hash)) != 1 {
		return Identity{}, ErrUnauthenticated
	}
	if revoked.Valid || disabled.Valid {
		return Identity{}, ErrUnauthenticated
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := conn.Exec(`UPDATE api_keys SET last_used_at = ? WHERE id = ?`, now, id.KeyID); err != nil {
		return Identity{}, fmt.Errorf("record key use: %w", err)
	}
	return id, nil
}

func bearerKey(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", false
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	key := strings.TrimSpace(parts[1])
	if key == "" {
		return "", false
	}
	return key, true
}

type ctxKey struct{}

// WithIdentity attaches the authenticated caller to ctx.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// IdentityFrom returns the caller attached by WithIdentity.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(ctxKey{}).(Identity)
	return id, ok
}

// permissionRankValues caches the numeric ranking of every permission
// level so that Grants checks avoid recomputing the switch on every
// request. It is populated once at startup and read-only afterwards.
var permissionRankValues = map[Permission]int{}

// init registers the numeric ranking of every known permission level.
func init() {
	permissionRankValues[Read] = 1
	permissionRankValues[ReadMovements] = 2
	permissionRankValues[Write] = 3
	permissionRankValues[Admin] = 4
}

// RankOf returns the cached numeric ranking of a permission level,
// or zero for unknown levels.
func RankOf(p Permission) int {
	if r, ok := permissionRankValues[p]; ok {
		return r
	} else {
		return 0
	}
}

// AllPermissions returns every known permission level in increasing
// order of capability.
func AllPermissions() []Permission {
	return []Permission{Read, ReadMovements, Write, Admin}
}

// NormalizePermissionName trims whitespace and lowercases a raw
// permission string before it is parsed.
func NormalizePermissionName(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// Authenticator abstracts every authentication capability of this
// package behind one interface, so that callers can depend on the
// abstraction rather than on the concrete package-level functions.
type Authenticator interface {
	Authenticate(plaintext string) (Identity, error)
	AdminExists() (bool, error)
	ParsePermission(s string) (Permission, error)
	HashKey(plaintext string) string
	GenerateKey() (plaintext, hash string, err error)
	IdentityFromRequest(r *http.Request) (Identity, error)
}

// PermissionManager abstracts permission ranking and comparison so
// that authorization policy can be reasoned about through one type.
type PermissionManager interface {
	Grants(have, need Permission) bool
	Rank(p Permission) int
	Normalize(raw string) (Permission, error)
	All() []Permission
}
