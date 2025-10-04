package cache

import (
	"time"
)

// Cache defines the interface for caching platform reputation data
// Implementations must be safe for concurrent use
type Cache interface {
	// Get retrieves a value from the cache
	// Returns (value, true) if found, (nil, false) if not found
	Get(key string) (any, bool)

	// Set stores a value in the cache with a TTL
	Set(key string, value any, ttl time.Duration)

	// Delete removes a value from the cache
	Delete(key string)

	// Clear removes all values from the cache
	Clear()

	// Stats returns cache statistics
	Stats() CacheStats
}

// CacheStats contains statistics about cache performance
type CacheStats struct {
	Hits        int64   `json:"hits"`         // Number of cache hits
	Misses      int64   `json:"misses"`       // Number of cache misses
	Evictions   int64   `json:"evictions"`    // Number of evicted entries
	Size        int     `json:"size"`         // Current number of entries
	HitRate     float64 `json:"hit_rate"`     // Hit rate (0.0-1.0)
	MemoryBytes int64   `json:"memory_bytes"` // Approximate memory usage
}
