package httpapi

import (
	"fmt"
	"net/http"
	"testing"
)

// seedStockRow inserts a raw stock row for read tests. No mutation API
// exists until Phase 5, so tests write the row directly.
func seedStockRow(t *testing.T, s *Server, vendorItemID, warehouseID, locationID, qty int64) {
	t.Helper()
	if _, err := s.db.Exec(
		`INSERT INTO stock (vendor_item_id, warehouse_id, location_id, quantity)
		 VALUES (?, ?, ?, ?)`,
		vendorItemID, warehouseID, locationID, qty,
	); err != nil {
		t.Fatalf("seed stock: %v", err)
	}
}

func TestStockReadsFull(t *testing.T) {
	s := newTestServer(t)
	_, admin := bootstrap(t, s, "admin")
	read := mintKey(t, s, admin, "ro", "read")

	vID := idOf(t, post(t, s, "/api/vendors",
		map[string]string{"name": "Acme", "code": "ACME"}, admin, http.StatusCreated), "vendor")
	iID := idOf(t, post(t, s, "/api/items",
		map[string]any{"sku": "W-1", "name": "Widget"}, admin, http.StatusCreated), "item")
	viID := idOf(t, post(t, s, "/api/vendor-items", map[string]any{
		"vendor_id": vID, "item_id": iID, "vendor_sku": "BW-123",
	}, admin, http.StatusCreated), "vendor_item")
	wID := idOf(t, post(t, s, "/api/warehouses",
		map[string]any{"name": "Berlin", "code": "WH-BER"}, admin, http.StatusCreated), "warehouse")
	l1 := idOf(t, post(t, s, fmt.Sprintf("/api/warehouses/%d/locations", wID),
		map[string]string{"code": "A-01", "name": "Shelf 1"}, admin, http.StatusCreated), "location")
	l2 := idOf(t, post(t, s, fmt.Sprintf("/api/warehouses/%d/locations", wID),
		map[string]string{"code": "A-02", "name": "Shelf 2"}, admin, http.StatusCreated), "location")
	seedStockRow(t, s, viID, wID, l1, 250)
	seedStockRow(t, s, viID, wID, l2, 30)

	// Read-level key sees current stock.
	body := get(t, s, "/api/stock", read, http.StatusOK)
	rows := body["stock"].([]any)
	if len(rows) != 2 {
		t.Fatalf("stock rows = %d, want 2", len(rows))
	}
	first := rows[0].(map[string]any)
	if first["quantity"] != 250.0 || first["vendor_code"] != "ACME" ||
		first["location_code"] != "A-01" || first["item_sku"] != "W-1" {
		t.Fatalf("unenriched row: %v", first)
	}

	// Each filter narrows; unknown IDs give an empty list.
	filtered := get(t, s, fmt.Sprintf("/api/stock?location_id=%d", l2), read, http.StatusOK)
	if len(filtered["stock"].([]any)) != 1 {
		t.Fatalf("by location: %v", filtered)
	}
	for _, q := range []string{
		fmt.Sprintf("vendor_item_id=%d", viID),
		fmt.Sprintf("warehouse_id=%d", wID),
		fmt.Sprintf("item_id=%d", iID),
		fmt.Sprintf("vendor_id=%d", vID),
		fmt.Sprintf("warehouse_id=%d&location_id=%d", wID, l1),
	} {
		b := get(t, s, "/api/stock?"+q, read, http.StatusOK)
		if len(b["stock"].([]any)) == 0 {
			t.Fatalf("filter %q returned nothing", q)
		}
	}
	empty := get(t, s, "/api/stock?warehouse_id=999", read, http.StatusOK)
	if len(empty["stock"].([]any)) != 0 {
		t.Fatalf("unknown warehouse: %v", empty)
	}

	// Unauthenticated reads are rejected; bad filters are 400.
	rec := call(t, s, "GET", "/api/stock", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no key = %d, want 401", rec.Code)
	}
	rec = call(t, s, "GET", "/api/stock?vendor_id=abc", nil, read)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad filter = %d, want 400", rec.Code)
	}
}
