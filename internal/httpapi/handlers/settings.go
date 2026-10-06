package handlers

import (
	"net/http"

	"forma3d/server/internal/models"
)

// AdminGetStripeSettings — GET /admin/settings/stripe: статус настроек
// Stripe без самих ключей.
func (h *Handler) AdminGetStripeSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.settings.StripeStatus())
}

// AdminUpdateStripeSettings — PUT /admin/settings/stripe: валидация ключа
// через Stripe API и сохранение в БД в зашифрованном виде.
func (h *Handler) AdminUpdateStripeSettings(w http.ResponseWriter, r *http.Request) {
	var req models.StripeSettingsUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	status, err := h.settings.SetStripe(r.Context(), req.SecretKey, req.WebhookSecret)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// AdminDeleteStripeSettings — DELETE /admin/settings/stripe → 204:
// очистка БД-настроек; env-фолбэк продолжает работать.
func (h *Handler) AdminDeleteStripeSettings(w http.ResponseWriter, r *http.Request) {
	if err := h.settings.ClearStripe(r.Context()); err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
