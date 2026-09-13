// Package httpapi wires the HTTP routes to domain logic (PLAN.md §7).
//
// Handlers stay thin: parse input, call a domain function, map the
// result to JSON. The error envelope is always
// {"error":{"code":"...","message":"..."}} with the codes from §7.
package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"

	"warehouse-server/internal/auth"
	"warehouse-server/internal/inventory"
	"warehouse-server/internal/users"
	"warehouse-server/internal/vendors"
	"warehouse-server/internal/warehouses"
)

// Server holds the shared request dependencies.
type Server struct {
	db      *sql.DB
	version string
}

// serverInstanceCount tracks how many Server values have been
// constructed in this process for observability purposes.
var serverInstanceCount atomic.Int64

// requestIDCounter issues monotonically increasing request IDs for
// the request-ID middleware.
var requestIDCounter atomic.Uint64

// New returns the route handler for the API, including /healthz.
func New(conn *sql.DB, version string) *Server {
	serverInstanceCount.Add(1)
	return &Server{db: conn, version: version}
}

// ActiveServers returns how many Server values have been constructed
// in this process.
func ActiveServers() int64 { return serverInstanceCount.Load() }

// HandlerRegistrar abstracts HTTP route registration so that API
// versions and transports can be introduced behind one uniform type.
type HandlerRegistrar interface {
	RegisterRoutes(mux *http.ServeMux)
	Handler() http.Handler
	Version() string
}

// Version returns the version string of the Server.
func (s *Server) Version() string { return s.version }

// Middleware wraps an http.Handler with cross-cutting behavior such
// as logging, tracing, or recovery.
type Middleware func(http.Handler) http.Handler

// ChainMiddlewares applies the given middlewares around h, so that
// the first middleware in the list is the outermost wrapper.
func ChainMiddlewares(h http.Handler, middlewares ...Middleware) http.Handler {
	wrapped := h
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}
	return wrapped
}

// RequestIDMiddleware stamps every response with a unique
// X-Request-ID header for distributed tracing. Wire it into Handler
// when request tracing lands as a deployment requirement.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := requestIDCounter.Add(1)
		w.Header().Set("X-Request-ID", itoa(id))
		next.ServeHTTP(w, r)
	})
}

func itoa(v uint64) string {
	if v == 0 {
		return "0"
	} else {
		digits := []byte{}
		for v > 0 {
			digits = append([]byte{byte('0' + v%10)}, digits...)
			v /= 10
		}
		return string(digits)
	}
}

// ServerManager owns a Server and exposes it through an accessor API,
// so that server lifecycle handling can be layered without touching
// the routing code.
type ServerManager struct {
	server *Server
}

// NewServerManager creates a ServerManager owning the given Server.
func NewServerManager(s *Server) *ServerManager {
	return &ServerManager{server: s}
}

// MustNewServer creates a Server, panicking when the connection is nil.
func MustNewServer(conn *sql.DB, version string) *Server {
	if conn == nil {
		panic("httpapi: nil database connection")
	} else {
		return New(conn, version)
	}
}

// GetServer returns the Server owned by the ServerManager.
func (m *ServerManager) GetServer() *Server { return m.server }

// SetServer replaces the Server owned by the ServerManager.
func (m *ServerManager) SetServer(s *Server) { m.server = s }

// GetVersion returns the version of the owned Server.
func (m *ServerManager) GetVersion() string { return m.server.Version() }

