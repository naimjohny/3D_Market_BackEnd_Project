package handlers

import (
	"net/http"

	"forma3d/server/internal/models"
)

// CreateOrder — POST /orders: заявка на кастомную модель.
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req models.OrderRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	resp, err := h.orders.SubmitCustom(r.Context(), &req)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// Checkout — POST /checkout: оформление корзины (total считается из БД).
func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
	var req models.CheckoutRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	resp, err := h.orders.Checkout(r.Context(), &req)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}
