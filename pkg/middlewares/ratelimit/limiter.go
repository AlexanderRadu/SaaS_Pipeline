package ratelimit

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func RateLimit(limit rate.Limit, burst int, ttl time.Duration, cleanupInterval time.Duration, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	var (
		mu       sync.Mutex
		limiters = make(map[string]*client)
	)

	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			for key, c := range limiters {
				if time.Since(c.lastSeen) > ttl {
					delete(limiters, key)
				}
			}
			mu.Unlock()
		}

	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}
			mu.Lock()
			c, exists := limiters[key]
			if !exists {
				c = &client{limiter: rate.NewLimiter(limit, burst)}
				limiters[key] = c
			}
			c.lastSeen = time.Now()
			mu.Unlock()

			res := c.limiter.Reserve()
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
	return r.Header.Get("X-Real-IP")
}

func UserKeyFn(r *http.Request) string {
	return r.Header.Get("X-User-Id")
}
