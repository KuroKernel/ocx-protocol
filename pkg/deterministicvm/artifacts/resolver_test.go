package artifacts

import (
	"crypto/sha256"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
)

func TestArtifactResolver_ResolveArtifact(t *testing.T) {
	// Create test configuration
	config := &ArtifactConfig{
		Cache: struct {
			MemorySizeMB  int    `yaml:"memory_size_mb"`
			DiskSizeGB    int    `yaml:"disk_size_gb"`
			TTLMinutes    int    `yaml:"ttl_minutes"`
			BaseDirectory string `yaml:"base_directory"`
			ShardCount    int    `yaml:"shard_count"`
		}{
			MemorySizeMB:  64,
			DiskSizeGB:    1,
			TTLMinutes:    60,
			BaseDirectory: t.TempDir(),
			ShardCount:    4,
		},
		Remote: struct {
			Sources                []ArtifactSource `yaml:"sources"`
			TimeoutSeconds         int              `yaml:"timeout_seconds"`
			MaxRetries             int              `yaml:"max_retries"`
			BackoffSeconds         int              `yaml:"backoff_seconds"`
			MaxConcurrentDownloads int              `yaml:"max_concurrent_downloads"`
		}{
			Sources: []ArtifactSource{
				{
					URL:          "https://test.artifacts.local",
					Priority:     1,
					AuthRequired: false,
					Timeout:      5 * time.Second,
					RateLimit:    100,
				},
			},
			TimeoutSeconds:         5,
			MaxRetries:             2,
			BackoffSeconds:         1,
			MaxConcurrentDownloads: 5,
		},
		Security: struct {
			VerifySignatures bool   `yaml:"verify_signatures"`
			PublicKeyPath    string `yaml:"public_key_path"`
			TLSCertPath      string `yaml:"tls_cert_path"`
		}{
			VerifySignatures: false,
		},
		Performance: struct {
			EnableMetrics   bool `yaml:"enable_metrics"`
			MetricsPort     int  `yaml:"metrics_port"`
			EnableProfiling bool `yaml:"enable_profiling"`
			ProfilingPort   int  `yaml:"profiling_port"`
		}{
			EnableMetrics:   false,
			MetricsPort:     9090,
			EnableProfiling: false,
			ProfilingPort:   6060,
		},
	}

	// Create resolver
	resolver, err := NewArtifactResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Close()

	// Test data
	testData := []byte("test artifact data")
	hash := sha256.Sum256(testData)

	// Test resolving non-existent artifact
	_, err = resolver.ResolveArtifact(hash)
	if err == nil {
		t.Error("Expected error for non-existent artifact")
	}

	// Test with mock remote store
	resolver.remoteStore = &MockRemoteStore{
		artifacts: map[string][]byte{
			fmt.Sprintf("%x", hash): testData,
		},
	}

	// Test successful resolution
	artifact, err := resolver.ResolveArtifact(hash)
	if err != nil {
		t.Fatalf("Failed to resolve artifact: %v", err)
	}

	if artifact == nil {
		t.Fatal("Expected artifact, got nil")
	}

	if artifact.Hash != hash {
		t.Errorf("Expected hash %x, got %x", hash, artifact.Hash)
	}

	if artifact.Source != "https://test.artifacts.local" {
		t.Errorf("Expected source 'https://test.artifacts.local', got '%s'", artifact.Source)
	}
}

func TestArtifactResolver_ResolveArtifactBatch(t *testing.T) {
	config := createTestConfig(t)
	resolver, err := NewArtifactResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Close()

	// Test data
	testData1 := []byte("test artifact 1")
	testData2 := []byte("test artifact 2")
	hash1 := sha256.Sum256(testData1)
	hash2 := sha256.Sum256(testData2)

	// Mock remote store
	resolver.remoteStore = &MockRemoteStore{
		artifacts: map[string][]byte{
			fmt.Sprintf("%x", hash1): testData1,
			fmt.Sprintf("%x", hash2): testData2,
		},
	}

	// Test batch resolution
	hashes := [][32]byte{hash1, hash2}
	artifacts, err := resolver.ResolveArtifactBatch(hashes)
	if err != nil {
		t.Fatalf("Failed to resolve batch: %v", err)
	}

	if len(artifacts) != 2 {
		t.Errorf("Expected 2 artifacts, got %d", len(artifacts))
	}

	// Verify artifacts
	for i, artifact := range artifacts {
		if artifact.Hash != hashes[i] {
			t.Errorf("Artifact %d: expected hash %x, got %x", i, hashes[i], artifact.Hash)
		}
	}
}

