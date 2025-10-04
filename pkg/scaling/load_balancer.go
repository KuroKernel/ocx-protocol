package scaling

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// LoadBalancerConfig defines configuration for load balancing
type LoadBalancerConfig struct {
	// Algorithm selection
	Algorithm string `json:"algorithm"` // "round_robin", "least_connections", "weighted_round_robin", "ip_hash"

	// Health check settings
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
	HealthCheckPath     string        `json:"health_check_path"`

	// Backend configuration
	Backends []BackendConfig `json:"backends"`

	// Circuit breaker settings
	CircuitBreakerEnabled bool          `json:"circuit_breaker_enabled"`
	FailureThreshold      int           `json:"failure_threshold"`
	RecoveryTimeout       time.Duration `json:"recovery_timeout"`

	// Sticky session settings
	StickySessionEnabled bool          `json:"sticky_session_enabled"`
	SessionCookieName    string        `json:"session_cookie_name"`
	SessionTTL           time.Duration `json:"session_ttl"`
}

// BackendConfig defines a backend server configuration
type BackendConfig struct {
	ID       string  `json:"id"`
	URL      string  `json:"url"`
	Weight   int     `json:"weight"`
	Priority int     `json:"priority"`
	Enabled  bool    `json:"enabled"`
	Health   string  `json:"health"`  // "healthy", "unhealthy", "unknown"
	Latency  float64 `json:"latency"` // in milliseconds
}

// LoadBalancer manages load balancing across multiple backends
type LoadBalancer struct {
	config    LoadBalancerConfig
	backends  map[string]*Backend
	algorithm LoadBalancingAlgorithm
	mu        sync.RWMutex

	// Health monitoring
	healthChecker *HealthChecker
	ctx           context.Context
	cancel        context.CancelFunc

	// Session management
	sessions  map[string]*StickySession
	sessionMu sync.RWMutex
}

// Backend represents a backend server
type Backend struct {
	config BackendConfig
	stats  BackendStats
	mu     sync.RWMutex
}

// BackendStats tracks backend performance metrics
type BackendStats struct {
	TotalRequests   int64               `json:"total_requests"`
	ActiveRequests  int64               `json:"active_requests"`
	FailedRequests  int64               `json:"failed_requests"`
	AverageLatency  time.Duration       `json:"average_latency"`
	LastHealthCheck time.Time           `json:"last_health_check"`
	CircuitBreaker  CircuitBreakerState `json:"circuit_breaker"`
}

// CircuitBreakerState tracks circuit breaker status
type CircuitBreakerState struct {
	State        string    `json:"state"` // "closed", "open", "half_open"
	FailureCount int       `json:"failure_count"`
	LastFailure  time.Time `json:"last_failure"`
	NextAttempt  time.Time `json:"next_attempt"`
}

// StickySession represents a sticky session for load balancing
type StickySession struct {
	BackendID string        `json:"backend_id"`
	CreatedAt time.Time     `json:"created_at"`
	LastUsed  time.Time     `json:"last_used"`
	TTL       time.Duration `json:"ttl"`
}

// LoadBalancingAlgorithm interface for different load balancing strategies
type LoadBalancingAlgorithm interface {
	SelectBackend(backends map[string]*Backend, sessionID string) (*Backend, error)
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(config LoadBalancerConfig) (*LoadBalancer, error) {
	ctx, cancel := context.WithCancel(context.Background())

	lb := &LoadBalancer{
		config:   config,
		backends: make(map[string]*Backend),
		sessions: make(map[string]*StickySession),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Initialize backends
	for _, backendConfig := range config.Backends {
		backend := &Backend{
			config: backendConfig,
			stats: BackendStats{
				CircuitBreaker: CircuitBreakerState{
					State: "closed",
				},
			},
		}
		lb.backends[backendConfig.ID] = backend
	}

	// Initialize load balancing algorithm
	switch config.Algorithm {
	case "round_robin":
		lb.algorithm = &RoundRobinAlgorithm{}
	case "least_connections":
		lb.algorithm = &LeastConnectionsAlgorithm{}
	case "weighted_round_robin":
		lb.algorithm = &WeightedRoundRobinAlgorithm{}
	case "ip_hash":
		lb.algorithm = &IPHashAlgorithm{}
	default:
		lb.algorithm = &RoundRobinAlgorithm{}
	}

	// Initialize health checker
	lb.healthChecker = NewHealthChecker(config.HealthCheckInterval, config.HealthCheckTimeout, config.HealthCheckPath)

	// Start health checking
	go lb.startHealthChecking()

	// Start session cleanup
	go lb.startSessionCleanup()

	return lb, nil
}

// SelectBackend selects a backend using the configured algorithm
func (lb *LoadBalancer) SelectBackend(r *http.Request) (*Backend, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Check for sticky session
	var sessionID string
	if lb.config.StickySessionEnabled {
		sessionID = lb.getSessionID(r)
		if sessionID != "" {
			if session, exists := lb.sessions[sessionID]; exists {
				if backend, exists := lb.backends[session.BackendID]; exists && lb.isBackendHealthy(backend) {
					// Update session last used time
					lb.updateSessionLastUsed(sessionID)
					return backend, nil
				}
			}
		}
	}

	// Filter healthy backends
	healthyBackends := make(map[string]*Backend)
	for id, backend := range lb.backends {
		if lb.isBackendHealthy(backend) {
			healthyBackends[id] = backend
		}
	}

	if len(healthyBackends) == 0 {
		return nil, fmt.Errorf("no healthy backends available")
	}

	// Select backend using algorithm
	selectedBackend, err := lb.algorithm.SelectBackend(healthyBackends, sessionID)
	if err != nil {
		return nil, err
	}

	// Create or update session if sticky sessions are enabled
	if lb.config.StickySessionEnabled && sessionID != "" {
		lb.createOrUpdateSession(sessionID, selectedBackend.config.ID)
	}

	return selectedBackend, nil
}

// RecordRequest records a request to a backend
func (lb *LoadBalancer) RecordRequest(backendID string, success bool, latency time.Duration) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	backend, exists := lb.backends[backendID]
	if !exists {
		return
	}

	backend.mu.Lock()
	defer backend.mu.Unlock()

	backend.stats.TotalRequests++
	if success {
		backend.stats.ActiveRequests++
		// Update average latency
		if backend.stats.AverageLatency == 0 {
			backend.stats.AverageLatency = latency
		} else {
			backend.stats.AverageLatency = (backend.stats.AverageLatency + latency) / 2
		}
	} else {
		backend.stats.FailedRequests++
		lb.updateCircuitBreaker(backend, false)
	}
}

// GetBackendStats returns statistics for all backends
func (lb *LoadBalancer) GetBackendStats() map[string]BackendStats {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	stats := make(map[string]BackendStats)
	for id, backend := range lb.backends {
		backend.mu.RLock()
		stats[id] = backend.stats
		backend.mu.RUnlock()
	}
	return stats
}

// AddBackend adds a new backend
func (lb *LoadBalancer) AddBackend(config BackendConfig) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	backend := &Backend{
		config: config,
		stats: BackendStats{
			CircuitBreaker: CircuitBreakerState{
				State: "closed",
			},
		},
	}
	lb.backends[config.ID] = backend
}

