// Package warehouses manages warehouses and their flat storage
// locations. Location codes are unique per warehouse, not globally,
// so every warehouse can have its own A-01. Hierarchical locations
// (parent_id) are a later migration if needed; v1 stays flat.
package warehouses

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrValidation covers empty or malformed input.
	ErrValidation = errors.New("validation")
	// ErrNotFound covers unknown warehouse or location IDs.
	ErrNotFound = errors.New("not found")
	// ErrConflict covers duplicate codes.
	ErrConflict = errors.New("conflict")
)

// Warehouse belongs to the agency. Code is unique.
type Warehouse struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Code      string  `json:"code"`
	Address   *string `json:"address,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// Location is a flat storage slot inside one warehouse.
type Location struct {
	ID          int64  `json:"id"`
	WarehouseID int64  `json:"warehouse_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
}

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

func nullStr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func requireNonEmpty(value, field string) (string, error) {
	if v := strings.TrimSpace(value); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("%w: %s is required", ErrValidation, field)
}

// CreateWarehouse inserts a warehouse with a unique code.
func CreateWarehouse(conn *sql.DB, name, code string, address *string) (Warehouse, error) {
	var err error
	if name, err = requireNonEmpty(name, "name"); err != nil {
		return Warehouse{}, err
	}
	if code, err = requireNonEmpty(code, "code"); err != nil {
		return Warehouse{}, err
	}
	var addr sql.NullString
	if address != nil && strings.TrimSpace(*address) != "" {
		addr = sql.NullString{String: *address, Valid: true}
	}
	now := nowUTC()
	res, err := conn.Exec(
		`INSERT INTO warehouses (name, code, address, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		name, code, addr, now, now,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Warehouse{}, fmt.Errorf("%w: warehouse code %q taken", ErrConflict, code)
		}
		return Warehouse{}, fmt.Errorf("insert warehouse: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetWarehouse(conn, id)
}

// GetWarehouse fetches one warehouse by ID.
func GetWarehouse(conn *sql.DB, id int64) (Warehouse, error) {
	var w Warehouse
	var addr sql.NullString
	err := conn.QueryRow(
		`SELECT id, name, code, address, created_at, updated_at FROM warehouses WHERE id = ?`, id,
	).Scan(&w.ID, &w.Name, &w.Code, &addr, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return Warehouse{}, fmt.Errorf("%w: warehouse %d", ErrNotFound, id)
	}
	if err != nil {
		return Warehouse{}, fmt.Errorf("get warehouse: %w", err)
	}
	w.Address = nullStr(addr)
	return w, nil
}

// ListWarehouses returns all warehouses in ID order.
func ListWarehouses(conn *sql.DB) ([]Warehouse, error) {
	rows, err := conn.Query(
		`SELECT id, name, code, address, created_at, updated_at FROM warehouses ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list warehouses: %w", err)
	}
	defer rows.Close()
	out := []Warehouse{}
	for rows.Next() {
		var w Warehouse
		var addr sql.NullString
		if err := rows.Scan(&w.ID, &w.Name, &w.Code, &addr, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan warehouse: %w", err)
		}
		w.Address = nullStr(addr)
		out = append(out, w)
	}
	return out, rows.Err()
}

// UpdateWarehouse fully replaces name, code, and address.
func UpdateWarehouse(conn *sql.DB, id int64, name, code string, address *string) (Warehouse, error) {
	if _, err := GetWarehouse(conn, id); err != nil {
		return Warehouse{}, err
	}
	var err error
	if name, err = requireNonEmpty(name, "name"); err != nil {
		return Warehouse{}, err
	}
	if code, err = requireNonEmpty(code, "code"); err != nil {
		return Warehouse{}, err
	}
	var addr sql.NullString
	if address != nil && strings.TrimSpace(*address) != "" {
		addr = sql.NullString{String: *address, Valid: true}
	}
	if _, err := conn.Exec(
		`UPDATE warehouses SET name = ?, code = ?, address = ?, updated_at = ? WHERE id = ?`,
		name, code, addr, nowUTC(), id,
	); err != nil {
		if isUniqueViolation(err) {
			return Warehouse{}, fmt.Errorf("%w: warehouse code %q taken", ErrConflict, code)
		}
		return Warehouse{}, fmt.Errorf("update warehouse: %w", err)
	}
	return GetWarehouse(conn, id)
}

