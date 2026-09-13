package httpapi

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// setupChain creates vendor/item/vendor-item/warehouse/A-01/A-02 via
// the API, returning their IDs.
func setupChain(t *testing.T, s *Server, admin string) (viID, wID, l1, l2 int64) {
	t.Helper()
	vID := idOf(t, post(t, s, "/api/vendors",
		map[string]string{"name": "Acme", "code": "ACME"}, admin, http.StatusCreated), "vendor")
	iID := idOf(t, post(t, s, "/api/items",
		map[string]any{"sku": "W-1", "name": "Widget"}, admin, http.StatusCreated), "item")
	viID = idOf(t, post(t, s, "/api/vendor-items", map[string]any{
		"vendor_id": vID, "item_id": iID, "vendor_sku": "BW-123",
	}, admin, http.StatusCreated), "vendor_item")
	wID = idOf(t, post(t, s, "/api/warehouses",
		map[string]any{"name": "Berlin", "code": "WH-BER"}, admin, http.StatusCreated), "warehouse")
	l1 = idOf(t, post(t, s, fmt.Sprintf("/api/warehouses/%d/locations", wID),
		map[string]string{"code": "A-01", "name": "Shelf 1"}, admin, http.StatusCreated), "location")
	l2 = idOf(t, post(t, s, fmt.Sprintf("/api/warehouses/%d/locations", wID),
		map[string]string{"code": "A-02", "name": "Shelf 2"}, admin, http.StatusCreated), "location")
	return viID, wID, l1, l2
}

func movementOf(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	m, ok := body["movement"].(map[string]any)
	if !ok {
		t.Fatalf("no movement in %v", body)
	}
	return m
}

