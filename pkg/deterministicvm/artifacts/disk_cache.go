package artifacts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"go.uber.org/atomic"
)

// DiskCache provides production-grade disk caching with atomic operations and LRU eviction
type DiskCache struct {
	baseDir      string
	maxSizeBytes int64
	currentSize  *atomic.Int64

	// LRU eviction
	accessTracker *AccessTracker
	evictionMutex sync.Mutex

	// Integrity verification
	checksumStore *ChecksumStore

	// Metrics
	hitCounter  *atomic.Uint64
	missCounter *atomic.Uint64

	// Configuration
	shardCount int
	shardMutex []sync.RWMutex

	// Cache state
	cacheState *CacheState
	stateMutex sync.RWMutex
}

// AccessTracker tracks access patterns for LRU eviction
type AccessTracker struct {
	accessTimes map[string]time.Time
	accessCount map[string]int64
	mutex       sync.RWMutex
}

// ChecksumStore stores and verifies file checksums
type ChecksumStore struct {
	checksums map[string]string
	mutex     sync.RWMutex
}

// CacheState represents the persistent state of the disk cache
type CacheState struct {
	Entries  map[string]CacheEntry `json:"entries"`
	LRU      []string              `json:"lru"`
	TotalSize int64                `json:"total_size"`
}

// CacheEntry represents a single cached file entry
type CacheEntry struct {
	Hash         string    `json:"hash"`
	Size         int64     `json:"size"`
	LastAccessed time.Time `json:"last_accessed"`
	AccessCount  int64     `json:"access_count"`
}

// NewDiskCache creates a new production-grade disk cache
func NewDiskCache(baseDir string, maxSizeBytes int64) (*DiskCache, error) {
	if maxSizeBytes <= 0 {
		return nil, fmt.Errorf("max size must be positive")
	}

	// Create base directory
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Create shard directories
	shardCount := 256 // 256 shards for good distribution
	shardMutex := make([]sync.RWMutex, shardCount)

	for i := 0; i < shardCount; i++ {
		shardDir := filepath.Join(baseDir, fmt.Sprintf("shard_%03d", i))
		if err := os.MkdirAll(shardDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create shard directory %d: %w", i, err)
		}
	}

	dc := &DiskCache{
		baseDir:      baseDir,
		maxSizeBytes: maxSizeBytes,
		currentSize:  atomic.NewInt64(0),
		accessTracker: &AccessTracker{
			accessTimes: make(map[string]time.Time),
			accessCount: make(map[string]int64),
		},
		checksumStore: &ChecksumStore{
			checksums: make(map[string]string),
		},
		hitCounter:  atomic.NewUint64(0),
		missCounter: atomic.NewUint64(0),
		shardCount:  shardCount,
		shardMutex:  shardMutex,
	}

	// Load existing cache state
	if err := dc.loadCacheState(); err != nil {
		return nil, fmt.Errorf("failed to load cache state: %w", err)
	}

	// Start background maintenance
	go dc.backgroundMaintenance()

	return dc, nil
}

// Get retrieves an artifact from disk cache
func (dc *DiskCache) Get(hash [32]byte) (*os.File, error) {
	hashStr := fmt.Sprintf("%x", hash)
	shardIndex := dc.getShardIndex(hash)

	dc.shardMutex[shardIndex].RLock()
	defer dc.shardMutex[shardIndex].RUnlock()

	// Construct file path
	filePath := dc.getFilePath(hash)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		dc.missCounter.Inc()
		return nil, fmt.Errorf("artifact not found: %s", hashStr)
	}

	// Verify integrity
	if err := dc.verifyFileIntegrity(hash, filePath); err != nil {
		dc.missCounter.Inc()
		return nil, fmt.Errorf("integrity check failed: %w", err)
	}

	// Update access tracking
	dc.accessTracker.mutex.Lock()
	dc.accessTracker.accessTimes[hashStr] = time.Now()
	dc.accessTracker.accessCount[hashStr]++
	dc.accessTracker.mutex.Unlock()

	// Open file for reading
	file, err := os.Open(filePath)
	if err != nil {
		dc.missCounter.Inc()
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	dc.hitCounter.Inc()
	return file, nil
}

