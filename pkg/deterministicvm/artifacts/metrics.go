package artifacts

import (
	"sync"
	"time"
)

// ArtifactMetrics provides production-grade metrics for artifact resolution
type ArtifactMetrics struct {
	// Cache performance
	CacheHitRate  map[string]int64
	CacheMissRate map[string]int64
	EvictionRate  map[string]int64

	// Download performance
	DownloadDuration map[string]time.Duration
	DownloadBytes    map[string]int64
	DownloadErrors   map[string]int64

	// System resources
	DiskUsage        int64
	MemoryUsage      int64
	NetworkBandwidth float64

	// Resolution performance
	ResolutionLatency time.Duration
	ResolutionTotal   int64
	ResolutionErrors  int64

	// Cache operations
	CacheOperations map[string]int64
	CacheSize       map[string]int64

	// Preload operations
	PreloadTotal  int64
	PreloadBytes  int64
	PreloadErrors int64

	// Maintenance operations
	MaintenanceErrors   int64
	MaintenanceDuration time.Duration

	// Mutex for thread safety
	mutex sync.RWMutex
}

// NewArtifactMetrics creates a new metrics instance
func NewArtifactMetrics() *ArtifactMetrics {
	return &ArtifactMetrics{
		CacheHitRate:     make(map[string]int64),
		CacheMissRate:    make(map[string]int64),
		EvictionRate:     make(map[string]int64),
		DownloadDuration: make(map[string]time.Duration),
		DownloadBytes:    make(map[string]int64),
		DownloadErrors:   make(map[string]int64),
		CacheOperations:  make(map[string]int64),
		CacheSize:        make(map[string]int64),
	}
}

// RecordCacheHit records a cache hit
func (am *ArtifactMetrics) RecordCacheHit(source string) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.CacheHitRate[source]++
	am.CacheOperations["hit_"+source]++
}

// RecordCacheMiss records a cache miss
func (am *ArtifactMetrics) RecordCacheMiss(source string) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.CacheMissRate[source]++
	am.CacheOperations["miss_"+source]++
}

// RecordDownloadLatency records download latency
func (am *ArtifactMetrics) RecordDownloadLatency(source string, duration time.Duration) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.DownloadDuration[source] = duration
}

// RecordDownloadError records a download error
func (am *ArtifactMetrics) RecordDownloadError(source string, err error) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	errorType := "unknown"
	if err != nil {
		errorType = err.Error()
	}
	am.DownloadErrors[source+"_"+errorType]++
}

// RecordDownloadSuccess records a successful download
func (am *ArtifactMetrics) RecordDownloadSuccess(source string, bytes int64) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.DownloadBytes[source] += bytes
}

// RecordResolutionLatency records resolution latency
func (am *ArtifactMetrics) RecordResolutionLatency(duration time.Duration) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.ResolutionLatency = duration
	am.ResolutionTotal++
}

// RecordResolutionError records a resolution error
func (am *ArtifactMetrics) RecordResolutionError() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.ResolutionErrors++
}

// RecordCacheError records a cache error
func (am *ArtifactMetrics) RecordCacheError(operation string, err error) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	errorType := "unknown"
	if err != nil {
		errorType = err.Error()
	}
	am.CacheOperations[operation+"_"+errorType]++
}

// RecordPreload records a preload operation
func (am *ArtifactMetrics) RecordPreload(bytes int64) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.PreloadTotal++
	am.PreloadBytes += bytes
}

// RecordPreloadError records a preload error
func (am *ArtifactMetrics) RecordPreloadError() {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.PreloadErrors++
}

// RecordMaintenanceError records a maintenance error
func (am *ArtifactMetrics) RecordMaintenanceError(err error) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.MaintenanceErrors++
}

// RecordMaintenanceDuration records maintenance duration
func (am *ArtifactMetrics) RecordMaintenanceDuration(duration time.Duration) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.MaintenanceDuration = duration
}

// UpdateDiskUsage updates disk usage metric
func (am *ArtifactMetrics) UpdateDiskUsage(bytes int64) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.DiskUsage = bytes
	am.CacheSize["disk"] = bytes
}

// UpdateMemoryUsage updates memory usage metric
func (am *ArtifactMetrics) UpdateMemoryUsage(bytes int64) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.MemoryUsage = bytes
	am.CacheSize["memory"] = bytes
}

// UpdateNetworkBandwidth updates network bandwidth metric
func (am *ArtifactMetrics) UpdateNetworkBandwidth(bytesPerSecond float64) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.NetworkBandwidth = bytesPerSecond
}

// RecordEviction records a cache eviction
func (am *ArtifactMetrics) RecordEviction(reason string) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.EvictionRate[reason]++
	am.CacheOperations["evict_"+reason]++
}
