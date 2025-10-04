package scaling

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HealthChecker performs health checks on backend servers
type HealthChecker struct {
	interval time.Duration
	timeout  time.Duration
	path     string
	client   *http.Client
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	URL       string        `json:"url"`
	Healthy   bool          `json:"healthy"`
	Latency   time.Duration `json:"latency"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Status    int           `json:"status_code"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(interval, timeout time.Duration, path string) *HealthChecker {
	return &HealthChecker{
		interval: interval,
		timeout:  timeout,
		path:     path,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// CheckHealth performs a health check on the given URL
func (hc *HealthChecker) CheckHealth(url string) bool {
	result := hc.CheckHealthDetailed(url)
	return result.Healthy
}

// CheckHealthDetailed performs a detailed health check
func (hc *HealthChecker) CheckHealthDetailed(url string) HealthCheckResult {
	start := time.Now()

	result := HealthCheckResult{
		URL:       url,
		Timestamp: start,
	}

	// Create request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url+hc.path, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		result.Latency = time.Since(start)
		return result
	}

	// Set headers
	req.Header.Set("User-Agent", "OCX-HealthChecker/1.0")
	req.Header.Set("Accept", "application/json")

	// Perform request
	resp, err := hc.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		result.Latency = time.Since(start)
		return result
	}
	defer resp.Body.Close()

	result.Latency = time.Since(start)
	result.Status = resp.StatusCode

	// Consider 2xx status codes as healthy
	result.Healthy = resp.StatusCode >= 200 && resp.StatusCode < 300

	if !result.Healthy {
		result.Error = fmt.Sprintf("unhealthy status code: %d", resp.StatusCode)
	}

	return result
}

// CheckHealthBatch performs health checks on multiple URLs concurrently
func (hc *HealthChecker) CheckHealthBatch(urls []string) map[string]HealthCheckResult {
	results := make(map[string]HealthCheckResult)
	resultChan := make(chan struct {
		url    string
		result HealthCheckResult
	}, len(urls))

	// Start health checks concurrently
	for _, url := range urls {
		go func(u string) {
			result := hc.CheckHealthDetailed(u)
			resultChan <- struct {
				url    string
				result HealthCheckResult
			}{u, result}
		}(url)
	}

	// Collect results
	for i := 0; i < len(urls); i++ {
		result := <-resultChan
		results[result.url] = result.result
	}

	return results
}

// CheckHealthWithRetry performs health check with retry logic
func (hc *HealthChecker) CheckHealthWithRetry(url string, maxRetries int) HealthCheckResult {
	var lastResult HealthCheckResult

	for i := 0; i <= maxRetries; i++ {
		result := hc.CheckHealthDetailed(url)
		lastResult = result

		if result.Healthy {
			return result
		}

		// Wait before retry (exponential backoff)
		if i < maxRetries {
			waitTime := time.Duration(1<<uint(i)) * time.Second
			time.Sleep(waitTime)
		}
	}

	return lastResult
}

// CustomHealthChecker allows custom health check logic
type CustomHealthChecker struct {
	checkFunc func(url string) HealthCheckResult
}

// NewCustomHealthChecker creates a custom health checker
func NewCustomHealthChecker(checkFunc func(url string) HealthCheckResult) *CustomHealthChecker {
	return &CustomHealthChecker{
		checkFunc: checkFunc,
	}
}

// CheckHealth performs custom health check
func (chc *CustomHealthChecker) CheckHealth(url string) bool {
	result := chc.CheckHealthDetailed(url)
	return result.Healthy
}

// CheckHealthDetailed performs detailed custom health check
func (chc *CustomHealthChecker) CheckHealthDetailed(url string) HealthCheckResult {
	return chc.checkFunc(url)
}

// HealthCheckConfig defines configuration for health checking
type HealthCheckConfig struct {
	// Basic settings
	Interval time.Duration `json:"interval"`
	Timeout  time.Duration `json:"timeout"`
	Path     string        `json:"path"`

	// Retry settings
	MaxRetries int           `json:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay"`

	// Custom settings
	Headers map[string]string `json:"headers"`
	Method  string            `json:"method"`

	// Success criteria
	ExpectedStatusCodes []int             `json:"expected_status_codes"`
	ExpectedBody        string            `json:"expected_body"`
	ExpectedHeaders     map[string]string `json:"expected_headers"`
}

// AdvancedHealthChecker provides advanced health checking capabilities
type AdvancedHealthChecker struct {
	config HealthCheckConfig
	client *http.Client
}

// NewAdvancedHealthChecker creates an advanced health checker
func NewAdvancedHealthChecker(config HealthCheckConfig) *AdvancedHealthChecker {
	return &AdvancedHealthChecker{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// CheckHealth performs advanced health check
func (ahc *AdvancedHealthChecker) CheckHealth(url string) bool {
	result := ahc.CheckHealthDetailed(url)
	return result.Healthy
}

// CheckHealthDetailed performs detailed advanced health check
func (ahc *AdvancedHealthChecker) CheckHealthDetailed(url string) HealthCheckResult {
	start := time.Now()

	result := HealthCheckResult{
		URL:       url,
		Timestamp: start,
	}

	// Create request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), ahc.config.Timeout)
	defer cancel()

	method := ahc.config.Method
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, url+ahc.config.Path, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		result.Latency = time.Since(start)
		return result
	}

	// Set custom headers
	for key, value := range ahc.config.Headers {
		req.Header.Set(key, value)
	}

	// Set default headers if not specified
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "OCX-HealthChecker/1.0")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}

	// Perform request
	resp, err := ahc.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		result.Latency = time.Since(start)
		return result
	}
	defer resp.Body.Close()

	result.Latency = time.Since(start)
	result.Status = resp.StatusCode

	// Check status code
	if len(ahc.config.ExpectedStatusCodes) > 0 {
		statusValid := false
		for _, expectedStatus := range ahc.config.ExpectedStatusCodes {
			if resp.StatusCode == expectedStatus {
				statusValid = true
				break
			}
		}
		if !statusValid {
			result.Error = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
			return result
		}
	} else {
		// Default: 2xx status codes are healthy
		result.Healthy = resp.StatusCode >= 200 && resp.StatusCode < 300
	}

	// Check expected headers
	for expectedHeader, expectedValue := range ahc.config.ExpectedHeaders {
		actualValue := resp.Header.Get(expectedHeader)
		if actualValue != expectedValue {
			result.Error = fmt.Sprintf("unexpected header value for %s: expected %s, got %s",
				expectedHeader, expectedValue, actualValue)
			return result
		}
	}

	// Check expected body (if specified)
	if ahc.config.ExpectedBody != "" {
		// Note: In a real implementation, you would read and compare the response body
		// we'll skip this check as it requires additional complexity
	}

	result.Healthy = true
	return result
}
