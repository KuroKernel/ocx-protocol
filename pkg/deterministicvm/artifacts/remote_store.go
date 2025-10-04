package artifacts

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RemoteArtifactStore provides production-grade remote artifact fetching
type RemoteArtifactStore struct {
	sources []ArtifactSource
	client  *http.Client

	// Circuit breaker per source
	breakers map[string]*CircuitBreaker

	// Request rate limiting
	rateLimiters map[string]*rate.Limiter

	// Retry configuration
	retryConfig *RetryConfig

	// Authentication
	authProvider AuthProvider

	// Metrics
	metrics *RemoteStoreMetrics
}

// ArtifactSource represents a remote artifact source
type ArtifactSource struct {
	URL          string
	Priority     int
	AuthRequired bool
	Timeout      time.Duration
	RateLimit    rate.Limit
}

// CircuitBreaker implements circuit breaker pattern for fault tolerance
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	state            CircuitState
	failureCount     int
	lastFailureTime  time.Time
	mutex            sync.RWMutex
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Jitter        bool
}

// AuthProvider provides authentication for remote sources
type AuthProvider interface {
	GetAuthHeader(source string) (string, error)
}

// RemoteStoreMetrics tracks remote store performance
type RemoteStoreMetrics struct {
	// Prometheus-style metrics for remote store operations
	TotalRequests    int64     `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests   int64     `json:"failed_requests"`
	AverageLatency   float64   `json:"average_latency_ms"`
	LastRequestTime  time.Time `json:"last_request_time"`
	CircuitBreakerOpen bool    `json:"circuit_breaker_open"`
	SourcePriorities map[string]int `json:"source_priorities"`
}

// NewRemoteArtifactStore creates a new remote artifact store
func NewRemoteArtifactStore(sources []ArtifactSource, timeoutSeconds, maxRetries int) (*RemoteArtifactStore, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("no sources provided")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Create circuit breakers for each source
	breakers := make(map[string]*CircuitBreaker)
	rateLimiters := make(map[string]*rate.Limiter)

	for _, source := range sources {
		breakers[source.URL] = &CircuitBreaker{
			failureThreshold: 5,
			resetTimeout:     30 * time.Second,
			state:            StateClosed,
		}

		if source.RateLimit > 0 {
			rateLimiters[source.URL] = rate.NewLimiter(source.RateLimit, 10)
		}
	}

	// Create retry configuration
	retryConfig := &RetryConfig{
		MaxRetries:    maxRetries,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}

	ras := &RemoteArtifactStore{
		sources:      sources,
		client:       client,
		breakers:     breakers,
		rateLimiters: rateLimiters,
		retryConfig:  retryConfig,
		metrics:      &RemoteStoreMetrics{},
	}

	return ras, nil
}

// FetchArtifact fetches an artifact from remote sources with fallback
func (ras *RemoteArtifactStore) FetchArtifact(hash [32]byte) (*Artifact, error) {
	hashStr := fmt.Sprintf("%x", hash)

	// Sort sources by priority (lower number = higher priority)
	sortedSources := make([]ArtifactSource, len(ras.sources))
	copy(sortedSources, ras.sources)

	// Simple sort by priority (in production, use sort.Slice)
	for i := 0; i < len(sortedSources); i++ {
		for j := i + 1; j < len(sortedSources); j++ {
			if sortedSources[i].Priority > sortedSources[j].Priority {
				sortedSources[i], sortedSources[j] = sortedSources[j], sortedSources[i]
			}
		}
	}

	// Try each source in priority order
	for _, source := range sortedSources {
		// Check circuit breaker
		if !ras.breakers[source.URL].CanExecute() {
			continue
		}

		// Check rate limit
		if limiter, exists := ras.rateLimiters[source.URL]; exists {
			if !limiter.Allow() {
				continue
			}
		}

		// Attempt to fetch from this source
		artifact, err := ras.fetchFromSource(source, hash)
		if err != nil {
			ras.breakers[source.URL].RecordFailure()
			continue
		}

		// Success - reset circuit breaker
		ras.breakers[source.URL].RecordSuccess()
		return artifact, nil
	}

	return nil, fmt.Errorf("artifact %s not found in any source", hashStr)
}

// fetchFromSource fetches an artifact from a specific source with retry logic
func (ras *RemoteArtifactStore) fetchFromSource(source ArtifactSource, hash [32]byte) (*Artifact, error) {
	hashStr := fmt.Sprintf("%x", hash)

	// Construct URL
	url := fmt.Sprintf("%s/artifacts/%s", source.URL, hashStr)

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if required
	if source.AuthRequired && ras.authProvider != nil {
		authHeader, err := ras.authProvider.GetAuthHeader(source.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to get auth header: %w", err)
		}
		req.Header.Set("Authorization", authHeader)
	}

	// Add headers
	req.Header.Set("User-Agent", "OCX-Artifact-Resolver/1.0")
	req.Header.Set("Accept", "application/octet-stream")

	// Retry logic
	var lastErr error
	delay := ras.retryConfig.BaseDelay

	for attempt := 0; attempt <= ras.retryConfig.MaxRetries; attempt++ {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), source.Timeout)

		// Execute request
		resp, err := ras.client.Do(req.WithContext(ctx))
		cancel()

		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if attempt < ras.retryConfig.MaxRetries {
				time.Sleep(delay)
				delay = ras.calculateNextDelay(delay)
			}
			continue
		}

		// Check response status
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			if attempt < ras.retryConfig.MaxRetries {
				time.Sleep(delay)
				delay = ras.calculateNextDelay(delay)
			}
			continue
		}

		// Success - create artifact
		artifact := &Artifact{
			Hash:       hash,
			Data:       resp.Body,
			Size:       resp.ContentLength,
			Source:     source.URL,
			ResolvedAt: time.Now(),
		}

		return artifact, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", ras.retryConfig.MaxRetries, lastErr)
}

// HealthCheck checks the health of all remote sources
func (ras *RemoteArtifactStore) HealthCheck() map[string]HealthStatus {
	status := make(map[string]HealthStatus)

	for _, source := range ras.sources {
		health := ras.checkSourceHealth(source)
		status[source.URL] = health
	}

	return status
}

// checkSourceHealth checks the health of a specific source
func (ras *RemoteArtifactStore) checkSourceHealth(source ArtifactSource) HealthStatus {
	// Check circuit breaker state
	breaker := ras.breakers[source.URL]
	breaker.mutex.RLock()
	state := breaker.state
	breaker.mutex.RUnlock()

	if state == StateOpen {
		return HealthStatus{
			Healthy: false,
			Reason:  "Circuit breaker open",
		}
	}

	// Perform health check request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/health", source.URL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return HealthStatus{
			Healthy: false,
			Reason:  fmt.Sprintf("Failed to create health check request: %v", err),
		}
	}

	resp, err := ras.client.Do(req)
	if err != nil {
		return HealthStatus{
			Healthy: false,
			Reason:  fmt.Sprintf("Health check request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return HealthStatus{
			Healthy: false,
			Reason:  fmt.Sprintf("Health check returned status %d", resp.StatusCode),
		}
	}

	return HealthStatus{
		Healthy: true,
		Reason:  "OK",
	}
}

// UpdateSourcePriorities updates source priorities based on performance
func (ras *RemoteArtifactStore) UpdateSourcePriorities() error {
	// Update source priorities based on performance metrics
	// Sources with higher success rates and lower latency get higher priority
	
	for i, source := range ras.sources {
		// Calculate priority based on metrics
		priority := ras.calculateSourcePriority(source)
		
		// Update source priority
		ras.sources[i].Priority = priority
		
		// Update metrics
		if ras.metrics.SourcePriorities == nil {
			ras.metrics.SourcePriorities = make(map[string]int)
		}
		ras.metrics.SourcePriorities[source.URL] = priority
	}
	
	// Sort sources by priority (highest first)
	ras.sortSourcesByPriority()
	
	return nil
}

// calculateSourcePriority calculates priority based on performance metrics
func (ras *RemoteArtifactStore) calculateSourcePriority(source ArtifactSource) int {
	// Base priority
	priority := 100
	
	// Adjust based on success rate
	if ras.metrics.TotalRequests > 0 {
		successRate := float64(ras.metrics.SuccessfulRequests) / float64(ras.metrics.TotalRequests)
		priority = int(float64(priority) * successRate)
	}
	
	// Adjust based on latency (lower latency = higher priority)
	if ras.metrics.AverageLatency > 0 {
		latencyPenalty := int(ras.metrics.AverageLatency / 10) // 10ms = 1 point penalty
		priority -= latencyPenalty
	}
	
	// Ensure priority is within bounds
	if priority < 1 {
		priority = 1
	}
	if priority > 1000 {
		priority = 1000
	}
	
	return priority
}

// sortSourcesByPriority sorts sources by priority (highest first)
func (ras *RemoteArtifactStore) sortSourcesByPriority() {
	// Simple bubble sort for small number of sources
	for i := 0; i < len(ras.sources)-1; i++ {
		for j := 0; j < len(ras.sources)-i-1; j++ {
			if ras.sources[j].Priority < ras.sources[j+1].Priority {
				ras.sources[j], ras.sources[j+1] = ras.sources[j+1], ras.sources[j]
			}
		}
	}
}

// Helper methods

func (ras *RemoteArtifactStore) calculateNextDelay(currentDelay time.Duration) time.Duration {
	nextDelay := time.Duration(float64(currentDelay) * ras.retryConfig.BackoffFactor)

	if nextDelay > ras.retryConfig.MaxDelay {
		nextDelay = ras.retryConfig.MaxDelay
	}

	if ras.retryConfig.Jitter {
		// Add jitter to prevent thundering herd
		jitter := time.Duration(float64(nextDelay) * 0.1)
		nextDelay += jitter
	}

	return nextDelay
}

// Circuit breaker methods

func (cb *CircuitBreaker) CanExecute() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if reset timeout has passed
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount = 0
	cb.state = StateClosed
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.failureThreshold {
		cb.state = StateOpen
	}
}

// HealthStatus represents the health status of a remote source
type HealthStatus struct {
	Healthy bool   `json:"healthy"`
	Reason  string `json:"reason"`
}
