package inventory

import (
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"warehouse-server/internal/users"
	"warehouse-server/internal/warehouses"
)

// setup returns a migrated DB with a catalog chain, one operator user,
// and the chain IDs.
func setup(t *testing.T) (*sql.DB, int64, int64, int64, int64, int64) {
	t.Helper()
	conn := openMigrated(t)
	vi, ber, a01, a02 := seedChain(t, conn)
	op, err := users.CreateUser(conn, "op", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return conn, op.ID, vi, ber, a01, a02
}

func level(t *testing.T, conn *sql.DB, vi, wh, loc int64) int64 {
	t.Helper()
	rows, err := ListStock(conn, StockFilter{VendorItemID: &vi, WarehouseID: &wh, LocationID: &loc})
	if err != nil {
		t.Fatalf("ListStock: %v", err)
	}
	if len(rows) == 0 {
		return 0
	}
	return rows[0].Quantity
}

func future() *time.Time {
	t := time.Now().UTC().Add(24 * time.Hour)
	return &t
}

// createWarehouse inserts a warehouse, returning its ID.
func createWarehouse(t *testing.T, conn *sql.DB, name, code string) (int64, error) {
	t.Helper()
	wh, err := warehouses.CreateWarehouse(conn, name, code, nil)
	if err != nil {
		return 0, err
	}
	return wh.ID, nil
}

// createLocation inserts a location in wh, returning its ID.
func createLocation(t *testing.T, conn *sql.DB, warehouseID int64, code string) int64 {
	t.Helper()
	loc, err := warehouses.CreateLocation(conn, warehouseID, code, code)
	if err != nil {
		t.Fatalf("CreateLocation: %v", err)
	}
	return loc.ID
}

func TestReceiveRemove(t *testing.T) {
	conn, op, vi, ber, a01, _ := setup(t)

	res, err := Receive(conn, vi, ber, a01, 100, "DELIVERY-1", "initial", nil, op)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if res.Movement.Status != StatusExecuted || res.Movement.Type != TypeReceive {
		t.Fatalf("movement: %+v", res.Movement)
	}
	if res.Movement.ToLocationID == nil || res.Movement.FromLocationID != nil {
		t.Fatalf("receive sides wrong: %+v", res.Movement)
	}
	if res.Movement.Reference == nil || *res.Movement.Reference != "DELIVERY-1" {
		t.Fatalf("reference lost: %+v", res.Movement)
	}
	if res.Movement.CreatedBy != op || res.Movement.CreatedByUsername != "op" {
		t.Fatalf("creator lost: %+v", res.Movement)
	}
	if level(t, conn, vi, ber, a01) != 100 {
		t.Fatal("level != 100 after receive")
	}

	if _, err := Remove(conn, vi, ber, a01, 30, "SALES-1", "", nil, op); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if level(t, conn, vi, ber, a01) != 70 {
		t.Fatal("level != 70 after remove")
	}
	if _, err := Remove(conn, vi, ber, a01, 71, "", "", nil, op); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("oversell err = %v, want ErrInsufficient", err)
	}
	if level(t, conn, vi, ber, a01) != 70 {
		t.Fatal("failed remove changed the level")
	}
	if _, err := Remove(conn, vi, ber, a01, 70, "", "", nil, op); err != nil {
		t.Fatalf("exact Remove: %v", err)
	}
	if level(t, conn, vi, ber, a01) != 0 {
		t.Fatal("level != 0 after exact remove")
	}
	if _, err := Receive(conn, vi, ber, a01, 0, "", "", nil, op); !errors.Is(err, ErrValidation) {
		t.Fatalf("zero qty err = %v, want ErrValidation", err)
	}
	if _, err := Receive(conn, 9999, ber, a01, 5, "", "", nil, op); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing vendor item err = %v, want ErrNotFound", err)
	}
}

func TestTransfer(t *testing.T) {
	conn, op, vi, ber, a01, a02 := setup(t)
	if _, err := Receive(conn, vi, ber, a01, 100, "", "", nil, op); err != nil {
		t.Fatalf("Receive: %v", err)
	}
	res, err := Transfer(conn, vi, ber, a01, a02, 40, "MOVE-1", "", nil, op)
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if res.Movement.FromLocationID == nil || res.Movement.ToLocationID == nil {
		t.Fatalf("transfer sides missing: %+v", res.Movement)
	}
	if len(res.Stock) != 2 {
		t.Fatalf("transfer stock rows = %d, want 2", len(res.Stock))
	}
	if level(t, conn, vi, ber, a01) != 60 || level(t, conn, vi, ber, a02) != 40 {
		t.Fatal("levels wrong after transfer")
	}
	// Atomicity: failed transfer leaves both sides untouched.
	if _, err := Transfer(conn, vi, ber, a01, a02, 61, "", "", nil, op); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("oversell transfer err = %v, want ErrInsufficient", err)
	}
	if level(t, conn, vi, ber, a01) != 60 || level(t, conn, vi, ber, a02) != 40 {
		t.Fatal("failed transfer moved stock")
	}
	if _, err := Transfer(conn, vi, ber, a01, a01, 5, "", "", nil, op); !errors.Is(err, ErrValidation) {
		t.Fatalf("same-slot transfer err = %v, want ErrValidation", err)
	}
	// Only one movement row per transfer.
	moves, err := ListMovements(conn, MovementFilter{Type: strptr(TypeTransfer)})
	if err != nil || len(moves) != 1 {
		t.Fatalf("transfer movements = %+v, %v", moves, err)
	}
}

