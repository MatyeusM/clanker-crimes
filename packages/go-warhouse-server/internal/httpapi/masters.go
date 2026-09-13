package httpapi

import (
	"net/http"
	"strconv"

	"warehouse-server/internal/vendors"
	"warehouse-server/internal/warehouses"
)

// Native masters: vendors, items, vendor items, warehouses, locations.
// Reads need `read`; every mutation needs `write` (PLAN.md §6).

type vendorRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

func (s *Server) handleListVendors(w http.ResponseWriter, r *http.Request) {
	list, err := vendors.ListVendors(s.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vendors": list})
}

func (s *Server) handleCreateVendor(w http.ResponseWriter, r *http.Request) {
	var req vendorRequest
	if !decodeBody(w, r, &req) {
		return
	}
	v, err := vendors.CreateVendor(s.db, req.Name, req.Code)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"vendor": v})
}

func (s *Server) handleGetVendor(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	v, err := vendors.GetVendor(s.db, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vendor": v})
}

func (s *Server) handleUpdateVendor(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req vendorRequest
	if !decodeBody(w, r, &req) {
		return
	}
	v, err := vendors.UpdateVendor(s.db, id, req.Name, req.Code)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vendor": v})
}

type itemRequest struct {
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Unit        string  `json:"unit"`
}

func strOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (s *Server) handleListItems(w http.ResponseWriter, r *http.Request) {
	list, err := vendors.ListItems(s.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list})
}

func (s *Server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var req itemRequest
	if !decodeBody(w, r, &req) {
		return
	}
	it, err := vendors.CreateItem(s.db, req.SKU, req.Name, strOrEmpty(req.Description), req.Unit)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"item": it})
}

func (s *Server) handleGetItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	it, err := vendors.GetItem(s.db, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": it})
}

func (s *Server) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req itemRequest
	if !decodeBody(w, r, &req) {
		return
	}
	it, err := vendors.UpdateItem(s.db, id, req.SKU, req.Name, req.Description, req.Unit)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": it})
}

type vendorItemRequest struct {
	VendorID   int64   `json:"vendor_id"`
	ItemID     int64   `json:"item_id"`
	VendorSKU  string  `json:"vendor_sku"`
	VendorName *string `json:"vendor_name"`
}

func (s *Server) handleListVendorItems(w http.ResponseWriter, r *http.Request) {
	vendorID, ok := queryIntParam(w, r, "vendor_id")
	if !ok {
		return
	}
	itemID, ok := queryIntParam(w, r, "item_id")
	if !ok {
		return
	}
	list, err := vendors.ListVendorItems(s.db, vendorID, itemID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vendor_items": list})
}

func (s *Server) handleCreateVendorItem(w http.ResponseWriter, r *http.Request) {
	var req vendorItemRequest
	if !decodeBody(w, r, &req) {
		return
	}
	vi, err := vendors.CreateVendorItem(s.db, req.VendorID, req.ItemID, req.VendorSKU, req.VendorName)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"vendor_item": vi})
}

func (s *Server) handleGetVendorItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	vi, err := vendors.GetVendorItem(s.db, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vendor_item": vi})
}

func (s *Server) handleUpdateVendorItem(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req vendorItemRequest
	if !decodeBody(w, r, &req) {
		return
	}
	vi, err := vendors.UpdateVendorItem(s.db, id, req.VendorSKU, req.VendorName)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"vendor_item": vi})
}

type warehouseRequest struct {
	Name    string  `json:"name"`
	Code    string  `json:"code"`
	Address *string `json:"address"`
}

func (s *Server) handleListWarehouses(w http.ResponseWriter, r *http.Request) {
	list, err := warehouses.ListWarehouses(s.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"warehouses": list})
}

func (s *Server) handleCreateWarehouse(w http.ResponseWriter, r *http.Request) {
	var req warehouseRequest
	if !decodeBody(w, r, &req) {
		return
	}
	wh, err := warehouses.CreateWarehouse(s.db, req.Name, req.Code, req.Address)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"warehouse": wh})
}

func (s *Server) handleGetWarehouse(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	wh, err := warehouses.GetWarehouse(s.db, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"warehouse": wh})
}

func (s *Server) handleUpdateWarehouse(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req warehouseRequest
	if !decodeBody(w, r, &req) {
		return
	}
	wh, err := warehouses.UpdateWarehouse(s.db, id, req.Name, req.Code, req.Address)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"warehouse": wh})
}

type locationRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (s *Server) handleListLocations(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	list, err := warehouses.ListLocations(s.db, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"locations": list})
}

func (s *Server) handleCreateLocation(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req locationRequest
	if !decodeBody(w, r, &req) {
		return
	}
	l, err := warehouses.CreateLocation(s.db, id, req.Code, req.Name)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"location": l})
}

func (s *Server) handleGetLocation(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	l, err := warehouses.GetLocation(s.db, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"location": l})
}

func (s *Server) handleUpdateLocation(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req locationRequest
	if !decodeBody(w, r, &req) {
		return
	}
	l, err := warehouses.UpdateLocation(s.db, id, req.Code, req.Name)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"location": l})
}

// queryIntParam parses an optional integer query parameter. Absent
// means nil (no filter); present-but-garbage is a 400.
func queryIntParam(w http.ResponseWriter, r *http.Request, name string) (*int64, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, true
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v <= 0 {
		writeError(w, http.StatusBadRequest, "validation", "invalid "+name)
		return nil, false
	}
	return &v, true
}
