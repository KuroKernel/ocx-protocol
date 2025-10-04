package health

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// Check represents a health check
type Check struct {
	Name        string
	Description string
	CheckFunc   func(ctx context.Context) error
	Timeout     time.Duration
	Critical    bool
}

// HealthChecker manages health checks
type HealthChecker struct {
	checks []Check
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make([]Check, 0),
	}
}

// AddCheck adds a health check
func (hc *HealthChecker) AddCheck(check Check) {
	hc.checks = append(hc.checks, check)
}

// RunChecks runs all health checks
func (hc *HealthChecker) RunChecks(ctx context.Context) *HealthStatus {
	status := &HealthStatus{
		Overall:   "healthy",
		Timestamp: time.Now().UTC(),
		Checks:    make(map[string]CheckResult),
		Version:   "1.0.0",
		Uptime:    time.Since(startTime),
	}

	criticalFailures := 0
	totalFailures := 0

	for _, check := range hc.checks {
		result := hc.runCheck(ctx, check)
		status.Checks[check.Name] = result

		if result.Status != "healthy" {
			totalFailures++
			if check.Critical {
				criticalFailures++
			}
		}
	}

	// Determine overall status
	if criticalFailures > 0 {
		status.Overall = "unhealthy"
	} else if totalFailures > 0 {
		status.Overall = "degraded"
	}

	// Add system information
	status.System = getSystemInfo()

	return status
}

// runCheck runs a single health check
func (hc *HealthChecker) runCheck(ctx context.Context, check Check) CheckResult {
	timeout := check.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	err := check.CheckFunc(checkCtx)
	duration := time.Since(start)

	result := CheckResult{
		Name:        check.Name,
		Description: check.Description,
		Duration:    duration,
		Timestamp:   time.Now().UTC(),
	}

	if err != nil {
		result.Status = "unhealthy"
		result.Error = err.Error()
	} else {
		result.Status = "healthy"
	}

	return result
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Overall   string                 `json:"overall"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
	Version   string                 `json:"version"`
	Uptime    time.Duration          `json:"uptime"`
	System    SystemInfo             `json:"system"`
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Status      string        `json:"status"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time     `json:"timestamp"`
	Error       string        `json:"error,omitempty"`
}

// SystemInfo represents system information
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumGoroutine int    `json:"num_goroutines"`
	NumCPU       int    `json:"num_cpu"`
	MemoryStats  struct {
		Alloc      uint64 `json:"alloc"`
		TotalAlloc uint64 `json:"total_alloc"`
		Sys        uint64 `json:"sys"`
		NumGC      uint32 `json:"num_gc"`
	} `json:"memory_stats"`
}

// getSystemInfo collects system information
func getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		MemoryStats: struct {
			Alloc      uint64 `json:"alloc"`
			TotalAlloc uint64 `json:"total_alloc"`
			Sys        uint64 `json:"sys"`
			NumGC      uint32 `json:"num_gc"`
		}{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
		},
	}
}

// Common health checks

// DatabaseCheck creates a database health check
func DatabaseCheck(checkFunc func(ctx context.Context) error) Check {
	return Check{
		Name:        "database",
		Description: "Database connectivity and basic operations",
		CheckFunc:   checkFunc,
		Timeout:     5 * time.Second,
		Critical:    true,
	}
}

// KeystoreCheck creates a keystore health check
func KeystoreCheck(checkFunc func(ctx context.Context) error) Check {
	return Check{
		Name:        "keystore",
		Description: "Keystore availability and key access",
		CheckFunc:   checkFunc,
		Timeout:     2 * time.Second,
		Critical:    true,
	}
}

// VerifierCheck creates a verifier health check
func VerifierCheck(checkFunc func(ctx context.Context) error) Check {
	return Check{
		Name:        "verifier",
		Description: "Receipt verifier availability and functionality",
		CheckFunc:   checkFunc,
		Timeout:     3 * time.Second,
		Critical:    true,
	}
}

// DVMCheck creates a deterministic VM health check
func DVMCheck(checkFunc func(ctx context.Context) error) Check {
	return Check{
		Name:        "deterministic_vm",
		Description: "Deterministic VM execution engine",
		CheckFunc:   checkFunc,
		Timeout:     5 * time.Second,
		Critical:    false,
	}
}

// SystemCheck creates a system resource health check
func SystemCheck() Check {
	return Check{
		Name:        "system",
		Description: "System resources and performance",
		CheckFunc: func(ctx context.Context) error {
			// Check memory usage
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Alert if memory usage is very high (> 1GB)
			if m.Alloc > 1024*1024*1024 {
				return fmt.Errorf("high memory usage: %d bytes", m.Alloc)
			}

			// Check goroutine count
			if runtime.NumGoroutine() > 1000 {
				return fmt.Errorf("high goroutine count: %d", runtime.NumGoroutine())
			}

			return nil
		},
		Timeout:  1 * time.Second,
		Critical: false,
	}
}

var startTime = time.Now()
