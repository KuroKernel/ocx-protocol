package api

import (
	"net/http"
	"time"
)

func (s *Server) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Note: Active connections tracking would be implemented with connection pooling
		// metrics.ActiveConnections.Inc()
		// defer metrics.ActiveConnections.Dec()

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
				s.metrics.RecordRequest(r.Method, r.URL.Path, "200", duration, 0, 0)
			} else {
				s.metrics.RecordRequest(r.Method, r.URL.Path, "error", duration, 0, 0)
			}
		case "/verify":
			if wrapped.statusCode == 200 {
				s.metrics.RecordVerify("go", "success", "", duration)
			} else {
				s.metrics.RecordVerify("go", "error", "http_error", duration)
			}
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
