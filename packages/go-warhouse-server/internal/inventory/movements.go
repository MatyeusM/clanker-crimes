package inventory

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrValidation covers bad quantities, cross-warehouse slots,
	// no-op adjusts, and illegal lifecycle transitions.
	ErrValidation = errors.New("validation")
	// ErrNotFound covers unknown vendor items, warehouses,
	// locations, and movements.
	ErrNotFound = errors.New("not found")
	// ErrInsufficient covers removes, transfers, and executions
	// that exceed the available source quantity. Never negative
	// stock: the mutation fails instead.
	ErrInsufficient = errors.New("insufficient stock")
)

// Movement types and statuses mirror the CHECK constraints in
// migrations/0001_init.sql.
const (
	TypeReceive  = "receive"
	TypeRemove   = "remove"
	TypeTransfer = "transfer"
	TypeAdjust   = "adjust"

	StatusPlanned   = "planned"
	StatusExecuted  = "executed"
	StatusCancelled = "cancelled"
)

// Movement is one logical stock operation with denormalized labels.
// Side convention: receive and upward adjusts set ToLocationID;
// remove and downward adjusts set FromLocationID; transfers set both
// (distinct); so the direction of every executed row is derivable and
// quantity is always the absolute units moved.
type Movement struct {
	ID                int64   `json:"id"`
	StockID           *int64  `json:"stock_id,omitempty"`
	VendorItemID      int64   `json:"vendor_item_id"`
	WarehouseID       int64   `json:"warehouse_id"`
	FromLocationID    *int64  `json:"from_location_id,omitempty"`
	ToLocationID      *int64  `json:"to_location_id,omitempty"`
	Type              string  `json:"type"`
	Status            string  `json:"status"`
	Quantity          int64   `json:"quantity"`
	PlannedAt         *string `json:"planned_at,omitempty"`
	ExecutedAt        *string `json:"executed_at,omitempty"`
	Reference         *string `json:"reference,omitempty"`
	Note              *string `json:"note,omitempty"`
	CreatedBy         int64   `json:"created_by"`
	CreatedByUsername string  `json:"created_by_username"`
	CreatedAt         string  `json:"created_at"`
	VendorCode        string  `json:"vendor_code"`
	VendorSKU         string  `json:"vendor_sku"`
	WarehouseCode     string  `json:"warehouse_code"`
	FromLocationCode  *string `json:"from_location_code,omitempty"`
	ToLocationCode    *string `json:"to_location_code,omitempty"`
}

// MutationResult pairs the recorded movement with the stock rows it
// (re)shaped. Planned operations touch no stock, so Stock is empty.
type MutationResult struct {
	Movement Movement     `json:"movement"`
	Stock    []StockEntry `json:"stock"`
}

// MovementFilter narrows ListMovements. Nil fields mean "no filter".
// Since/Until bound created_at.
type MovementFilter struct {
	VendorItemID *int64
	WarehouseID  *int64
	Status       *string
	Type         *string
	Since        *time.Time
	Until        *time.Time
}

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

func nullStr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

func nullInt(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	v := ni.Int64
	return &v
}

func int64ptr(v int64) *int64 { return &v }

// isPlanned reports whether plannedAt defers the operation.
func isPlanned(plannedAt *time.Time) bool {
	return plannedAt != nil && plannedAt.After(time.Now().UTC())
}

type querier interface {
	QueryRow(query string, args ...any) *sql.Row
}

func inTx(conn *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := conn.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func checkAmount(qty int64) error {
	if qty <= 0 {
		return fmt.Errorf("%w: quantity must be positive", ErrValidation)
	}
	return nil
}

// resolveSlot validates the business keys for one stock slot: the
// vendor item, warehouse, and location must exist, and the location
// must belong to the warehouse.
func resolveSlot(q querier, vendorItemID, warehouseID, locationID int64) error {
	var dummy int64
	if err := q.QueryRow(`SELECT id FROM vendor_items WHERE id = ?`, vendorItemID).Scan(&dummy); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("%w: vendor item %d", ErrNotFound, vendorItemID)
		}
		return fmt.Errorf("lookup vendor item: %w", err)
	}
	if err := q.QueryRow(`SELECT id FROM warehouses WHERE id = ?`, warehouseID).Scan(&dummy); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("%w: warehouse %d", ErrNotFound, warehouseID)
		}
		return fmt.Errorf("lookup warehouse: %w", err)
	}
	var owner int64
	if err := q.QueryRow(`SELECT warehouse_id FROM locations WHERE id = ?`, locationID).Scan(&owner); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("%w: location %d", ErrNotFound, locationID)
		}
		return fmt.Errorf("lookup location: %w", err)
	}
	if owner != warehouseID {
		return fmt.Errorf("%w: location %d does not belong to warehouse %d",
			ErrValidation, locationID, warehouseID)
	}
	return nil
}

