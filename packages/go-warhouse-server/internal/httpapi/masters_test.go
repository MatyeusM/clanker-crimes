package httpapi

import (
	"fmt"
	"net/http"
	"testing"
)

func post(t *testing.T, s *Server, path string, body any, key string, want int) map[string]any {
	t.Helper()
	rec := call(t, s, "POST", path, body, key)
	if rec.Code != want {
		t.Fatalf("POST %s = %d (%s), want %d", path, rec.Code, rec.Body.String(), want)
	}
	return decode(t, rec)
}

func get(t *testing.T, s *Server, path, key string, want int) map[string]any {
	t.Helper()
	rec := call(t, s, "GET", path, nil, key)
	if rec.Code != want {
		t.Fatalf("GET %s = %d (%s), want %d", path, rec.Code, rec.Body.String(), want)
	}
	return decode(t, rec)
}

func put(t *testing.T, s *Server, path string, body any, key string, want int) map[string]any {
	t.Helper()
	rec := call(t, s, "PUT", path, body, key)
	if rec.Code != want {
		t.Fatalf("PUT %s = %d (%s), want %d", path, rec.Code, rec.Body.String(), want)
	}
	return decode(t, rec)
}

func idOf(t *testing.T, body map[string]any, resource string) int64 {
	t.Helper()
	res, ok := body[resource].(map[string]any)
	if !ok {
		t.Fatalf("no %q in %v", resource, body)
	}
	id, ok := res["id"].(float64)
	if !ok {
		t.Fatalf("no id in %v", res)
	}
	return int64(id)
}

func mintKey(t *testing.T, s *Server, adminKey, name, perm string) string {
	t.Helper()
	body := post(t, s, "/api/users/1/keys",
		map[string]string{"name": name, "permission": perm}, adminKey, http.StatusCreated)
	key, _ := body["api_key"].(string)
	return key
}

