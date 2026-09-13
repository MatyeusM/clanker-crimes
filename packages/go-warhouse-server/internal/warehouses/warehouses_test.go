package warehouses

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

func TestWarehouseCRUD(t *testing.T) {
	conn := openMigrated(t)
	w, err := CreateWarehouse(conn, "Berlin", "WH-BER", strptr("Street 1"))
	if err != nil {
		t.Fatalf("CreateWarehouse: %v", err)
	}
	if w.Code != "WH-BER" || w.Address == nil {
		t.Fatalf("unexpected warehouse: %+v", w)
	}
	if _, err := CreateWarehouse(conn, "Dupe", "WH-BER", nil); !errors.Is(err, ErrConflict) {
		t.Fatalf("dup code err = %v, want ErrConflict", err)
	}
	u, err := UpdateWarehouse(conn, w.ID, "Berlin Ost", "WH-BER", nil)
	if err != nil {
		t.Fatalf("UpdateWarehouse: %v", err)
	}
	if u.Name != "Berlin Ost" || u.Address != nil {
		t.Fatalf("unexpected update: %+v", u)
	}
	if _, err := GetWarehouse(conn, 9999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing warehouse err = %v, want ErrNotFound", err)
	}
}

func TestLocationScoping(t *testing.T) {
	conn := openMigrated(t)
	ber, _ := CreateWarehouse(conn, "Berlin", "WH-BER", nil)
	ham, _ := CreateWarehouse(conn, "Hamburg", "WH-HAM", nil)

	a1, err := CreateLocation(conn, ber.ID, "A-01", "Shelf A-01")
	if err != nil {
		t.Fatalf("CreateLocation: %v", err)
	}
	// Same code in another warehouse is fine.
	if _, err := CreateLocation(conn, ham.ID, "A-01", "Shelf A-01"); err != nil {
		t.Fatalf("cross-warehouse code: %v", err)
	}
	// Same warehouse is a conflict.
	if _, err := CreateLocation(conn, ber.ID, "A-01", "Dupe"); !errors.Is(err, ErrConflict) {
		t.Fatalf("dup code err = %v, want ErrConflict", err)
	}
	// Unknown warehouse maps to NotFound.
	if _, err := CreateLocation(conn, 9999, "Z-9", "Nowhere"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing warehouse err = %v, want ErrNotFound", err)
	}

	list, err := ListLocations(conn, ber.ID)
	if err != nil || len(list) != 1 || list[0].ID != a1.ID {
		t.Fatalf("list = %v, %v", list, err)
	}
	if _, err := ListLocations(conn, 9999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("list missing warehouse err = %v, want ErrNotFound", err)
	}

	u, err := UpdateLocation(conn, a1.ID, "A-02", "Shelf A-02")
	if err != nil {
		t.Fatalf("UpdateLocation: %v", err)
	}
	if u.Code != "A-02" {
		t.Fatalf("unexpected update: %+v", u)
	}
	if _, err := GetLocation(conn, 9999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing location err = %v, want ErrNotFound", err)
	}
}
