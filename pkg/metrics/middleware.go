package metrics

import (
	"net/http"
	"strconv"
	"time"
)

// ResponseWriter wraps http.ResponseWriter to capture status code and response size
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int64
}

// NewResponseWriter creates a new ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += int64(size)
	return size, err
}

// StatusCode returns the captured status code
func (rw *ResponseWriter) StatusCode() int {
	return rw.statusCode
}

// Size returns the captured response size
func (rw *ResponseWriter) Size() int64 {
	return rw.size
}

// MetricsMiddleware provides HTTP request instrumentation
type MetricsMiddleware struct {
	metrics *SimpleMetrics
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(metrics *SimpleMetrics) *MetricsMiddleware {
	return &MetricsMiddleware{
		metrics: metrics,
	}
}

// Middleware returns the metrics middleware function
func (mm *MetricsMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture metrics
		rw := NewResponseWriter(w)

		// Get request size
		requestSize := r.ContentLength
		if requestSize < 0 {
			requestSize = 0
		}

		// Process the request
		next.ServeHTTP(rw, r)

		// Calculate duration
		duration := time.Since(start)

		// Extract route from path (simplified)
		route := extractRoute(r.URL.Path)

		// Record metrics
		statusCode := strconv.Itoa(rw.StatusCode())
		mm.metrics.RecordRequest(
			r.Method,
			route,
			statusCode,
			duration,
			requestSize,
			rw.Size(),
		)
	})
}

// extractRoute extracts a simplified route from the path
func extractRoute(path string) string {
	// Map specific paths to route names for better metrics grouping
	switch {
	case path == "/livez":
		return "/livez"
	case path == "/readyz":
		return "/readyz"
	case path == "/metrics":
		return "/metrics"
	case path == "/verify":
		return "/verify"
	case path == "/batch-verify":
		return "/batch-verify"
	case path == "/extract-fields":
		return "/extract-fields"
	case path == "/status":
		return "/status"
	case path == "/api/v1/execute":
		return "/api/v1/execute"
	case path == "/api/v1/artifact/info":
		return "/api/v1/artifact/info"
	case len(path) > 15 && path[:15] == "/api/v1/receipts/":
		return "/api/v1/receipts/{id}"
	default:
		return "unknown"
	}
}
