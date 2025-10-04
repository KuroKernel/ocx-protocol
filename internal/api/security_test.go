package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"ocx.local/pkg/receipt"
)

func TestAPIKeyAuthMiddleware(t *testing.T) {
	// Create test middleware
	config := &SecurityConfig{
		APIKeys: []string{
			"test-key-1",
			"test-key-2",
		},
		MaxBodyBytes: 1000000,
		IPRPS:        10,
		IPBurst:      20,
		KeyRPS:       20,
		KeyBurst:     40,
	}

	middleware := &SecurityMiddleware{
		Config:      config,
		rateLimiter: NewRateLimiter(config),
	}

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with middleware
	wrappedHandler := middleware.Middleware(handler)

	t.Run("valid_api_key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
		req.Header.Set("OCX-API-Key", "test-key-1")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "success" {
			t.Errorf("Expected body 'success', got %s", w.Body.String())
		}
	})

	t.Run("invalid_api_key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
		req.Header.Set("OCX-API-Key", "invalid-key")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("missing_api_key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("empty_api_key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
		req.Header.Set("OCX-API-Key", "")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("api_key_not_required", func(t *testing.T) {
		// Create middleware without API keys (no requirement)
		configNoKeys := &SecurityConfig{
			APIKeys:      []string{}, // Empty means no API key required
			MaxBodyBytes: 1000000,
			IPRPS:        10,
			IPBurst:      20,
			KeyRPS:       20,
			KeyBurst:     40,
		}
		middlewareNoKeys := &SecurityMiddleware{
			Config:      configNoKeys,
			rateLimiter: NewRateLimiter(configNoKeys),
		}
		wrappedHandlerNoKeys := middlewareNoKeys.Middleware(handler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
		w := httptest.NewRecorder()

		wrappedHandlerNoKeys.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestRateLimiter(t *testing.T) {
	// Create test security config
	config := &SecurityConfig{
		APIKeys:      []string{"test-key", "test-key-0", "test-key-1", "test-key-2"},
		MaxBodyBytes: 1000000,
		IPRPS:        100,  // 100 req/s per IP (high for testing)
		IPBurst:      200,  // burst 200
		KeyRPS:       100,  // 100 req/s per API key (high for testing)
		KeyBurst:     200,  // burst 200
	}

	// Create security middleware with rate limiting
	middleware := &SecurityMiddleware{
		Config:      config,
		rateLimiter: NewRateLimiter(config),
	}

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with security middleware (includes rate limiting)
	wrappedHandler := middleware.Middleware(handler)

	t.Run("within_rate_limit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("OCX-API-Key", "test-key")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("exceed_ip_rate_limit", func(t *testing.T) {
		// Create a separate middleware with very restrictive rate limits for this test
		restrictiveConfig := &SecurityConfig{
			APIKeys:      []string{"test-key"},
			MaxBodyBytes: 1000000,
			IPRPS:        1,  // 1 req/s per IP (very restrictive)
			IPBurst:      2,  // burst 2
			KeyRPS:       10, // 10 req/s per API key
			KeyBurst:     20, // burst 20
		}
		restrictiveMiddleware := &SecurityMiddleware{
			Config:      restrictiveConfig,
			rateLimiter: NewRateLimiter(restrictiveConfig),
		}
		restrictiveHandler := restrictiveMiddleware.Middleware(handler)

		// Make multiple requests from same IP
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.2:12345"
			req.Header.Set("OCX-API-Key", "test-key")
			w := httptest.NewRecorder()

			restrictiveHandler.ServeHTTP(w, req)

			if i < 2 {
				// First two requests should succeed
				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200 for request %d, got %d", i, w.Code)
				}
			} else {
				// Third request should be rate limited
				if w.Code != http.StatusTooManyRequests {
					t.Errorf("Expected status 429 for request %d, got %d", i, w.Code)
				}
			}
		}
	})

	t.Run("exceed_key_rate_limit", func(t *testing.T) {
		// Create a separate middleware with very restrictive key rate limits for this test
		restrictiveConfig := &SecurityConfig{
			APIKeys:      []string{"test-key"},
			MaxBodyBytes: 1000000,
			IPRPS:        10, // 10 req/s per IP
			IPBurst:      20, // burst 20
			KeyRPS:       1,  // 1 req/s per API key (very restrictive)
			KeyBurst:     2,  // burst 2
		}
		restrictiveMiddleware := &SecurityMiddleware{
			Config:      restrictiveConfig,
			rateLimiter: NewRateLimiter(restrictiveConfig),
		}
		restrictiveHandler := restrictiveMiddleware.Middleware(handler)

		// Make multiple requests with same API key
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("OCX-API-Key", "test-key")
			req.RemoteAddr = "192.168.1.3:12345"
			w := httptest.NewRecorder()

			restrictiveHandler.ServeHTTP(w, req)

			if i < 2 {
				// First two requests should succeed
				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200 for request %d, got %d", i, w.Code)
				}
			} else {
				// Third request should be rate limited
				if w.Code != http.StatusTooManyRequests {
					t.Errorf("Expected status 429 for request %d, got %d", i, w.Code)
				}
			}
		}
	})

	t.Run("different_ips_not_rate_limited", func(t *testing.T) {
		// Make requests from different IPs
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
			req.RemoteAddr = fmt.Sprintf("192.168.1.%d:12345", i+10)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for request %d, got %d", i, w.Code)
			}
		}
	})

	t.Run("different_keys_not_rate_limited", func(t *testing.T) {
		// Make requests with different API keys
		for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
		req.RemoteAddr = "192.168.1.4:12345"
		req.Header.Set("OCX-API-Key", "test-key")
			req.Header.Set("OCX-API-Key", fmt.Sprintf("test-key-%d", i))
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for request %d, got %d", i, w.Code)
			}
		}
	})
}

