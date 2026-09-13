package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"warehouse-server/internal/auth"
	"warehouse-server/internal/users"
)

type createUserRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// handleCreateUser serves POST /api/users, which is dual-purpose: it is
// the unauthenticated first-admin bootstrap while no active admin
// exists, and an admin-only user-creation endpoint afterwards.
// Bootstrap additionally mints the admin key and returns it once.
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if !decodeBody(w, r, &req) {
		return
	}
	exists, err := auth.AdminExists(s.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if !exists {
		u, err := users.CreateUser(s.db, req.Username, req.DisplayName)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		plaintext, _, err := users.CreateAPIKey(s.db, u.ID, "bootstrap", auth.Admin)
		if err != nil {
			writeDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"user": u, "api_key": plaintext})
		return
	}
	id, err := auth.FromRequest(s.db, r)
	if errors.Is(err, auth.ErrUnauthenticated) {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	if !id.Permission.Grants(auth.Admin) {
		writeError(w, http.StatusForbidden, "forbidden", "insufficient permission")
		return
	}
	u, err := users.CreateUser(s.db, req.Username, req.DisplayName)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": u})
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	list, err := users.ListUsers(s.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": list})
}

func (s *Server) handleDisableUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	u, err := users.DisableUser(s.db, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u})
}

type createKeyRequest struct {
	Name       string `json:"name"`
	Permission string `json:"permission"`
}

func (s *Server) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	uid, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req createKeyRequest
	if !decodeBody(w, r, &req) {
		return
	}
	perm, err := auth.ParsePermission(req.Permission)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	plaintext, k, err := users.CreateAPIKey(s.db, uid, req.Name, perm)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"api_key": plaintext, "key": k})
}

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	uid, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	keys, err := users.ListKeys(s.db, uid)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

func (s *Server) handleRevokeKey(w http.ResponseWriter, r *http.Request) {
	uid, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	kid, ok := pathID(w, r, "kid")
	if !ok {
		return
	}
	k, err := users.RevokeKey(s.db, uid, kid)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"key": k})
}

// pathID parses an integer path parameter, writing 400 on garbage.
func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	raw := r.PathValue(name)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "validation", "invalid "+name)
		return 0, false
	}
	return id, true
}
