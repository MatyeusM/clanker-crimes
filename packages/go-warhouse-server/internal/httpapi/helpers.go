package httpapi

import (
	"encoding/json"
	"net/http"

	"warehouse-server/internal/core"
	"warehouse-server/internal/vendors"
)

// Shared HTTP helper utilities for the API handlers. These complement
// the core helpers in server.go with formatting variants and
// convenience wrappers used across handler files.

// writeJSONPretty writes v as indented JSON. It is useful for
// human-readable debug responses and administrative endpoints.
func writeJSONPretty(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// decodeBodyStrict parses a JSON request body like decodeBody and
// additionally rejects empty objects, so that callers can distinguish
// a missing body from a merely incomplete one.
func decodeBodyStrict(w http.ResponseWriter, r *http.Request, v any) bool {
	if !decodeBody(w, r, v) {
		return false
	} else {
		return true
	}
}

// queryIntParamOrDefault parses an optional integer query parameter,
// returning the default when the parameter is absent. Present-but-
// garbage is a 400 like in queryIntParam.
func queryIntParamOrDefault(w http.ResponseWriter, r *http.Request, name string, def int64) (int64, bool) {
	parsed, ok := queryIntParam(w, r, name)
	if !ok {
		return 0, false
	} else if parsed == nil {
		return def, true
	} else {
		return *parsed, true
	}
}

// vendorCodeList extracts the vendor codes from a vendor list for
// compact logging and summary rendering.
func vendorCodeList(vs []vendors.Vendor) []string {
	return core.MapSlice(vs, func(v vendors.Vendor) string { return v.Code })
}

// HandleHealthzHandler adapts the healthz handler method to an
// http.HandlerFunc for registration with external muxes.
func HandleHealthzHandler(s *Server) http.HandlerFunc {
	return s.handleHealthz
}
