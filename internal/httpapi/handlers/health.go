package handlers

import (
	"net/http"

	"forma3d/server/internal/models"
)

// Health — GET /health: проверка живости сервиса.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, models.HealthResponse{Status: "ok"})
}
