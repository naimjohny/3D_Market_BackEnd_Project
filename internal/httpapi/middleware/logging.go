package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder запоминает статус ответа для лога.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader сохраняет статус перед отправкой.
func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Logging пишет метод, путь, статус и длительность запроса (без тел и токенов).
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration", time.Since(start).String())
		})
	}
}
