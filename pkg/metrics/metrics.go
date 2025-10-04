package metrics

import (
	"fmt"
	"sync"
	"time"
)

// SimpleMetrics provides a simplified metrics implementation for Go 1.18
type SimpleMetrics struct {
	mu sync.RWMutex

	// HTTP Request Metrics
	requestsTotal   map[string]int64
	requestDuration map[string][]time.Duration
	requestSize     map[string][]int64
	responseSize    map[string][]int64

	// OCX Protocol Specific Metrics
	executeTotal      map[string]int64
	executeDuration   map[string][]time.Duration
	executeGasUsed    map[string][]uint64
	executeMemoryUsed map[string][]uint64

	// Verification Metrics
	verifyTotal    map[string]int64
	verifyDuration map[string][]time.Duration
	verifySuccess  map[string]int64
	verifyFailure  map[string]int64

	// Receipt Metrics
	receiptsStored    map[string]int64
	receiptsRetrieved map[string]int64
	receiptSize       map[string][]int64

	// Security Metrics
	authFailures    map[string]int64
	rateLimitHits   map[string]int64
	idempotencyHits map[string]int64

	// System Metrics
	activeConnections int64
	memoryUsage       float64
	cpuUsage          float64
}

// NewMetrics creates a new simplified metrics instance
func NewMetrics() *SimpleMetrics {
	return &SimpleMetrics{
		requestsTotal:     make(map[string]int64),
		requestDuration:   make(map[string][]time.Duration),
		requestSize:       make(map[string][]int64),
		responseSize:      make(map[string][]int64),
		executeTotal:      make(map[string]int64),
		executeDuration:   make(map[string][]time.Duration),
		executeGasUsed:    make(map[string][]uint64),
		executeMemoryUsed: make(map[string][]uint64),
		verifyTotal:       make(map[string]int64),
		verifyDuration:    make(map[string][]time.Duration),
		verifySuccess:     make(map[string]int64),
		verifyFailure:     make(map[string]int64),
		receiptsStored:    make(map[string]int64),
		receiptsRetrieved: make(map[string]int64),
		receiptSize:       make(map[string][]int64),
		authFailures:      make(map[string]int64),
		rateLimitHits:     make(map[string]int64),
		idempotencyHits:   make(map[string]int64),
	}
}

// RecordRequest records HTTP request metrics
func (m *SimpleMetrics) RecordRequest(method, route, statusCode string, duration time.Duration, requestSize, responseSize int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s_%s_%s", method, route, statusCode)
	m.requestsTotal[key]++

	durationKey := fmt.Sprintf("%s_%s", method, route)
	m.requestDuration[durationKey] = append(m.requestDuration[durationKey], duration)
	m.requestSize[durationKey] = append(m.requestSize[durationKey], requestSize)
	m.responseSize[durationKey] = append(m.responseSize[durationKey], responseSize)
}

// RecordExecute records artifact execution metrics
func (m *SimpleMetrics) RecordExecute(issuerID, status string, duration time.Duration, gasUsed, memoryUsed uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s_%s", status, issuerID)
	m.executeTotal[key]++

	m.executeDuration[issuerID] = append(m.executeDuration[issuerID], duration)
	m.executeGasUsed[issuerID] = append(m.executeGasUsed[issuerID], gasUsed)
	m.executeMemoryUsed[issuerID] = append(m.executeMemoryUsed[issuerID], memoryUsed)
}

// RecordVerify records receipt verification metrics
func (m *SimpleMetrics) RecordVerify(verifierType, status, reason string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s_%s", status, verifierType)
	m.verifyTotal[key]++

	m.verifyDuration[verifierType] = append(m.verifyDuration[verifierType], duration)

	if status == "success" {
		m.verifySuccess[verifierType]++
	} else {
		failureKey := fmt.Sprintf("%s_%s", verifierType, reason)
		m.verifyFailure[failureKey]++
	}
}

// RecordReceipt records receipt storage/retrieval metrics
func (m *SimpleMetrics) RecordReceipt(issuerID, operation, status string, size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if operation == "store" {
		key := fmt.Sprintf("%s_%s", issuerID, status)
		m.receiptsStored[key]++
	} else if operation == "retrieve" {
		key := fmt.Sprintf("%s_%s", issuerID, status)
		m.receiptsRetrieved[key]++
	}

	if size > 0 {
		m.receiptSize[issuerID] = append(m.receiptSize[issuerID], size)
	}
}

// RecordDBQuery records database query metrics
func (m *SimpleMetrics) RecordDBQuery(operation, table, status string, duration time.Duration) {
	// Simplified implementation - just log for now
	fmt.Printf("DB Query: %s on %s - %s (took %v)\n", operation, table, status, duration)
}

// RecordAuthFailure records authentication failure metrics
func (m *SimpleMetrics) RecordAuthFailure(reason, apiKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s_%s", reason, apiKey)
	m.authFailures[key]++
}

// RecordRateLimit records rate limit hit metrics
func (m *SimpleMetrics) RecordRateLimit(limitType, identifier string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s_%s", limitType, identifier)
	m.rateLimitHits[key]++
}

// RecordIdempotency records idempotency hit metrics
func (m *SimpleMetrics) RecordIdempotency(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.idempotencyHits[status]++
}

// UpdateSystemMetrics updates system-level metrics
func (m *SimpleMetrics) UpdateSystemMetrics(activeConnections int, memoryUsage, cpuUsage float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.activeConnections = int64(activeConnections)
	m.memoryUsage = memoryUsage
	m.cpuUsage = cpuUsage
}

// UpdateDBConnections updates database connection metrics
func (m *SimpleMetrics) UpdateDBConnections(active, idle, total int) {
	// Simplified implementation - just log for now
	fmt.Printf("DB Connections: active=%d, idle=%d, total=%d\n", active, idle, total)
}

// GetStats returns current metrics statistics
func (m *SimpleMetrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["requests_total"] = m.requestsTotal
	stats["execute_total"] = m.executeTotal
	stats["verify_total"] = m.verifyTotal
	stats["receipts_stored"] = m.receiptsStored
	stats["receipts_retrieved"] = m.receiptsRetrieved
	stats["auth_failures"] = m.authFailures
	stats["rate_limit_hits"] = m.rateLimitHits
	stats["idempotency_hits"] = m.idempotencyHits
	stats["active_connections"] = m.activeConnections
	stats["memory_usage"] = m.memoryUsage
	stats["cpu_usage"] = m.cpuUsage

	return stats
}
