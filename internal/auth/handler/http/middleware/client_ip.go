package middleware

import (
	"context"
	"net/http"
)

func XRealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			ctx := context.WithValue(r.Context(), "X-Real-IP", realIP)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}
