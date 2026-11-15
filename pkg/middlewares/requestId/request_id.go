package requestId

import (
	"SaaS_Pipeline/pkg/logger"
	"net/http"

	"github.com/google/uuid"
)

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		r.Header.Set("X-Request-Id", requestID)
		w.Header().Set("X-Request-Id", requestID)
		ctx := logger.ContextWithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
