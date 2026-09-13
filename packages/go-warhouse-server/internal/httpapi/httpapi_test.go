package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"warehouse-server/internal/db"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := db.Migrate(conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return New(conn, "test")
}

func call(t *testing.T, s *Server, method, path string, body any, apiKey string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	body := decode(t, rec)
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("no error envelope in %v", body)
	}
	code, _ := errObj["code"].(string)
	return code
}

func bootstrap(t *testing.T, s *Server, username string) (userID float64, adminKey string) {
	t.Helper()
	rec := call(t, s, "POST", "/api/users", map[string]string{"username": username}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("bootstrap status = %d (%s)", rec.Code, rec.Body.String())
	}
	body := decode(t, rec)
	key, _ := body["api_key"].(string)
	if !strings.HasPrefix(key, "wks_") {
		t.Fatalf("no plaintext api_key in %v", body)
	}
	user, _ := body["user"].(map[string]any)
	id, _ := user["id"].(float64)
	return id, key
}

func TestHealthz(t *testing.T) {
	s := newTestServer(t)
	rec := call(t, s, "GET", "/healthz", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decode(t, rec)
	if body["status"] != "ok" || body["version"] != "test" {
		t.Fatalf("unexpected body: %v", body)
	}
}

func TestBootstrapFlow(t *testing.T) {
	s := newTestServer(t)

	_, adminKey := bootstrap(t, s, "admin")

	// Bootstrap closes after the first admin.
	rec := call(t, s, "POST", "/api/users", map[string]string{"username": "second"}, "")
	if rec.Code != http.StatusUnauthorized || errCode(t, rec) != "unauthenticated" {
		t.Fatalf("second bootstrap = %d %s, want 401 unauthenticated", rec.Code, rec.Body.String())
	}

	// Admin path works.
	rec = call(t, s, "GET", "/api/users", nil, adminKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("list users = %d %s", rec.Code, rec.Body.String())
	}
	if users := decode(t, rec)["users"].([]any); len(users) != 1 {
		t.Fatalf("users = %v, want 1", users)
	}

	// Admin creates a user without a key.
	rec = call(t, s, "POST", "/api/users",
		map[string]string{"username": "bob", "display_name": "Bob"}, adminKey)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create bob = %d %s", rec.Code, rec.Body.String())
	}
	body := decode(t, rec)
	if _, hasKey := body["api_key"]; hasKey {
		t.Fatalf("admin-created user must not include api_key: %v", body)
	}
	bobID := body["user"].(map[string]any)["id"].(float64)

	// Duplicate username conflicts.
	rec = call(t, s, "POST", "/api/users", map[string]string{"username": "bob"}, adminKey)
	if rec.Code != http.StatusConflict || errCode(t, rec) != "conflict" {
		t.Fatalf("duplicate = %d %s, want 409 conflict", rec.Code, rec.Body.String())
	}

	// Mint a read key for bob; list shows the hash, never plaintext.
	rec = call(t, s, "POST", fmt.Sprintf("/api/users/%d/keys", int64(bobID)),
		map[string]string{"name": "ro", "permission": "read"}, adminKey)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create key = %d %s", rec.Code, rec.Body.String())
	}
	keyBody := decode(t, rec)
	bobKey, _ := keyBody["api_key"].(string)
	if !strings.HasPrefix(bobKey, "wks_") {
		t.Fatalf("no plaintext key: %v", keyBody)
	}
	bobKeyID := int64(keyBody["key"].(map[string]any)["id"].(float64))
	rec = call(t, s, "GET", fmt.Sprintf("/api/users/%d/keys", int64(bobID)), nil, adminKey)
	keysBody := decode(t, rec)
	keys := keysBody["keys"].([]any)
	if len(keys) != 1 {
		t.Fatalf("keys = %v", keysBody)
	}
	k := keys[0].(map[string]any)
	if _, hasPlain := k["api_key"]; hasPlain {
		t.Fatalf("key listing leaks plaintext: %v", k)
	}
	if k["key_hash"] == bobKey {
		t.Fatal("key_hash equals plaintext")
	}

	// Read key is forbidden on admin routes.
	rec = call(t, s, "GET", "/api/users", nil, bobKey)
	if rec.Code != http.StatusForbidden || errCode(t, rec) != "forbidden" {
		t.Fatalf("read key on admin route = %d %s, want 403 forbidden", rec.Code, rec.Body.String())
	}

	// Revoke bob's key; it stops working.
	rec = call(t, s, "POST", fmt.Sprintf("/api/users/%d/keys/%d/revoke", int64(bobID), bobKeyID), nil, adminKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke = %d %s", rec.Code, rec.Body.String())
	}
	rec = call(t, s, "GET", "/api/users", nil, bobKey)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("use after admin still... bob revoked key = %d, want 401", rec.Code)
	}

	// Disable bob; listing still shows him with disabled_at.
	rec = call(t, s, "POST", fmt.Sprintf("/api/users/%d/disable", int64(bobID)), nil, adminKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable = %d %s", rec.Code, rec.Body.String())
	}
	if u := decode(t, rec)["user"].(map[string]any); u["disabled_at"] == nil {
		t.Fatalf("disabled_at missing: %v", u)
	}
}