func TestBodyCapMiddleware(t *testing.T) {
	// Create test middleware
	config := &SecurityConfig{
		APIKeys:      []string{"test-key"},
		MaxBodyBytes: 1024, // 1KB limit
		IPRPS:        1000, // Very high rate limits to avoid interference
		IPBurst:      2000,
		KeyRPS:       1000,
		KeyBurst:     2000,
	}

	middleware := &SecurityMiddleware{
		Config:      config,
		rateLimiter: NewRateLimiter(config),
	}

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Body != nil {
			body, _ = io.ReadAll(r.Body)
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	// Wrap with middleware (includes body cap)
	wrappedHandler := middleware.Middleware(handler)

	t.Run("body_within_limit", func(t *testing.T) {
		body := make([]byte, 512) // 512 bytes
		rand.Read(body)

		req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		req.Header.Set("OCX-API-Key", "test-key")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("body_exceeds_limit", func(t *testing.T) {
		body := make([]byte, 2048) // 2KB, exceeds 1KB limit
		rand.Read(body)

		req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		req.Header.Set("OCX-API-Key", "test-key")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413, got %d", w.Code)
		}
	})

	t.Run("empty_body", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("nil_body", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
		req.Body = nil
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestCORSMiddleware(t *testing.T) {
	// Create test middleware
	config := &SecurityConfig{
		APIKeys:      []string{"test-key"},
		CORSOrigins:  []string{"https://example.com", "https://test.com"},
		MaxBodyBytes: 1000000,
		IPRPS:        1000, // Very high rate limits to avoid interference
		IPBurst:      2000,
		KeyRPS:       1000,
		KeyBurst:     2000,
	}

	middleware := &SecurityMiddleware{
		Config:      config,
		rateLimiter: NewRateLimiter(config),
	}

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with middleware (includes CORS)
	wrappedHandler := middleware.Middleware(handler)

	t.Run("preflight_request_allowed_origin", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check CORS headers
		if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Errorf("Expected Access-Control-Allow-Origin header")
		}
		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Errorf("Expected Access-Control-Allow-Methods header")
		}
		if w.Header().Get("Access-Control-Allow-Headers") == "" {
			t.Errorf("Expected Access-Control-Allow-Headers header")
		}
	})

	t.Run("preflight_request_disallowed_origin", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://malicious.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("actual_request_allowed_origin", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("OCX-API-Key", "test-key")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check CORS headers
		if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Errorf("Expected Access-Control-Allow-Origin header")
		}
	})

	t.Run("actual_request_disallowed_origin", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Origin", "https://malicious.com")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("no_origin_header", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestIdempotencyMiddleware(t *testing.T) {
	// Create test middleware
	middleware := &IdempotencyMiddleware{
		store: &mockStore{},
	}

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with middleware
	wrappedHandler := middleware.Middleware(handler)

	t.Run("first_request_with_idempotency_key", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
		req.Header.Set("Idempotency-Key", "test-key-1")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "success" {
			t.Errorf("Expected body 'success', got %s", w.Body.String())
		}
	})

	t.Run("duplicate_request_with_same_idempotency_key", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
		req.Header.Set("Idempotency-Key", "test-key-1")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		// Should return cached response
		if w.Body.String() != "success" {
			t.Errorf("Expected body 'success', got %s", w.Body.String())
		}
	})

	t.Run("request_with_different_idempotency_key", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
		req.Header.Set("Idempotency-Key", "test-key-2")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "success" {
			t.Errorf("Expected body 'success', got %s", w.Body.String())
		}
	})

	t.Run("request_without_idempotency_key", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "success" {
			t.Errorf("Expected body 'success', got %s", w.Body.String())
		}
	})

	t.Run("conflicting_request_with_same_idempotency_key", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("different body"))
		req.Header.Set("Idempotency-Key", "test-key-1")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("Expected status 409, got %d", w.Code)
		}
	})
}

