package ratelimit

import (
	"net"
	"net/http"
	"strconv"
	"sync"

	"golang.org/x/time/rate"
)

func RateLimit(limit rate.Limit, burst int, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	var (
		mu       sync.Mutex
		limiters = make(map[string]*rate.Limiter)
	)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}
			mu.Lock()
			limiter, exists := limiters[key]
			if !exists {
				limiter = rate.NewLimiter(limit, burst)
				limiters[key] = limiter
			}
			mu.Unlock()

			res := limiter.Reserve()
			if !res.OK() {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			if delay := res.Delay(); delay > 0 {
				res.Cancel()

				retryAfter := int(delay.Seconds())
				if retryAfter == 0 {
					retryAfter = 1
				}

				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func IPKeyFn(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func UserKeyFn(r *http.Request) string {
	return r.Header.Get("X-User-Id")
}
