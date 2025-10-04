package artifacts

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// RemoteStoreInterface defines the interface for remote artifact stores
type RemoteStoreInterface interface {
	FetchArtifact(hash [32]byte) (*Artifact, error)
	HealthCheck() map[string]HealthStatus
	UpdateSourcePriorities() error
}

// KeyLoader interface for loading public keys
type KeyLoader interface {
	PublicKeyForIssuer(issuer string) (ed25519.PublicKey, error)
}

// ArtifactResolver provides production-grade artifact resolution with multi-tier caching
type ArtifactResolver struct {
	// Multi-tier caching: memory + disk + network
	memCache    *lru.Cache[string, *ArtifactMetadata]
	diskCache   *DiskCache
	remoteStore RemoteStoreInterface

	// Cryptographic integrity
	hashVerifier      hash.Hash
	signatureVerifier ed25519.PublicKey
	keyLoader         KeyLoader

	// Performance monitoring
	metrics *ArtifactMetrics

	// Concurrent access control
	downloadSemaphore chan struct{}
	cacheMutex        sync.RWMutex

	// Configuration
	config *ArtifactConfig

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// ArtifactMetadata contains metadata about a cached artifact
type ArtifactMetadata struct {
	Hash        [32]byte
	Size        int64
	LastAccess  time.Time
	AccessCount int64
	LocalPath   string
	Signature   []byte
	CreatedAt   time.Time
}

// Artifact represents a resolved artifact with all necessary information
type Artifact struct {
	Hash       [32]byte
	Data       io.ReadCloser
	Size       int64
	LocalPath  string
	Signature  []byte
	Source     string
	ResolvedAt time.Time
}

// NewArtifactResolver creates a new production-grade artifact resolver
func NewArtifactResolver(config *ArtifactConfig) (*ArtifactResolver, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create memory cache with LRU eviction
	memCache, err := lru.New[string, *ArtifactMetadata](config.Cache.MemorySizeMB * 1024 * 1024 / 1024) // Rough estimate
	if err != nil {
		return nil, fmt.Errorf("failed to create memory cache: %w", err)
	}

	// Create disk cache
	diskCache, err := NewDiskCache(config.Cache.BaseDirectory, int64(config.Cache.DiskSizeGB)*1024*1024*1024)
	if err != nil {
		return nil, fmt.Errorf("failed to create disk cache: %w", err)
	}

	// Create remote store
	remoteStore, err := NewRemoteArtifactStore(config.Remote.Sources, config.Remote.TimeoutSeconds, config.Remote.MaxRetries)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote store: %w", err)
	}

	// Create metrics
	metrics := NewArtifactMetrics()

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	ar := &ArtifactResolver{
		memCache:          memCache,
		diskCache:         diskCache,
		remoteStore:       remoteStore,
		hashVerifier:      sha256.New(),
		signatureVerifier: nil, // Will be set if signature verification is enabled
		metrics:           metrics,
		downloadSemaphore: make(chan struct{}, config.Remote.MaxConcurrentDownloads),
		config:            config,
		ctx:               ctx,
		cancel:            cancel,
	}

	// Load signature verification key if enabled
	if config.Security.VerifySignatures && config.Security.PublicKeyPath != "" {
		if err := ar.loadSignatureKey(config.Security.PublicKeyPath); err != nil {
			return nil, fmt.Errorf("failed to load signature key: %w", err)
		}
	}

	// Start background maintenance
	go ar.backgroundMaintenance()

	return ar, nil
}

// ResolveArtifact resolves a single artifact by hash with full fallback chain
func (ar *ArtifactResolver) ResolveArtifact(hash [32]byte) (*Artifact, error) {
	// Check for context cancellation first
	select {
	case <-ar.ctx.Done():
		return nil, ar.ctx.Err()
	default:
	}

	start := time.Now()
	defer func() {
		ar.metrics.RecordResolutionLatency(time.Since(start))
	}()

	hashStr := fmt.Sprintf("%x", hash)

	// Check memory cache first
	ar.cacheMutex.RLock()
	if metadata, ok := ar.memCache.Get(hashStr); ok {
		ar.cacheMutex.RUnlock()
		ar.metrics.RecordCacheHit("memory")

		// Verify artifact still exists on disk
		if artifact, err := ar.diskCache.Get(hash); err == nil {
			// Get file size
			stat, err := artifact.Stat()
			if err != nil {
				return nil, fmt.Errorf("failed to stat artifact: %w", err)
			}

			// Update access tracking atomically
			ar.cacheMutex.Lock()
			if updatedMetadata, ok := ar.memCache.Get(hashStr); ok {
				updatedMetadata.LastAccess = time.Now()
				updatedMetadata.AccessCount++
			}
			ar.cacheMutex.Unlock()

			return &Artifact{
				Hash:       hash,
				Data:       artifact,
				Size:       stat.Size(),
				LocalPath:  metadata.LocalPath,
				Signature:  metadata.Signature,
				Source:     "memory_cache",
				ResolvedAt: time.Now(),
			}, nil
		}
	} else {
		ar.cacheMutex.RUnlock()
	}

	// Check for context cancellation before disk cache
	select {
	case <-ar.ctx.Done():
		return nil, ar.ctx.Err()
	default:
	}

	// Check disk cache
	if artifact, err := ar.diskCache.Get(hash); err == nil {
		ar.metrics.RecordCacheHit("disk")

		// Get file size and name
		stat, err := artifact.Stat()
		if err != nil {
			return nil, fmt.Errorf("failed to stat artifact: %w", err)
		}

		// Add to memory cache
		ar.cacheMutex.Lock()
		ar.memCache.Add(hashStr, &ArtifactMetadata{
			Hash:        hash,
			Size:        stat.Size(),
			LastAccess:  time.Now(),
			AccessCount: 1,
			LocalPath:   artifact.Name(),
			CreatedAt:   time.Now(),
		})
		ar.cacheMutex.Unlock()

		return &Artifact{
			Hash:       hash,
			Data:       artifact,
			Size:       stat.Size(),
			LocalPath:  artifact.Name(),
			Source:     "disk_cache",
			ResolvedAt: time.Now(),
		}, nil
	}

	// Check for context cancellation before remote download
	select {
	case <-ar.ctx.Done():
		return nil, ar.ctx.Err()
	default:
	}

	// Download from remote sources
	ar.metrics.RecordCacheMiss("all")
	return ar.downloadArtifact(hash)
}

