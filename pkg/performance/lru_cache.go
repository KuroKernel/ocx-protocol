package performance

import (
	"sync"
	"time"
)

// LRUCache provides a simple LRU cache implementation
type LRUCache struct {
	// Configuration
	maxSize int
	ttl     time.Duration

	// Storage
	items map[string]*CacheItem
	order []string
	mutex sync.RWMutex

	// Statistics
	stats      CacheStats
	statsMutex sync.RWMutex
}

// CacheItem represents a cached item
type CacheItem struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	AccessCount int64     `json:"access_count"`
	LastAccess  time.Time `json:"last_access"`
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits      int64 `json:"hits"`
	Misses    int64 `json:"misses"`
	Evictions int64 `json:"evictions"`
	Size      int   `json:"size"`
	MaxSize   int   `json:"max_size"`
}

// NewLRUCache creates a new LRU cache
func NewLRUCache(maxSize int) *LRUCache {
	return &LRUCache{
		maxSize: maxSize,
		ttl:     time.Hour * 24, // Default 24 hour TTL
		items:   make(map[string]*CacheItem),
		order:   make([]string, 0),
		stats: CacheStats{
			MaxSize: maxSize,
		},
	}
}

// Set stores a value in the cache
func (c *LRUCache) Set(key, value string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()

	// Check if key already exists
	if item, exists := c.items[key]; exists {
		// Update existing item
		item.Value = value
		item.LastAccess = now
		item.AccessCount++
		c.moveToFront(key)
		return
	}

	// Create new item
	item := &CacheItem{
		Key:         key,
		Value:       value,
		CreatedAt:   now,
		ExpiresAt:   now.Add(c.ttl),
		AccessCount: 1,
		LastAccess:  now,
	}

	// Add to cache
	c.items[key] = item
	c.order = append([]string{key}, c.order...)

	// Update stats
	c.statsMutex.Lock()
	c.stats.Size = len(c.items)
	c.statsMutex.Unlock()

	// Evict if necessary
	if len(c.items) > c.maxSize {
		c.evictLRU()
	}
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) (string, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	item, exists := c.items[key]
	if !exists {
		c.statsMutex.Lock()
		c.stats.Misses++
		c.statsMutex.Unlock()
		return "", false
	}

	// Check if expired
	if time.Now().After(item.ExpiresAt) {
		delete(c.items, key)
		c.removeFromOrder(key)
		c.statsMutex.Lock()
		c.stats.Misses++
		c.statsMutex.Unlock()
		return "", false
	}

	// Update access info
	item.LastAccess = time.Now()
	item.AccessCount++
	c.moveToFront(key)

	c.statsMutex.Lock()
	c.stats.Hits++
	c.statsMutex.Unlock()

	return item.Value, true
}

// Delete removes a value from the cache
func (c *LRUCache) Delete(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	_, exists := c.items[key]
	if !exists {
		return false
	}

	delete(c.items, key)
	c.removeFromOrder(key)

	c.statsMutex.Lock()
	c.stats.Size = len(c.items)
	c.statsMutex.Unlock()

	return true
}

// Clear removes all items from the cache
func (c *LRUCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[string]*CacheItem)
	c.order = make([]string, 0)

	c.statsMutex.Lock()
	c.stats.Size = 0
	c.statsMutex.Unlock()
}

// GetSize returns the current cache size
func (c *LRUCache) GetSize() int {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()
	return c.stats.Size
}

// GetHitRate returns the cache hit rate
func (c *LRUCache) GetHitRate() float64 {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()

	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0.0
	}

	return float64(c.stats.Hits) / float64(total) * 100.0
}

// GetEvictions returns the number of evictions
func (c *LRUCache) GetEvictions() int64 {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()
	return c.stats.Evictions
}

// GetStats returns cache statistics
func (c *LRUCache) GetStats() CacheStats {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()

	stats := c.stats
	stats.Size = len(c.items)
	return stats
}

// moveToFront moves an item to the front of the order
func (c *LRUCache) moveToFront(key string) {
	c.removeFromOrder(key)
	c.order = append([]string{key}, c.order...)
}

// removeFromOrder removes an item from the order
func (c *LRUCache) removeFromOrder(key string) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

// evictLRU removes the least recently used item
func (c *LRUCache) evictLRU() {
	if len(c.order) == 0 {
		return
	}

	// Remove the last item (least recently used)
	key := c.order[len(c.order)-1]
	delete(c.items, key)
	c.order = c.order[:len(c.order)-1]

	c.statsMutex.Lock()
	c.stats.Evictions++
	c.stats.Size = len(c.items)
	c.statsMutex.Unlock()
}

// Cleanup removes expired items
func (c *LRUCache) Cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(c.items, key)
		c.removeFromOrder(key)
	}

	c.statsMutex.Lock()
	c.stats.Size = len(c.items)
	c.statsMutex.Unlock()
}
