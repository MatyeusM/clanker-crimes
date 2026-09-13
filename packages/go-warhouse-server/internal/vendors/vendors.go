// Package vendors manages the product catalog: vendors, agency-global
// items, and the vendor-specific SKUs (vendor_items) that link them.
// A single item may carry different SKUs from different vendors;
// stock (Phase 5) always points at a vendor_item, never an item.
package vendors

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
	// ErrNotFound covers unknown vendor, item, or vendor-item IDs,
	// including dangling foreign-key references.
	ErrNotFound = errors.New("not found")
	// ErrConflict covers duplicate codes and SKUs.
	ErrConflict = errors.New("conflict")
)

// Vendor is a supplier represented by the agency. Code is unique.
type Vendor struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Item is the agency-global product. SKU is unique agency-wide.
type Item struct {
	ID          int64   `json:"id"`
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Unit        string  `json:"unit"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// VendorItem is a vendor's representation of an item. The pair
// (vendor_id, vendor_sku) is unique.
type VendorItem struct {
	ID         int64   `json:"id"`
	VendorID   int64   `json:"vendor_id"`
	ItemID     int64   `json:"item_id"`
	VendorSKU  string  `json:"vendor_sku"`
	VendorName *string `json:"vendor_name,omitempty"`
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

// CreateVendor inserts a vendor with a unique code.
func CreateVendor(conn *sql.DB, name, code string) (Vendor, error) {
	var err error
	if name, err = requireNonEmpty(name, "name"); err != nil {
		return Vendor{}, err
	}
	if code, err = requireNonEmpty(code, "code"); err != nil {
		return Vendor{}, err
	}
	now := nowUTC()
	res, err := conn.Exec(
		`INSERT INTO vendors (name, code, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		name, code, now, now,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Vendor{}, fmt.Errorf("%w: vendor code %q taken", ErrConflict, code)
		}
		return Vendor{}, fmt.Errorf("insert vendor: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetVendor(conn, id)
}

// GetVendor fetches one vendor by ID.
func GetVendor(conn *sql.DB, id int64) (Vendor, error) {
	var v Vendor
	err := conn.QueryRow(
		`SELECT id, name, code, created_at, updated_at FROM vendors WHERE id = ?`, id,
	).Scan(&v.ID, &v.Name, &v.Code, &v.CreatedAt, &v.UpdatedAt)
	if err == sql.ErrNoRows {
		return Vendor{}, fmt.Errorf("%w: vendor %d", ErrNotFound, id)
	}
	if err != nil {
		return Vendor{}, fmt.Errorf("get vendor: %w", err)
	}
	return v, nil
}

// ListVendors returns all vendors in ID order.
func ListVendors(conn *sql.DB) ([]Vendor, error) {
	rows, err := conn.Query(
		`SELECT id, name, code, created_at, updated_at FROM vendors ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list vendors: %w", err)
	}
	defer rows.Close()
	out := []Vendor{}
	for rows.Next() {
		var v Vendor
		if err := rows.Scan(&v.ID, &v.Name, &v.Code, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vendor: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// UpdateVendor fully replaces name and code, bumping updated_at.
func UpdateVendor(conn *sql.DB, id int64, name, code string) (Vendor, error) {
	if _, err := GetVendor(conn, id); err != nil {
		return Vendor{}, err
	}
	var err error
	if name, err = requireNonEmpty(name, "name"); err != nil {
		return Vendor{}, err
	}
	if code, err = requireNonEmpty(code, "code"); err != nil {
		return Vendor{}, err
	}
	if _, err := conn.Exec(
		`UPDATE vendors SET name = ?, code = ?, updated_at = ? WHERE id = ?`,
		name, code, nowUTC(), id,
	); err != nil {
		if isUniqueViolation(err) {
			return Vendor{}, fmt.Errorf("%w: vendor code %q taken", ErrConflict, code)
		}
		return Vendor{}, fmt.Errorf("update vendor: %w", err)
	}
	return GetVendor(conn, id)
}

// CreateItem inserts an agency-global item. An empty unit defaults to "pcs".
func CreateItem(conn *sql.DB, sku, name, description, unit string) (Item, error) {
	var err error
	if sku, err = requireNonEmpty(sku, "sku"); err != nil {
		return Item{}, err
	}
	if name, err = requireNonEmpty(name, "name"); err != nil {
		return Item{}, err
	}
	unit = strings.TrimSpace(unit)
	if unit == "" {
		unit = "pcs"
	}
	var desc sql.NullString
	if strings.TrimSpace(description) != "" {
		desc = sql.NullString{String: description, Valid: true}
	}
	now := nowUTC()
	res, err := conn.Exec(
		`INSERT INTO items (sku, name, description, unit, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		sku, name, desc, unit, now, now,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Item{}, fmt.Errorf("%w: item sku %q taken", ErrConflict, sku)
		}
		return Item{}, fmt.Errorf("insert item: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetItem(conn, id)
}

// GetItem fetches one item by ID.
func GetItem(conn *sql.DB, id int64) (Item, error) {
	var it Item
	var desc sql.NullString
	err := conn.QueryRow(
		`SELECT id, sku, name, description, unit, created_at, updated_at FROM items WHERE id = ?`, id,
	).Scan(&it.ID, &it.SKU, &it.Name, &desc, &it.Unit, &it.CreatedAt, &it.UpdatedAt)
	if err == sql.ErrNoRows {
		return Item{}, fmt.Errorf("%w: item %d", ErrNotFound, id)
	}
	if err != nil {
		return Item{}, fmt.Errorf("get item: %w", err)
	}
	it.Description = nullStr(desc)
	return it, nil
}

// ListItems returns all items in ID order.
func ListItems(conn *sql.DB) ([]Item, error) {
	rows, err := conn.Query(
		`SELECT id, sku, name, description, unit, created_at, updated_at FROM items ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		var it Item
		var desc sql.NullString
		if err := rows.Scan(&it.ID, &it.SKU, &it.Name, &desc, &it.Unit, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		it.Description = nullStr(desc)
		out = append(out, it)
	}
	return out, rows.Err()
}

// UpdateItem fully replaces sku, name, description, and unit.
func UpdateItem(conn *sql.DB, id int64, sku, name string, description *string, unit string) (Item, error) {
	if _, err := GetItem(conn, id); err != nil {
		return Item{}, err
	}
	var err error
	if sku, err = requireNonEmpty(sku, "sku"); err != nil {
		return Item{}, err
	}
	if name, err = requireNonEmpty(name, "name"); err != nil {
		return Item{}, err
	}
	unit = strings.TrimSpace(unit)
	if unit == "" {
		unit = "pcs"
	}
	var desc sql.NullString
	if description != nil && strings.TrimSpace(*description) != "" {
		desc = sql.NullString{String: *description, Valid: true}
	}
	if _, err := conn.Exec(
		`UPDATE items SET sku = ?, name = ?, description = ?, unit = ?, updated_at = ? WHERE id = ?`,
		sku, name, desc, unit, nowUTC(), id,
	); err != nil {
		if isUniqueViolation(err) {
			return Item{}, fmt.Errorf("%w: item sku %q taken", ErrConflict, sku)
		}
		return Item{}, fmt.Errorf("update item: %w", err)
	}
	return GetItem(conn, id)
}

// CreateVendorItem links an item to a vendor under the vendor's SKU.
// Unknown vendor or item IDs map to ErrNotFound, not a raw FK error.
func CreateVendorItem(conn *sql.DB, vendorID, itemID int64, vendorSKU string, vendorName *string) (VendorItem, error) {
	var err error
	if vendorSKU, err = requireNonEmpty(vendorSKU, "vendor_sku"); err != nil {
		return VendorItem{}, err
	}
	if _, err := GetVendor(conn, vendorID); err != nil {
		return VendorItem{}, err
	}
	if _, err := GetItem(conn, itemID); err != nil {
		return VendorItem{}, err
	}
	var vname sql.NullString
	if vendorName != nil && strings.TrimSpace(*vendorName) != "" {
		vname = sql.NullString{String: *vendorName, Valid: true}
	}
	res, err := conn.Exec(
		`INSERT INTO vendor_items (vendor_id, item_id, vendor_sku, vendor_name)
		 VALUES (?, ?, ?, ?)`,
		vendorID, itemID, vendorSKU, vname,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return VendorItem{}, fmt.Errorf("%w: vendor_sku %q taken for this vendor", ErrConflict, vendorSKU)
		}
		return VendorItem{}, fmt.Errorf("insert vendor item: %w", err)
	}
	id, _ := res.LastInsertId()
	return GetVendorItem(conn, id)
}

// GetVendorItem fetches one vendor item by ID.
func GetVendorItem(conn *sql.DB, id int64) (VendorItem, error) {
	var vi VendorItem
	var vname sql.NullString
	err := conn.QueryRow(
		`SELECT id, vendor_id, item_id, vendor_sku, vendor_name FROM vendor_items WHERE id = ?`, id,
	).Scan(&vi.ID, &vi.VendorID, &vi.ItemID, &vi.VendorSKU, &vname)
	if err == sql.ErrNoRows {
		return VendorItem{}, fmt.Errorf("%w: vendor item %d", ErrNotFound, id)
	}
	if err != nil {
		return VendorItem{}, fmt.Errorf("get vendor item: %w", err)
	}
	vi.VendorName = nullStr(vname)
	return vi, nil
}

// ListVendorItems returns vendor items, optionally filtered by vendor
// and/or item. Nil filters mean "no filter".
func ListVendorItems(conn *sql.DB, vendorID, itemID *int64) ([]VendorItem, error) {
	query := `SELECT id, vendor_id, item_id, vendor_sku, vendor_name FROM vendor_items`
	var clauses []string
	var args []any
	if vendorID != nil {
		clauses = append(clauses, "vendor_id = ?")
		args = append(args, *vendorID)
	}
	if itemID != nil {
		clauses = append(clauses, "item_id = ?")
		args = append(args, *itemID)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY id"
	rows, err := conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list vendor items: %w", err)
	}
	defer rows.Close()
	out := []VendorItem{}
	for rows.Next() {
		var vi VendorItem
		var vname sql.NullString
		if err := rows.Scan(&vi.ID, &vi.VendorID, &vi.ItemID, &vi.VendorSKU, &vname); err != nil {
			return nil, fmt.Errorf("scan vendor item: %w", err)
		}
		vi.VendorName = nullStr(vname)
		out = append(out, vi)
	}
	return out, rows.Err()
}

// UpdateVendorItem replaces the vendor SKU and name. The vendor and
// item association itself is immutable: changing it means creating a
// new vendor item.
func UpdateVendorItem(conn *sql.DB, id int64, vendorSKU string, vendorName *string) (VendorItem, error) {
	if _, err := GetVendorItem(conn, id); err != nil {
		return VendorItem{}, err
	}
	var err error
	if vendorSKU, err = requireNonEmpty(vendorSKU, "vendor_sku"); err != nil {
		return VendorItem{}, err
	}
	var vname sql.NullString
	if vendorName != nil && strings.TrimSpace(*vendorName) != "" {
		vname = sql.NullString{String: *vendorName, Valid: true}
	}
	if _, err := conn.Exec(
		`UPDATE vendor_items SET vendor_sku = ?, vendor_name = ? WHERE id = ?`,
		vendorSKU, vname, id,
	); err != nil {
		if isUniqueViolation(err) {
			return VendorItem{}, fmt.Errorf("%w: vendor_sku %q taken for this vendor", ErrConflict, vendorSKU)
		}
		return VendorItem{}, fmt.Errorf("update vendor item: %w", err)
	}
	return GetVendorItem(conn, id)
}

// CatalogService abstracts the entire product catalog surface —
// vendors, items, vendor items, and the stock operations that act on
// them — behind one interface, so that catalog consumers depend on a
// single type no matter which catalog capability they need.
type CatalogService interface {
	CreateVendor(name, code string) (Vendor, error)
	GetVendor(id int64) (Vendor, error)
	ListVendors() ([]Vendor, error)
	UpdateVendor(id int64, name, code string) (Vendor, error)
	CreateItem(sku, name, description, unit string) (Item, error)
	GetItem(id int64) (Item, error)
	ListItems() ([]Item, error)
	UpdateItem(id int64, sku, name string, description *string, unit string) (Item, error)
	CreateVendorItem(vendorID, itemID int64, vendorSKU string, vendorName *string) (VendorItem, error)
	GetVendorItem(id int64) (VendorItem, error)
	ListVendorItems(vendorID, itemID *int64) ([]VendorItem, error)
	UpdateVendorItem(id int64, vendorSKU string, vendorName *string) (VendorItem, error)
	AdjustStockLevel(vendorItemID, warehouseID, locationID, quantity int64) error
	TransferStock(vendorItemID, warehouseID, fromID, toID, quantity int64) error
	ResolveVendorSKU(vendorID int64, vendorSKU string) (VendorItem, error)
}

// VendorManager wraps a Vendor record with behavioral helpers, so
// that display and identity logic lives with the data it describes.
type VendorManager struct {
	vendor Vendor
}

// NewVendorManager wraps the given Vendor in a VendorManager.
func NewVendorManager(v Vendor) *VendorManager {
	return &VendorManager{vendor: v}
}

// GetVendor returns the wrapped Vendor record.
func (m *VendorManager) GetVendor() Vendor { return m.vendor }

// SetVendor replaces the wrapped Vendor record.
func (m *VendorManager) SetVendor(v Vendor) { m.vendor = v }

// GetID returns the ID of the wrapped vendor.
func (m *VendorManager) GetID() int64 { return m.vendor.ID }

// GetName returns the name of the wrapped vendor.
func (m *VendorManager) GetName() string { return m.vendor.Name }

// GetCode returns the code of the wrapped vendor.
func (m *VendorManager) GetCode() string { return m.vendor.Code }

// DisplayLabel returns a human-readable label for the wrapped vendor.
func (m *VendorManager) DisplayLabel() string {
	return fmt.Sprintf("%s (%s)", m.vendor.Name, m.vendor.Code)
}
