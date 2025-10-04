package scaling

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// RoundRobinAlgorithm implements round-robin load balancing
type RoundRobinAlgorithm struct {
	current int
	mu      sync.Mutex
}

// SelectBackend selects the next backend in round-robin fashion
func (rr *RoundRobinAlgorithm) SelectBackend(backends map[string]*Backend, sessionID string) (*Backend, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	backendList := make([]*Backend, 0, len(backends))
	for _, backend := range backends {
		backendList = append(backendList, backend)
	}

	if len(backendList) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	selected := backendList[rr.current%len(backendList)]
	rr.current++
	return selected, nil
}

// LeastConnectionsAlgorithm implements least connections load balancing
type LeastConnectionsAlgorithm struct{}

// SelectBackend selects the backend with the least active connections
func (lc *LeastConnectionsAlgorithm) SelectBackend(backends map[string]*Backend, sessionID string) (*Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	var selected *Backend
	minConnections := int64(^uint64(0) >> 1) // Max int64

	for _, backend := range backends {
		backend.mu.RLock()
		activeConnections := backend.stats.ActiveRequests
		backend.mu.RUnlock()

		if activeConnections < minConnections {
			minConnections = activeConnections
			selected = backend
		}
	}

	if selected == nil {
		return nil, fmt.Errorf("no backends available")
	}

	return selected, nil
}

// WeightedRoundRobinAlgorithm implements weighted round-robin load balancing
type WeightedRoundRobinAlgorithm struct {
	current int
	weights []int
	mu      sync.Mutex
}

// SelectBackend selects backend based on weights
func (wrr *WeightedRoundRobinAlgorithm) SelectBackend(backends map[string]*Backend, sessionID string) (*Backend, error) {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	// Convert map to sorted slice for consistent ordering
	type backendWeight struct {
		backend *Backend
		weight  int
	}

	backendWeights := make([]backendWeight, 0, len(backends))
	for _, backend := range backends {
		backendWeights = append(backendWeights, backendWeight{
			backend: backend,
			weight:  backend.config.Weight,
		})
	}

	// Sort by weight (descending)
	sort.Slice(backendWeights, func(i, j int) bool {
		return backendWeights[i].weight > backendWeights[j].weight
	})

	if len(backendWeights) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	// Calculate total weight
	totalWeight := 0
	for _, bw := range backendWeights {
		totalWeight += bw.weight
	}

	if totalWeight == 0 {
		// If all weights are 0, fall back to round-robin
		selected := backendWeights[wrr.current%len(backendWeights)]
		wrr.current++
		return selected.backend, nil
	}

	// Weighted selection
	currentWeight := wrr.current % totalWeight
	accumulatedWeight := 0

	for _, bw := range backendWeights {
		accumulatedWeight += bw.weight
		if currentWeight < accumulatedWeight {
			wrr.current++
			return bw.backend, nil
		}
	}

	// Fallback (should not reach here)
	return backendWeights[0].backend, nil
}

// IPHashAlgorithm implements IP hash load balancing
type IPHashAlgorithm struct{}

// SelectBackend selects backend based on client IP hash
func (ih *IPHashAlgorithm) SelectBackend(backends map[string]*Backend, sessionID string) (*Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	// Use sessionID as hash input (could be IP or session identifier)
	hashInput := sessionID
	if hashInput == "" {
		// Fallback to random selection if no session ID
		rand.Seed(time.Now().UnixNano())
		hashInput = fmt.Sprintf("%d", rand.Int())
	}

	// Calculate hash
	hash := md5.Sum([]byte(hashInput))
	hashValue := int(hash[0])<<24 | int(hash[1])<<16 | int(hash[2])<<8 | int(hash[3])

	// Convert map to slice for consistent ordering
	backendList := make([]*Backend, 0, len(backends))
	for _, backend := range backends {
		backendList = append(backendList, backend)
	}

	// Sort for consistent ordering
	sort.Slice(backendList, func(i, j int) bool {
		return backendList[i].config.ID < backendList[j].config.ID
	})

	// Select backend based on hash
	index := hashValue % len(backendList)
	if index < 0 {
		index = -index
	}

	return backendList[index], nil
}

// LatencyBasedAlgorithm implements latency-based load balancing
type LatencyBasedAlgorithm struct {
	latencyWindow time.Duration
}

// NewLatencyBasedAlgorithm creates a new latency-based algorithm
func NewLatencyBasedAlgorithm(latencyWindow time.Duration) *LatencyBasedAlgorithm {
	return &LatencyBasedAlgorithm{
		latencyWindow: latencyWindow,
	}
}

// SelectBackend selects backend with lowest average latency
func (lb *LatencyBasedAlgorithm) SelectBackend(backends map[string]*Backend, sessionID string) (*Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	var selected *Backend
	minLatency := time.Duration(^uint64(0) >> 1) // Max duration

	for _, backend := range backends {
		backend.mu.RLock()
		latency := backend.stats.AverageLatency
		backend.mu.RUnlock()

		// Only consider recent latency measurements
		if latency > 0 && latency < minLatency {
			minLatency = latency
			selected = backend
		}
	}

	// If no latency data available, fall back to round-robin
	if selected == nil {
		backendList := make([]*Backend, 0, len(backends))
		for _, backend := range backends {
			backendList = append(backendList, backend)
		}
		if len(backendList) > 0 {
			selected = backendList[rand.Intn(len(backendList))]
		}
	}

	if selected == nil {
		return nil, fmt.Errorf("no backends available")
	}

	return selected, nil
}

// PriorityBasedAlgorithm implements priority-based load balancing
type PriorityBasedAlgorithm struct{}

// SelectBackend selects backend with highest priority
func (pb *PriorityBasedAlgorithm) SelectBackend(backends map[string]*Backend, sessionID string) (*Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	var selected *Backend
	highestPriority := -1

	for _, backend := range backends {
		if backend.config.Priority > highestPriority {
			highestPriority = backend.config.Priority
			selected = backend
		}
	}

	if selected == nil {
		return nil, fmt.Errorf("no backends available")
	}

	return selected, nil
}

// RandomAlgorithm implements random load balancing
type RandomAlgorithm struct{}

// SelectBackend randomly selects a backend
func (r *RandomAlgorithm) SelectBackend(backends map[string]*Backend, sessionID string) (*Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	backendList := make([]*Backend, 0, len(backends))
	for _, backend := range backends {
		backendList = append(backendList, backend)
	}

	rand.Seed(time.Now().UnixNano())
	selected := backendList[rand.Intn(len(backendList))]
	return selected, nil
}
