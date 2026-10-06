// Package handlers — тонкие HTTP-хендлеры: парсинг → сервис → ответ.
package handlers

import (
	"log/slog"

	"forma3d/server/internal/services"
)

// Handler объединяет сервисы, доступные HTTP-слою.
type Handler struct {
	products *services.ProductService
	orders   *services.OrderService
	auth     *services.AuthService
	payments *services.PaymentService
	settings *services.SettingsService
	logger   *slog.Logger
}

// New создаёт корневой HTTP-хендлер со всеми зависимостями.
func New(
	products *services.ProductService,
	orders *services.OrderService,
	auth *services.AuthService,
	payments *services.PaymentService,
	settings *services.SettingsService,
	logger *slog.Logger,
) *Handler {
	return &Handler{products: products, orders: orders, auth: auth, payments: payments, settings: settings, logger: logger}
}
