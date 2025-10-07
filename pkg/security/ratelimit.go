package security

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	clients map[string]*clientLimit
	mu      sync.RWMutex

	// Configuration
	requestsPerSecond int
	burst             int
	cleanupInterval   time.Duration
}

// clientLimit tracks rate limit state for a single client
type clientLimit struct {
	tokens         float64
	lastRefill     time.Time
	requestCount   int
	firstRequest   time.Time
	mu             sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond, burst int) *RateLimiter {
	rl := &RateLimiter{
		clients:           make(map[string]*clientLimit),
		requestsPerSecond: requestsPerSecond,
		burst:             burst,
		cleanupInterval:   time.Minute * 5,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mu.Lock()
	client, exists := rl.clients[clientID]
	if !exists {
		client = &clientLimit{
			tokens:       float64(rl.burst),
			lastRefill:   time.Now(),
			requestCount: 0,
			firstRequest: time.Now(),
		}
		rl.clients[clientID] = client
	}
	rl.mu.Unlock()

	client.mu.Lock()
	defer client.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(client.lastRefill)
	client.tokens += elapsed.Seconds() * float64(rl.requestsPerSecond)

	// Cap at burst limit
	if client.tokens > float64(rl.burst) {
		client.tokens = float64(rl.burst)
	}

	client.lastRefill = now

	// Check if we have tokens
	if client.tokens >= 1.0 {
		client.tokens -= 1.0
		client.requestCount++
		return true
	}

	return false
}

// cleanup removes stale client entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for clientID, client := range rl.clients {
			client.mu.Lock()
			if now.Sub(client.lastRefill) > time.Hour {
				delete(rl.clients, clientID)
			}
			client.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// Middleware returns an HTTP middleware that enforces rate limiting
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use IP address as client ID
		clientID := r.RemoteAddr

		// Try to get API key for better identification
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			clientID = "api:" + apiKey
		}

		if !rl.Allow(clientID) {
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetStats returns rate limiting statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"total_clients":       len(rl.clients),
		"requests_per_second": rl.requestsPerSecond,
		"burst":               rl.burst,
	}
}
