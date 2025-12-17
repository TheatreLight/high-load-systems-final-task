package utils

import (
	"net/http"

	"golang.org/x/time/rate"
)

// RateLimiter wraps the rate.Limiter for HTTP rate limiting
type RateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter creates a new RateLimiter with specified rate and burst
func NewRateLimiter(rateLimit rate.Limit, burst int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rateLimit, burst),
	}
}

// Allow checks if a request is allowed under the rate limit
func (rl *RateLimiter) Allow() bool {
	return rl.limiter.Allow()
}

// RateLimitMiddleware creates middleware for rate limiting
// Rate limiter: 1000 req/s with burst 100000 for stable work under load
// Burst is significantly increased to handle peak loads from wrk -c500 test
// With 12 threads and 500 connections, wrk generates ~6500 req/s
// Large burst allows handling initial request spike without blocking,
// after which speed stabilizes at ~1000 req/s
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