// RemoveBackend removes a backend
func (lb *LoadBalancer) RemoveBackend(backendID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	delete(lb.backends, backendID)
}

// UpdateBackend updates backend configuration
func (lb *LoadBalancer) UpdateBackend(backendID string, config BackendConfig) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if backend, exists := lb.backends[backendID]; exists {
		backend.config = config
	}
}

// Close shuts down the load balancer
func (lb *LoadBalancer) Close() {
	lb.cancel()
}

// Helper methods

func (lb *LoadBalancer) isBackendHealthy(backend *Backend) bool {
	backend.mu.RLock()
	defer backend.mu.RUnlock()

	if !backend.config.Enabled {
		return false
	}

	// Check circuit breaker
	if lb.config.CircuitBreakerEnabled {
		state := backend.stats.CircuitBreaker.State
		if state == "open" {
			// Check if recovery timeout has passed
			if time.Since(backend.stats.CircuitBreaker.LastFailure) < lb.config.RecoveryTimeout {
				return false
			}
			// Move to half-open state
			backend.stats.CircuitBreaker.State = "half_open"
		}
	}

	return backend.config.Health == "healthy"
}

func (lb *LoadBalancer) getSessionID(r *http.Request) string {
	if cookie, err := r.Cookie(lb.config.SessionCookieName); err == nil {
		return cookie.Value
	}
	return ""
}

func (lb *LoadBalancer) createOrUpdateSession(sessionID, backendID string) {
	lb.sessionMu.Lock()
	defer lb.sessionMu.Unlock()

	lb.sessions[sessionID] = &StickySession{
		BackendID: backendID,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
		TTL:       lb.config.SessionTTL,
	}
}

func (lb *LoadBalancer) updateSessionLastUsed(sessionID string) {
	lb.sessionMu.Lock()
	defer lb.sessionMu.Unlock()

	if session, exists := lb.sessions[sessionID]; exists {
		session.LastUsed = time.Now()
	}
}

func (lb *LoadBalancer) updateCircuitBreaker(backend *Backend, success bool) {
	if !lb.config.CircuitBreakerEnabled {
		return
	}

	cb := &backend.stats.CircuitBreaker

	if success {
		// Reset circuit breaker on success
		cb.State = "closed"
		cb.FailureCount = 0
	} else {
		cb.FailureCount++
		cb.LastFailure = time.Now()

		if cb.FailureCount >= lb.config.FailureThreshold {
			cb.State = "open"
			cb.NextAttempt = time.Now().Add(lb.config.RecoveryTimeout)
		}
	}
}

func (lb *LoadBalancer) startHealthChecking() {
	ticker := time.NewTicker(lb.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-lb.ctx.Done():
			return
		case <-ticker.C:
			lb.performHealthChecks()
		}
	}
}

func (lb *LoadBalancer) performHealthChecks() {
	lb.mu.RLock()
	backends := make(map[string]*Backend)
	for id, backend := range lb.backends {
		backends[id] = backend
	}
	lb.mu.RUnlock()

	for id, backend := range backends {
		go func(backendID string, b *Backend) {
			healthy := lb.healthChecker.CheckHealth(b.config.URL)

			lb.mu.Lock()
			if backend, exists := lb.backends[backendID]; exists {
				backend.mu.Lock()
				if healthy {
					backend.config.Health = "healthy"
				} else {
					backend.config.Health = "unhealthy"
				}
				backend.stats.LastHealthCheck = time.Now()
				backend.mu.Unlock()
			}
			lb.mu.Unlock()
		}(id, backend)
	}
}

func (lb *LoadBalancer) startSessionCleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-lb.ctx.Done():
			return
		case <-ticker.C:
			lb.cleanupExpiredSessions()
		}
	}
}

func (lb *LoadBalancer) cleanupExpiredSessions() {
	lb.sessionMu.Lock()
	defer lb.sessionMu.Unlock()

	now := time.Now()
	for sessionID, session := range lb.sessions {
		if now.Sub(session.LastUsed) > session.TTL {
			delete(lb.sessions, sessionID)
		}
	}
}