func TestBootstrapReopensAfterAdminDisabled(t *testing.T) {
	s := newTestServer(t)
	_, adminKey := bootstrap(t, s, "admin")

	rec := call(t, s, "POST", "/api/users/1/disable", nil, adminKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable admin = %d %s", rec.Code, rec.Body.String())
	}
	// Old key stops working and bootstrap reopens (no lockout).
	rec = call(t, s, "GET", "/api/users", nil, adminKey)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("disabled admin key = %d, want 401", rec.Code)
	}
	rec = call(t, s, "POST", "/api/users", map[string]string{"username": "admin2"}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("re-bootstrap = %d %s, want 201", rec.Code, rec.Body.String())
	}
}

func TestPermissionMatrix(t *testing.T) {
	s := newTestServer(t)
	_, adminKey := bootstrap(t, s, "admin")

	// Mint one key per level through the API.
	levels := []string{"read", "read-movements", "write", "admin"}
	keys := map[string]string{}
	for _, lvl := range levels {
		rec := call(t, s, "POST", "/api/users/1/keys",
			map[string]string{"name": lvl, "permission": lvl}, adminKey)
		if rec.Code != http.StatusCreated {
			t.Fatalf("mint %s key = %d %s", lvl, rec.Code, rec.Body.String())
		}
		keys[lvl] = decode(t, rec)["api_key"].(string)
	}

	for _, lvl := range levels {
		rec := call(t, s, "GET", "/api/users", nil, keys[lvl])
		want := http.StatusForbidden
		if lvl == "admin" {
			want = http.StatusOK
		}
		if rec.Code != want {
			t.Fatalf("%s key on admin route = %d, want %d", lvl, rec.Code, want)
		}
	}

	// No key, garbage key, malformed header.
	rec := call(t, s, "GET", "/api/users", nil, "")
	if rec.Code != http.StatusUnauthorized || errCode(t, rec) != "unauthenticated" {
		t.Fatalf("missing key = %d, want 401 unauthenticated", rec.Code)
	}
	rec = call(t, s, "GET", "/api/users", nil, "wks_garbage")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("garbage key = %d, want 401", rec.Code)
	}
	req := httptest.NewRequest("GET", "/api/users", nil)
	req.Header.Set("Authorization", "Token abc")
	rec2 := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec2, req)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("non-bearer scheme = %d, want 401", rec2.Code)
	}
}

func TestUserValidation(t *testing.T) {
	s := newTestServer(t)
	_, adminKey := bootstrap(t, s, "admin")

	rec := call(t, s, "POST", "/api/users", map[string]string{"username": " "}, adminKey)
	if rec.Code != http.StatusBadRequest || errCode(t, rec) != "validation" {
		t.Fatalf("empty username = %d, want 400 validation", rec.Code)
	}
	rec = call(t, s, "POST", "/api/users/1/keys",
		map[string]string{"name": "x", "permission": "root"}, adminKey)
	if rec.Code != http.StatusBadRequest || errCode(t, rec) != "validation" {
		t.Fatalf("bad permission = %d, want 400 validation", rec.Code)
	}
	rec = call(t, s, "GET", "/api/users/abc/keys", nil, adminKey)
	if rec.Code != http.StatusBadRequest || errCode(t, rec) != "validation" {
		t.Fatalf("bad id = %d, want 400 validation", rec.Code)
	}
	rec = call(t, s, "GET", "/api/users/999/keys", nil, adminKey)
	if rec.Code != http.StatusNotFound || errCode(t, rec) != "not_found" {
		t.Fatalf("missing user = %d, want 404 not_found", rec.Code)
	}
}