// Put stores an artifact in disk cache with atomic operations
func (dc *DiskCache) Put(hash [32]byte, data io.Reader) error {
	hashStr := fmt.Sprintf("%x", hash)
	shardIndex := dc.getShardIndex(hash)

	dc.shardMutex[shardIndex].Lock()
	defer dc.shardMutex[shardIndex].Unlock()

	// Construct file paths
	filePath := dc.getFilePath(hash)
	tempPath := filePath + ".tmp"
	metaPath := filePath + ".meta"

	// Create temporary file
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempPath)

	// Copy data and compute checksum
	hasher := sha256.New()
	multiWriter := io.MultiWriter(tempFile, hasher)

	size, err := io.Copy(multiWriter, data)
	if err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to copy data: %w", err)
	}

	tempFile.Close()

	// Check size limits
	if dc.currentSize.Load()+size > dc.maxSizeBytes {
		// Try to free space
		if err := dc.EvictLRU(size); err != nil {
			return fmt.Errorf("insufficient space and eviction failed: %w", err)
		}
	}

	// Compute checksum
	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	// Create metadata
	metadata := FileMetadata{
		Hash:        hashStr,
		Size:        size,
		Checksum:    checksum,
		CreatedAt:   time.Now(),
		LastAccess:  time.Now(),
		AccessCount: 1,
	}

	// Write metadata
	metaData, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metaPath+".tmp", metaData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Atomic move operations
	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("failed to move temp file: %w", err)
	}

	if err := os.Rename(metaPath+".tmp", metaPath); err != nil {
		os.Remove(filePath) // Clean up on error
		return fmt.Errorf("failed to move metadata: %w", err)
	}

	// Update cache state
	dc.currentSize.Add(size)
	dc.accessTracker.mutex.Lock()
	dc.accessTracker.accessTimes[hashStr] = time.Now()
	dc.accessTracker.accessCount[hashStr] = 1
	dc.accessTracker.mutex.Unlock()

	dc.checksumStore.mutex.Lock()
	dc.checksumStore.checksums[hashStr] = checksum
	dc.checksumStore.mutex.Unlock()

	return nil
}

// Delete removes an artifact from disk cache
func (dc *DiskCache) Delete(hash [32]byte) error {
	hashStr := fmt.Sprintf("%x", hash)
	shardIndex := dc.getShardIndex(hash)

	dc.shardMutex[shardIndex].Lock()
	defer dc.shardMutex[shardIndex].Unlock()

	// Construct file paths
	filePath := dc.getFilePath(hash)
	metaPath := filePath + ".meta"

	// Get file size before deletion
	var size int64
	if stat, err := os.Stat(filePath); err == nil {
		size = stat.Size()
	}

	// Remove files
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove metadata: %w", err)
	}

	// Update cache state
	dc.currentSize.Sub(size)
	dc.accessTracker.mutex.Lock()
	delete(dc.accessTracker.accessTimes, hashStr)
	delete(dc.accessTracker.accessCount, hashStr)
	dc.accessTracker.mutex.Unlock()

	dc.checksumStore.mutex.Lock()
	delete(dc.checksumStore.checksums, hashStr)
	dc.checksumStore.mutex.Unlock()

	return nil
}

