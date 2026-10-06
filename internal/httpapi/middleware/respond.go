// Package middleware — HTTP-middleware: recovery, логирование, CORS,
// лимит тела, rate limit и Bearer-JWT-авторизация.
package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"forma3d/server/internal/models"
)

// writeError пишет единый конверт ошибки API из middleware-слоя.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.NewAPIError(code, message))
}

// clientIP извлекает IP клиента. По умолчанию — из RemoteAddr (заголовкам
// прокси не доверяем); при trustProxy=true (env TRUSTED_PROXY, сервер за
// доверенным обратным прокси) — первый адрес X-Forwarded-For с фолбэком
// на RemoteAddr, если заголовок пуст.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
				return first
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
