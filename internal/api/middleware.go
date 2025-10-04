package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// SecurityConfig holds security configuration
type SecurityConfig struct {
	APIKeys      []string
	MaxBodyBytes int64
	IPRPS        int
	IPBurst      int
	KeyRPS       int
	KeyBurst     int
	CORSOrigins  []string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// LoadSecurityConfig loads security configuration from environment variables
func LoadSecurityConfig() *SecurityConfig {
	config := &SecurityConfig{
		MaxBodyBytes: 1000000, // 1MB default
		IPRPS:        10,      // 10 req/s per IP
		IPBurst:      20,      // burst 20
		KeyRPS:       20,      // 20 req/s per API key
		KeyBurst:     40,      // burst 40
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Load API keys
	if keys := os.Getenv("OCX_API_KEYS"); keys != "" {
		config.APIKeys = strings.Split(keys, ",")
		for i, key := range config.APIKeys {
			config.APIKeys[i] = strings.TrimSpace(key)
		}
	}

	// Load body size limit
	if size := os.Getenv("OCX_MAX_BODY_BYTES"); size != "" {
		if parsed, err := strconv.ParseInt(size, 10, 64); err == nil {
			config.MaxBodyBytes = parsed
		}
	}

	// Load rate limits
	if rps := os.Getenv("OCX_RL_IP_RPS"); rps != "" {
		if parsed, err := strconv.Atoi(rps); err == nil {
			config.IPRPS = parsed
		}
	}
	if burst := os.Getenv("OCX_RL_IP_BURST"); burst != "" {
		if parsed, err := strconv.Atoi(burst); err == nil {
			config.IPBurst = parsed
		}
	}
	if rps := os.Getenv("OCX_RL_KEY_RPS"); rps != "" {
		if parsed, err := strconv.Atoi(rps); err == nil {
			config.KeyRPS = parsed
		}
	}
	if burst := os.Getenv("OCX_RL_KEY_BURST"); burst != "" {
		if parsed, err := strconv.Atoi(burst); err == nil {
			config.KeyBurst = parsed
		}
	}

	// Load CORS origins
	if origins := os.Getenv("OCX_CORS_ORIGINS"); origins != "" {
		config.CORSOrigins = strings.Split(origins, ",")
		for i, origin := range config.CORSOrigins {
			config.CORSOrigins[i] = strings.TrimSpace(origin)
		}
	}

	return config
}

// RateLimiter manages rate limiting for IPs and API keys
type RateLimiter struct {
	ipLimiters  map[string]*rate.Limiter
	keyLimiters map[string]*rate.Limiter
	config      *SecurityConfig
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *SecurityConfig) *RateLimiter {
	return &RateLimiter{
		ipLimiters:  make(map[string]*rate.Limiter),
		keyLimiters: make(map[string]*rate.Limiter),
		config:      config,
	}
}

// GetIPLimiter gets or creates a rate limiter for an IP
func (rl *RateLimiter) GetIPLimiter(ip string) *rate.Limiter {
	if limiter, exists := rl.ipLimiters[ip]; exists {
		return limiter
	}

	limiter := rate.NewLimiter(rate.Limit(rl.config.IPRPS), rl.config.IPBurst)
	rl.ipLimiters[ip] = limiter
	return limiter
}

// GetKeyLimiter gets or creates a rate limiter for an API key
func (rl *RateLimiter) GetKeyLimiter(key string) *rate.Limiter {
	if limiter, exists := rl.keyLimiters[key]; exists {
		return limiter
	}

	limiter := rate.NewLimiter(rate.Limit(rl.config.KeyRPS), rl.config.KeyBurst)
	rl.keyLimiters[key] = limiter
	return limiter
}

// SecurityMiddleware provides comprehensive security for HTTP handlers
type SecurityMiddleware struct {
	Config      *SecurityConfig
	rateLimiter *RateLimiter
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware() *SecurityMiddleware {
	config := LoadSecurityConfig()
	return &SecurityMiddleware{
		Config:      config,
		rateLimiter: NewRateLimiter(config),
	}
}

// Middleware returns the security middleware function
func (sm *SecurityMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. CORS handling
		if !sm.handleCORS(w, r) {
			return
		}

		// 2. API key authentication
		apiKey, ok := sm.authenticate(w, r)
		if !ok {
			return
		}

		// 3. Rate limiting
		if !sm.rateLimit(w, r, apiKey) {
			return
		}

		// 4. Body size limiting
		if !sm.limitBodySize(w, r) {
			return
		}

		// 5. Input validation
		if !sm.validateInput(w, r) {
			return
		}

		// 6. Add security headers
		sm.addSecurityHeaders(w)

		// 7. Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// handleCORS handles CORS preflight and origin validation
func (sm *SecurityMiddleware) handleCORS(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		if sm.isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, OCX-API-Key, Idempotency-Key")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusOK)
			return false
		}
		http.Error(w, "CORS origin not allowed", http.StatusForbidden)
		return false
	}

	// Validate origin for actual requests
	if origin != "" && !sm.isAllowedOrigin(origin) {
		http.Error(w, "CORS origin not allowed", http.StatusForbidden)
		return false
	}

	// Set CORS headers for actual requests
	if sm.isAllowedOrigin(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	return true
}

// isAllowedOrigin checks if an origin is allowed
func (sm *SecurityMiddleware) isAllowedOrigin(origin string) bool {
	if len(sm.Config.CORSOrigins) == 0 {
		return true // Allow all if no origins configured
	}

	for _, allowed := range sm.Config.CORSOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

// authenticate validates API key authentication
func (sm *SecurityMiddleware) authenticate(w http.ResponseWriter, r *http.Request) (string, bool) {
	// If no API keys are configured, allow all requests
	if len(sm.Config.APIKeys) == 0 {
		return "", true
	}

	// Support both header names
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = r.Header.Get("OCX-API-Key")
	}
	if apiKey == "" {
		sm.sendError(w, "E001", "Missing API key", http.StatusUnauthorized)
		return "", false
	}

	// Validate API key
	if !sm.isValidAPIKey(apiKey) {
		sm.sendError(w, "E002", "Invalid API key", http.StatusUnauthorized)
		return "", false
	}

	return apiKey, true
}

// isValidAPIKey checks if an API key is valid
func (sm *SecurityMiddleware) isValidAPIKey(key string) bool {
	if len(sm.Config.APIKeys) == 0 {
		return true // Allow all if no keys configured
	}

	for _, validKey := range sm.Config.APIKeys {
		if key == validKey {
			return true
		}
	}
	return false
}

// rateLimit applies rate limiting
func (sm *SecurityMiddleware) rateLimit(w http.ResponseWriter, r *http.Request, apiKey string) bool {
	// Get client IP
	ip := sm.getClientIP(r)

	// Check IP rate limit
	ipLimiter := sm.rateLimiter.GetIPLimiter(ip)
	if !ipLimiter.Allow() {
		sm.sendError(w, "E003", "Rate limit exceeded (IP)", http.StatusTooManyRequests)
		return false
	}

	// Check API key rate limit
	keyLimiter := sm.rateLimiter.GetKeyLimiter(apiKey)
	if !keyLimiter.Allow() {
		sm.sendError(w, "E004", "Rate limit exceeded (API key)", http.StatusTooManyRequests)
		return false
	}

	return true
}

// getClientIP extracts the real client IP
func (sm *SecurityMiddleware) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

// limitBodySize limits request body size
func (sm *SecurityMiddleware) limitBodySize(w http.ResponseWriter, r *http.Request) bool {
	if r.ContentLength > sm.Config.MaxBodyBytes {
		sm.sendError(w, "E005", "Request body too large", http.StatusRequestEntityTooLarge)
		return false
	}

	// Only wrap the request body with a limited reader if it's not nil
	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, sm.Config.MaxBodyBytes)
	}
	return true
}

// validateInput performs basic input validation
func (sm *SecurityMiddleware) validateInput(w http.ResponseWriter, r *http.Request) bool {
	// Validate Content-Type for POST requests
	// Allow CBOR for verify endpoints
	if r.Method == "POST" {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" && contentType != "application/cbor" {
			sm.sendError(w, "E006", "Content-Type must be application/json or application/cbor", http.StatusBadRequest)
			return false
		}
	}

	return true
}

