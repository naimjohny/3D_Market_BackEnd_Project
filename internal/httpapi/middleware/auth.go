package middleware

import (
	"net/http"
	"strings"
)

// bearerPrefix — ожидаемая схема заголовка Authorization.
const bearerPrefix = "Bearer "

// Auth пропускает запрос только с валидным Bearer-JWT (проверка — validate).
func Auth(validate func(token string) error) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, bearerPrefix) {
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
			if token == "" || validate(token) != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
