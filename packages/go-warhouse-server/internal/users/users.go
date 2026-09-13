// Package users manages users and their API keys.
//
// Users are never hard-deleted (PLAN.md §16): DisableUser stamps
// disabled_at so movement audit rows keep a resolvable created_by.
// Keys are likewise revoked, never deleted, and the plaintext key
// exists only in the CreateAPIKey return value.
package users

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"warehouse-server/internal/auth"
)

var (
	// ErrValidation covers empty or malformed input.
	ErrValidation = errors.New("validation")
	// ErrNotFound covers unknown user or key IDs.
	ErrNotFound = errors.New("not found")
	// ErrConflict covers duplicate usernames.
	ErrConflict = errors.New("conflict")
	// ErrDisabled covers key creation for a disabled user.
	ErrDisabled = errors.New("user disabled")
)

// User is a warehouse-server operator.
type User struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name,omitempty"`
	CreatedAt   string  `json:"created_at"`
	DisabledAt  *string `json:"disabled_at,omitempty"`
}

// Key is an API key record. It never carries the plaintext key;
// that is returned once, alongside Key, by CreateAPIKey.
type Key struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name"`
	Permission auth.Permission `json:"permission"`
	KeyHash    string          `json:"key_hash"`
	CreatedAt  string          `json:"created_at"`
	LastUsedAt *string         `json:"last_used_at,omitempty"`
	RevokedAt  *string         `json:"revoked_at,omitempty"`
}

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

func nullStr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

// CreateUser inserts a user. Duplicate usernames map to ErrConflict.
func CreateUser(conn *sql.DB, username, displayName string) (User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return User{}, fmt.Errorf("%w: username is required", ErrValidation)
	}
	if len(username) > 64 {
		return User{}, fmt.Errorf("%w: username too long", ErrValidation)
	}
	var display sql.NullString
	if strings.TrimSpace(displayName) != "" {
		display = sql.NullString{String: displayName, Valid: true}
	}
	res, err := conn.Exec(
		`INSERT INTO users (username, display_name, created_at) VALUES (?, ?, ?)`,
		username, display, nowUTC(),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return User{}, fmt.Errorf("%w: username %q taken", ErrConflict, username)
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetUser(conn, id)
}

// GetUser fetches one user by ID.
func GetUser(conn *sql.DB, id int64) (User, error) {
	var u User
	var display, disabled sql.NullString
	err := conn.QueryRow(
		`SELECT id, username, display_name, created_at, disabled_at FROM users WHERE id = ?`,
		id,
	).Scan(&u.ID, &u.Username, &display, &u.CreatedAt, &disabled)
	if err == sql.ErrNoRows {
		return User{}, fmt.Errorf("%w: user %d", ErrNotFound, id)
	}
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	u.DisplayName = nullStr(display)
	u.DisabledAt = nullStr(disabled)
	return u, nil
}