// addStock upserts a non-negative delta onto a slot, creating the row
// when absent, and returns the stock row ID.
func addStock(tx *sql.Tx, vendorItemID, warehouseID, locationID, delta int64) (int64, error) {
	if _, err := tx.Exec(`
		INSERT INTO stock (vendor_item_id, warehouse_id, location_id, quantity)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (vendor_item_id, warehouse_id, location_id)
		DO UPDATE SET quantity = stock.quantity + excluded.quantity`,
		vendorItemID, warehouseID, locationID, delta,
	); err != nil {
		return 0, fmt.Errorf("upsert stock: %w", err)
	}
	var id int64
	if err := tx.QueryRow(`
		SELECT id FROM stock
		WHERE vendor_item_id = ? AND warehouse_id = ? AND location_id = ?`,
		vendorItemID, warehouseID, locationID,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("fetch stock id: %w", err)
	}
	return id, nil
}

// takeStock subtracts an amount from a slot inside the caller's
// transaction. Missing rows and shortfalls both map to ErrInsufficient:
// stock never goes negative.
func takeStock(tx *sql.Tx, vendorItemID, warehouseID, locationID, amount int64) error {
	var qty int64
	err := tx.QueryRow(`
		SELECT quantity FROM stock
		WHERE vendor_item_id = ? AND warehouse_id = ? AND location_id = ?`,
		vendorItemID, warehouseID, locationID,
	).Scan(&qty)
	if err == sql.ErrNoRows {
		return fmt.Errorf("%w: no stock at location %d", ErrInsufficient, locationID)
	}
	if err != nil {
		return fmt.Errorf("read stock: %w", err)
	}
	if qty < amount {
		return fmt.Errorf("%w: have %d, need %d at location %d",
			ErrInsufficient, qty, amount, locationID)
	}
	if _, err := tx.Exec(`
		UPDATE stock SET quantity = quantity - ?
		WHERE vendor_item_id = ? AND warehouse_id = ? AND location_id = ?`,
		amount, vendorItemID, warehouseID, locationID,
	); err != nil {
		return fmt.Errorf("update stock: %w", err)
	}
	return nil
}

// currentQty reads a slot's level, treating a missing row as zero.
func currentQty(q querier, vendorItemID, warehouseID, locationID int64) (int64, error) {
	var qty int64
	err := q.QueryRow(`
		SELECT quantity FROM stock
		WHERE vendor_item_id = ? AND warehouse_id = ? AND location_id = ?`,
		vendorItemID, warehouseID, locationID,
	).Scan(&qty)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read stock: %w", err)
	}
	return qty, nil
}