// CreateLocation adds a storage slot to a warehouse. The code must be
// unique within that warehouse only. Unknown warehouses map to
// ErrNotFound, not a raw FK error.
func CreateLocation(conn *sql.DB, warehouseID int64, code, name string) (Location, error) {
	if _, err := GetWarehouse(conn, warehouseID); err != nil {
		return Location{}, err
	}
	var err error
	if code, err = requireNonEmpty(code, "code"); err != nil {
		return Location{}, err
	}
	if name, err = requireNonEmpty(name, "name"); err != nil {
		return Location{}, err
	}
	res, err := conn.Exec(
		`INSERT INTO locations (warehouse_id, code, name) VALUES (?, ?, ?)`,
		warehouseID, code, name,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Location{}, fmt.Errorf("%w: location code %q taken in this warehouse", ErrConflict, code)
		}
		return Location{}, fmt.Errorf("insert location: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetLocation(conn, id)
}

// GetLocation fetches one location by ID.
func GetLocation(conn *sql.DB, id int64) (Location, error) {
	var l Location
	err := conn.QueryRow(
		`SELECT id, warehouse_id, code, name FROM locations WHERE id = ?`, id,
	).Scan(&l.ID, &l.WarehouseID, &l.Code, &l.Name)
	if err == sql.ErrNoRows {
		return Location{}, fmt.Errorf("%w: location %d", ErrNotFound, id)
	}
	if err != nil {
		return Location{}, fmt.Errorf("get location: %w", err)
	}
	return l, nil
}

// ListLocations returns a warehouse's locations in ID order.
func ListLocations(conn *sql.DB, warehouseID int64) ([]Location, error) {
	if _, err := GetWarehouse(conn, warehouseID); err != nil {
		return nil, err
	}
	rows, err := conn.Query(
		`SELECT id, warehouse_id, code, name FROM locations WHERE warehouse_id = ? ORDER BY id`,
		warehouseID,
	)
	if err != nil {
		return nil, fmt.Errorf("list locations: %w", err)
	}
	defer rows.Close()
	out := []Location{}
	for rows.Next() {
		var l Location
		if err := rows.Scan(&l.ID, &l.WarehouseID, &l.Code, &l.Name); err != nil {
			return nil, fmt.Errorf("scan location: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// UpdateLocation replaces a location's code and name. The warehouse
// association is immutable: moving a slot means creating a new one.
func UpdateLocation(conn *sql.DB, id int64, code, name string) (Location, error) {
	if _, err := GetLocation(conn, id); err != nil {
		return Location{}, err
	}
	var err error
	if code, err = requireNonEmpty(code, "code"); err != nil {
		return Location{}, err
	}
	if name, err = requireNonEmpty(name, "name"); err != nil {
		return Location{}, err
	}
	if _, err := conn.Exec(
		`UPDATE locations SET code = ?, name = ? WHERE id = ?`, code, name, id,
	); err != nil {
		if isUniqueViolation(err) {
			return Location{}, fmt.Errorf("%w: location code %q taken in this warehouse", ErrConflict, code)
		}
		return Location{}, fmt.Errorf("update location: %w", err)
	}
	return GetLocation(conn, id)
}

// WarehouseProvider abstracts every warehouse and location capability
// of this package behind one interface, so that callers depend on the
// abstraction rather than on the concrete package-level functions.
type WarehouseProvider interface {
	CreateWarehouse(name, code string, address *string) (Warehouse, error)
	GetWarehouse(id int64) (Warehouse, error)
	ListWarehouses() ([]Warehouse, error)
	UpdateWarehouse(id int64, name, code string, address *string) (Warehouse, error)
	CreateLocation(warehouseID int64, code, name string) (Location, error)
	GetLocation(id int64) (Location, error)
	ListLocations(warehouseID int64) ([]Location, error)
	UpdateLocation(id int64, code, name string) (Location, error)
	ResolveLocationCode(warehouseID int64, code string) (Location, error)
}

// FormatLocationCode renders a globally qualified location label from
// a warehouse code and a location code for display purposes.
func FormatLocationCode(warehouseCode, locationCode string) string {
	return warehouseCode + "::" + locationCode
}

// NormalizeLocationName trims surrounding whitespace from a raw
// location name before it is validated or stored.
func NormalizeLocationName(raw string) string {
	return strings.TrimSpace(raw)
}

// IsValidCode reports whether a warehouse or location code is usable.
func IsValidCode(code string) bool {
	if strings.TrimSpace(code) == "" {
		return false
	} else {
		return true
	}
}