func TestArtifactResolver_PreloadArtifacts(t *testing.T) {
	config := createTestConfig(t)
	resolver, err := NewArtifactResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Close()

	// Test data
	testData := []byte("test artifact for preload")
	hash := sha256.Sum256(testData)

	// Mock remote store
	resolver.remoteStore = &MockRemoteStore{
		artifacts: map[string][]byte{
			fmt.Sprintf("%x", hash): testData,
		},
	}

	// Test preload
	hashes := [][32]byte{hash}
	err = resolver.PreloadArtifacts(hashes)
	if err != nil {
		t.Fatalf("Failed to preload artifacts: %v", err)
	}

	// Verify artifact is cached
	artifact, err := resolver.ResolveArtifact(hash)
	if err != nil {
		t.Fatalf("Failed to resolve preloaded artifact: %v", err)
	}

	if artifact.Source != "memory_cache" && artifact.Source != "disk_cache" {
		t.Errorf("Expected cached artifact, got source: %s", artifact.Source)
	}
}

func TestArtifactResolver_ValidateArtifactIntegrity(t *testing.T) {
	config := createTestConfig(t)
	resolver, err := NewArtifactResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Close()

	// Test data
	testData := []byte("test artifact for integrity")
	hash := sha256.Sum256(testData)

	// Create artifact with correct hash
	artifact := &Artifact{
		Hash: hash,
		Data: &MockReadCloser{data: testData},
		Size: int64(len(testData)),
	}

	// Test valid artifact
	err = resolver.ValidateArtifactIntegrity(artifact)
	if err != nil {
		t.Errorf("Expected valid artifact, got error: %v", err)
	}

	// Test artifact with wrong hash
	wrongHash := sha256.Sum256([]byte("wrong data"))
	artifact.Hash = wrongHash
	err = resolver.ValidateArtifactIntegrity(artifact)
	if err == nil {
		t.Error("Expected error for artifact with wrong hash")
	}

	// Test nil artifact
	err = resolver.ValidateArtifactIntegrity(nil)
	if err == nil {
		t.Error("Expected error for nil artifact")
	}
}

