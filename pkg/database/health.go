package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthChecker provides comprehensive database health monitoring
type HealthChecker struct {
	pool   *pgxpool.Pool
	logger Logger
}

// Logger interface for health check logging
type Logger interface {
	Error(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
}

// NewHealthChecker creates a health checker for database monitoring
func NewHealthChecker(pool *pgxpool.Pool, logger Logger) *HealthChecker {
	return &HealthChecker{
		pool:   pool,
		logger: logger,
	}
}

// HealthStatus represents database health state
type HealthStatus struct {
	Healthy       bool          `json:"healthy"`
	ConnectionsOK bool          `json:"connections_ok"`
	QueryLatency  time.Duration `json:"query_latency_ms"`
	ActiveConns   int32         `json:"active_connections"`
	IdleConns     int32         `json:"idle_connections"`
	MaxConns      int32         `json:"max_connections"`
	ErrorMessage  string        `json:"error,omitempty"`
	CheckedAt     time.Time     `json:"checked_at"`
}

// Check performs comprehensive database health check
func (hc *HealthChecker) Check(ctx context.Context) HealthStatus {
	start := time.Now()
	status := HealthStatus{
		CheckedAt: start,
		Healthy:   false,
	}

	// Check 1: Connection pool stats
	if hc.pool != nil {
		stats := hc.pool.Stat()
		status.ActiveConns = stats.AcquiredConns()
		status.IdleConns = stats.IdleConns()
		status.MaxConns = stats.MaxConns()

		// Connection pool exhaustion check
		if status.ActiveConns >= status.MaxConns {
			status.ErrorMessage = "connection pool exhausted"
			return status
		}
		status.ConnectionsOK = true
	}

	// Check 2: Query execution
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	startQuery := time.Now()
	err := hc.pool.Ping(queryCtx)
	status.QueryLatency = time.Since(startQuery)

	if err != nil {
		status.ErrorMessage = fmt.Sprintf("query failed: %v", err)
		if hc.logger != nil {
			hc.logger.Error("Database health check failed",
				"error", err,
				"latency", status.QueryLatency)
		}
		return status
	}

	// Check 3: Latency threshold (warn if >100ms)
	if status.QueryLatency > 100*time.Millisecond {
		if hc.logger != nil {
			hc.logger.Warn("Database latency high",
				"latency", status.QueryLatency)
		}
	}

	// All checks passed
	status.Healthy = true
	return status
}

// CheckWithRetry performs health check with retry logic
func (hc *HealthChecker) CheckWithRetry(ctx context.Context, maxRetries int) HealthStatus {
	var status HealthStatus

	for attempt := 0; attempt < maxRetries; attempt++ {
		status = hc.Check(ctx)
		if status.Healthy {
			return status
		}

		if attempt < maxRetries-1 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond
			select {
			case <-ctx.Done():
				status.ErrorMessage = "health check cancelled"
				return status
			case <-time.After(backoff):
				continue
			}
		}
	}

	return status
}

// IsReady checks if database is ready for traffic
func (hc *HealthChecker) IsReady(ctx context.Context) bool {
	status := hc.Check(ctx)
	return status.Healthy &&
		status.ConnectionsOK &&
		status.QueryLatency < 200*time.Millisecond
}

// IsAlive checks if database connection is alive
func (hc *HealthChecker) IsAlive(ctx context.Context) bool {
	status := hc.Check(ctx)
	return status.Healthy
}
