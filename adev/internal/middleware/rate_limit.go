package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimitConfig holds per-IP rate limiter settings.
type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
}

// DefaultRateLimitConfig returns sensible defaults (10 req/s, burst 20).
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 10,
		Burst:             20,
	}
}

// RateLimit returns middleware that applies per-IP token bucket rate limiting.
func RateLimit(cfg RateLimitConfig) Middleware {
	var mu sync.Mutex
	limiters := make(map[string]*rate.Limiter)

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		if lim, ok := limiters[ip]; ok {
			return lim
		}
		lim := rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.Burst)
		limiters[ip] = lim
		return lim
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ip == "" {
				ip = r.RemoteAddr
			}
			if !getLimiter(ip).Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
