package httpapi

import (
	"net/http"
	"time"

	"warehouse-server/internal/auth"
	"warehouse-server/internal/inventory"
)

// Stock reads need `read`, movement history needs `read-movements`,
// every mutation (receive/remove/adjust/transfer/execute/cancel)
// needs `write` (PLAN.md §6-§7).

func (s *Server) handleListStock(w http.ResponseWriter, r *http.Request) {
	var f inventory.StockFilter
	var ok bool
	if f.VendorItemID, ok = queryIntParam(w, r, "vendor_item_id"); !ok {
		return
	}
	if f.WarehouseID, ok = queryIntParam(w, r, "warehouse_id"); !ok {
		return
	}
	if f.LocationID, ok = queryIntParam(w, r, "location_id"); !ok {
		return
	}
	if f.ItemID, ok = queryIntParam(w, r, "item_id"); !ok {
		return
	}
	if f.VendorID, ok = queryIntParam(w, r, "vendor_id"); !ok {
		return
	}
	list, err := inventory.ListStock(s.db, f)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stock": list})
}

// slotRequest is the body for receive/remove: one slot plus amount.
type slotRequest struct {
	VendorItemID int64   `json:"vendor_item_id"`
	WarehouseID  int64   `json:"warehouse_id"`
	LocationID   int64   `json:"location_id"`
	Quantity     int64   `json:"quantity"`
	Reference    *string `json:"reference"`
	Note         *string `json:"note"`
	PlannedAt    *string `json:"planned_at"`
}

// transferRequest moves quantity between two slots of one warehouse.
type transferRequest struct {
	VendorItemID   int64   `json:"vendor_item_id"`
	WarehouseID    int64   `json:"warehouse_id"`
	FromLocationID int64   `json:"from_location_id"`
	ToLocationID   int64   `json:"to_location_id"`
	Quantity       int64   `json:"quantity"`
	Reference      *string `json:"reference"`
	Note           *string `json:"note"`
	PlannedAt      *string `json:"planned_at"`
}

// adjustRequest sets a slot to an absolute level. Note is the required
// audit reason (PLAN.md §12: absolute quantity + required note).
type adjustRequest struct {
	VendorItemID int64   `json:"vendor_item_id"`
	WarehouseID  int64   `json:"warehouse_id"`
	LocationID   int64   `json:"location_id"`
	Quantity     int64   `json:"quantity"`
	Note         *string `json:"note"`
	PlannedAt    *string `json:"planned_at"`
}

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// parsePlannedAt parses an optional RFC3339 timestamp. Empty means
// "execute immediately"; garbage is a 400.
func parsePlannedAt(w http.ResponseWriter, raw *string) (*time.Time, bool) {
	if raw == nil || *raw == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation", "invalid planned_at (want RFC3339)")
		return nil, false
	}
	utc := t.UTC()
	return &utc, true
}

func (s *Server) callerID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := auth.IdentityFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "authentication required")
		return 0, false
	}
	return id.UserID, true
}

func (s *Server) handleReceive(w http.ResponseWriter, r *http.Request) {
	var req slotRequest
	if !decodeBody(w, r, &req) {
		return
	}
	planned, ok := parsePlannedAt(w, req.PlannedAt)
	if !ok {
		return
	}
	createdBy, ok := s.callerID(w, r)
	if !ok {
		return
	}
	res, err := inventory.Receive(s.db, req.VendorItemID, req.WarehouseID, req.LocationID,
		req.Quantity, strVal(req.Reference), strVal(req.Note), planned, createdBy)
	if err != nil {
		writeInventoryError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (s *Server) handleRemove(w http.ResponseWriter, r *http.Request) {
	var req slotRequest
	if !decodeBody(w, r, &req) {
		return
	}
	planned, ok := parsePlannedAt(w, req.PlannedAt)
	if !ok {
		return
	}
	createdBy, ok := s.callerID(w, r)
	if !ok {
		return
	}
	res, err := inventory.Remove(s.db, req.VendorItemID, req.WarehouseID, req.LocationID,
		req.Quantity, strVal(req.Reference), strVal(req.Note), planned, createdBy)
	if err != nil {
		writeInventoryError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (s *Server) handleAdjust(w http.ResponseWriter, r *http.Request) {
	var req adjustRequest
	if !decodeBody(w, r, &req) {
		return
	}
	planned, ok := parsePlannedAt(w, req.PlannedAt)
	if !ok {
		return
	}
	createdBy, ok := s.callerID(w, r)
	if !ok {
		return
	}
	res, err := inventory.Adjust(s.db, req.VendorItemID, req.WarehouseID, req.LocationID,
		req.Quantity, strVal(req.Note), planned, createdBy)
	if err != nil {
		writeInventoryError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (s *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	var req transferRequest
	if !decodeBody(w, r, &req) {
		return
	}
	planned, ok := parsePlannedAt(w, req.PlannedAt)
	if !ok {
		return
	}
	createdBy, ok := s.callerID(w, r)
	if !ok {
		return
	}
	res, err := inventory.Transfer(s.db, req.VendorItemID, req.WarehouseID,
		req.FromLocationID, req.ToLocationID, req.Quantity,
		strVal(req.Reference), strVal(req.Note), planned, createdBy)
	if err != nil {
		writeInventoryError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (s *Server) handleListMovements(w http.ResponseWriter, r *http.Request) {
	var f inventory.MovementFilter
	var ok bool
	if f.VendorItemID, ok = queryIntParam(w, r, "vendor_item_id"); !ok {
		return
	}
	if f.WarehouseID, ok = queryIntParam(w, r, "warehouse_id"); !ok {
		return
	}
	if f.Status, ok = queryStrParam(w, r, "status"); !ok {
		return
	}
	if f.Type, ok = queryStrParam(w, r, "type"); !ok {
		return
	}
	if f.Since, ok = queryTimeParam(w, r, "since"); !ok {
		return
	}
	if f.Until, ok = queryTimeParam(w, r, "until"); !ok {
		return
	}
	list, err := inventory.ListMovements(s.db, f)
	if err != nil {
		writeInventoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"movements": list})
}

func (s *Server) handleExecuteMovement(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	res, err := inventory.Execute(s.db, id)
	if err != nil {
		writeInventoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleCancelMovement(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	m, err := inventory.Cancel(s.db, id)
	if err != nil {
		writeInventoryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"movement": m})
}

// queryStrParam parses an optional string query parameter.
func queryStrParam(w http.ResponseWriter, r *http.Request, name string) (*string, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, true
	}
	return &raw, true
}

// queryTimeParam parses an optional RFC3339 query parameter.
func queryTimeParam(w http.ResponseWriter, r *http.Request, name string) (*time.Time, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation", "invalid "+name+" (want RFC3339)")
		return nil, false
	}
	utc := t.UTC()
	return &utc, true
}
