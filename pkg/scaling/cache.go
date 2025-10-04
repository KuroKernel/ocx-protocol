package scaling

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// CacheConfig defines configuration for distributed caching
type CacheConfig struct {
	// Cache settings
	DefaultTTL      time.Duration `json:"default_ttl"`
	MaxSize         int64         `json:"max_size"`
	CleanupInterval time.Duration `json:"cleanup_interval"`

	// Distributed cache settings
	DistributedEnabled bool     `json:"distributed_enabled"`
	ClusterNodes       []string `json:"cluster_nodes"`
	ReplicationFactor  int      `json:"replication_factor"`

	// Eviction policy
	EvictionPolicy string `json:"eviction_policy"` // "lru", "lfu", "ttl", "random"

	// Persistence
	PersistenceEnabled bool   `json:"persistence_enabled"`
	PersistencePath    string `json:"persistence_path"`

	// Compression
	CompressionEnabled bool `json:"compression_enabled"`

	// Security
	EncryptionEnabled bool   `json:"encryption_enabled"`
	EncryptionKey     string `json:"encryption_key"`
}

// CacheEntry represents a cache entry
type CacheEntry struct {
	Key         string        `json:"key"`
	Value       interface{}   `json:"value"`
	TTL         time.Duration `json:"ttl"`
	CreatedAt   time.Time     `json:"created_at"`
	AccessedAt  time.Time     `json:"accessed_at"`
	AccessCount int64         `json:"access_count"`
	Size        int64         `json:"size"`
	Version     int64         `json:"version"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalEntries  int64         `json:"total_entries"`
	TotalSize     int64         `json:"total_size"`
	HitCount      int64         `json:"hit_count"`
	MissCount     int64         `json:"miss_count"`
	EvictionCount int64         `json:"eviction_count"`
	AverageTTL    time.Duration `json:"average_ttl"`
	HitRate       float64       `json:"hit_rate"`
	LastCleanup   time.Time     `json:"last_cleanup"`
}

// DistributedCache manages distributed caching across cluster nodes
type DistributedCache struct {
	config     CacheConfig
	localCache map[string]*CacheEntry
	cacheMu    sync.RWMutex
	stats      CacheStats
	statsMu    sync.RWMutex

	// Cluster communication
	clusterManager *ClusterManager

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Event handlers
	onCacheMiss   func(string)
	onCacheEvict  func(string, *CacheEntry)
	onCacheUpdate func(string, interface{})
}

// CacheOperation represents a cache operation
type CacheOperation struct {
	Type      string        `json:"type"` // "get", "set", "delete", "clear"
	Key       string        `json:"key"`
	Value     interface{}   `json:"value,omitempty"`
	TTL       time.Duration `json:"ttl,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	NodeID    string        `json:"node_id"`
}