// addSecurityHeaders adds security headers
func (sm *SecurityMiddleware) addSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// sendError sends a standardized error response
func (sm *SecurityMiddleware) sendError(w http.ResponseWriter, code, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// IdempotencyMiddleware handles idempotency keys
type IdempotencyMiddleware struct {
	store Store
}

// NewIdempotencyMiddleware creates a new idempotency middleware
func NewIdempotencyMiddleware(store Store) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{store: store}
}

// Middleware returns the idempotency middleware function
func (im *IdempotencyMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only apply to POST requests
		if r.Method != "POST" {
			next.ServeHTTP(w, r)
			return
		}

		// Check for idempotency key (optional)
		idemKey := r.Header.Get("Idempotency-Key")
		if idemKey == "" {
			// No idempotency key provided, proceed without idempotency checking
			next.ServeHTTP(w, r)
			return
		}

		// Read and hash request body
		bodyBytes, err := im.readRequestBody(r)
		if err != nil {
			im.sendError(w, "E009", "Failed to read request body", http.StatusBadRequest)
			return
		}

		reqHash := sha256.Sum256(bodyBytes)

		// Check for existing idempotency key
		if existingHash, existingResp, found, err := im.store.GetIdempotent(r.Context(), idemKey); err == nil && found {
			if existingHash == reqHash {
				// Same request - return cached response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(existingResp)
				return
			} else {
				// Different request with same key - conflict
				im.sendError(w, "E007", "Idempotency key/body mismatch", http.StatusConflict)
				return
			}
		}

		// Create response recorder to capture response
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(recorder, r)

		// Store response for idempotency
		if recorder.statusCode == http.StatusOK {
			im.store.PutIdempotent(r.Context(), idemKey, reqHash, recorder.body)
		}
	})
}

// readRequestBody reads and restores request body
func (im *IdempotencyMiddleware) readRequestBody(r *http.Request) ([]byte, error) {
	body := make([]byte, r.ContentLength)
	if _, err := r.Body.Read(body); err != nil && err.Error() != "EOF" {
		return nil, err
	}

	// Restore body for next handler
	r.Body = &bodyReader{data: body}
	return body, nil
}

// sendError sends a standardized error response
func (im *IdempotencyMiddleware) sendError(w http.ResponseWriter, code, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// responseRecorder captures HTTP response
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(data []byte) (int, error) {
	rr.body = append(rr.body, data...)
	return rr.ResponseWriter.Write(data)
}

// bodyReader implements io.ReadCloser for request body
type bodyReader struct {
	data []byte
	pos  int
}

func (br *bodyReader) Read(p []byte) (n int, err error) {
	if br.pos >= len(br.data) {
		return 0, fmt.Errorf("EOF")
	}

	n = copy(p, br.data[br.pos:])
	br.pos += n
	return n, nil
}

func (br *bodyReader) Close() error {
	return nil
}

// Store interface for idempotency (matches receipt.Store)
type Store interface {
	PutIdempotent(ctx context.Context, key string, reqHash [32]byte, respCBOR []byte) (bool, bool, []byte, error)
	GetIdempotent(ctx context.Context, key string) ([32]byte, []byte, bool, error)
}
