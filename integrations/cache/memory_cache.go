package cache

import (
	"sync"
	"time"
)

// cacheEntry represents a single cached value with expiration
type cacheEntry struct {
	Value      any
	Expiration time.Time
}

// MemoryCache is an in-memory cache implementation with TTL support
type MemoryCache struct {
	entries map[string]*cacheEntry
	stats   CacheStats
	mu      sync.RWMutex

	// Cleanup goroutine control
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// NewMemoryCache creates a new in-memory cache with periodic cleanup
func NewMemoryCache(cleanupInterval time.Duration) *MemoryCache {
	cache := &MemoryCache{
		entries:         make(map[string]*cacheEntry),
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// Get retrieves a value from the cache
func (c *MemoryCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		c.stats.Misses++
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.Expiration) {
		c.stats.Misses++
		return nil, false
	}

	c.stats.Hits++
	return entry.Value, true
}

// Set stores a value in the cache with a TTL
func (c *MemoryCache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}
}

// Delete removes a value from the cache
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// Clear removes all values from the cache
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
	c.stats.Evictions += int64(len(c.entries))
}

// Stats returns cache statistics
func (c *MemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.Size = len(c.entries)

	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}

	return stats
}

// Stop stops the cleanup goroutine
func (c *MemoryCache) Stop() {
	close(c.stopCleanup)
}

// cleanupLoop periodically removes expired entries
func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanup removes expired entries (called by cleanupLoop)
func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	evicted := 0

	for key, entry := range c.entries {
		if now.After(entry.Expiration) {
			delete(c.entries, key)
			evicted++
		}
	}

	if evicted > 0 {
		c.stats.Evictions += int64(evicted)
	}
}