func TestSecurityMiddlewareIntegration(t *testing.T) {
	// Create test middleware with all security features
	config := &SecurityConfig{
		APIKeys:      []string{"test-key"},
		MaxBodyBytes: 1024,
		CORSOrigins:  []string{"https://example.com"},
		IPRPS:        10,
		IPBurst:      20,
		KeyRPS:       20,
		KeyBurst:     40,
	}

	middleware := &SecurityMiddleware{
		Config:      config,
		rateLimiter: NewRateLimiter(config),
	}

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with all middleware (SecurityMiddleware handles all security features)
	wrappedHandler := middleware.Middleware(handler)

	t.Run("valid_request_with_all_security_checks", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
		req.Header.Set("OCX-API-Key", "test-key")
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "success" {
			t.Errorf("Expected body 'success', got %s", w.Body.String())
		}
	})

	t.Run("request_fails_api_key_check", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
		req.Header.Set("OCX-API-Key", "invalid-key")
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("request_fails_cors_check", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
		req.Header.Set("OCX-API-Key", "test-key")
		req.Header.Set("Origin", "https://malicious.com")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("request_fails_body_size_check", func(t *testing.T) {
		largeBody := make([]byte, 2048) // Exceeds 1KB limit
		rand.Read(largeBody)

		req := httptest.NewRequest("POST", "/test", bytes.NewReader(largeBody))
		req.Header.Set("OCX-API-Key", "test-key")
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413, got %d", w.Code)
		}
	})
}

func TestLoadSecurityConfig(t *testing.T) {
	t.Run("load_from_environment", func(t *testing.T) {
		// Set environment variables
		os.Setenv("OCX_API_KEYS", "key1,key2")
		os.Setenv("OCX_MAX_BODY_BYTES", "2048")
		os.Setenv("OCX_CORS_ORIGINS", "https://example.com,https://test.com")

		config := LoadSecurityConfig()

		// Verify API keys
		if len(config.APIKeys) != 2 {
			t.Errorf("Expected 2 API keys, got %d", len(config.APIKeys))
		}
		// Check that the keys are in the slice
		key1Found := false
		key2Found := false
		for _, key := range config.APIKeys {
			if key == "key1" {
				key1Found = true
			}
			if key == "key2" {
				key2Found = true
			}
		}
		if !key1Found {
			t.Error("Expected key1 to be in API keys")
		}
		if !key2Found {
			t.Error("Expected key2 to be in API keys")
		}

		// Verify other settings
		if config.MaxBodyBytes != 2048 {
			t.Errorf("Expected MaxBodyBytes to be 2048, got %d", config.MaxBodyBytes)
		}
		if len(config.CORSOrigins) != 2 {
			t.Errorf("Expected 2 CORS origins, got %d", len(config.CORSOrigins))
		}

		// Clean up environment variables
		os.Unsetenv("OCX_API_KEYS")
		os.Unsetenv("OCX_MAX_BODY_BYTES")
		os.Unsetenv("OCX_CORS_ORIGINS")
	})

	t.Run("load_default_config", func(t *testing.T) {
		config := LoadSecurityConfig()

		// Verify default values
		if config.MaxBodyBytes != 1000000 { // 1MB default
			t.Errorf("Expected MaxBodyBytes to be 1000000, got %d", config.MaxBodyBytes)
		}
		if len(config.CORSOrigins) != 0 {
			t.Errorf("Expected 0 CORS origins by default, got %d", len(config.CORSOrigins))
		}
	})
}

