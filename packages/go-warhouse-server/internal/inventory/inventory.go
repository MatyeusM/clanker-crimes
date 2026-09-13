// Package inventory owns stock levels and their movement history.
//
// Stock is keyed by (vendor_item, warehouse, location): the same item
// from different vendors, or in different slots, is a separate row.
// Current levels live in `stock` for fast reads; every change is also
// recorded in `stock_movements` as the audit trail (PLAN.md §11).
//
// This file holds the read side. Mutations (receive, remove, adjust,
// transfer, planned/execute/cancel) live in movements.go.
package inventory

import (
	"database/sql"
	"fmt"
	"strings"
)

// StockEntry is one stock row with denormalized labels so readers do
// not need N extra lookups to render it.
type StockEntry struct {
	ID            int64  `json:"id"`
	VendorItemID  int64  `json:"vendor_item_id"`
	WarehouseID   int64  `json:"warehouse_id"`
	LocationID    int64  `json:"location_id"`
	Quantity      int64  `json:"quantity"`
	VendorCode    string `json:"vendor_code"`
	VendorSKU     string `json:"vendor_sku"`
	ItemSKU       string `json:"item_sku"`
	ItemName      string `json:"item_name"`
	WarehouseCode string `json:"warehouse_code"`
	LocationCode  string `json:"location_code"`
}

// StockFilter narrows ListStock. Nil fields mean "no filter".
type StockFilter struct {
	VendorItemID *int64
	WarehouseID  *int64
	LocationID   *int64
	ItemID       *int64
	VendorID     *int64
}

// ListStock returns current levels in ID order, optionally filtered.
// Unknown filter IDs yield an empty list, never an error.
func ListStock(conn *sql.DB, f StockFilter) ([]StockEntry, error) {
	query := `
		SELECT s.id, s.vendor_item_id, s.warehouse_id, s.location_id, s.quantity,
		       v.code, vi.vendor_sku, i.sku, i.name, w.code, l.code
		FROM stock s
		JOIN vendor_items vi ON vi.id = s.vendor_item_id
		JOIN vendors v ON v.id = vi.vendor_id
		JOIN items i ON i.id = vi.item_id
		JOIN warehouses w ON w.id = s.warehouse_id
		JOIN locations l ON l.id = s.location_id`
	var clauses []string
	var args []any
	if f.VendorItemID != nil {
		clauses = append(clauses, "s.vendor_item_id = ?")
		args = append(args, *f.VendorItemID)
	}
	if f.WarehouseID != nil {
		clauses = append(clauses, "s.warehouse_id = ?")
		args = append(args, *f.WarehouseID)
	}
	if f.LocationID != nil {
		clauses = append(clauses, "s.location_id = ?")
		args = append(args, *f.LocationID)
	}
	if f.ItemID != nil {
		clauses = append(clauses, "vi.item_id = ?")
		args = append(args, *f.ItemID)
	}
	if f.VendorID != nil {
		clauses = append(clauses, "vi.vendor_id = ?")
		args = append(args, *f.VendorID)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY s.id"
	rows, err := conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list stock: %w", err)
	}
	defer rows.Close()
	out := []StockEntry{}
	for rows.Next() {
		var e StockEntry
		if err := rows.Scan(
			&e.ID, &e.VendorItemID, &e.WarehouseID, &e.LocationID, &e.Quantity,
			&e.VendorCode, &e.VendorSKU, &e.ItemSKU, &e.ItemName,
			&e.WarehouseCode, &e.LocationCode,
		); err != nil {
			return nil, fmt.Errorf("scan stock: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