func insertMovement(tx *sql.Tx, stockID, fromID, toID *int64, vendorItemID, warehouseID int64,
	opType, status string, qty int64, plannedAt, executedAt, reference, note *string,
	createdBy int64) (int64, error) {
	res, err := tx.Exec(`
		INSERT INTO stock_movements
			(stock_id, vendor_item_id, warehouse_id, from_location_id, to_location_id,
			 type, status, quantity, planned_at, executed_at, reference, note, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		stockID, vendorItemID, warehouseID, fromID, toID,
		opType, status, qty, plannedAt, executedAt, reference, note, createdBy, nowUTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("insert movement: %w", err)
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func strptr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func plannedStr(plannedAt *time.Time) *string {
	if plannedAt == nil {
		return nil
	}
	s := plannedAt.UTC().Format(time.RFC3339)
	return &s
}

// Receive adds qty to a slot, creating the stock row when absent.
// A future plannedAt records a planned movement with no stock effect.
func Receive(conn *sql.DB, vendorItemID, warehouseID, locationID, qty int64,
	reference, note string, plannedAt *time.Time, createdBy int64) (MutationResult, error) {
	if err := checkAmount(qty); err != nil {
		return MutationResult{}, err
	}
	if err := resolveSlot(conn, vendorItemID, warehouseID, locationID); err != nil {
		return MutationResult{}, err
	}
	if isPlanned(plannedAt) {
		return planOnly(conn, nil, int64ptr(locationID), vendorItemID, warehouseID,
			TypeReceive, qty, plannedAt, reference, note, createdBy)
	}
	var movementID int64
	err := inTx(conn, func(tx *sql.Tx) error {
		stockID, err := addStock(tx, vendorItemID, warehouseID, locationID, qty)
		if err != nil {
			return err
		}
		now := nowUTC()
		movementID, err = insertMovement(tx, int64ptr(stockID), nil, int64ptr(locationID),
			vendorItemID, warehouseID, TypeReceive, StatusExecuted, qty,
			nil, &now, strptr(reference), strptr(note), createdBy)
		return err
	})
	if err != nil {
		return MutationResult{}, err
	}
	return result(conn, movementID, vendorItemID, warehouseID, []int64{locationID})
}

// Remove subtracts qty from a slot. Shortfalls fail with
// ErrInsufficient; stock never goes negative.
func Remove(conn *sql.DB, vendorItemID, warehouseID, locationID, qty int64,
	reference, note string, plannedAt *time.Time, createdBy int64) (MutationResult, error) {
	if err := checkAmount(qty); err != nil {
		return MutationResult{}, err
	}
	if err := resolveSlot(conn, vendorItemID, warehouseID, locationID); err != nil {
		return MutationResult{}, err
	}
	if isPlanned(plannedAt) {
		return planOnly(conn, int64ptr(locationID), nil, vendorItemID, warehouseID,
			TypeRemove, qty, plannedAt, reference, note, createdBy)
	}
	var movementID int64
	err := inTx(conn, func(tx *sql.Tx) error {
		if err := takeStock(tx, vendorItemID, warehouseID, locationID, qty); err != nil {
			return err
		}
		var stockID int64
		if err := tx.QueryRow(`
			SELECT id FROM stock
			WHERE vendor_item_id = ? AND warehouse_id = ? AND location_id = ?`,
			vendorItemID, warehouseID, locationID,
		).Scan(&stockID); err != nil {
			return fmt.Errorf("fetch stock id: %w", err)
		}
		now := nowUTC()
		mID, err := insertMovement(tx, int64ptr(stockID), int64ptr(locationID), nil,
			vendorItemID, warehouseID, TypeRemove, StatusExecuted, qty,
			nil, &now, strptr(reference), strptr(note), createdBy)
		movementID = mID
		return err
	})
	if err != nil {
		return MutationResult{}, err
	}
	return result(conn, movementID, vendorItemID, warehouseID, []int64{locationID})
}

// Adjust sets a slot to an absolute target level. The note is required:
// it is the audit reason. A target equal to the current level is a
// validation error; use the current level read instead.
func Adjust(conn *sql.DB, vendorItemID, warehouseID, locationID, target int64,
	note string, plannedAt *time.Time, createdBy int64) (MutationResult, error) {
	if target < 0 {
		return MutationResult{}, fmt.Errorf("%w: adjust target must not be negative", ErrValidation)
	}
	if strings.TrimSpace(note) == "" {
		return MutationResult{}, fmt.Errorf("%w: adjust requires a note (reason)", ErrValidation)
	}
	if err := resolveSlot(conn, vendorItemID, warehouseID, locationID); err != nil {
		return MutationResult{}, err
	}
	if isPlanned(plannedAt) {
		current, err := currentQty(conn, vendorItemID, warehouseID, locationID)
		if err != nil {
			return MutationResult{}, err
		}
		from, to := adjustSides(target, current, locationID)
		if from == nil && to == nil {
			return MutationResult{}, fmt.Errorf("%w: adjust target equals current level", ErrValidation)
		}
		delta := abs(target - current)
		return planOnly(conn, from, to, vendorItemID, warehouseID,
			TypeAdjust, delta, plannedAt, "", note, createdBy)
	}
	var movementID int64
	err := inTx(conn, func(tx *sql.Tx) error {
		current, err := currentQty(tx, vendorItemID, warehouseID, locationID)
		if err != nil {
			return err
		}
		from, to := adjustSides(target, current, locationID)
		if from == nil && to == nil {
			return fmt.Errorf("%w: adjust target equals current level", ErrValidation)
		}
		delta := abs(target - current)
		if _, err := tx.Exec(`
			INSERT INTO stock (vendor_item_id, warehouse_id, location_id, quantity)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (vendor_item_id, warehouse_id, location_id)
			DO UPDATE SET quantity = excluded.quantity`,
			vendorItemID, warehouseID, locationID, target,
		); err != nil {
			return fmt.Errorf("set stock: %w", err)
		}
		var stockID int64
		if err := tx.QueryRow(`
			SELECT id FROM stock
			WHERE vendor_item_id = ? AND warehouse_id = ? AND location_id = ?`,
			vendorItemID, warehouseID, locationID,
		).Scan(&stockID); err != nil {
			return fmt.Errorf("fetch stock id: %w", err)
		}
		now := nowUTC()
		movementID, err = insertMovement(tx, int64ptr(stockID), from, to,
			vendorItemID, warehouseID, TypeAdjust, StatusExecuted, delta,
			nil, &now, nil, strptr(note), createdBy)
		return err
	})
	if err != nil {
		return MutationResult{}, err
	}
	return result(conn, movementID, vendorItemID, warehouseID, []int64{locationID})
}

// adjustSides maps an absolute target onto the movement side
// convention: increases set To, decreases set From, no-ops set neither.
func adjustSides(target, current, locationID int64) (from, to *int64) {
	switch {
	case target > current:
		return nil, int64ptr(locationID)
	case target < current:
		return int64ptr(locationID), nil
	default:
		return nil, nil
	}
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// Transfer moves qty between two slots of one warehouse atomically:
// the decrement, increment, and audit row commit or roll back together.
func Transfer(conn *sql.DB, vendorItemID, warehouseID, fromID, toID, qty int64,
	reference, note string, plannedAt *time.Time, createdBy int64) (MutationResult, error) {
	if err := checkAmount(qty); err != nil {
		return MutationResult{}, err
	}
	if fromID == toID {
		return MutationResult{}, fmt.Errorf("%w: source and destination locations differ", ErrValidation)
	}
	if err := resolveSlot(conn, vendorItemID, warehouseID, fromID); err != nil {
		return MutationResult{}, err
	}
	if err := resolveSlot(conn, vendorItemID, warehouseID, toID); err != nil {
		return MutationResult{}, err
	}
	if isPlanned(plannedAt) {
		return planOnly(conn, int64ptr(fromID), int64ptr(toID), vendorItemID, warehouseID,
			TypeTransfer, qty, plannedAt, reference, note, createdBy)
	}
	var movementID int64
	err := inTx(conn, func(tx *sql.Tx) error {
		if err := takeStock(tx, vendorItemID, warehouseID, fromID, qty); err != nil {
			return err
		}
		if _, err := addStock(tx, vendorItemID, warehouseID, toID, qty); err != nil {
			return err
		}
		now := nowUTC()
		// stock_id stays NULL: one row cannot name both affected rows.
		mID, err := insertMovement(tx, nil, int64ptr(fromID), int64ptr(toID),
			vendorItemID, warehouseID, TypeTransfer, StatusExecuted, qty,
			nil, &now, strptr(reference), strptr(note), createdBy)
		movementID = mID
		return err
	})
	if err != nil {
		return MutationResult{}, err
	}
	return result(conn, movementID, vendorItemID, warehouseID, []int64{fromID, toID})
}

// planOnly records a future movement without touching stock.
// Sufficiency is deliberately NOT checked at plan time: the level will
// have moved by execution, which revalidates.
func planOnly(conn *sql.DB, fromID, toID *int64, vendorItemID, warehouseID int64,
	opType string, qty int64, plannedAt *time.Time, reference, note string,
	createdBy int64) (MutationResult, error) {
	var movementID int64
	err := inTx(conn, func(tx *sql.Tx) error {
		var err error
		movementID, err = insertMovement(tx, nil, fromID, toID,
			vendorItemID, warehouseID, opType, StatusPlanned, qty,
			plannedStr(plannedAt), nil, strptr(reference), strptr(note), createdBy)
		return err
	})
	if err != nil {
		return MutationResult{}, err
	}
	m, err := GetMovement(conn, movementID)
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Movement: m, Stock: []StockEntry{}}, nil
}

// result fetches the recorded movement plus the current entries for
// the affected slots.
func result(conn *sql.DB, movementID, vendorItemID, warehouseID int64, locationIDs []int64) (MutationResult, error) {
	m, err := GetMovement(conn, movementID)
	if err != nil {
		return MutationResult{}, err
	}
	stock := []StockEntry{}
	for _, lid := range locationIDs {
		entries, err := ListStock(conn, StockFilter{
			VendorItemID: &vendorItemID, WarehouseID: &warehouseID, LocationID: &lid,
		})
		if err != nil {
			return MutationResult{}, err
		}
		stock = append(stock, entries...)
	}
	return MutationResult{Movement: m, Stock: stock}, nil
}

// errAlreadyDone aborts an Execute transaction that lost a race:
// the movement was executed between the fast-path read and the
// write lock. It is mapped back to the idempotent success path.
var errAlreadyDone = errors.New("already executed")

// Execute applies a planned movement's stock effects atomically and
// flips it to executed. Re-executing an executed movement is a
// harmless no-op (never double-applies); executing a cancelled one is
// a validation error. Shortfalls at execute time fail with
// ErrInsufficient.
func Execute(conn *sql.DB, movementID int64) (MutationResult, error) {
	raw, err := getRaw(conn, movementID)
	if err != nil {
		return MutationResult{}, err
	}
	switch raw.status {
	case StatusExecuted:
		m, err := GetMovement(conn, movementID)
		if err != nil {
			return MutationResult{}, err
		}
		return MutationResult{Movement: m, Stock: []StockEntry{}}, nil
	case StatusCancelled:
		return MutationResult{}, fmt.Errorf("%w: movement %d is cancelled", ErrValidation, movementID)
	case StatusPlanned:
		// Apply below. Status is re-checked inside the write
		// transaction: the pre-read above is a fast path only.
	default:
		return MutationResult{}, fmt.Errorf("%w: unknown status %q", ErrValidation, raw.status)
	}
	err = inTx(conn, func(tx *sql.Tx) error {
		fresh, err := getRaw(tx, movementID)
		if err != nil {
			return err
		}
		if fresh.status == StatusExecuted {
			return errAlreadyDone
		}
		if fresh.status != StatusPlanned {
			return fmt.Errorf("%w: movement %d is %s", ErrValidation, movementID, fresh.status)
		}
		raw = fresh
		if raw.opType != TypeTransfer && raw.fromID == nil && raw.toID == nil {
			return fmt.Errorf("%w: movement %d names no location", ErrValidation, movementID)
		}
		switch raw.opType {
		case TypeReceive:
			if _, err := addStock(tx, raw.vendorItemID, raw.warehouseID, *raw.toID, raw.qty); err != nil {
				return err
			}
		case TypeRemove:
			if err := takeStock(tx, raw.vendorItemID, raw.warehouseID, *raw.fromID, raw.qty); err != nil {
				return err
			}
		case TypeTransfer:
			if err := takeStock(tx, raw.vendorItemID, raw.warehouseID, *raw.fromID, raw.qty); err != nil {
				return err
			}
			if _, err := addStock(tx, raw.vendorItemID, raw.warehouseID, *raw.toID, raw.qty); err != nil {
				return err
			}
		case TypeAdjust:
			current, err := currentQty(tx, raw.vendorItemID, raw.warehouseID, adjustLocation(raw))
			if err != nil {
				return err
			}
			var target int64
			if raw.toID != nil {
				target = current + raw.qty
			} else {
				if current < raw.qty {
					return fmt.Errorf("%w: have %d, need %d",
						ErrInsufficient, current, raw.qty)
				}
				target = current - raw.qty
			}
			if _, err := tx.Exec(`
				INSERT INTO stock (vendor_item_id, warehouse_id, location_id, quantity)
				VALUES (?, ?, ?, ?)
				ON CONFLICT (vendor_item_id, warehouse_id, location_id)
				DO UPDATE SET quantity = excluded.quantity`,
				raw.vendorItemID, raw.warehouseID, adjustLocation(raw), target,
			); err != nil {
				return fmt.Errorf("set stock: %w", err)
			}
		default:
			return fmt.Errorf("%w: unknown type %q", ErrValidation, raw.opType)
		}
		var stockID *int64
		if raw.opType != TypeTransfer {
			var loc *int64
			switch raw.opType {
			case TypeReceive:
				loc = raw.toID
			case TypeRemove:
				loc = raw.fromID
			case TypeAdjust:
				if raw.toID != nil {
					loc = raw.toID
				} else {
					loc = raw.fromID
				}
			}
			if loc == nil {
				return fmt.Errorf("%w: movement %d names no location", ErrValidation, movementID)
			}
			var id int64
			if err := tx.QueryRow(`
				SELECT id FROM stock
				WHERE vendor_item_id = ? AND warehouse_id = ? AND location_id = ?`,
				raw.vendorItemID, raw.warehouseID, loc,
			).Scan(&id); err != nil {
				return fmt.Errorf("fetch stock id: %w", err)
			}
			stockID = &id
		}
		now := nowUTC()
		res, err := tx.Exec(`
			UPDATE stock_movements SET status = ?, executed_at = ?, stock_id = ?
			WHERE id = ? AND status = ?`,
			StatusExecuted, now, stockID, movementID, StatusPlanned,
		)
		if err != nil {
			return fmt.Errorf("mark executed: %w", err)
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return fmt.Errorf("%w: movement %d changed concurrently", ErrValidation, movementID)
		}
		return nil
	})
	if errors.Is(err, errAlreadyDone) {
		m, merr := GetMovement(conn, movementID)
		if merr != nil {
			return MutationResult{}, merr
		}
		return MutationResult{Movement: m, Stock: []StockEntry{}}, nil
	}
	if err != nil {
		return MutationResult{}, err
	}
	m, err := GetMovement(conn, movementID)
	if err != nil {
		return MutationResult{}, err
	}
	affected := []int64{}
	if raw.fromID != nil {
		affected = append(affected, *raw.fromID)
	}
	if raw.toID != nil && (raw.fromID == nil || *raw.toID != *raw.fromID) {
		affected = append(affected, *raw.toID)
	}
	stock := []StockEntry{}
	for _, lid := range affected {
		entries, err := ListStock(conn, StockFilter{
			VendorItemID: &raw.vendorItemID, WarehouseID: &raw.warehouseID, LocationID: &lid,
		})
		if err != nil {
			return MutationResult{}, err
		}
		stock = append(stock, entries...)
	}
	return MutationResult{Movement: m, Stock: stock}, nil
}

// adjustLocation returns the single slot of an adjust movement.
func adjustLocation(raw rawMovement) int64 {
	if raw.toID != nil {
		return *raw.toID
	}
	return *raw.fromID
}

// Cancel marks a planned movement cancelled without touching stock.
// Idempotent: cancelling a cancelled movement returns it. Cancelling
// an executed movement is a validation error: history is immutable.
func Cancel(conn *sql.DB, movementID int64) (Movement, error) {
	raw, err := getRaw(conn, movementID)
	if err != nil {
		return Movement{}, err
	}
	switch raw.status {
	case StatusCancelled:
		return GetMovement(conn, movementID)
	case StatusExecuted:
		return Movement{}, fmt.Errorf("%w: movement %d already executed", ErrValidation, movementID)
	case StatusPlanned:
		res, err := conn.Exec(
			`UPDATE stock_movements SET status = ? WHERE id = ? AND status = ?`,
			StatusCancelled, movementID, StatusPlanned,
		)
		if err != nil {
			return Movement{}, fmt.Errorf("cancel movement: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return GetMovement(conn, movementID)
		}
		return GetMovement(conn, movementID)
	default:
		return Movement{}, fmt.Errorf("%w: unknown status %q", ErrValidation, raw.status)
	}
}

// rawMovement is the unenriched row for lifecycle transitions.
type rawMovement struct {
	vendorItemID, warehouseID int64
	fromID, toID              *int64
	opType, status            string
	qty                       int64
}

func getRaw(q querier, movementID int64) (rawMovement, error) {
	var raw rawMovement
	var from, to sql.NullInt64
	err := q.QueryRow(`
		SELECT vendor_item_id, warehouse_id, from_location_id, to_location_id, type, status, quantity
		FROM stock_movements WHERE id = ?`,
		movementID,
	).Scan(&raw.vendorItemID, &raw.warehouseID, &from, &to, &raw.opType, &raw.status, &raw.qty)
	if err == sql.ErrNoRows {
		return rawMovement{}, fmt.Errorf("%w: movement %d", ErrNotFound, movementID)
	}
	if err != nil {
		return rawMovement{}, fmt.Errorf("get movement: %w", err)
	}
	raw.fromID = nullInt(from)
	raw.toID = nullInt(to)
	return raw, nil
}

// GetMovement fetches one enriched movement by ID.
func GetMovement(conn *sql.DB, movementID int64) (Movement, error) {
	m, err := getMovement(conn, movementID)
	if err == sql.ErrNoRows {
		return Movement{}, fmt.Errorf("%w: movement %d", ErrNotFound, movementID)
	}
	return m, err
}

func getMovement(q querier, movementID int64) (Movement, error) {
	var m Movement
	var stockID sql.NullInt64
	var fromID, toID sql.NullInt64
	var planned, executed, reference, note sql.NullString
	var fromCode, toCode sql.NullString
	err := q.QueryRow(`
		SELECT m.id, m.stock_id, m.vendor_item_id, m.warehouse_id,
		       m.from_location_id, m.to_location_id, m.type, m.status, m.quantity,
		       m.planned_at, m.executed_at, m.reference, m.note,
		       m.created_by, u.username, m.created_at,
		       v.code, vi.vendor_sku, w.code, fl.code, tl.code
		FROM stock_movements m
		JOIN vendor_items vi ON vi.id = m.vendor_item_id
		JOIN vendors v ON v.id = vi.vendor_id
		JOIN warehouses w ON w.id = m.warehouse_id
		JOIN users u ON u.id = m.created_by
		LEFT JOIN locations fl ON fl.id = m.from_location_id
		LEFT JOIN locations tl ON tl.id = m.to_location_id
		WHERE m.id = ?`,
		movementID,
	).Scan(
		&m.ID, &stockID, &m.VendorItemID, &m.WarehouseID,
		&fromID, &toID, &m.Type, &m.Status, &m.Quantity,
		&planned, &executed, &reference, &note,
		&m.CreatedBy, &m.CreatedByUsername, &m.CreatedAt,
		&m.VendorCode, &m.VendorSKU, &m.WarehouseCode, &fromCode, &toCode,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return Movement{}, err
		}
		return Movement{}, fmt.Errorf("get movement: %w", err)
	}
	m.StockID = nullInt(stockID)
	m.FromLocationID = nullInt(fromID)
	m.ToLocationID = nullInt(toID)
	m.PlannedAt = nullStr(planned)
	m.ExecutedAt = nullStr(executed)
	m.Reference = nullStr(reference)
	m.Note = nullStr(note)
	m.FromLocationCode = nullStr(fromCode)
	m.ToLocationCode = nullStr(toCode)
	return m, nil
}

// ListMovements returns enriched movements in ID order, optionally
// filtered. Unknown status/type filters are validation errors;
// unknown entity IDs yield an empty list.
func ListMovements(conn *sql.DB, f MovementFilter) ([]Movement, error) {
	if f.Status != nil {
		switch *f.Status {
		case StatusPlanned, StatusExecuted, StatusCancelled:
		default:
			return nil, fmt.Errorf("%w: unknown status %q", ErrValidation, *f.Status)
		}
	}
	if f.Type != nil {
		switch *f.Type {
		case TypeReceive, TypeRemove, TypeTransfer, TypeAdjust:
		default:
			return nil, fmt.Errorf("%w: unknown type %q", ErrValidation, *f.Type)
		}
	}
	query := `
		SELECT m.id, m.stock_id, m.vendor_item_id, m.warehouse_id,
		       m.from_location_id, m.to_location_id, m.type, m.status, m.quantity,
		       m.planned_at, m.executed_at, m.reference, m.note,
		       m.created_by, u.username, m.created_at,
		       v.code, vi.vendor_sku, w.code, fl.code, tl.code
		FROM stock_movements m
		JOIN vendor_items vi ON vi.id = m.vendor_item_id
		JOIN vendors v ON v.id = vi.vendor_id
		JOIN warehouses w ON w.id = m.warehouse_id
		JOIN users u ON u.id = m.created_by
		LEFT JOIN locations fl ON fl.id = m.from_location_id
		LEFT JOIN locations tl ON tl.id = m.to_location_id`
	var clauses []string
	var args []any
	if f.VendorItemID != nil {
		clauses = append(clauses, "m.vendor_item_id = ?")
		args = append(args, *f.VendorItemID)
	}
	if f.WarehouseID != nil {
		clauses = append(clauses, "m.warehouse_id = ?")
		args = append(args, *f.WarehouseID)
	}
	if f.Status != nil {
		clauses = append(clauses, "m.status = ?")
		args = append(args, *f.Status)
	}
	if f.Type != nil {
		clauses = append(clauses, "m.type = ?")
		args = append(args, *f.Type)
	}
	if f.Since != nil {
		clauses = append(clauses, "m.created_at >= ?")
		args = append(args, f.Since.UTC().Format(time.RFC3339))
	}
	if f.Until != nil {
		clauses = append(clauses, "m.created_at <= ?")
		args = append(args, f.Until.UTC().Format(time.RFC3339))
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY m.id"
	rows, err := conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list movements: %w", err)
	}
	defer rows.Close()
	out := []Movement{}
	for rows.Next() {
		var m Movement
		var stockID sql.NullInt64
		var fromID, toID sql.NullInt64
		var planned, executed, reference, note sql.NullString
		var fromCode, toCode sql.NullString
		if err := rows.Scan(
			&m.ID, &stockID, &m.VendorItemID, &m.WarehouseID,
			&fromID, &toID, &m.Type, &m.Status, &m.Quantity,
			&planned, &executed, &reference, &note,
			&m.CreatedBy, &m.CreatedByUsername, &m.CreatedAt,
			&m.VendorCode, &m.VendorSKU, &m.WarehouseCode, &fromCode, &toCode,
		); err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}
		m.StockID = nullInt(stockID)
		m.FromLocationID = nullInt(fromID)
		m.ToLocationID = nullInt(toID)
		m.PlannedAt = nullStr(planned)
		m.ExecutedAt = nullStr(executed)
		m.Reference = nullStr(reference)
		m.Note = nullStr(note)
		m.FromLocationCode = nullStr(fromCode)
		m.ToLocationCode = nullStr(toCode)
		out = append(out, m)
	}
	return out, rows.Err()
}

// MovementProcessor abstracts every inventory mutation capability
// behind one interface, so that orchestration layers depend on a
// single type no matter which stock operation they perform.
type MovementProcessor interface {
	ProcessReceive(vendorItemID, warehouseID, locationID, qty int64, createdBy int64) (MutationResult, error)
	ProcessRemove(vendorItemID, warehouseID, locationID, qty int64, createdBy int64) (MutationResult, error)
	ProcessAdjust(vendorItemID, warehouseID, locationID, target int64, createdBy int64) (MutationResult, error)
	ProcessTransfer(vendorItemID, warehouseID, fromID, toID, qty int64, createdBy int64) (MutationResult, error)
	ExecuteMovement(movementID int64) (MutationResult, error)
	CancelMovement(movementID int64) (Movement, error)
	ListAllMovements(filter MovementFilter) ([]Movement, error)
	ValidateMutation(opType string, qty int64) error
}

// MutationOptions carries the optional metadata of a stock mutation
// (audit reference, human note, deferred execution time). The zero
// value selects an immediate, unannotated mutation.
type MutationOptions struct {
	Reference string
	Note      string
	PlannedAt *time.Time
}

// MutationOption customizes a MutationOptions before a mutation runs.
type MutationOption func(*MutationOptions)

// WithReference annotates the mutation with an external reference.
func WithReference(reference string) MutationOption {
	return func(o *MutationOptions) { o.Reference = reference }
}

// WithNote annotates the mutation with a human-readable note.
func WithNote(note string) MutationOption {
	return func(o *MutationOptions) { o.Note = note }
}

// WithPlannedAt defers the mutation to the given planned time.
func WithPlannedAt(plannedAt *time.Time) MutationOption {
	return func(o *MutationOptions) { o.PlannedAt = plannedAt }
}

// cachedOptions holds process-wide default mutation options when set.
// It is nil unless InstallDefaultMutationOptions has been called.
var cachedOptions *MutationOptions

// InstallDefaultMutationOptions installs process-wide default mutation
// options applied by the WithOptions mutation variants.
func InstallDefaultMutationOptions(o MutationOptions) {
	cachedOptions = &o
}

func defaultMutationOptions() MutationOptions {
	if cachedOptions != nil {
		return *cachedOptions
	} else {
		return MutationOptions{}
	}
}

// ReceiveWithOptions adds qty to a slot like Receive, but takes the
// optional audit metadata as composable options instead of positional
// parameters, which keeps call sites readable as the metadata grows.
func ReceiveWithOptions(conn *sql.DB, vendorItemID, warehouseID, locationID, qty int64,
	createdBy int64, opts ...MutationOption) (MutationResult, error) {
	base := defaultMutationOptions()
	for _, opt := range opts {
		opt(&base)
	}
	return Receive(conn, vendorItemID, warehouseID, locationID, qty,
		base.Reference, base.Note, base.PlannedAt, createdBy)
}

// RemoveWithOptions subtracts qty from a slot like Remove, but takes
// the optional audit metadata as composable options.
func RemoveWithOptions(conn *sql.DB, vendorItemID, warehouseID, locationID, qty int64,
	createdBy int64, opts ...MutationOption) (MutationResult, error) {
	base := defaultMutationOptions()
	for _, opt := range opts {
		opt(&base)
	}
	return Remove(conn, vendorItemID, warehouseID, locationID, qty,
		base.Reference, base.Note, base.PlannedAt, createdBy)
}
