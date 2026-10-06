// Package httpapi собирает все роуты (зеркало src/app/config/api-routes.config.ts)
// и middleware-цепочку в один http.Handler.
package httpapi

import (
	"log/slog"
	"net/http"

	"forma3d/server/internal/httpapi/handlers"
	"forma3d/server/internal/httpapi/middleware"
)

// prefix — общий префикс всех эндпоинтов API.
const prefix = "/api/v1"

// Лимиты запросов в минуту на IP.
const (
	apiRatePerMinute   = 60
	loginRatePerMinute = 5
)

// Options — зависимости для сборки роутера.
type Options struct {
	// Handlers — корневой HTTP-хендлер.
	Handlers *handlers.Handler
	// Logger — логгер запросов и паник.
	Logger *slog.Logger
	// CORSOrigins — allowlist источников.
	CORSOrigins []string
	// ValidateToken — проверка Bearer-JWT для /admin/*.
	ValidateToken func(token string) error
	// TrustedProxy — брать клиентский IP из X-Forwarded-For (сервер за прокси).
	TrustedProxy bool
}

// New строит роутер с цепочкой recovery → логирование → CORS → лимит тела → rate limit.
func New(opts Options) http.Handler {
	h := opts.Handlers
	adminOnly := middleware.Auth(opts.ValidateToken)
	loginLimiter := middleware.NewIPRateLimiter(loginRatePerMinute, opts.TrustedProxy)

	mux := http.NewServeMux()

	// Публичные роуты.
	mux.HandleFunc("GET "+prefix+"/health", h.Health)
	mux.HandleFunc("GET "+prefix+"/products", h.ListProducts)
	mux.HandleFunc("GET "+prefix+"/products/{id}", h.GetProduct)
	mux.HandleFunc("POST "+prefix+"/orders", h.CreateOrder)
	mux.HandleFunc("POST "+prefix+"/checkout", h.Checkout)
	mux.HandleFunc("POST "+prefix+"/stripe/webhook", h.StripeWebhook)
	mux.Handle("POST "+prefix+"/auth/login", loginLimiter.Middleware(http.HandlerFunc(h.Login)))

	// Админские роуты — только с валидным Bearer-JWT.
	mux.Handle("GET "+prefix+"/admin/orders", adminOnly(http.HandlerFunc(h.AdminListOrders)))
	mux.Handle("PATCH "+prefix+"/admin/orders/{id}", adminOnly(http.HandlerFunc(h.AdminUpdateOrderStatus)))
	mux.Handle("GET "+prefix+"/admin/stats", adminOnly(http.HandlerFunc(h.AdminStats)))
	mux.Handle("POST "+prefix+"/admin/products", adminOnly(http.HandlerFunc(h.AdminCreateProduct)))
	mux.Handle("PUT "+prefix+"/admin/products/{id}", adminOnly(http.HandlerFunc(h.AdminUpdateProduct)))
	mux.Handle("DELETE "+prefix+"/admin/products/{id}", adminOnly(http.HandlerFunc(h.AdminDeleteProduct)))
	mux.Handle("GET "+prefix+"/admin/settings/stripe", adminOnly(http.HandlerFunc(h.AdminGetStripeSettings)))
	mux.Handle("PUT "+prefix+"/admin/settings/stripe", adminOnly(http.HandlerFunc(h.AdminUpdateStripeSettings)))
	mux.Handle("DELETE "+prefix+"/admin/settings/stripe", adminOnly(http.HandlerFunc(h.AdminDeleteStripeSettings)))

	var handler http.Handler = mux
	handler = middleware.NewIPRateLimiter(apiRatePerMinute, opts.TrustedProxy).Middleware(handler)
	handler = middleware.BodyLimit(handler)
	handler = middleware.CORS(opts.CORSOrigins)(handler)
	handler = middleware.Logging(opts.Logger)(handler)
	handler = middleware.Recovery(opts.Logger)(handler)
	return handler
}
