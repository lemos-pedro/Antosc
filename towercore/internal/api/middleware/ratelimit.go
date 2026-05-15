package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"towercore/pkg/apierror"
)

type clientWindow struct {
	windowStart time.Time
	count       int
}

// RateLimitPerMinute limita requests por cliente (X-API-Key + IP).
func RateLimitPerMinute(limit int) func(http.Handler) http.Handler {
	if limit <= 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	var (
		mu      sync.Mutex
		clients = map[string]clientWindow{}
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := rateLimitKey(r)
			now := time.Now().UTC()
			window := now.Truncate(time.Minute)

			mu.Lock()
			state := clients[key]
			if state.windowStart.IsZero() || state.windowStart.Before(window) {
				state.windowStart = window
				state.count = 0
			}
			state.count++
			clients[key] = state
			allowed := state.count <= limit
			mu.Unlock()

			if !allowed {
				apierror.Write(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitKey(r *http.Request) string {
	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = host
	}
	apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if apiKey == "" {
		apiKey = "-"
	}
	return apiKey + "|" + ip
}
