package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Параметры очистки неактивных IP из памяти лимитера.
const (
	cleanupInterval = time.Minute
	visitorTTL      = 3 * time.Minute
)

// visitor — лимитер одного IP и момент последнего обращения.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter — per-IP token-bucket лимитер с фоновой очисткой старых записей.
type IPRateLimiter struct {
	mu         sync.Mutex
	visitors   map[string]*visitor
	limit      rate.Limit
	burst      int
	trustProxy bool
}

// NewIPRateLimiter создаёт лимитер на perMinute запросов в минуту на IP;
// trustProxy — брать IP из X-Forwarded-For (см. clientIP).
func NewIPRateLimiter(perMinute int, trustProxy bool) *IPRateLimiter {
	l := &IPRateLimiter{
		visitors:   make(map[string]*visitor),
		limit:      rate.Limit(float64(perMinute) / 60.0),
		burst:      perMinute,
		trustProxy: trustProxy,
	}
	go l.cleanupLoop()
	return l
}

// Allow сообщает, разрешён ли очередной запрос с данного IP.
func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	v, ok := l.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v.limiter.Allow()
}

// Middleware отвечает 429 при превышении лимита.
func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(clientIP(r, l.trustProxy)) {
			writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// cleanupLoop периодически удаляет IP, не появлявшиеся дольше visitorTTL.
func (l *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-visitorTTL)
		l.mu.Lock()
		for ip, v := range l.visitors {
			if v.lastSeen.Before(cutoff) {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}
