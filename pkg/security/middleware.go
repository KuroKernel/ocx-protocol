package security

import (
	"net/http"
)

// RequestSizeLimiter limits the size of incoming requests
type RequestSizeLimiter struct {
	maxBytes int64
}

// NewRequestSizeLimiter creates a new request size limiter
func NewRequestSizeLimiter(maxBytes int64) *RequestSizeLimiter {
	return &RequestSizeLimiter{
		maxBytes: maxBytes,
	}
}

// Middleware returns an HTTP middleware that enforces request size limits
func (rsl *RequestSizeLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Limit request body size
		r.Body = http.MaxBytesReader(w, r.Body, rsl.maxBytes)

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware handles CORS headers
type CORSMiddleware struct {
	allowedOrigins []string
	allowedMethods []string
	allowedHeaders []string
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(origins, methods, headers []string) *CORSMiddleware {
	if len(methods) == 0 {
		methods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(headers) == 0 {
		headers = []string{"Content-Type", "Authorization", "X-API-Key"}
	}

	return &CORSMiddleware{
		allowedOrigins: origins,
		allowedMethods: methods,
		allowedHeaders: headers,
	}
}

// Middleware returns an HTTP middleware that adds CORS headers
func (cm *CORSMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range cm.allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", joinStrings(cm.allowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", joinStrings(cm.allowedHeaders, ", "))
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		// Handle preflight request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware adds security headers to responses
type SecurityHeadersMiddleware struct{}

// NewSecurityHeadersMiddleware creates a new security headers middleware
func NewSecurityHeadersMiddleware() *SecurityHeadersMiddleware {
	return &SecurityHeadersMiddleware{}
}

// Middleware returns an HTTP middleware that adds security headers
func (shm *SecurityHeadersMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// joinStrings joins strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}

	return result
}
