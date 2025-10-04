package integrations

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter provides per-platform rate limiting to avoid API quota exhaustion
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

// NewRateLimiter creates a new rate limiter with per-platform limits
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: map[string]*rate.Limiter{
			// GitHub: 5000 requests/hour = ~1.39 req/sec
			"github": rate.NewLimiter(rate.Every(720*time.Millisecond), 5),

			// LinkedIn: 500 requests/hour = ~0.14 req/sec
			"linkedin": rate.NewLimiter(rate.Every(7200*time.Millisecond), 2),

			// Uber: 1000 requests/hour = ~0.28 req/sec
			"uber": rate.NewLimiter(rate.Every(3600*time.Millisecond), 3),
		},
	}
}

// Wait blocks until the rate limiter allows the request to proceed
// Returns error if context is cancelled or platform is unknown
func (r *RateLimiter) Wait(ctx context.Context, platform string) error {
	r.mu.RLock()
	limiter, ok := r.limiters[platform]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("unknown platform: %s", platform)
	}

	return limiter.Wait(ctx)
}

// Allow checks if a request can proceed without blocking
// Returns true if the request is allowed, false otherwise
func (r *RateLimiter) Allow(platform string) bool {
	r.mu.RLock()
	limiter, ok := r.limiters[platform]
	r.mu.RUnlock()

	if !ok {
		return false
	}

	return limiter.Allow()
}

// Reserve reserves a token and returns a Reservation that can be used to wait
func (r *RateLimiter) Reserve(platform string) (*rate.Reservation, error) {
	r.mu.RLock()
	limiter, ok := r.limiters[platform]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown platform: %s", platform)
	}

	return limiter.Reserve(), nil
}

// SetLimit updates the rate limit for a specific platform
func (r *RateLimiter) SetLimit(platform string, limit rate.Limit, burst int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.limiters[platform]; !ok {
		return fmt.Errorf("unknown platform: %s", platform)
	}

	r.limiters[platform] = rate.NewLimiter(limit, burst)
	return nil
}

// GetStats returns current rate limit statistics for all platforms
func (r *RateLimiter) GetStats() map[string]RateLimitStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]RateLimitStats)
	for platform, limiter := range r.limiters {
		stats[platform] = RateLimitStats{
			Platform: platform,
			Limit:    limiter.Limit(),
			Burst:    limiter.Burst(),
			Tokens:   limiter.Tokens(),
		}
	}
	return stats
}

// RateLimitStats contains statistics for a single platform's rate limiter
type RateLimitStats struct {
	Platform string      `json:"platform"`
	Limit    rate.Limit  `json:"limit"`     // Requests per second
	Burst    int         `json:"burst"`     // Maximum burst size
	Tokens   float64     `json:"tokens"`    // Currently available tokens
}
