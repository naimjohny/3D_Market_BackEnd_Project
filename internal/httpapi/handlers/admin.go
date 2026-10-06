package handlers

import (
	"net/http"

	"forma3d/server/internal/models"
)

// AdminListOrders — GET /admin/orders: все заказы, новые сверху.
func (h *Handler) AdminListOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.orders.ListAll(r.Context())
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

// AdminUpdateOrderStatus — PATCH /admin/orders/{id}: смена статуса заказа.
func (h *Handler) AdminUpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	var req models.OrderStatusUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	order, err := h.orders.UpdateStatus(r.Context(), r.PathValue("id"), req.Status)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

// AdminStats — GET /admin/stats: сводка по заказам.
func (h *Handler) AdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.orders.Stats(r.Context())
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// AdminCreateProduct — POST /admin/products: новый продукт (id генерируется при пустом).
func (h *Handler) AdminCreateProduct(w http.ResponseWriter, r *http.Request) {
	var product models.Product
	if !decodeJSON(w, r, &product) {
		return
	}
	created, err := h.products.Create(r.Context(), product)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// AdminUpdateProduct — PUT /admin/products/{id}: полная перезапись продукта.
func (h *Handler) AdminUpdateProduct(w http.ResponseWriter, r *http.Request) {
	var product models.Product
	if !decodeJSON(w, r, &product) {
		return
	}
	updated, err := h.products.Update(r.Context(), r.PathValue("id"), product)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// AdminDeleteProduct — DELETE /admin/products/{id} → 204.
func (h *Handler) AdminDeleteProduct(w http.ResponseWriter, r *http.Request) {
	if err := h.products.Delete(r.Context(), r.PathValue("id")); err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
