package middleware

import "net/http"

// maxBodyBytes — лимит тела запроса (1 MB).
const maxBodyBytes = 1 << 20

// BodyLimit оборачивает тело запроса в MaxBytesReader (1 MB).
func BodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}