func TestStockMutationsFlow(t *testing.T) {
	s := newTestServer(t)
	_, admin := bootstrap(t, s, "admin")
	read := mintKey(t, s, admin, "ro", "read")
	mover := mintKey(t, s, admin, "mo", "read-movements")
	write := mintKey(t, s, admin, "rw", "write")
	viID, wID, l1, l2 := setupChain(t, s, admin)

	slot := func(loc int64, qty int64) map[string]any {
		return map[string]any{
			"vendor_item_id": viID, "warehouse_id": wID,
			"location_id": loc, "quantity": qty,
		}
	}

	// Receive 100 via write key; read key sees it.
	body := post(t, s, "/api/stock/receive", slot(l1, 100), write, http.StatusCreated)
	if m := movementOf(t, body); m["status"] != "executed" || m["type"] != "receive" {
		t.Fatalf("receive movement: %v", m)
	}
	got := get(t, s, fmt.Sprintf("/api/stock?location_id=%d", l1), read, http.StatusOK)
	if rows := got["stock"].([]any); len(rows) != 1 || rows[0].(map[string]any)["quantity"] != 100.0 {
		t.Fatalf("stock after receive: %v", got)
	}

	// Remove 30, then oversell → 422 insufficient_stock with level unchanged.
	post(t, s, "/api/stock/remove", slot(l1, 30), write, http.StatusCreated)
	rr := call(t, s, "POST", "/api/stock/remove", slot(l1, 71), write)
	if rr.Code != 422 || errCode(t, rr) != "insufficient_stock" {
		t.Fatalf("oversell = %d %s, want 422 insufficient_stock", rr.Code, rr.Body.String())
	}

	// Transfer 40 atomically; failed transfer leaves both sides alone.
	post(t, s, "/api/stock/transfer", map[string]any{
		"vendor_item_id": viID, "warehouse_id": wID,
		"from_location_id": l1, "to_location_id": l2, "quantity": 40,
	}, write, http.StatusCreated)
	rr = call(t, s, "POST", "/api/stock/transfer", map[string]any{
		"vendor_item_id": viID, "warehouse_id": wID,
		"from_location_id": l1, "to_location_id": l2, "quantity": 31,
	}, write)
	// l1 holds 70-40=30 after the good transfer; moving 31 must fail.
	if rr.Code != 422 || errCode(t, rr) != "insufficient_stock" {
		t.Fatalf("oversell transfer = %d %s, want 422", rr.Code, rr.Body.String())
	}
	got = get(t, s, fmt.Sprintf("/api/stock?location_id=%d", l1), read, http.StatusOK)
	if rows := got["stock"].([]any); rows[0].(map[string]any)["quantity"] != 30.0 {
		t.Fatalf("source after failed transfer: %v", got)
	}

	// Same-slot transfer is a 400.
	rr = call(t, s, "POST", "/api/stock/transfer", map[string]any{
		"vendor_item_id": viID, "warehouse_id": wID,
		"from_location_id": l1, "to_location_id": l1, "quantity": 1,
	}, write)
	if rr.Code != http.StatusBadRequest || errCode(t, rr) != "validation" {
		t.Fatalf("same-slot = %d %s, want 400 validation", rr.Code, rr.Body.String())
	}

	// Adjust is absolute + requires a note.
	rr = call(t, s, "POST", "/api/stock/adjust", map[string]any{
		"vendor_item_id": viID, "warehouse_id": wID, "location_id": l1, "quantity": 25,
	}, write)
	if rr.Code != http.StatusBadRequest || errCode(t, rr) != "validation" {
		t.Fatalf("noteless adjust = %d %s, want 400 validation", rr.Code, rr.Body.String())
	}
	adj := post(t, s, "/api/stock/adjust", map[string]any{
		"vendor_item_id": viID, "warehouse_id": wID,
		"location_id": l1, "quantity": 25, "note": "recount",
	}, write, http.StatusCreated)
	if m := movementOf(t, adj); m["quantity"] != 5.0 {
		t.Fatalf("adjust delta: %v", m)
	}

	// Planned receive: no stock effect, hidden from `read` history.
	future := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	planBody := slot(l1, 50)
	planBody["planned_at"] = future
	planBody["reference"] = "PO-9"
	plan := post(t, s, "/api/stock/receive", planBody, write, http.StatusCreated)
	pm := movementOf(t, plan)
	if pm["status"] != "planned" {
		t.Fatalf("planned: %v", pm)
	}
	planID := int64(pm["id"].(float64))

	rr = call(t, s, "GET", "/api/stock/movements", nil, read)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("read key on movements = %d, want 403", rr.Code)
	}
	moves := get(t, s, "/api/stock/movements?status=planned", mover, http.StatusOK)
	if len(moves["movements"].([]any)) != 1 {
		t.Fatalf("planned movements: %v", moves)
	}

	// Execute applies; re-execute is a no-op; cancel-after-execute is 400.
	exec := post(t, s, fmt.Sprintf("/api/stock/movements/%d/execute", planID), nil, write, http.StatusOK)
	if movementOf(t, exec)["status"] != "executed" {
		t.Fatalf("execute: %v", exec)
	}
	post(t, s, fmt.Sprintf("/api/stock/movements/%d/execute", planID), nil, write, http.StatusOK)
	rr = call(t, s, "POST", fmt.Sprintf("/api/stock/movements/%d/cancel", planID), nil, write)
	if rr.Code != http.StatusBadRequest || errCode(t, rr) != "validation" {
		t.Fatalf("cancel executed = %d %s, want 400", rr.Code, rr.Body.String())
	}

	// Plan + cancel a remove; executing it afterwards is a 400.
	planBody = slot(l1, 5)
	planBody["planned_at"] = future
	plan2 := post(t, s, "/api/stock/remove", planBody, write, http.StatusCreated)
	cancelID := int64(movementOf(t, plan2)["id"].(float64))
	cancelled := post(t, s, fmt.Sprintf("/api/stock/movements/%d/cancel", cancelID), nil, write, http.StatusOK)
	if movementOf(t, cancelled)["status"] != "cancelled" {
		t.Fatalf("cancel: %v", cancelled)
	}
	rr = call(t, s, "POST", fmt.Sprintf("/api/stock/movements/%d/execute", cancelID), nil, write)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("execute cancelled = %d, want 400", rr.Code)
	}

	// Unknown movement is a 404; garbage planned_at is a 400.
	rr = call(t, s, "POST", "/api/stock/movements/999/execute", nil, write)
	if rr.Code != http.StatusNotFound || errCode(t, rr) != "not_found" {
		t.Fatalf("execute missing = %d %s, want 404", rr.Code, rr.Body.String())
	}
	bad := slot(l1, 1)
	bad["planned_at"] = "not-a-time"
	rr = call(t, s, "POST", "/api/stock/receive", bad, write)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad planned_at = %d, want 400", rr.Code)
	}

	// Permissions: read key cannot mutate; missing key is 401.
	rr = call(t, s, "POST", "/api/stock/receive", slot(l1, 1), read)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("mutate with read = %d, want 403", rr.Code)
	}
	rr = call(t, s, "POST", "/api/stock/receive", slot(l1, 1), "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("mutate without key = %d, want 401", rr.Code)
	}
}
