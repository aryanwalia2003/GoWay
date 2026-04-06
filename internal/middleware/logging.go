package middleware

import (
	"context"
	"net/http"
	"time"

	"awb-gen/internal/logger"
)

type loggingRec struct {
	http.ResponseWriter
	status int
}

func (r *loggingRec) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Logging injects a trace_id and records request lifecycle telemetry.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		traceID := logger.GenerateTraceID()
		r = r.WithContext(context.WithValue(r.Context(), "trace_id", traceID))

		rec := &loggingRec{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		logger.LogRequest(traceID, r.Method, r.URL.Path, rec.status, time.Since(start).Milliseconds(), nil)
	})
}