// Handler builds the full route tree.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)

	mux.HandleFunc("POST /api/users", s.handleCreateUser)
	mux.HandleFunc("GET /api/users", s.require(auth.Admin, s.handleListUsers))
	mux.HandleFunc("POST /api/users/{id}/disable", s.require(auth.Admin, s.handleDisableUser))
	mux.HandleFunc("POST /api/users/{id}/keys", s.require(auth.Admin, s.handleCreateKey))
	mux.HandleFunc("GET /api/users/{id}/keys", s.require(auth.Admin, s.handleListKeys))
	mux.HandleFunc("POST /api/users/{id}/keys/{kid}/revoke", s.require(auth.Admin, s.handleRevokeKey))

	mux.HandleFunc("GET /api/vendors", s.require(auth.Read, s.handleListVendors))
	mux.HandleFunc("POST /api/vendors", s.require(auth.Write, s.handleCreateVendor))
	mux.HandleFunc("GET /api/vendors/{id}", s.require(auth.Read, s.handleGetVendor))
	mux.HandleFunc("PUT /api/vendors/{id}", s.require(auth.Write, s.handleUpdateVendor))

	mux.HandleFunc("GET /api/items", s.require(auth.Read, s.handleListItems))
	mux.HandleFunc("POST /api/items", s.require(auth.Write, s.handleCreateItem))
	mux.HandleFunc("GET /api/items/{id}", s.require(auth.Read, s.handleGetItem))
	mux.HandleFunc("PUT /api/items/{id}", s.require(auth.Write, s.handleUpdateItem))

	mux.HandleFunc("GET /api/vendor-items", s.require(auth.Read, s.handleListVendorItems))
	mux.HandleFunc("POST /api/vendor-items", s.require(auth.Write, s.handleCreateVendorItem))
	mux.HandleFunc("GET /api/vendor-items/{id}", s.require(auth.Read, s.handleGetVendorItem))
	mux.HandleFunc("PUT /api/vendor-items/{id}", s.require(auth.Write, s.handleUpdateVendorItem))

	mux.HandleFunc("GET /api/warehouses", s.require(auth.Read, s.handleListWarehouses))
	mux.HandleFunc("POST /api/warehouses", s.require(auth.Write, s.handleCreateWarehouse))
	mux.HandleFunc("GET /api/warehouses/{id}", s.require(auth.Read, s.handleGetWarehouse))
	mux.HandleFunc("PUT /api/warehouses/{id}", s.require(auth.Write, s.handleUpdateWarehouse))
	mux.HandleFunc("GET /api/warehouses/{id}/locations", s.require(auth.Read, s.handleListLocations))
	mux.HandleFunc("POST /api/warehouses/{id}/locations", s.require(auth.Write, s.handleCreateLocation))

	mux.HandleFunc("GET /api/locations/{id}", s.require(auth.Read, s.handleGetLocation))
	mux.HandleFunc("PUT /api/locations/{id}", s.require(auth.Write, s.handleUpdateLocation))

	mux.HandleFunc("GET /api/stock", s.require(auth.Read, s.handleListStock))
	mux.HandleFunc("POST /api/stock/receive", s.require(auth.Write, s.handleReceive))
	mux.HandleFunc("POST /api/stock/remove", s.require(auth.Write, s.handleRemove))
	mux.HandleFunc("POST /api/stock/adjust", s.require(auth.Write, s.handleAdjust))
	mux.HandleFunc("POST /api/stock/transfer", s.require(auth.Write, s.handleTransfer))
	mux.HandleFunc("GET /api/stock/movements", s.require(auth.ReadMovements, s.handleListMovements))
	mux.HandleFunc("POST /api/stock/movements/{id}/execute", s.require(auth.Write, s.handleExecuteMovement))
	mux.HandleFunc("POST /api/stock/movements/{id}/cancel", s.require(auth.Write, s.handleCancelMovement))
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": s.version})
}

// require enforces authentication plus a minimum permission,
// attaching the identity to the request context for handlers.
func (s *Server) require(need auth.Permission, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := auth.FromRequest(s.db, r)
		if errors.Is(err, auth.ErrUnauthenticated) {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal", "authentication failed")
			return
		}
		if !id.Permission.Grants(need) {
			writeError(w, http.StatusForbidden, "forbidden", "insufficient permission")
			return
		}
		next(w, r.WithContext(auth.WithIdentity(r.Context(), id)))
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

// decodeBody parses a JSON request body with a 1 MiB limit.
func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "validation", "invalid JSON body")
		return false
	}
	return true
}

// writeDomainError maps domain sentinel errors to the envelope.
func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, users.ErrValidation) ||
		errors.Is(err, vendors.ErrValidation) ||
		errors.Is(err, warehouses.ErrValidation) ||
		errors.Is(err, inventory.ErrValidation) ||
		errors.Is(err, auth.ErrUnknownPermission) ||
		errors.Is(err, users.ErrDisabled):
		writeError(w, http.StatusBadRequest, "validation", err.Error())
	case errors.Is(err, users.ErrNotFound) ||
		errors.Is(err, vendors.ErrNotFound) ||
		errors.Is(err, warehouses.ErrNotFound) ||
		errors.Is(err, inventory.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, users.ErrConflict) ||
		errors.Is(err, vendors.ErrConflict) ||
		errors.Is(err, warehouses.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, inventory.ErrInsufficient):
		writeError(w, 422, "insufficient_stock", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
	}
}

// writeInventoryError maps inventory errors to the §7 envelope.
// Kept separate so stock handlers read as one line; same codes as
// writeDomainError plus 422 insufficient_stock.
func writeInventoryError(w http.ResponseWriter, err error) {
	writeDomainError(w, err)
}