// NewDistributedCache creates a new distributed cache
func NewDistributedCache(config CacheConfig, clusterManager *ClusterManager) (*DistributedCache, error) {
	ctx, cancel := context.WithCancel(context.Background())

	dc := &DistributedCache{
		config:         config,
		localCache:     make(map[string]*CacheEntry),
		clusterManager: clusterManager,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Start background tasks
	dc.startCleanup()
	dc.startReplication()

	return dc, nil
}

// Get retrieves a value from the cache
func (dc *DistributedCache) Get(key string) (interface{}, bool) {
	dc.cacheMu.RLock()
	entry, exists := dc.localCache[key]
	dc.cacheMu.RUnlock()

	if !exists {
		dc.recordMiss()
		if dc.onCacheMiss != nil {
			dc.onCacheMiss(key)
		}
		return nil, false
	}

	// Check if entry has expired
	if dc.isExpired(entry) {
		dc.Delete(key)
		dc.recordMiss()
		if dc.onCacheMiss != nil {
			dc.onCacheMiss(key)
		}
		return nil, false
	}

	// Update access statistics
	dc.updateAccessStats(entry)
	dc.recordHit()

	return entry.Value, true
}

// Set stores a value in the cache
func (dc *DistributedCache) Set(key string, value interface{}, ttl time.Duration) error {
	if ttl == 0 {
		ttl = dc.config.DefaultTTL
	}

	// Calculate entry size
	size := dc.calculateSize(key, value)

	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		TTL:         ttl,
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 1,
		Size:        size,
		Version:     time.Now().UnixNano(),
	}

	dc.cacheMu.Lock()
	dc.localCache[key] = entry
	dc.cacheMu.Unlock()

	// Update statistics
	dc.updateStats()

	// Replicate to other nodes if distributed
	if dc.config.DistributedEnabled {
		dc.replicateOperation(CacheOperation{
			Type:      "set",
			Key:       key,
			Value:     value,
			TTL:       ttl,
			Timestamp: time.Now(),
			NodeID:    dc.clusterManager.config.NodeID,
		})
	}

	// Trigger event handler
	if dc.onCacheUpdate != nil {
		dc.onCacheUpdate(key, value)
	}

	return nil
}

// Delete removes a value from the cache
func (dc *DistributedCache) Delete(key string) bool {
	dc.cacheMu.Lock()
	entry, exists := dc.localCache[key]
	if exists {
		delete(dc.localCache, key)
	}
	dc.cacheMu.Unlock()

	if exists {
		// Update statistics
		dc.updateStats()

		// Replicate to other nodes if distributed
		if dc.config.DistributedEnabled {
			dc.replicateOperation(CacheOperation{
				Type:      "delete",
				Key:       key,
				Timestamp: time.Now(),
				NodeID:    dc.clusterManager.config.NodeID,
			})
		}

		// Trigger event handler
		if dc.onCacheEvict != nil {
			dc.onCacheEvict(key, entry)
		}

		return true
	}

	return false
}

// Clear removes all entries from the cache
func (dc *DistributedCache) Clear() {
	dc.cacheMu.Lock()
	dc.localCache = make(map[string]*CacheEntry)
	dc.cacheMu.Unlock()

	// Update statistics
	dc.updateStats()

	// Replicate to other nodes if distributed
	if dc.config.DistributedEnabled {
		dc.replicateOperation(CacheOperation{
			Type:      "clear",
			Timestamp: time.Now(),
			NodeID:    dc.clusterManager.config.NodeID,
		})
	}
}

// GetStats returns cache statistics
func (dc *DistributedCache) GetStats() CacheStats {
	dc.statsMu.RLock()
	defer dc.statsMu.RUnlock()
	return dc.stats
}

// GetKeys returns all cache keys
func (dc *DistributedCache) GetKeys() []string {
	dc.cacheMu.RLock()
	defer dc.cacheMu.RUnlock()

	keys := make([]string, 0, len(dc.localCache))
	for key := range dc.localCache {
		keys = append(keys, key)
	}
	return keys
}

// GetSize returns the total cache size
func (dc *DistributedCache) GetSize() int64 {
	dc.statsMu.RLock()
	defer dc.statsMu.RUnlock()
	return dc.stats.TotalSize
}

// SetEventHandlers sets event handlers for cache events
func (dc *DistributedCache) SetEventHandlers(
	onCacheMiss func(string),
	onCacheEvict func(string, *CacheEntry),
	onCacheUpdate func(string, interface{}),
) {
	dc.onCacheMiss = onCacheMiss
	dc.onCacheEvict = onCacheEvict
	dc.onCacheUpdate = onCacheUpdate
}

// Close shuts down the distributed cache
func (dc *DistributedCache) Close() {
	dc.cancel()
	dc.wg.Wait()
}

// Helper methods

func (dc *DistributedCache) isExpired(entry *CacheEntry) bool {
	return time.Since(entry.CreatedAt) > entry.TTL
}