// EvictLRU evicts least recently used artifacts to free space
func (dc *DiskCache) EvictLRU(bytesToFree int64) error {
	dc.evictionMutex.Lock()
	defer dc.evictionMutex.Unlock()

	// Collect all files with access information
	type fileInfo struct {
		hash        string
		lastAccess  time.Time
		accessCount int64
		size        int64
	}

	var files []fileInfo
	dc.accessTracker.mutex.RLock()
	for hash, lastAccess := range dc.accessTracker.accessTimes {
		accessCount := dc.accessTracker.accessCount[hash]

		// Get file size
		filePath := dc.getFilePathFromHash(hash)
		if stat, err := os.Stat(filePath); err == nil {
			files = append(files, fileInfo{
				hash:        hash,
				lastAccess:  lastAccess,
				accessCount: accessCount,
				size:        stat.Size(),
			})
		}
	}
	dc.accessTracker.mutex.RUnlock()

	// Sort by access time (oldest first) and access count (least accessed first)
	sort.Slice(files, func(i, j int) bool {
		if files[i].lastAccess.Equal(files[j].lastAccess) {
			return files[i].accessCount < files[j].accessCount
		}
		return files[i].lastAccess.Before(files[j].lastAccess)
	})

	// Evict files until we have enough space
	var freedBytes int64
	for _, file := range files {
		if freedBytes >= bytesToFree {
			break
		}

		hashBytes := [32]byte{}
		if _, err := fmt.Sscanf(file.hash, "%x", &hashBytes); err != nil {
			continue
		}

		if err := dc.Delete(hashBytes); err != nil {
			continue // Continue on error
		}

		freedBytes += file.size
	}

	if freedBytes < bytesToFree {
		return fmt.Errorf("insufficient space to free %d bytes (freed %d)", bytesToFree, freedBytes)
	}

	return nil
}

// VerifyIntegrity verifies the integrity of all cached files
func (dc *DiskCache) VerifyIntegrity() error {
	var errors []error

	// Check all shards
	for i := 0; i < dc.shardCount; i++ {
		shardDir := filepath.Join(dc.baseDir, fmt.Sprintf("shard_%03d", i))

		entries, err := os.ReadDir(shardDir)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to read shard %d: %w", i, err))
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || entry.Name() == "" {
				continue
			}

			// Skip metadata files
			if filepath.Ext(entry.Name()) == ".meta" {
				continue
			}

			// Extract hash from filename
			hashStr := entry.Name()
			hashBytes := [32]byte{}
			if _, err := fmt.Sscanf(hashStr, "%x", &hashBytes); err != nil {
				errors = append(errors, fmt.Errorf("invalid hash in filename %s: %w", entry.Name(), err))
				continue
			}

			// Verify file integrity
			filePath := filepath.Join(shardDir, entry.Name())
			if err := dc.verifyFileIntegrity(hashBytes, filePath); err != nil {
				errors = append(errors, fmt.Errorf("integrity check failed for %s: %w", hashStr, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("integrity verification failed: %v", errors)
	}

	return nil
}

// CompactAndRepair performs cache compaction and repair
func (dc *DiskCache) CompactAndRepair() error {
	// Remove orphaned files
	if err := dc.removeOrphanedFiles(); err != nil {
		return fmt.Errorf("failed to remove orphaned files: %w", err)
	}

	// Rebuild access tracking
	if err := dc.rebuildAccessTracking(); err != nil {
		return fmt.Errorf("failed to rebuild access tracking: %w", err)
	}

	// Recalculate cache size
	if err := dc.recalculateSize(); err != nil {
		return fmt.Errorf("failed to recalculate size: %w", err)
	}

	return nil
}

// Close gracefully shuts down the disk cache
func (dc *DiskCache) Close() error {
	// Save cache state
	if err := dc.saveCacheState(); err != nil {
		return fmt.Errorf("failed to save cache state: %w", err)
	}

	return nil
}

// Helper methods

func (dc *DiskCache) getShardIndex(hash [32]byte) int {
	return int(hash[0]) % dc.shardCount
}

func (dc *DiskCache) getFilePath(hash [32]byte) string {
	hashStr := fmt.Sprintf("%x", hash)
	shardIndex := dc.getShardIndex(hash)
	shardDir := filepath.Join(dc.baseDir, fmt.Sprintf("shard_%03d", shardIndex))
	return filepath.Join(shardDir, hashStr)
}

func (dc *DiskCache) getFilePathFromHash(hashStr string) string {
	hashBytes := [32]byte{}
	fmt.Sscanf(hashStr, "%x", &hashBytes)
	return dc.getFilePath(hashBytes)
}

func (dc *DiskCache) verifyFileIntegrity(hash [32]byte, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}

	computedHash := hasher.Sum(nil)
	if !bytesEqual(computedHash, hash[:]) {
		return fmt.Errorf("hash mismatch")
	}

	return nil
}

func (dc *DiskCache) loadCacheState() error {
	// Load cache state from disk metadata file
	stateFile := filepath.Join(dc.baseDir, "cache_state.json")
	
	// Check if state file exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		// No state file exists, initialize with empty state
		dc.cacheState = &CacheState{
			Entries: make(map[string]CacheEntry),
			LRU:     make([]string, 0),
		}
		return nil
	}
	
	// Read and parse state file
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return fmt.Errorf("failed to read cache state: %w", err)
	}
	
	var state CacheState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse cache state: %w", err)
	}
	
	dc.cacheState = &state
	return nil
}