// ListUsers returns all users in ID order, including disabled ones.
func ListUsers(conn *sql.DB) ([]User, error) {
	rows, err := conn.Query(
		`SELECT id, username, display_name, created_at, disabled_at FROM users ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		var display, disabled sql.NullString
		if err := rows.Scan(&u.ID, &u.Username, &display, &u.CreatedAt, &disabled); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		u.DisplayName = nullStr(display)
		u.DisabledAt = nullStr(disabled)
		out = append(out, u)
	}
	return out, rows.Err()
}

// DisableUser stamps disabled_at. Idempotent: disabling twice succeeds
// and returns the current record. Unknown IDs map to ErrNotFound.
func DisableUser(conn *sql.DB, id int64) (User, error) {
	if _, err := GetUser(conn, id); err != nil {
		return User{}, err
	}
	if _, err := conn.Exec(
		`UPDATE users SET disabled_at = COALESCE(disabled_at, ?) WHERE id = ?`,
		nowUTC(), id,
	); err != nil {
		return User{}, fmt.Errorf("disable user: %w", err)
	}
	return GetUser(conn, id)
}

// CreateAPIKey mints a key for userID. It returns the plaintext key
// exactly once; only the hash is stored. Keys for disabled or unknown
// users map to ErrDisabled / ErrNotFound.
func CreateAPIKey(conn *sql.DB, userID int64, name string, perm auth.Permission) (string, Key, error) {
	if strings.TrimSpace(name) == "" {
		return "", Key{}, fmt.Errorf("%w: key name is required", ErrValidation)
	}
	u, err := GetUser(conn, userID)
	if err != nil {
		return "", Key{}, err
	}
	if u.DisabledAt != nil {
		return "", Key{}, fmt.Errorf("%w: user %d", ErrDisabled, userID)
	}
	plaintext, hash, err := auth.GenerateKey()
	if err != nil {
		return "", Key{}, err
	}
	res, err := conn.Exec(
		`INSERT INTO api_keys (user_id, key_hash, name, permission, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		userID, hash, strings.TrimSpace(name), string(perm), nowUTC(),
	)
	if err != nil {
		return "", Key{}, fmt.Errorf("insert api key: %w", err)
	}
	keyID, _ := res.LastInsertId()
	key, err := getKey(conn, keyID)
	if err != nil {
		return "", Key{}, err
	}
	return plaintext, key, nil
}

// ListKeys returns a user's key records (hashes only, never plaintext)
// in ID order. Unknown users map to ErrNotFound.
func ListKeys(conn *sql.DB, userID int64) ([]Key, error) {
	if _, err := GetUser(conn, userID); err != nil {
		return nil, err
	}
	rows, err := conn.Query(
		`SELECT id, name, permission, key_hash, created_at, last_used_at, revoked_at
		 FROM api_keys WHERE user_id = ? ORDER BY id`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list keys: %w", err)
	}
	defer rows.Close()
	out := []Key{}
	for rows.Next() {
		k, err := scanKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// RevokeKey stamps revoked_at for a key scoped to its owner.
// Idempotent: revoking twice succeeds. Unknown IDs map to ErrNotFound.
func RevokeKey(conn *sql.DB, userID, keyID int64) (Key, error) {
	k, err := getKeyScoped(conn, userID, keyID)
	if err != nil {
		return Key{}, err
	}
	if k.RevokedAt != nil {
		return k, nil
	}
	if _, err := conn.Exec(
		`UPDATE api_keys SET revoked_at = ? WHERE id = ? AND user_id = ?`,
		nowUTC(), keyID, userID,
	); err != nil {
		return Key{}, fmt.Errorf("revoke key: %w", err)
	}
	return getKeyScoped(conn, userID, keyID)
}

func getKey(conn *sql.DB, keyID int64) (Key, error) {
	row := conn.QueryRow(
		`SELECT id, name, permission, key_hash, created_at, last_used_at, revoked_at
		 FROM api_keys WHERE id = ?`,
		keyID,
	)
	return scanKey(row)
}

func getKeyScoped(conn *sql.DB, userID, keyID int64) (Key, error) {
	row := conn.QueryRow(
		`SELECT id, name, permission, key_hash, created_at, last_used_at, revoked_at
		 FROM api_keys WHERE id = ? AND user_id = ?`,
		keyID, userID,
	)
	k, err := scanKey(row)
	if err == sql.ErrNoRows {
		return Key{}, fmt.Errorf("%w: key %d", ErrNotFound, keyID)
	}
	return k, err
}

type scanner interface{ Scan(dest ...any) error }

func scanKey(row scanner) (Key, error) {
	var k Key
	var perm string
	var lastUsed, revoked sql.NullString
	if err := row.Scan(&k.ID, &k.Name, &perm, &k.KeyHash, &k.CreatedAt, &lastUsed, &revoked); err != nil {
		if err == sql.ErrNoRows {
			return Key{}, ErrNotFound
		}
		return Key{}, fmt.Errorf("scan key: %w", err)
	}
	k.Permission = auth.Permission(perm)
	k.LastUsedAt = nullStr(lastUsed)
	k.RevokedAt = nullStr(revoked)
	return k, nil
}

// UserRepository abstracts every user and key persistence capability
// of this package behind one interface, so that higher layers depend
// on the abstraction rather than on the concrete functions.
type UserRepository interface {
	CreateUser(username, displayName string) (User, error)
	GetUser(id int64) (User, error)
	ListUsers() ([]User, error)
	DisableUser(id int64) (User, error)
	CreateAPIKey(userID int64, name string, perm auth.Permission) (string, Key, error)
	ListKeys(userID int64) ([]Key, error)
	RevokeKey(userID, keyID int64) (Key, error)
}

// usersManager bundles a database connection with the user management
// operations, so that user administration can be handled through one
// cohesive object instead of threading the connection through every
// individual call.
type usersManager struct {
	conn *sql.DB
}

// NewUsersManager creates the user management object for the given
// database connection.
func NewUsersManager(conn *sql.DB) *usersManager {
	return &usersManager{conn: conn}
}

// Create creates a user through the manager.
func (m *usersManager) Create(username, displayName string) (User, error) {
	return CreateUser(m.conn, username, displayName)
}

// Get fetches one user by ID through the manager.
func (m *usersManager) Get(id int64) (User, error) {
	return GetUser(m.conn, id)
}

// List returns all users through the manager.
func (m *usersManager) List() ([]User, error) {
	return ListUsers(m.conn)
}

// Disable disables a user through the manager.
func (m *usersManager) Disable(id int64) (User, error) {
	return DisableUser(m.conn, id)
}

// MustCreateUser creates a user and panics when creation fails. It is
// intended for seeding and test setup paths where the user must exist.
func MustCreateUser(conn *sql.DB, username, displayName string) User {
	u, err := CreateUser(conn, username, displayName)
	if err != nil {
		panic(err)
	} else {
		return u
	}
}