func (dc *DistributedCache) calculateSize(key string, value interface{}) int64 {
	// Simple size calculation - in a real implementation, this would be more accurate
	data, _ := json.Marshal(value)
	return int64(len(key) + len(data))
}

func (dc *DistributedCache) updateAccessStats(entry *CacheEntry) {
	dc.cacheMu.Lock()
	entry.AccessedAt = time.Now()
	entry.AccessCount++
	dc.cacheMu.Unlock()
}

func (dc *DistributedCache) recordHit() {
	dc.statsMu.Lock()
	dc.stats.HitCount++
	dc.statsMu.Unlock()
}

func (dc *DistributedCache) recordMiss() {
	dc.statsMu.Lock()
	dc.stats.MissCount++
	dc.statsMu.Unlock()
}

func (dc *DistributedCache) updateStats() {
	dc.cacheMu.RLock()
	totalEntries := int64(len(dc.localCache))
	totalSize := int64(0)
	totalTTL := time.Duration(0)

	for _, entry := range dc.localCache {
		totalSize += entry.Size
		totalTTL += entry.TTL
	}

	dc.cacheMu.RUnlock()

	dc.statsMu.Lock()
	dc.stats.TotalEntries = totalEntries
	dc.stats.TotalSize = totalSize
	if totalEntries > 0 {
		dc.stats.AverageTTL = totalTTL / time.Duration(totalEntries)
	}

	// Calculate hit rate
	totalRequests := dc.stats.HitCount + dc.stats.MissCount
	if totalRequests > 0 {
		dc.stats.HitRate = float64(dc.stats.HitCount) / float64(totalRequests)
	}
	dc.statsMu.Unlock()
}

func (dc *DistributedCache) startCleanup() {
	dc.wg.Add(1)
	go func() {
		defer dc.wg.Done()
		ticker := time.NewTicker(dc.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-dc.ctx.Done():
				return
			case <-ticker.C:
				dc.cleanup()
			}
		}
	}()
}

func (dc *DistributedCache) cleanup() {
	dc.cacheMu.Lock()
	defer dc.cacheMu.Unlock()

	now := time.Now()
	evicted := 0

	// Remove expired entries
	for key, entry := range dc.localCache {
		if now.Sub(entry.CreatedAt) > entry.TTL {
			delete(dc.localCache, key)
			evicted++
		}
	}

	// Check if we need to evict based on size
	if dc.config.MaxSize > 0 && dc.stats.TotalSize > dc.config.MaxSize {
		evicted += dc.evictEntries()
	}

	// Update statistics
	dc.statsMu.Lock()
	dc.stats.EvictionCount += int64(evicted)
	dc.stats.LastCleanup = now
	dc.statsMu.Unlock()

	if evicted > 0 {
		log.Printf("Cache cleanup: evicted %d entries", evicted)
	}
}

func (dc *DistributedCache) evictEntries() int {
	switch dc.config.EvictionPolicy {
	case "lru":
		return dc.evictLRU()
	case "lfu":
		return dc.evictLFU()
	case "ttl":
		return dc.evictTTL()
	case "random":
		return dc.evictRandom()
	default:
		return dc.evictLRU()
	}
}

func (dc *DistributedCache) evictLRU() int {
	// Find least recently used entry
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range dc.localCache {
		if oldestKey == "" || entry.AccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.AccessedAt
		}
	}

	if oldestKey != "" {
		delete(dc.localCache, oldestKey)
		return 1
	}
	return 0
}

func (dc *DistributedCache) evictLFU() int {
	// Find least frequently used entry
	var lfuKey string
	var minAccessCount int64 = -1

	for key, entry := range dc.localCache {
		if minAccessCount == -1 || entry.AccessCount < minAccessCount {
			lfuKey = key
			minAccessCount = entry.AccessCount
		}
	}

	if lfuKey != "" {
		delete(dc.localCache, lfuKey)
		return 1
	}
	return 0
}

