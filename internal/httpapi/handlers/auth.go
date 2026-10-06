package handlers

import (
	"net/http"

	"forma3d/server/internal/models"
)

// Login — POST /auth/login: выдача JWT администратору.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	resp, err := h.auth.Login(&req)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
