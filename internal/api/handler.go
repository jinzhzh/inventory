package api

import (
	"code3-inventory/internal/db"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

type Handler struct {
	store *db.Store
}

func New(store *db.Store) *Handler {
	return &Handler{store: store}
}

// ---- SKU ----

func (h *Handler) ListSKUs(w http.ResponseWriter, r *http.Request) {
	skus, err := h.store.ListSKUs(r.Context())
	if err != nil {
		writeError(w, 500, "查询失败: "+err.Error())
		return
	}
	writeJSON(w, 200, skus)
}

func (h *Handler) CreateSKU(w http.ResponseWriter, r *http.Request) {
	var sku db.SKU
	if err := readJSON(r, &sku); err != nil {
		writeError(w, 400, "JSON解析失败")
		return
	}
	if sku.Name == "" {
		writeError(w, 400, "商品名称不能为空")
		return
	}
	if sku.SKUCode == "" {
		writeError(w, 400, "SKU编码不能为空")
		return
	}
	sku.ID = uuid.New().String()
	if err := h.store.CreateSKU(r.Context(), sku); err != nil {
		writeError(w, 500, "创建失败: "+err.Error())
		return
	}
	writeJSON(w, 201, sku)
}

func (h *Handler) GetSKU(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sku, err := h.store.GetSKU(r.Context(), id)
	if err != nil {
		writeError(w, 404, "SKU不存在")
		return
	}
	writeJSON(w, 200, sku)
}

func (h *Handler) UpdateSKU(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var sku db.SKU
	if err := readJSON(r, &sku); err != nil {
		writeError(w, 400, "JSON解析失败")
		return
	}
	sku.ID = id
	if err := h.store.UpdateSKU(r.Context(), sku); err != nil {
		writeError(w, 500, "更新失败: "+err.Error())
		return
	}
	writeJSON(w, 200, sku)
}

func (h *Handler) DeleteSKU(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteSKU(r.Context(), id); err != nil {
		writeError(w, 500, "删除失败: "+err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"message": "已删除"})
}

// ---- Inbound / Outbound ----

type StockReq struct {
	SKU      string  `json:"sku"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
	Remark   string  `json:"remark"`
	Operator string  `json:"operator"`
}

func (h *Handler) Inbound(w http.ResponseWriter, r *http.Request) {
	var req StockReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, 400, "JSON解析失败")
		return
	}
	if req.SKU == "" || req.Quantity <= 0 {
		writeError(w, 400, "SKU和数量必须填写，数量需大于0")
		return
	}
	ctx := r.Context()

	// Check SKU exists
	_, err := h.store.GetSKU(ctx, req.SKU)
	if err != nil {
		writeError(w, 404, "SKU不存在")
		return
	}

	tx := db.Transaction{
		ID:       uuid.New().String(),
		SKU:      req.SKU,
		Type:     "inbound",
		Quantity: req.Quantity,
		Price:    req.Price,
		Total:    float64(req.Quantity) * req.Price,
		Remark:   req.Remark,
		Operator: req.Operator,
	}
	if err := h.store.AddTransaction(ctx, tx); err != nil {
		writeError(w, 500, "入库失败: "+err.Error())
		return
	}
	if err := h.store.AdjustStock(ctx, req.SKU, req.Quantity); err != nil {
		writeError(w, 500, "库存更新失败: "+err.Error())
		return
	}
	writeJSON(w, 201, tx)
}

func (h *Handler) Outbound(w http.ResponseWriter, r *http.Request) {
	var req StockReq
	if err := readJSON(r, &req); err != nil {
		writeError(w, 400, "JSON解析失败")
		return
	}
	if req.SKU == "" || req.Quantity <= 0 {
		writeError(w, 400, "SKU和数量必须填写，数量需大于0")
		return
	}
	ctx := r.Context()

	// Check stock
	sku, err := h.store.GetSKU(ctx, req.SKU)
	if err != nil {
		writeError(w, 404, "SKU不存在")
		return
	}
	if sku.CurrentQty < req.Quantity {
		writeError(w, 400, fmt.Sprintf("库存不足：当前 %d，申请 %d", sku.CurrentQty, req.Quantity))
		return
	}

	tx := db.Transaction{
		ID:       uuid.New().String(),
		SKU:      req.SKU,
		Type:     "outbound",
		Quantity: req.Quantity,
		Price:    req.Price,
		Total:    float64(req.Quantity) * req.Price,
		Remark:   req.Remark,
		Operator: req.Operator,
	}
	if err := h.store.AddTransaction(ctx, tx); err != nil {
		writeError(w, 500, "出库失败: "+err.Error())
		return
	}
	if err := h.store.AdjustStock(ctx, req.SKU, -req.Quantity); err != nil {
		writeError(w, 500, "库存更新失败: "+err.Error())
		return
	}
	writeJSON(w, 201, tx)
}

// ---- Inventory / Alerts / Ledger / Stats ----

func (h *Handler) Inventory(w http.ResponseWriter, r *http.Request) {
	inv, err := h.store.Inventory(r.Context())
	if err != nil {
		writeError(w, 500, "查询失败: "+err.Error())
		return
	}
	writeJSON(w, 200, inv)
}

func (h *Handler) Alerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.store.Alerts(r.Context())
	if err != nil {
		writeError(w, 500, "查询失败: "+err.Error())
		return
	}
	writeJSON(w, 200, alerts)
}

func (h *Handler) Ledger(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	txs, err := h.store.Ledger(r.Context(), limit, offset)
	if err != nil {
		writeError(w, 500, "查询失败: "+err.Error())
		return
	}
	writeJSON(w, 200, txs)
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.Stats(r.Context())
	if err != nil {
		writeError(w, 500, "查询失败: "+err.Error())
		return
	}
	writeJSON(w, 200, stats)
}

// helpers

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}