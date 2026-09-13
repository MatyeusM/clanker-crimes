package vendors

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"warehouse-server/internal/db"
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

func strptr(s string) *string { return &s }

func TestVendorCRUD(t *testing.T) {
	conn := openMigrated(t)
	v, err := CreateVendor(conn, "Acme GmbH", "ACME")
	if err != nil {
		t.Fatalf("CreateVendor: %v", err)
	}
	if v.Code != "ACME" || v.Name != "Acme GmbH" {
		t.Fatalf("unexpected vendor: %+v", v)
	}
	if _, err := CreateVendor(conn, "Other", "ACME"); !errors.Is(err, ErrConflict) {
		t.Fatalf("dup code err = %v, want ErrConflict", err)
	}
	if _, err := CreateVendor(conn, "", "X"); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty name err = %v, want ErrValidation", err)
	}
	u, err := UpdateVendor(conn, v.ID, "Acme AG", "ACME2")
	if err != nil {
		t.Fatalf("UpdateVendor: %v", err)
	}
	if u.Name != "Acme AG" || u.Code != "ACME2" {
		t.Fatalf("unexpected update: %+v", u)
	}
	if _, err := GetVendor(conn, 9999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing vendor err = %v, want ErrNotFound", err)
	}
	list, err := ListVendors(conn)
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %v, %v", list, err)
	}
}

func TestItemCRUD(t *testing.T) {
	conn := openMigrated(t)
	it, err := CreateItem(conn, "ABC-123", "Industrial Widget", "heavy", "")
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if it.Unit != "pcs" {
		t.Fatalf("unit = %q, want default pcs", it.Unit)
	}
	if _, err := CreateItem(conn, "ABC-123", "Dupe", "", "pcs"); !errors.Is(err, ErrConflict) {
		t.Fatalf("dup sku err = %v, want ErrConflict", err)
	}
	u, err := UpdateItem(conn, it.ID, "ABC-123", "Widget v2", strptr("lighter"), "kg")
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if u.Name != "Widget v2" || u.Unit != "kg" || u.Description == nil {
		t.Fatalf("unexpected update: %+v", u)
	}
	// Nil description clears.
	u, err = UpdateItem(conn, it.ID, "ABC-123", "Widget v2", nil, "kg")
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if u.Description != nil {
		t.Fatalf("description not cleared: %+v", u)
	}
}

func TestVendorItemCRUD(t *testing.T) {
	conn := openMigrated(t)
	v, _ := CreateVendor(conn, "Acme", "ACME")
	v2, _ := CreateVendor(conn, "Globex", "GLOBEX")
	it, _ := CreateItem(conn, "W-1", "Widget", "", "pcs")

	vi, err := CreateVendorItem(conn, v.ID, it.ID, "BW-123", strptr("Blue Widget"))
	if err != nil {
		t.Fatalf("CreateVendorItem: %v", err)
	}
	// Same SKU under another vendor is fine; same vendor is a conflict.
	if _, err := CreateVendorItem(conn, v2.ID, it.ID, "BW-123", nil); err != nil {
		t.Fatalf("cross-vendor sku: %v", err)
	}
	if _, err := CreateVendorItem(conn, v.ID, it.ID, "BW-123", nil); !errors.Is(err, ErrConflict) {
		t.Fatalf("dup vendor sku err = %v, want ErrConflict", err)
	}
	// Dangling references map to NotFound.
	if _, err := CreateVendorItem(conn, 9999, it.ID, "X", nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing vendor err = %v, want ErrNotFound", err)
	}
	if _, err := CreateVendorItem(conn, v.ID, 9999, "X", nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing item err = %v, want ErrNotFound", err)
	}

	all, _ := ListVendorItems(conn, nil, nil)
	if len(all) != 2 {
		t.Fatalf("unfiltered = %d, want 2", len(all))
	}
	byVendor, _ := ListVendorItems(conn, &v.ID, nil)
	if len(byVendor) != 1 {
		t.Fatalf("by vendor = %d, want 1", len(byVendor))
	}
	byItem, _ := ListVendorItems(conn, nil, &it.ID)
	if len(byItem) != 2 {
		t.Fatalf("by item = %d, want 2", len(byItem))
	}

	u, err := UpdateVendorItem(conn, vi.ID, "BW-124", nil)
	if err != nil {
		t.Fatalf("UpdateVendorItem: %v", err)
	}
	if u.VendorSKU != "BW-124" || u.VendorName != nil {
		t.Fatalf("unexpected update: %+v", u)
	}
}
