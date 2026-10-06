package handlers

import "net/http"

// ListProducts — GET /products?category=…: каталог с опциональным фильтром.
func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.products.List(r.Context(), r.URL.Query().Get("category"))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

// GetProduct — GET /products/{id}: продукт или 404.
func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	product, err := h.products.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, product)
}