func TestTransferCrossWarehouseLocation(t *testing.T) {
	conn, op, vi, ber, a01, _ := setup(t)
	ham, err := createWarehouse(t, conn, "Hamburg", "WH-HAM")
	if err != nil {
		t.Fatalf("warehouse: %v", err)
	}
	h1 := createLocation(t, conn, ham, "A-01")
	if _, err := Receive(conn, vi, ber, a01, 10, "", "", nil, op); err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if _, err := Transfer(conn, vi, ber, a01, h1, 5, "", "", nil, op); !errors.Is(err, ErrValidation) {
		t.Fatalf("cross-warehouse transfer err = %v, want ErrValidation", err)
	}
}

func TestAdjust(t *testing.T) {
	conn, op, vi, ber, a01, _ := setup(t)
	if _, err := Receive(conn, vi, ber, a01, 10, "", "", nil, op); err != nil {
		t.Fatalf("Receive: %v", err)
	}
	up, err := Adjust(conn, vi, ber, a01, 25, "recount found more", nil, op)
	if err != nil {
		t.Fatalf("Adjust up: %v", err)
	}
	if up.Movement.Quantity != 15 || up.Movement.ToLocationID == nil || up.Movement.FromLocationID != nil {
		t.Fatalf("adjust up movement wrong: %+v", up.Movement)
	}
	down, err := Adjust(conn, vi, ber, a01, 5, "damaged", nil, op)
	if err != nil {
		t.Fatalf("Adjust down: %v", err)
	}
	if down.Movement.Quantity != 20 || down.Movement.FromLocationID == nil || down.Movement.ToLocationID != nil {
		t.Fatalf("adjust down movement wrong: %+v", down.Movement)
	}
	if level(t, conn, vi, ber, a01) != 5 {
		t.Fatal("level != 5 after adjusts")
	}
	if _, err := Adjust(conn, vi, ber, a01, 5, "noop", nil, op); !errors.Is(err, ErrValidation) {
		t.Fatalf("unchanged adjust err = %v, want ErrValidation", err)
	}
	if _, err := Adjust(conn, vi, ber, a01, 6, "  ", nil, op); !errors.Is(err, ErrValidation) {
		t.Fatalf("noteless adjust err = %v, want ErrValidation", err)
	}
	if _, err := Adjust(conn, vi, ber, a01, -1, "neg", nil, op); !errors.Is(err, ErrValidation) {
		t.Fatalf("negative adjust err = %v, want ErrValidation", err)
	}
	// Adjusting an empty slot creates it.
	if _, err := Adjust(conn, vi, ber, a01, 0, "write off", nil, op); err != nil {
		t.Fatalf("adjust to zero: %v", err)
	}
	if level(t, conn, vi, ber, a01) != 0 {
		t.Fatal("level != 0 after write-off")
	}
}

func TestPlannedLifecycle(t *testing.T) {
	conn, op, vi, ber, a01, a02 := setup(t)

	// Planned receive touches nothing until executed.
	plan, err := Receive(conn, vi, ber, a01, 50, "PO-9", "", future(), op)
	if err != nil {
		t.Fatalf("plan receive: %v", err)
	}
	if plan.Movement.Status != StatusPlanned || plan.Movement.PlannedAt == nil {
		t.Fatalf("not planned: %+v", plan.Movement)
	}
	if len(plan.Stock) != 0 {
		t.Fatalf("planned op returned stock: %+v", plan)
	}
	if level(t, conn, vi, ber, a01) != 0 {
		t.Fatal("planned receive changed stock")
	}
	exec, err := Execute(conn, plan.Movement.ID)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if exec.Movement.Status != StatusExecuted || exec.Movement.ExecutedAt == nil {
		t.Fatalf("not executed: %+v", exec.Movement)
	}
	if level(t, conn, vi, ber, a01) != 50 {
		t.Fatal("level != 50 after execute")
	}
	// Re-executing is a no-op, never a double apply.
	again, err := Execute(conn, plan.Movement.ID)
	if err != nil {
		t.Fatalf("re-execute: %v", err)
	}
	if level(t, conn, vi, ber, a01) != 50 || again.Movement.Status != StatusExecuted {
		t.Fatalf("double apply: %+v", again)
	}

	// Oversized planned remove: allowed at plan, rejected at execute.
	big, err := Remove(conn, vi, ber, a01, 500, "", "", future(), op)
	if err != nil {
		t.Fatalf("plan oversized remove: %v", err)
	}
	if _, err := Execute(conn, big.Movement.ID); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("execute oversized err = %v, want ErrInsufficient", err)
	}

	// Planned transfer executes atomically.
	tr, err := Transfer(conn, vi, ber, a01, a02, 20, "", "", future(), op)
	if err != nil {
		t.Fatalf("plan transfer: %v", err)
	}
	if _, err := Execute(conn, tr.Movement.ID); err != nil {
		t.Fatalf("execute transfer: %v", err)
	}
	if level(t, conn, vi, ber, a01) != 30 || level(t, conn, vi, ber, a02) != 20 {
		t.Fatal("levels wrong after planned transfer")
	}

	// Cancel path.
	c, err := Remove(conn, vi, ber, a01, 5, "", "", future(), op)
	if err != nil {
		t.Fatalf("plan remove: %v", err)
	}
	cancelled, err := Cancel(conn, c.Movement.ID)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if cancelled.Status != StatusCancelled {
		t.Fatalf("not cancelled: %+v", cancelled)
	}
	if _, err := Execute(conn, c.Movement.ID); !errors.Is(err, ErrValidation) {
		t.Fatalf("execute cancelled err = %v, want ErrValidation", err)
	}
	if _, err := Cancel(conn, c.Movement.ID); err != nil {
		t.Fatalf("second Cancel: %v", err)
	}
	if _, err := Cancel(conn, plan.Movement.ID); !errors.Is(err, ErrValidation) {
		t.Fatalf("cancel executed err = %v, want ErrValidation", err)
	}
	if _, err := Execute(conn, 9999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("execute missing err = %v, want ErrNotFound", err)
	}
	if _, err := Cancel(conn, 9999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cancel missing err = %v, want ErrNotFound", err)
	}
}

