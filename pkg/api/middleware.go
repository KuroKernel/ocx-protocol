package api

import (
	"net/http"
	"time"

	"ocx.local/pkg/metrics"
)

func (s *Server) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Increment active connections
		metrics.ActiveConnections.Inc()
		defer metrics.ActiveConnections.Dec()
		
		// Wrap response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     200,
		}
		
		next.ServeHTTP(wrapped, r)
		
		// Record metrics based on endpoint
		duration := time.Since(start)
		
		switch r.URL.Path {
		case "/execute":
			if wrapped.statusCode == 200 {
				metrics.RecordExecution(0, duration, true) // Cycles will be set by handler
			} else {
				metrics.RecordExecution(0, duration, false)
			}
		case "/verify":
			metrics.RecordVerification(duration, wrapped.statusCode == 200)
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// BodySizeLimitMiddleware limits request body size
func BodySizeLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
