package inventory

import (
	"database/sql"
	"path/filepath"
	"testing"

	"warehouse-server/internal/db"
	"warehouse-server/internal/vendors"
	"warehouse-server/internal/warehouses"
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

// seedChain builds vendor Acme + item Widget + vendor item BW-123 in
// warehouse Berlin with locations A-01/A-02, returning their IDs.
func seedChain(t *testing.T, conn *sql.DB) (vendorItemID, berID, a01, a02 int64) {
	t.Helper()
	v, err := vendors.CreateVendor(conn, "Acme", "ACME")
	if err != nil {
		t.Fatalf("CreateVendor: %v", err)
	}
	it, err := vendors.CreateItem(conn, "W-1", "Widget", "", "pcs")
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	vi, err := vendors.CreateVendorItem(conn, v.ID, it.ID, "BW-123", nil)
	if err != nil {
		t.Fatalf("CreateVendorItem: %v", err)
	}
	wh, err := warehouses.CreateWarehouse(conn, "Berlin", "WH-BER", nil)
	if err != nil {
		t.Fatalf("CreateWarehouse: %v", err)
	}
	l1, err := warehouses.CreateLocation(conn, wh.ID, "A-01", "Shelf 1")
	if err != nil {
		t.Fatalf("CreateLocation: %v", err)
	}
	l2, err := warehouses.CreateLocation(conn, wh.ID, "A-02", "Shelf 2")
	if err != nil {
		t.Fatalf("CreateLocation: %v", err)
	}
	return vi.ID, wh.ID, l1.ID, l2.ID
}

// seedStock inserts a raw stock row. No mutation API exists until
// Phase 5, so tests write the row directly.
func seedStock(t *testing.T, conn *sql.DB, vendorItemID, warehouseID, locationID, qty int64) {
	t.Helper()
	if _, err := conn.Exec(
		`INSERT INTO stock (vendor_item_id, warehouse_id, location_id, quantity)
		 VALUES (?, ?, ?, ?)`,
		vendorItemID, warehouseID, locationID, qty,
	); err != nil {
		t.Fatalf("seed stock: %v", err)
	}
}

func intptr(v int64) *int64 { return &v }

func TestListStock(t *testing.T) {
	conn := openMigrated(t)
	vi, ber, a01, a02 := seedChain(t, conn)
	seedStock(t, conn, vi, ber, a01, 250)
	seedStock(t, conn, vi, ber, a02, 30)

	all, err := ListStock(conn, StockFilter{})
	if err != nil {
		t.Fatalf("ListStock: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("rows = %d, want 2", len(all))
	}
	e := all[0]
	if e.Quantity != 250 || e.VendorCode != "ACME" || e.VendorSKU != "BW-123" ||
		e.ItemSKU != "W-1" || e.ItemName != "Widget" ||
		e.WarehouseCode != "WH-BER" || e.LocationCode != "A-01" {
		t.Fatalf("unenriched entry: %+v", e)
	}

	byLoc, err := ListStock(conn, StockFilter{LocationID: intptr(a02)})
	if err != nil || len(byLoc) != 1 || byLoc[0].Quantity != 30 {
		t.Fatalf("by location = %+v, %v", byLoc, err)
	}
	byWH, err := ListStock(conn, StockFilter{WarehouseID: intptr(ber)})
	if err != nil || len(byWH) != 2 {
		t.Fatalf("by warehouse = %+v, %v", byWH, err)
	}
	byVI, err := ListStock(conn, StockFilter{VendorItemID: intptr(vi)})
	if err != nil || len(byVI) != 2 {
		t.Fatalf("by vendor item = %+v, %v", byVI, err)
	}
	combined, err := ListStock(conn, StockFilter{WarehouseID: intptr(ber), LocationID: intptr(a01)})
	if err != nil || len(combined) != 1 || combined[0].Quantity != 250 {
		t.Fatalf("combined = %+v, %v", combined, err)
	}
	missing, err := ListStock(conn, StockFilter{WarehouseID: intptr(9999)})
	if err != nil || len(missing) != 0 {
		t.Fatalf("unknown warehouse = %+v, %v (want empty)", missing, err)
	}
}

func TestListStockEmpty(t *testing.T) {
	conn := openMigrated(t)
	seedChain(t, conn)
	out, err := ListStock(conn, StockFilter{})
	if err != nil {
		t.Fatalf("ListStock: %v", err)
	}
	if out == nil || len(out) != 0 {
		t.Fatalf("want empty non-nil list, got %+v", out)
	}
}