func TestMastersFlow(t *testing.T) {
	s := newTestServer(t)
	_, admin := bootstrap(t, s, "admin")

	vendor := post(t, s, "/api/vendors",
		map[string]string{"name": "Acme GmbH", "code": "ACME"}, admin, http.StatusCreated)
	vendorID := idOf(t, vendor, "vendor")
	post(t, s, "/api/vendors",
		map[string]string{"name": "Dupe", "code": "ACME"}, admin, http.StatusConflict)
	put(t, s, fmt.Sprintf("/api/vendors/%d", vendorID),
		map[string]string{"name": "Acme AG", "code": "ACME"}, admin, http.StatusOK)
	got := get(t, s, fmt.Sprintf("/api/vendors/%d", vendorID), admin, http.StatusOK)
	if got["vendor"].(map[string]any)["name"] != "Acme AG" {
		t.Fatalf("vendor not updated: %v", got)
	}
	get(t, s, "/api/vendors/999", admin, http.StatusNotFound)

	item := post(t, s, "/api/items",
		map[string]any{"sku": "ABC-123", "name": "Widget", "unit": "pcs"}, admin, http.StatusCreated)
	itemID := idOf(t, item, "item")
	post(t, s, "/api/items",
		map[string]any{"sku": "ABC-123", "name": "Dupe"}, admin, http.StatusConflict)

	// Vendor item needs existing sides; unknown vendor is a 404.
	post(t, s, "/api/vendor-items",
		map[string]any{"vendor_id": 999, "item_id": itemID, "vendor_sku": "X"},
		admin, http.StatusNotFound)
	vi := post(t, s, "/api/vendor-items",
		map[string]any{"vendor_id": vendorID, "item_id": itemID, "vendor_sku": "BW-123"},
		admin, http.StatusCreated)
	viID := idOf(t, vi, "vendor_item")
	post(t, s, "/api/vendor-items",
		map[string]any{"vendor_id": vendorID, "item_id": itemID, "vendor_sku": "BW-123"},
		admin, http.StatusConflict)
	filtered := get(t, s, fmt.Sprintf("/api/vendor-items?vendor_id=%d", vendorID), admin, http.StatusOK)
	if len(filtered["vendor_items"].([]any)) != 1 {
		t.Fatalf("filtered vendor-items: %v", filtered)
	}
	rec := call(t, s, "GET", "/api/vendor-items?vendor_id=abc", nil, admin)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad filter = %d, want 400", rec.Code)
	}
	put(t, s, fmt.Sprintf("/api/vendor-items/%d", viID),
		map[string]any{"vendor_sku": "BW-124"}, admin, http.StatusOK)

	ber := post(t, s, "/api/warehouses",
		map[string]any{"name": "Berlin", "code": "WH-BER"}, admin, http.StatusCreated)
	berID := idOf(t, ber, "warehouse")
	ham := post(t, s, "/api/warehouses",
		map[string]any{"name": "Hamburg", "code": "WH-HAM"}, admin, http.StatusCreated)
	hamID := idOf(t, ham, "warehouse")
	post(t, s, "/api/warehouses",
		map[string]any{"name": "Dupe", "code": "WH-BER"}, admin, http.StatusConflict)

	loc := post(t, s, fmt.Sprintf("/api/warehouses/%d/locations", berID),
		map[string]string{"code": "A-01", "name": "Shelf"}, admin, http.StatusCreated)
	locID := idOf(t, loc, "location")
	// Same code in Hamburg is fine; in Berlin it conflicts.
	post(t, s, fmt.Sprintf("/api/warehouses/%d/locations", hamID),
		map[string]string{"code": "A-01", "name": "Shelf"}, admin, http.StatusCreated)
	post(t, s, fmt.Sprintf("/api/warehouses/%d/locations", berID),
		map[string]string{"code": "A-01", "name": "Dupe"}, admin, http.StatusConflict)
	post(t, s, "/api/warehouses/999/locations",
		map[string]string{"code": "Z-9", "name": "Nowhere"}, admin, http.StatusNotFound)
	put(t, s, fmt.Sprintf("/api/locations/%d", locID),
		map[string]string{"code": "A-02", "name": "Shelf 2"}, admin, http.StatusOK)
	get(t, s, fmt.Sprintf("/api/locations/%d", locID), admin, http.StatusOK)
	list := get(t, s, fmt.Sprintf("/api/warehouses/%d/locations", berID), admin, http.StatusOK)
	if len(list["locations"].([]any)) != 1 {
		t.Fatalf("berlin locations: %v", list)
	}
}

func TestMastersPermissions(t *testing.T) {
	s := newTestServer(t)
	_, admin := bootstrap(t, s, "admin")
	read := mintKey(t, s, admin, "ro", "read")
	movement := mintKey(t, s, admin, "mo", "read-movements")
	write := mintKey(t, s, admin, "rw", "write")

	// Reads work for every non-admin level.
	for _, key := range []string{read, movement, write, admin} {
		get(t, s, "/api/vendors", key, http.StatusOK)
		get(t, s, "/api/items", key, http.StatusOK)
		get(t, s, "/api/vendor-items", key, http.StatusOK)
		get(t, s, "/api/warehouses", key, http.StatusOK)
	}

	// Writes need `write` or `admin`.
	vendorBody := map[string]string{"name": "Acme", "code": "ACME"}
	post(t, s, "/api/vendors", vendorBody, write, http.StatusCreated)
	for _, key := range []string{read, movement} {
		rec := call(t, s, "POST", "/api/vendors", vendorBody, key)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("write with lower key = %d, want 403", rec.Code)
		}
	}
	rec := call(t, s, "POST", "/api/vendors", vendorBody, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("write without key = %d, want 401", rec.Code)
	}
	rec = call(t, s, "PUT", "/api/vendors/1", vendorBody, read)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("PUT with read key = %d, want 403", rec.Code)
	}
	rec = call(t, s, "POST", "/api/warehouses/1/locations",
		map[string]string{"code": "A-01", "name": "S"}, movement)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("location write with read-movements = %d, want 403", rec.Code)
	}
}