func (dc *DiskCache) saveCacheState() error {
	// Save cache state to disk metadata file
	stateFile := filepath.Join(dc.baseDir, "cache_state.json")
	
	// Marshal cache state to JSON
	data, err := json.Marshal(dc.cacheState)
	if err != nil {
		return fmt.Errorf("failed to marshal cache state: %w", err)
	}
	
	// Write to temporary file first, then rename for atomicity
	tempFile := stateFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache state: %w", err)
	}
	
	// Atomic rename
	if err := os.Rename(tempFile, stateFile); err != nil {
		os.Remove(tempFile) // Clean up temp file
		return fmt.Errorf("failed to rename cache state file: %w", err)
	}
	
	return nil
}

func (dc *DiskCache) backgroundMaintenance() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dc.CompactAndRepair()
		}
	}
}

func (dc *DiskCache) removeOrphanedFiles() error {
	// Remove files that exist on disk but not in cache state
	entries, err := os.ReadDir(dc.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}
	
	removedCount := 0
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "cache_state.json" {
			continue // Skip directories and state file
		}
		
		// Check if file exists in cache state
		if _, exists := dc.cacheState.Entries[entry.Name()]; !exists {
			filePath := filepath.Join(dc.baseDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				// Log error but continue with other files
				continue
			}
			removedCount++
		}
	}
	
	if removedCount > 0 {
		// Update cache state after cleanup
		dc.saveCacheState()
	}
	
	return nil
}

func (dc *DiskCache) rebuildAccessTracking() error {
	// Rebuild LRU access tracking from file modification times
	dc.cacheState.LRU = make([]string, 0, len(dc.cacheState.Entries))
	
	// Get file modification times and sort by access time
	type fileInfo struct {
		name    string
		modTime time.Time
	}
	
	var files []fileInfo
	for name, entry := range dc.cacheState.Entries {
		filePath := filepath.Join(dc.baseDir, name)
		if stat, err := os.Stat(filePath); err == nil {
			files = append(files, fileInfo{
				name:    name,
				modTime: stat.ModTime(),
			})
		} else {
			// File doesn't exist, use entry timestamp
			files = append(files, fileInfo{
				name:    name,
				modTime: entry.LastAccessed,
			})
		}
	}
	
	// Sort by modification time (oldest first for LRU)
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[i].modTime.After(files[j].modTime) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
	
	// Rebuild LRU list
	for _, file := range files {
		dc.cacheState.LRU = append(dc.cacheState.LRU, file.name)
	}
	
	return nil
}

func (dc *DiskCache) recalculateSize() error {
	// Recalculate total cache size by summing file sizes
	totalSize := int64(0)
	
	for name, entry := range dc.cacheState.Entries {
		filePath := filepath.Join(dc.baseDir, name)
		if stat, err := os.Stat(filePath); err == nil {
			fileSize := stat.Size()
			entry.Size = fileSize
			totalSize += fileSize
		} else {
			// File doesn't exist, remove from cache state
			delete(dc.cacheState.Entries, name)
		}
	}
	
	// Update total size
	dc.cacheState.TotalSize = totalSize
	
	// Save updated state
	return dc.saveCacheState()
}

// FileMetadata represents metadata for a cached file
type FileMetadata struct {
	Hash        string    `json:"hash"`
	Size        int64     `json:"size"`
	Checksum    string    `json:"checksum"`
	CreatedAt   time.Time `json:"created_at"`
	LastAccess  time.Time `json:"last_access"`
	AccessCount int64     `json:"access_count"`
}