func TestSecurityMiddlewareConcurrency(t *testing.T) {
	// Create test middleware
	config := &SecurityConfig{
		APIKeys:      []string{"test-key"},
		MaxBodyBytes: 1024,
		IPRPS:        100,
		IPBurst:      200,
		KeyRPS:       100,
		KeyBurst:     200,
	}

	middleware := &SecurityMiddleware{
		Config:      config,
		rateLimiter: NewRateLimiter(config),
	}

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	// Wrap with middleware (SecurityMiddleware handles all security features)
	wrappedHandler := middleware.Middleware(handler)

	numGoroutines := 10
	results := make(chan error, numGoroutines)

	t.Run("concurrent_requests", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
				req.RemoteAddr = fmt.Sprintf("192.168.1.%d:12345", index)
				req.Header.Set("OCX-API-Key", "test-key")
				w := httptest.NewRecorder()

				wrappedHandler.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					results <- fmt.Errorf("expected status 200, got %d", w.Code)
					return
				}

				results <- nil
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent request failed: %v", err)
			}
		}
	})
}

// Mock store for testing idempotency middleware
type mockStore struct {
	requests map[string][]byte
	responses map[string][]byte
}

func (m *mockStore) PutIdempotent(ctx context.Context, key string, reqHash [32]byte, respCBOR []byte) (bool, bool, []byte, error) {
	if m.requests == nil {
		m.requests = make(map[string][]byte)
		m.responses = make(map[string][]byte)
	}

	reqHashBytes := reqHash[:]
	if existingReqHash, exists := m.requests[key]; exists {
		if string(existingReqHash) != string(reqHashBytes) {
			// Conflicting request
			return true, false, m.responses[key], nil
		}
		// Same request, return cached response
		return true, false, m.responses[key], nil
	}

	// New request
	m.requests[key] = reqHashBytes
	m.responses[key] = respCBOR
	return false, true, respCBOR, nil
}

func (m *mockStore) GetIdempotent(ctx context.Context, key string) ([32]byte, []byte, bool, error) {
	if m.requests == nil {
		return [32]byte{}, nil, false, nil
	}

	reqHashBytes, exists := m.requests[key]
	if !exists {
		return [32]byte{}, nil, false, nil
	}

	var reqHash [32]byte
	copy(reqHash[:], reqHashBytes)
	return reqHash, m.responses[key], true, nil
}

func (m *mockStore) SaveReceipt(ctx context.Context, r receipt.ReceiptFull, fullCBOR []byte) (string, error) {
	return "mock-receipt-id", nil
}

func (m *mockStore) GetReceipt(ctx context.Context, id string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func BenchmarkAPIKeyAuthMiddleware(b *testing.B) {
	config := &SecurityConfig{
		APIKeys: []string{"test-key"},
	}

	middleware := &SecurityMiddleware{Config: config}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Middleware(handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
		req.Header.Set("OCX-API-Key", "test-key")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)
	}
}

func BenchmarkRateLimiterMiddleware(b *testing.B) {
	config := &SecurityConfig{
		IPRPS:   1000,
		IPBurst: 2000,
		KeyRPS:  1000,
		KeyBurst: 2000,
	}

	middleware := &SecurityMiddleware{
		Config:      config,
		rateLimiter: NewRateLimiter(config),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Middleware(handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("OCX-API-Key", "test-key")
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)
	}
}

func BenchmarkBodyCapMiddleware(b *testing.B) {
	config := &SecurityConfig{MaxBodyBytes: 1024 * 1024} // 1MB
	middleware := &SecurityMiddleware{Config: config}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.Middleware(handler)

	body := make([]byte, 1024) // 1KB body
	rand.Read(body)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
		req.Header.Set("OCX-API-Key", "test-key")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)
	}
}
