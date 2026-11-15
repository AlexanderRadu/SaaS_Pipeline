package logging

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func LoggingMiddleware(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			clientIP := r.Header.Get("X-Real-IP")
			if clientIP == "" {
				clientIP = r.Header.Get("X-Forwarded-For")
			}
			if clientIP == "" {
				clientIP = r.RemoteAddr
			}

			respWriter := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}
			next.ServeHTTP(respWriter, r)

			duration := time.Since(start)

			log.InfoContext(r.Context(), "HTTP Request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("status", strconv.Itoa(respWriter.status)),
				slog.String("duration", duration.String()),
				slog.String("client_ip", clientIP),
				slog.String("user_agent", r.UserAgent()),
			)
		})
	}
}