// ResolveArtifactBatch resolves multiple artifacts concurrently
func (ar *ArtifactResolver) ResolveArtifactBatch(hashes [][32]byte) ([]*Artifact, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	// Limit concurrent downloads
	semaphore := make(chan struct{}, ar.config.Remote.MaxConcurrentDownloads)
	results := make([]*Artifact, len(hashes))
	errors := make([]error, len(hashes))

	var wg sync.WaitGroup
	for i, hash := range hashes {
		wg.Add(1)
		go func(index int, h [32]byte) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			artifact, err := ar.ResolveArtifact(h)
			if err != nil {
				errors[index] = err
				return
			}
			results[index] = artifact
		}(i, hash)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("batch resolution failed: %w", err)
		}
	}

	return results, nil
}

// PreloadArtifacts preloads artifacts into cache for faster access
func (ar *ArtifactResolver) PreloadArtifacts(hashes [][32]byte) error {
	if len(hashes) == 0 {
		return nil
	}

	// Use batch resolution to preload
	artifacts, err := ar.ResolveArtifactBatch(hashes)
	if err != nil {
		return fmt.Errorf("preload failed: %w", err)
	}

	// Ensure all artifacts are cached
	for _, artifact := range artifacts {
		if artifact != nil {
			ar.metrics.RecordPreload(artifact.Size)
		}
	}

	return nil
}

// ValidateArtifactIntegrity validates the cryptographic integrity of an artifact
func (ar *ArtifactResolver) ValidateArtifactIntegrity(artifact *Artifact) error {
	if artifact == nil {
		return fmt.Errorf("artifact is nil")
	}

	// Verify hash
	ar.hashVerifier.Reset()
	if _, err := io.Copy(ar.hashVerifier, artifact.Data); err != nil {
		return fmt.Errorf("failed to compute hash: %w", err)
	}

	computedHash := ar.hashVerifier.Sum(nil)
	if !bytesEqual(computedHash, artifact.Hash[:]) {
		return fmt.Errorf("hash mismatch: expected %x, got %x", artifact.Hash, computedHash)
	}

	// Verify signature if enabled
	if ar.config.Security.VerifySignatures && ar.signatureVerifier != nil {
		if err := ar.verifySignature(artifact); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
	}

	return nil
}

// downloadArtifact downloads an artifact from remote sources
func (ar *ArtifactResolver) downloadArtifact(hash [32]byte) (*Artifact, error) {
	// Acquire download semaphore
	select {
	case ar.downloadSemaphore <- struct{}{}:
		defer func() { <-ar.downloadSemaphore }()
	case <-ar.ctx.Done():
		return nil, ar.ctx.Err()
	}

	// Try remote sources
	artifact, err := ar.remoteStore.FetchArtifact(hash)
	if err != nil {
		ar.metrics.RecordDownloadError("remote", err)
		return nil, err
	}

	// Cache the artifact
	if err := ar.cacheArtifact(hash, artifact); err != nil {
		ar.metrics.RecordCacheError("disk", err)
		// Continue even if caching fails
	}

	ar.metrics.RecordDownloadSuccess("remote", artifact.Size)
	return artifact, nil
}

// cacheArtifact caches an artifact to disk and memory
func (ar *ArtifactResolver) cacheArtifact(hash [32]byte, artifact *Artifact) error {
	// Cache to disk
	if err := ar.diskCache.Put(hash, artifact.Data); err != nil {
		return fmt.Errorf("failed to cache to disk: %w", err)
	}

	// Add to memory cache
	hashStr := fmt.Sprintf("%x", hash)
	ar.cacheMutex.Lock()
	ar.memCache.Add(hashStr, &ArtifactMetadata{
		Hash:        hash,
		Size:        artifact.Size,
		LastAccess:  time.Now(),
		AccessCount: 1,
		LocalPath:   artifact.LocalPath,
		Signature:   artifact.Signature,
		CreatedAt:   time.Now(),
	})
	ar.cacheMutex.Unlock()

	return nil
}