func TestConcurrentRemove(t *testing.T) {
	conn, op, vi, ber, a01, _ := setup(t)
	if _, err := Receive(conn, vi, ber, a01, 50, "", "", nil, op); err != nil {
		t.Fatalf("Receive: %v", err)
	}
	const workers = 10
	var wg sync.WaitGroup
	errs := make([]error, workers)
	for i := range workers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = Remove(conn, vi, ber, a01, 10, "", "", nil, op)
		}(i)
	}
	wg.Wait()
	var ok, short int
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrInsufficient):
			short++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 5 || short != 5 {
		t.Fatalf("ok=%d short=%d, want 5/5 (oversell or lost update)", ok, short)
	}
	if got := level(t, conn, vi, ber, a01); got != 0 {
		t.Fatalf("final level = %d, want 0", got)
	}
	// Audit must match reality: executed removes sum to the received 50.
	moves, err := ListMovements(conn, MovementFilter{Type: strptr(TypeRemove)})
	if err != nil {
		t.Fatalf("ListMovements: %v", err)
	}
	var total int64
	for _, m := range moves {
		if m.Status == StatusExecuted {
			total += m.Quantity
		}
	}
	if total != 50 {
		t.Fatalf("executed removes sum = %d, want 50", total)
	}
}

func TestListMovementsFilters(t *testing.T) {
	conn, op, vi, ber, a01, a02 := setup(t)
	if _, err := Receive(conn, vi, ber, a01, 100, "R-1", "note", nil, op); err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if _, err := Transfer(conn, vi, ber, a01, a02, 10, "", "", future(), op); err != nil {
		t.Fatalf("plan transfer: %v", err)
	}
	byType, err := ListMovements(conn, MovementFilter{Type: strptr(TypeReceive)})
	if err != nil || len(byType) != 1 {
		t.Fatalf("by type = %+v, %v", byType, err)
	}
	byStatus, err := ListMovements(conn, MovementFilter{Status: strptr(StatusPlanned)})
	if err != nil || len(byStatus) != 1 || byStatus[0].Type != TypeTransfer {
		t.Fatalf("by status = %+v, %v", byStatus, err)
	}
	byBoth, err := ListMovements(conn, MovementFilter{
		VendorItemID: &vi, WarehouseID: &ber, Status: strptr(StatusExecuted),
	})
	if err != nil || len(byBoth) != 1 {
		t.Fatalf("combined = %+v, %v", byBoth, err)
	}
	since := time.Now().UTC().Add(-time.Hour)
	until := time.Now().UTC().Add(time.Hour)
	byTime, err := ListMovements(conn, MovementFilter{Since: &since, Until: &until})
	if err != nil || len(byTime) != 2 {
		t.Fatalf("by time = %+v, %v", byTime, err)
	}
	if _, err := ListMovements(conn, MovementFilter{Status: strptr("bogus")}); !errors.Is(err, ErrValidation) {
		t.Fatalf("bad status err = %v, want ErrValidation", err)
	}
	// Audit row answers what/when/where/who/why.
	m := byType[0]
	if m.VendorSKU != "BW-123" || m.WarehouseCode != "WH-BER" ||
		m.ToLocationCode == nil || *m.ToLocationCode != "A-01" ||
		m.CreatedByUsername != "op" || m.Reference == nil || *m.Reference != "R-1" ||
		m.Note == nil || m.ExecutedAt == nil {
		t.Fatalf("incomplete audit row: %+v", m)
	}
}
