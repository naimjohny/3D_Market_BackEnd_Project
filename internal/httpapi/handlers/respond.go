package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"forma3d/server/internal/models"
	"forma3d/server/internal/services"
)

// writeJSON сериализует v с указанным статусом.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError пишет единый конверт ошибки: {"error":{"code","message"}}.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, models.NewAPIError(code, message))
}

// decodeJSON парсит тело запроса; ошибки формата отдаёт как 400/413.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	var maxBytesErr *http.MaxBytesError
	err := json.NewDecoder(r.Body).Decode(dst)
	switch {
	case err == nil:
		return true
	case errors.As(err, &maxBytesErr):
		writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds 1 MB")
	default:
		writeError(w, http.StatusBadRequest, "bad_json", "request body must be valid JSON")
	}
	return false
}

// writeServiceError мапит ошибки сервисного слоя на HTTP-статусы;
// непредвиденные ошибки логируются и уходят клиенту как 500 без деталей.
func (h *Handler) writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *services.ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeError(w, http.StatusBadRequest, "validation_error", validationErr.Msg)
	case errors.Is(err, services.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "entity not found")
	case errors.Is(err, services.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", "entity already exists")
	case errors.Is(err, services.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	case errors.Is(err, services.ErrStripeInvalidKey):
		writeError(w, http.StatusBadRequest, "stripe_invalid_key", "Stripe rejected the provided secret key")
	default:
		h.logger.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		writeError(w, http.StatusInternalServerError, "internal", "internal server error")
	}
}