// loadSignatureKey loads the Ed25519 public key for signature verification
func (ar *ArtifactResolver) loadSignatureKey(keyPath string) error {
	// Load the public key from file
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key file %s: %w", keyPath, err)
	}

	// Parse the public key - try hex first, then raw bytes
	keyStr := strings.TrimSpace(string(keyData))
	if key, err := hex.DecodeString(keyStr); err == nil && len(key) == ed25519.PublicKeySize {
		ar.signatureVerifier = ed25519.PublicKey(key)
		return nil
	}

	// Try raw bytes
	if len(keyData) == ed25519.PublicKeySize {
		ar.signatureVerifier = ed25519.PublicKey(keyData)
		return nil
	}

	return fmt.Errorf("invalid public key format in %s", keyPath)
}

// loadKey loads a public key for an issuer using the KeyLoader interface
func (ar *ArtifactResolver) loadKey(issuer string) (ed25519.PublicKey, error) {
	if ar.keyLoader != nil {
		return ar.keyLoader.PublicKeyForIssuer(issuer)
	}

	// Fallback to keystore files under OCX_KEYSTORE_PATH
	root := os.Getenv("OCX_KEYSTORE_PATH")
	if root == "" {
		return nil, errors.New("no keystore configured")
	}

	pem := filepath.Join(root, issuer+".pub")
	b, err := os.ReadFile(pem)
	if err != nil {
		return nil, err
	}

	// Expect hex or raw; try hex first
	if k, err := hex.DecodeString(strings.TrimSpace(string(b))); err == nil && len(k) == ed25519.PublicKeySize {
		return ed25519.PublicKey(k), nil
	}

	if len(b) == ed25519.PublicKeySize {
		return ed25519.PublicKey(b), nil
	}

	return nil, errors.New("invalid public key encoding")
}

// verifySignature verifies the Ed25519 signature of an artifact
func (ar *ArtifactResolver) verifySignature(artifact *Artifact) error {
	if len(artifact.Signature) == 0 {
		return fmt.Errorf("no signature provided")
	}

	if len(ar.signatureVerifier) == 0 {
		return fmt.Errorf("no signature key loaded")
	}

	// Verify signature length
	if len(artifact.Signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length: expected %d bytes, got %d", ed25519.SignatureSize, len(artifact.Signature))
	}

	// Create the message to verify (artifact content with domain separation)
	// Read the artifact data
	artifactData, err := io.ReadAll(artifact.Data)
	if err != nil {
		return fmt.Errorf("failed to read artifact data: %w", err)
	}
	msg := append([]byte("OCXv1|artifact|"), artifactData...)

	// Verify the Ed25519 signature
	if !ed25519.Verify(ar.signatureVerifier, msg, artifact.Signature) {
		return errors.New("artifact signature invalid")
	}

	return nil
}

// verifyArtifactSignature verifies an artifact signature with a specific issuer
func (ar *ArtifactResolver) verifyArtifactSignature(issuer string, content, signature []byte) error {
	pub, err := ar.loadKey(issuer)
	if err != nil {
		return err
	}
	msg := append([]byte("OCXv1|artifact|"), content...)
	if !ed25519.Verify(pub, msg, signature) {
		return errors.New("artifact signature invalid")
	}
	return nil
}

// parseEd25519PublicKey parses an Ed25519 public key from raw bytes
func parseEd25519PublicKey(keyData []byte) (ed25519.PublicKey, error) {
	// Accept raw Ed25519 public key bytes (32 bytes)
	if len(keyData) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: expected %d bytes, got %d", ed25519.PublicKeySize, len(keyData))
	}

	return ed25519.PublicKey(keyData), nil
}

// backgroundMaintenance performs background cache maintenance
func (ar *ArtifactResolver) backgroundMaintenance() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ar.performMaintenance()
		case <-ar.ctx.Done():
			return
		}
	}
}

// performMaintenance performs cache maintenance tasks
func (ar *ArtifactResolver) performMaintenance() {
	// Clean up expired memory cache entries
	ar.cacheMutex.Lock()
	// Note: LRU cache doesn't have PeekAll, so we'll skip this for now
	// In a production implementation, we'd need to track expiration separately
	ar.cacheMutex.Unlock()

	// Perform disk cache maintenance
	if err := ar.diskCache.CompactAndRepair(); err != nil {
		ar.metrics.RecordMaintenanceError(err)
	}
}

// Close gracefully shuts down the artifact resolver
func (ar *ArtifactResolver) Close() error {
	ar.cancel()

	if err := ar.diskCache.Close(); err != nil {
		return fmt.Errorf("failed to close disk cache: %w", err)
	}

	return nil
}

// bytesEqual compares two byte slices for equality
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