func TestArtifactResolver_ConcurrentAccess(t *testing.T) {
	config := createTestConfig(t)
	resolver, err := NewArtifactResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Close()

	// Test data
	testData := []byte("concurrent test artifact")
	hash := sha256.Sum256(testData)

	// Mock remote store
	resolver.remoteStore = &MockRemoteStore{
		artifacts: map[string][]byte{
			fmt.Sprintf("%x", hash): testData,
		},
	}

	// Test concurrent access
	const numGoroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			artifact, err := resolver.ResolveArtifact(hash)
			if err != nil {
				errors <- err
				return
			}
			if artifact == nil {
				errors <- fmt.Errorf("got nil artifact")
				return
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

func TestArtifactResolver_ContextCancellation(t *testing.T) {
	config := createTestConfig(t)
	resolver, err := NewArtifactResolver(config)
	if err != nil {
		t.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Close()

	// Test data
	testData := []byte("test artifact for cancellation")
	hash := sha256.Sum256(testData)

	// Mock remote store with delay
	resolver.remoteStore = &MockRemoteStore{
		artifacts: map[string][]byte{
			fmt.Sprintf("%x", hash): testData,
		},
		delay: 100 * time.Millisecond,
	}

	// Cancel context
	resolver.cancel()

	// Test resolution with cancelled context
	_, err = resolver.ResolveArtifact(hash)
	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}

// Benchmark tests

func BenchmarkArtifactResolver_ResolveArtifact(b *testing.B) {
	config := createTestConfig(b)
	resolver, err := NewArtifactResolver(config)
	if err != nil {
		b.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Close()

	// Test data
	testData := []byte("benchmark test artifact")
	hash := sha256.Sum256(testData)

	// Mock remote store
	resolver.remoteStore = &MockRemoteStore{
		artifacts: map[string][]byte{
			fmt.Sprintf("%x", hash): testData,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		artifact, err := resolver.ResolveArtifact(hash)
		if err != nil {
			b.Fatalf("Failed to resolve artifact: %v", err)
		}
		if artifact == nil {
			b.Fatal("Got nil artifact")
		}
	}
}

func BenchmarkArtifactResolver_ConcurrentResolution(b *testing.B) {
	config := createTestConfig(b)
	resolver, err := NewArtifactResolver(config)
	if err != nil {
		b.Fatalf("Failed to create resolver: %v", err)
	}
	defer resolver.Close()

	// Test data
	testData := []byte("concurrent benchmark test artifact")
	hash := sha256.Sum256(testData)

	// Mock remote store
	resolver.remoteStore = &MockRemoteStore{
		artifacts: map[string][]byte{
			fmt.Sprintf("%x", hash): testData,
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			artifact, err := resolver.ResolveArtifact(hash)
			if err != nil {
				b.Fatalf("Failed to resolve artifact: %v", err)
			}
			if artifact == nil {
				b.Fatal("Got nil artifact")
			}
		}
	})
}

// Helper functions

func createTestConfig(t testing.TB) *ArtifactConfig {
	return &ArtifactConfig{
		Cache: struct {
			MemorySizeMB  int    `yaml:"memory_size_mb"`
			DiskSizeGB    int    `yaml:"disk_size_gb"`
			TTLMinutes    int    `yaml:"ttl_minutes"`
			BaseDirectory string `yaml:"base_directory"`
			ShardCount    int    `yaml:"shard_count"`
		}{
			MemorySizeMB:  64,
			DiskSizeGB:    1,
			TTLMinutes:    60,
			BaseDirectory: t.TempDir(),
			ShardCount:    4,
		},
		Remote: struct {
			Sources                []ArtifactSource `yaml:"sources"`
			TimeoutSeconds         int              `yaml:"timeout_seconds"`
			MaxRetries             int              `yaml:"max_retries"`
			BackoffSeconds         int              `yaml:"backoff_seconds"`
			MaxConcurrentDownloads int              `yaml:"max_concurrent_downloads"`
		}{
			Sources: []ArtifactSource{
				{
					URL:          "https://test.artifacts.local",
					Priority:     1,
					AuthRequired: false,
					Timeout:      5 * time.Second,
					RateLimit:    100,
				},
			},
			TimeoutSeconds:         5,
			MaxRetries:             2,
			BackoffSeconds:         1,
			MaxConcurrentDownloads: 5,
		},
		Security: struct {
			VerifySignatures bool   `yaml:"verify_signatures"`
			PublicKeyPath    string `yaml:"public_key_path"`
			TLSCertPath      string `yaml:"tls_cert_path"`
		}{
			VerifySignatures: false,
		},
		Performance: struct {
			EnableMetrics   bool `yaml:"enable_metrics"`
			MetricsPort     int  `yaml:"metrics_port"`
			EnableProfiling bool `yaml:"enable_profiling"`
			ProfilingPort   int  `yaml:"profiling_port"`
		}{
			EnableMetrics:   false,
			MetricsPort:     9090,
			EnableProfiling: false,
			ProfilingPort:   6060,
		},
	}
}

// Mock implementations

type MockRemoteStore struct {
	artifacts map[string][]byte
	delay     time.Duration
}

func (m *MockRemoteStore) FetchArtifact(hash [32]byte) (*Artifact, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	hashStr := fmt.Sprintf("%x", hash)
	data, exists := m.artifacts[hashStr]
	if !exists {
		return nil, fmt.Errorf("artifact not found: %s", hashStr)
	}

	return &Artifact{
		Hash:       hash,
		Data:       &MockReadCloser{data: data},
		Size:       int64(len(data)),
		Source:     "https://test.artifacts.local",
		ResolvedAt: time.Now(),
	}, nil
}

func (m *MockRemoteStore) HealthCheck() map[string]HealthStatus {
	return map[string]HealthStatus{
		"https://test.artifacts.local": {
			Healthy: true,
			Reason:  "OK",
		},
	}
}

func (m *MockRemoteStore) UpdateSourcePriorities() error {
	return nil
}

type MockReadCloser struct {
	data []byte
	pos  int
}

func (m *MockReadCloser) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *MockReadCloser) Close() error {
	return nil
}

func (m *MockReadCloser) Size() int64 {
	return int64(len(m.data))
}

func (m *MockReadCloser) Name() string {
	return "mock-artifact"
}