func (dc *DistributedCache) evictTTL() int {
	// Find entry with shortest TTL remaining
	var shortestKey string
	var shortestTTL time.Duration = -1

	now := time.Now()
	for key, entry := range dc.localCache {
		remainingTTL := entry.TTL - now.Sub(entry.CreatedAt)
		if shortestTTL == -1 || remainingTTL < shortestTTL {
			shortestKey = key
			shortestTTL = remainingTTL
		}
	}

	if shortestKey != "" {
		delete(dc.localCache, shortestKey)
		return 1
	}
	return 0
}

func (dc *DistributedCache) evictRandom() int {
	// Randomly select an entry to evict
	for key := range dc.localCache {
		delete(dc.localCache, key)
		return 1
	}
	return 0
}

func (dc *DistributedCache) startReplication() {
	if !dc.config.DistributedEnabled {
		return
	}

	dc.wg.Add(1)
	go func() {
		defer dc.wg.Done()
		// Implementation: handle replication
		// we'll just log that replication is running
		log.Printf("Cache replication started")
	}()
}

func (dc *DistributedCache) replicateOperation(operation CacheOperation) {
	// Implementation: send the operation to other nodes
	// we'll just log the operation
	log.Printf("Replicating cache operation: %+v", operation)
}

// CacheShard represents a cache shard for distributed caching
type CacheShard struct {
	id    int
	cache *DistributedCache
}

// ShardedCache manages multiple cache shards
type ShardedCache struct {
	shards []*CacheShard
	config CacheConfig
}

// NewShardedCache creates a new sharded cache
func NewShardedCache(config CacheConfig, numShards int, clusterManager *ClusterManager) (*ShardedCache, error) {
	shards := make([]*CacheShard, numShards)

	for i := 0; i < numShards; i++ {
		cache, err := NewDistributedCache(config, clusterManager)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache shard %d: %v", i, err)
		}

		shards[i] = &CacheShard{
			id:    i,
			cache: cache,
		}
	}

	return &ShardedCache{
		shards: shards,
		config: config,
	}, nil
}

// Get retrieves a value from the appropriate shard
func (sc *ShardedCache) Get(key string) (interface{}, bool) {
	shard := sc.getShard(key)
	return shard.cache.Get(key)
}

// Set stores a value in the appropriate shard
func (sc *ShardedCache) Set(key string, value interface{}, ttl time.Duration) error {
	shard := sc.getShard(key)
	return shard.cache.Set(key, value, ttl)
}

// Delete removes a value from the appropriate shard
func (sc *ShardedCache) Delete(key string) bool {
	shard := sc.getShard(key)
	return shard.cache.Delete(key)
}

// GetStats returns combined statistics from all shards
func (sc *ShardedCache) GetStats() CacheStats {
	var combinedStats CacheStats

	for _, shard := range sc.shards {
		stats := shard.cache.GetStats()
		combinedStats.TotalEntries += stats.TotalEntries
		combinedStats.TotalSize += stats.TotalSize
		combinedStats.HitCount += stats.HitCount
		combinedStats.MissCount += stats.MissCount
		combinedStats.EvictionCount += stats.EvictionCount
	}

	// Calculate combined hit rate
	totalRequests := combinedStats.HitCount + combinedStats.MissCount
	if totalRequests > 0 {
		combinedStats.HitRate = float64(combinedStats.HitCount) / float64(totalRequests)
	}

	return combinedStats
}

// Close shuts down all shards
func (sc *ShardedCache) Close() {
	for _, shard := range sc.shards {
		shard.cache.Close()
	}
}

func (sc *ShardedCache) getShard(key string) *CacheShard {
	// Use consistent hashing to determine shard
	hash := sha256.Sum256([]byte(key))
	shardIndex := int(hash[0]) % len(sc.shards)
	return sc.shards[shardIndex]
}
