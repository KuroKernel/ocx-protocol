package performance

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// DiskCacheConfig defines configuration for the disk cache
type DiskCacheConfig struct {
	Dir      string        `json:"dir"`
	MaxSize  int64         `json:"max_size"`
	TTL      time.Duration `json:"ttl"`
	Cleanup  time.Duration `json:"cleanup"`
	Compress bool          `json:"compress"`
	Encrypt  bool          `json:"encrypt"`
	Key      []byte        `json:"key"`
}

// DiskCache provides persistent disk-based caching
type DiskCache struct {
	config     DiskCacheConfig
	index      map[string]*CacheIndex
	indexMutex sync.RWMutex
	stats      *DiskCacheStats
	ctx        context.Context
	cancel     context.CancelFunc
}

// CacheIndex tracks metadata for cached items
type CacheIndex struct {
	Key         string    `json:"key"`
	FilePath    string    `json:"file_path"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	AccessCount int64     `json:"access_count"`
	LastAccess  time.Time `json:"last_access"`
	Compressed  bool      `json:"compressed"`
	Encrypted   bool      `json:"encrypted"`
}

// DiskCacheStats tracks disk cache performance
type DiskCacheStats struct {
	Hits           int64     `json:"hits"`
	Misses         int64     `json:"misses"`
	Writes         int64     `json:"writes"`
	Deletes        int64     `json:"deletes"`
	Compressions   int64     `json:"compressions"`
	Decompressions int64     `json:"decompressions"`
	Encryptions    int64     `json:"encryptions"`
	Decryptions    int64     `json:"decryptions"`
	TotalSize      int64     `json:"total_size"`
	LastCleanup    time.Time `json:"last_cleanup"`
}

// NewDiskCache creates a new disk cache instance
func NewDiskCache(config DiskCacheConfig) (*DiskCache, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	dc := &DiskCache{
		config: config,
		index:  make(map[string]*CacheIndex),
		stats:  &DiskCacheStats{},
		ctx:    ctx,
		cancel: cancel,
	}

	// Load existing index
	if err := dc.loadIndex(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load cache index: %w", err)
	}

	// Start cleanup goroutine
	go dc.cleanupWorker()

	return dc, nil
}

// Get retrieves an item from the disk cache
func (dc *DiskCache) Get(key string) (*CacheItem, bool) {
	dc.indexMutex.RLock()
	index, exists := dc.index[key]
	dc.indexMutex.RUnlock()

	if !exists {
		dc.stats.Misses++
		return nil, false
	}

	// Check if expired
	if time.Now().After(index.ExpiresAt) {
		dc.Delete(key)
		dc.stats.Misses++
		return nil, false
	}

	// Read file
	data, err := dc.readFile(index.FilePath, index.Compressed, index.Encrypted)
	if err != nil {
		dc.stats.Misses++
		return nil, false
	}

	// Update access statistics
	dc.indexMutex.Lock()
	index.AccessCount++
	index.LastAccess = time.Now()
	dc.indexMutex.Unlock()

	dc.stats.Hits++

	// Create cache item
	item := &CacheItem{
		Key:         key,
		Value:       string(data),
		CreatedAt:   index.CreatedAt,
		ExpiresAt:   index.ExpiresAt,
		AccessCount: index.AccessCount,
		LastAccess:  index.LastAccess,
	}

	return item, true
}

// Set stores an item in the disk cache
func (dc *DiskCache) Set(key string, item *CacheItem) error {
	// Generate file path
	filePath := dc.getFilePath(key)

	// Write file
	data := []byte(item.Value)
	compressed := false
	encrypted := false

	if dc.config.Compress {
		var err error
		data, err = dc.compress(data)
		if err != nil {
			return fmt.Errorf("failed to compress data: %w", err)
		}
		compressed = true
		dc.stats.Compressions++
	}

	if dc.config.Encrypt {
		var err error
		data, err = dc.encrypt(data)
		if err != nil {
			return fmt.Errorf("failed to encrypt data: %w", err)
		}
		encrypted = true
		dc.stats.Encryptions++
	}

	if err := dc.writeFile(filePath, data); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Update index
	dc.indexMutex.Lock()
	dc.index[key] = &CacheIndex{
		Key:         key,
		FilePath:    filePath,
		Size:        int64(len(data)),
		CreatedAt:   item.CreatedAt,
		ExpiresAt:   item.ExpiresAt,
		AccessCount: item.AccessCount,
		LastAccess:  item.LastAccess,
		Compressed:  compressed,
		Encrypted:   encrypted,
	}
	dc.indexMutex.Unlock()

	dc.stats.Writes++
	dc.stats.TotalSize += int64(len(data))

	// Save index
	return dc.saveIndex()
}

// Remove removes an item from the disk cache
func (dc *DiskCache) Remove(key string) error {
	dc.indexMutex.Lock()
	index, exists := dc.index[key]
	if exists {
		delete(dc.index, key)
		dc.stats.TotalSize -= index.Size
	}
	dc.indexMutex.Unlock()

	if !exists {
		return nil
	}

	// Remove file
	if err := os.Remove(index.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	dc.stats.Deletes++

	// Save index
	return dc.saveIndex()
}

// Clear removes all items from the disk cache
func (dc *DiskCache) Clear() error {
	dc.indexMutex.Lock()
	defer dc.indexMutex.Unlock()

	// Remove all files
	for _, index := range dc.index {
		if err := os.Remove(index.FilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove file: %w", err)
		}
	}

	// Clear index
	dc.index = make(map[string]*CacheIndex)
	dc.stats.TotalSize = 0

	// Save index
	return dc.saveIndex()
}

// Count returns the number of items in the cache
func (dc *DiskCache) Count() int {
	dc.indexMutex.RLock()
	defer dc.indexMutex.RUnlock()
	return len(dc.index)
}

// Stats returns disk cache statistics
func (dc *DiskCache) Stats() DiskCacheStats {
	dc.indexMutex.RLock()
	defer dc.indexMutex.RUnlock()
	return *dc.stats
}

// Close closes the disk cache
func (dc *DiskCache) Close() error {
	dc.cancel()
	return dc.saveIndex()
}

// getFilePath generates a file path for a cache key
func (dc *DiskCache) getFilePath(key string) string {
	// Use first 2 characters as subdirectory to avoid too many files in one directory
	subdir := key[:2]
	dir := filepath.Join(dc.config.Dir, subdir)
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, key)
}

// readFile reads a file from disk with optional decompression and decryption
func (dc *DiskCache) readFile(filePath string, compressed, encrypted bool) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if encrypted {
		data, err = dc.decrypt(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt data: %w", err)
		}
		dc.stats.Decryptions++
	}

	if compressed {
		data, err = dc.decompress(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress data: %w", err)
		}
		dc.stats.Decompressions++
	}

	return data, nil
}

// writeFile writes data to a file
func (dc *DiskCache) writeFile(filePath string, data []byte) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write file atomically
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, filePath)
}

// compress compresses data using gzip
func (dc *DiskCache) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompress decompresses data using gzip
func (dc *DiskCache) decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// encrypt encrypts data using AES-GCM
func (dc *DiskCache) encrypt(data []byte) ([]byte, error) {
	if len(dc.config.Key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes")
	}

	block, err := aes.NewCipher(dc.config.Key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-GCM
func (dc *DiskCache) decrypt(data []byte) ([]byte, error) {
	if len(dc.config.Key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes")
	}

	block, err := aes.NewCipher(dc.config.Key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// loadIndex loads the cache index from disk
func (dc *DiskCache) loadIndex() error {
	indexPath := filepath.Join(dc.config.Dir, "index.json")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No index file exists yet
		}
		return err
	}

	var index map[string]*CacheIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return err
	}

	// Validate index entries and remove invalid ones
	validIndex := make(map[string]*CacheIndex)
	for key, entry := range index {
		if _, err := os.Stat(entry.FilePath); err == nil {
			validIndex[key] = entry
			dc.stats.TotalSize += entry.Size
		}
	}

	dc.index = validIndex
	return nil
}

// saveIndex saves the cache index to disk
func (dc *DiskCache) saveIndex() error {
	dc.indexMutex.RLock()
	data, err := json.MarshalIndent(dc.index, "", "  ")
	dc.indexMutex.RUnlock()

	if err != nil {
		return err
	}

	indexPath := filepath.Join(dc.config.Dir, "index.json")
	return os.WriteFile(indexPath, data, 0644)
}

// cleanupWorker runs periodic cleanup of expired items
func (dc *DiskCache) cleanupWorker() {
	ticker := time.NewTicker(dc.config.Cleanup)
	defer ticker.Stop()

	for {
		select {
		case <-dc.ctx.Done():
			return
		case <-ticker.C:
			dc.cleanup()
		}
	}
}

// cleanup removes expired items and enforces size limits
func (dc *DiskCache) cleanup() {
	dc.indexMutex.Lock()
	defer dc.indexMutex.Unlock()

	now := time.Now()
	expiredKeys := make([]string, 0)

	// Find expired items
	for key, index := range dc.index {
		if now.After(index.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// Remove expired items
	for _, key := range expiredKeys {
		if index, exists := dc.index[key]; exists {
			os.Remove(index.FilePath)
			delete(dc.index, key)
			dc.stats.TotalSize -= index.Size
		}
	}

	// Enforce size limits
	if dc.stats.TotalSize > dc.config.MaxSize {
		dc.enforceSizeLimit()
	}

	dc.stats.LastCleanup = now
	dc.saveIndex()
}

// enforceSizeLimit removes items to stay within size limits
func (dc *DiskCache) enforceSizeLimit() {
	// Sort by last access time (LRU)
	type item struct {
		key   string
		index *CacheIndex
	}

	items := make([]item, 0, len(dc.index))
	for key, index := range dc.index {
		items = append(items, item{key, index})
	}

	// Sort by last access time (oldest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].index.LastAccess.Before(items[j].index.LastAccess)
	})

	// Remove items until we're under the limit
	for _, item := range items {
		if dc.stats.TotalSize <= dc.config.MaxSize {
			break
		}

		os.Remove(item.index.FilePath)
		delete(dc.index, item.key)
		dc.stats.TotalSize -= item.index.Size
	}
}

// Delete removes a value from the disk cache
func (dc *DiskCache) Delete(key string) error {
	dc.indexMutex.Lock()
	defer dc.indexMutex.Unlock()

	// Check if key exists
	index, exists := dc.index[key]
	if !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	// Remove file
	if err := os.Remove(index.FilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Remove from index
	delete(dc.index, key)

	// Update stats
	dc.stats.TotalSize -= index.Size

	return nil
}
