package security

import (
    "crypto/subtle"
    "time"
)

// ConstantTimeCompare prevents timing attacks
func ConstantTimeCompare(a, b []byte) bool {
    return subtle.ConstantTimeCompare(a, b) == 1
}

// RateLimiter implements basic rate limiting
type RateLimiter struct {
    requests map[string][]time.Time
    limit    int
    window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

func (rl *RateLimiter) Allow(key string) bool {
    now := time.Now()
    times := rl.requests[key]
    
    // Remove old requests outside the window
    var validTimes []time.Time
    for _, t := range times {
        if now.Sub(t) <= rl.window {
            validTimes = append(validTimes, t)
        }
    }
    
    if len(validTimes) >= rl.limit {
        return false
    }
    
    validTimes = append(validTimes, now)
    rl.requests[key] = validTimes
    return true
}
