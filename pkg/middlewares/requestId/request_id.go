package requestId

import (
	"SaaS_Pipeline/pkg/logger"
	"net/http"

	"github.com/google/uuid"
)

const headerName = "X-Request-Id"

func RequestIDMiddleware(next http.Handler) http.Handler {
	return handleRequestId(next, true)
}

func InternalRequestIDMiddleware(next http.Handler) http.Handler {
	return handleRequestId(next, false)
}

func handleRequestId(next http.Handler, writeResponseHeader bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(headerName)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		r.Header.Set(headerName, requestID)
		if writeResponseHeader {
			w.Header().Set(headerName, requestID)

		}

		ctx := logger.ContextWithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
