package performance

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// PerformanceTester provides performance testing capabilities
type PerformanceTester struct {
	// Configuration
	config TestingConfig

	// Test scenarios
	scenarios map[string]*TestScenario

	// Results storage
	results map[string]*TestResult

	// Control
	ctx    context.Context
	cancel context.CancelFunc
}

// TestingConfig defines configuration for performance testing
type TestingConfig struct {
	// Test execution settings
	DefaultDuration     time.Duration `json:"default_duration"`
	DefaultConcurrency  int           `json:"default_concurrency"`
	DefaultWarmupTime   time.Duration `json:"default_warmup_time"`
	DefaultCooldownTime time.Duration `json:"default_cooldown_time"`

	// Test data settings
	TestDataSize      int `json:"test_data_size"`
	TestDataVariation int `json:"test_data_variation"`

	// Performance thresholds
	MaxLatencyP95 time.Duration `json:"max_latency_p95"`
	MaxLatencyP99 time.Duration `json:"max_latency_p99"`
	MinThroughput int64         `json:"min_throughput"`
	MaxErrorRate  float64       `json:"max_error_rate"`

	// Test reporting
	EnableDetailedReporting bool `json:"enable_detailed_reporting"`
	EnableRealTimeMetrics   bool `json:"enable_real_time_metrics"`
}

// TestScenario represents a performance test scenario
type TestScenario struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Config           BenchmarkConfig        `json:"config"`
	TestFunction     TestFunction           `json:"-"`
	SetupFunction    SetupFunction          `json:"-"`
	TeardownFunction TeardownFunction       `json:"-"`
	Data             map[string]interface{} `json:"data"`
}

// TestFunction represents a function to test
type TestFunction func(ctx context.Context, data map[string]interface{}) error

// SetupFunction represents a setup function for a test
type SetupFunction func(ctx context.Context) (map[string]interface{}, error)

// TeardownFunction represents a teardown function for a test
type TeardownFunction func(ctx context.Context, data map[string]interface{}) error

// TestResult represents the result of a performance test
type TestResult struct {
	ScenarioName    string           `json:"scenario_name"`
	StartTime       time.Time        `json:"start_time"`
	EndTime         time.Time        `json:"end_time"`
	Duration        time.Duration    `json:"duration"`
	Success         bool             `json:"success"`
	ErrorMessage    string           `json:"error_message,omitempty"`
	BenchmarkResult *BenchmarkResult `json:"benchmark_result,omitempty"`
	Passed          bool             `json:"passed"`
	Failures        []TestFailure    `json:"failures,omitempty"`
}

// TestFailure represents a test failure
type TestFailure struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Expected    string `json:"expected"`
	Actual      string `json:"actual"`
	Threshold   string `json:"threshold"`
}

// NewPerformanceTester creates a new performance tester
func NewPerformanceTester(config TestingConfig) *PerformanceTester {
	ctx, cancel := context.WithCancel(context.Background())

	return &PerformanceTester{
		config:    config,
		scenarios: make(map[string]*TestScenario),
		results:   make(map[string]*TestResult),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// AddScenario adds a test scenario
func (pt *PerformanceTester) AddScenario(scenario *TestScenario) {
	pt.scenarios[scenario.Name] = scenario
}

// RunScenario runs a specific test scenario
func (pt *PerformanceTester) RunScenario(ctx context.Context, scenarioName string) (*TestResult, error) {
	scenario, exists := pt.scenarios[scenarioName]
	if !exists {
		return nil, fmt.Errorf("scenario %s not found", scenarioName)
	}

	result := &TestResult{
		ScenarioName: scenarioName,
		StartTime:    time.Now(),
	}

	// Setup
	var setupData map[string]interface{}
	if scenario.SetupFunction != nil {
		var err error
		setupData, err = scenario.SetupFunction(ctx)
		if err != nil {
			result.Success = false
			result.ErrorMessage = fmt.Sprintf("setup failed: %v", err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			pt.results[scenarioName] = result
			return result, err
		}
	}

	// Merge setup data with scenario data
	testData := make(map[string]interface{})
	for k, v := range scenario.Data {
		testData[k] = v
	}
	for k, v := range setupData {
		testData[k] = v
	}

	// Create benchmark runner
	runner := NewBenchmarkRunner(scenario.Config)

	// Run benchmark
	benchmarkResult, err := runner.Run(ctx, func(ctx context.Context) error {
		return scenario.TestFunction(ctx, testData)
	})

	if err != nil {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("benchmark failed: %v", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		pt.results[scenarioName] = result
		return result, err
	}

	// Teardown
	if scenario.TeardownFunction != nil {
		if err := scenario.TeardownFunction(ctx, testData); err != nil {
			// Log teardown error but don't fail the test
			fmt.Printf("Teardown failed for scenario %s: %v\n", scenarioName, err)
		}
	}

	// Evaluate results
	result.BenchmarkResult = benchmarkResult
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = true
	result.Passed = pt.evaluateTestResult(benchmarkResult, &result.Failures)

	pt.results[scenarioName] = result
	return result, nil
}

// RunAllScenarios runs all test scenarios
func (pt *PerformanceTester) RunAllScenarios(ctx context.Context) (map[string]*TestResult, error) {
	results := make(map[string]*TestResult)

	for scenarioName := range pt.scenarios {
		result, err := pt.RunScenario(ctx, scenarioName)
		if err != nil {
			return results, fmt.Errorf("scenario %s failed: %w", scenarioName, err)
		}
		results[scenarioName] = result
	}

	return results, nil
}

// evaluateTestResult evaluates if a test result passes the performance criteria
func (pt *PerformanceTester) evaluateTestResult(result *BenchmarkResult, failures *[]TestFailure) bool {
	passed := true

	// Check latency P95
	if result.LatencyStats.P95 > pt.config.MaxLatencyP95 {
		*failures = append(*failures, TestFailure{
			Type:        "latency_p95",
			Description: "P95 latency exceeds maximum allowed",
			Expected:    pt.config.MaxLatencyP95.String(),
			Actual:      result.LatencyStats.P95.String(),
			Threshold:   "maximum",
		})
		passed = false
	}

	// Check latency P99
	if result.LatencyStats.P99 > pt.config.MaxLatencyP99 {
		*failures = append(*failures, TestFailure{
			Type:        "latency_p99",
			Description: "P99 latency exceeds maximum allowed",
			Expected:    pt.config.MaxLatencyP99.String(),
			Actual:      result.LatencyStats.P99.String(),
			Threshold:   "maximum",
		})
		passed = false
	}

	// Check throughput
	if result.Throughput < float64(pt.config.MinThroughput) {
		*failures = append(*failures, TestFailure{
			Type:        "throughput",
			Description: "Throughput below minimum required",
			Expected:    fmt.Sprintf("%d req/s", pt.config.MinThroughput),
			Actual:      fmt.Sprintf("%.2f req/s", result.Throughput),
			Threshold:   "minimum",
		})
		passed = false
	}

	// Check error rate
	if result.ErrorRate > pt.config.MaxErrorRate {
		*failures = append(*failures, TestFailure{
			Type:        "error_rate",
			Description: "Error rate exceeds maximum allowed",
			Expected:    fmt.Sprintf("%.2f%%", pt.config.MaxErrorRate*100),
			Actual:      fmt.Sprintf("%.2f%%", result.ErrorRate*100),
			Threshold:   "maximum",
		})
		passed = false
	}

	return passed
}

// GetResults returns all test results
func (pt *PerformanceTester) GetResults() map[string]*TestResult {
	return pt.results
}

// GenerateTestReport generates a comprehensive test report
func (pt *PerformanceTester) GenerateTestReport() string {
	results := pt.GetResults()

	report := "Performance Test Report\n"
	report += "======================\n\n"

	// Summary
	totalTests := len(results)
	passedTests := 0
	failedTests := 0

	for _, result := range results {
		if result.Passed {
			passedTests++
		} else {
			failedTests++
		}
	}

	report += fmt.Sprintf("Test Summary:\n")
	report += fmt.Sprintf("  Total Tests: %d\n", totalTests)
	report += fmt.Sprintf("  Passed: %d\n", passedTests)
	report += fmt.Sprintf("  Failed: %d\n", failedTests)
	report += fmt.Sprintf("  Success Rate: %.2f%%\n\n", float64(passedTests)/float64(totalTests)*100)

	// Detailed results
	for scenarioName, result := range results {
		report += fmt.Sprintf("Scenario: %s\n", scenarioName)
		report += fmt.Sprintf("  Status: %s\n", map[bool]string{true: "PASSED", false: "FAILED"}[result.Passed])
		report += fmt.Sprintf("  Duration: %v\n", result.Duration)

		if result.BenchmarkResult != nil {
			report += fmt.Sprintf("  Total Requests: %d\n", result.BenchmarkResult.TotalRequests)
			report += fmt.Sprintf("  Successful: %d\n", result.BenchmarkResult.SuccessfulRequests)
			report += fmt.Sprintf("  Failed: %d\n", result.BenchmarkResult.FailedRequests)
			report += fmt.Sprintf("  Error Rate: %.2f%%\n", result.BenchmarkResult.ErrorRate*100)
			report += fmt.Sprintf("  Throughput: %.2f req/s\n", result.BenchmarkResult.Throughput)
			report += fmt.Sprintf("  Latency P50: %v\n", result.BenchmarkResult.LatencyStats.Median)
			report += fmt.Sprintf("  Latency P95: %v\n", result.BenchmarkResult.LatencyStats.P95)
			report += fmt.Sprintf("  Latency P99: %v\n", result.BenchmarkResult.LatencyStats.P99)
		}

		if len(result.Failures) > 0 {
			report += fmt.Sprintf("  Failures:\n")
			for _, failure := range result.Failures {
				report += fmt.Sprintf("    - %s: %s (expected: %s, actual: %s)\n",
					failure.Type, failure.Description, failure.Expected, failure.Actual)
			}
		}

		report += "\n" + strings.Repeat("-", 50) + "\n\n"
	}

	return report
}

// LoadTestScenario represents a load test scenario
type LoadTestScenario struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Config       LoadTestConfig         `json:"config"`
	TestFunction TestFunction           `json:"-"`
	Data         map[string]interface{} `json:"data"`
}

// LoadTestConfig defines configuration for load testing
type LoadTestConfig struct {
	// Load test settings
	Duration     time.Duration `json:"duration"`
	RampUpTime   time.Duration `json:"ramp_up_time"`
	RampDownTime time.Duration `json:"ramp_down_time"`
	MaxUsers     int           `json:"max_users"`
	MinUsers     int           `json:"min_users"`

	// Load pattern
	LoadPattern string `json:"load_pattern"` // "linear", "exponential", "step", "constant"

	// Performance thresholds
	MaxLatencyP95 time.Duration `json:"max_latency_p95"`
	MaxLatencyP99 time.Duration `json:"max_latency_p99"`
	MinThroughput int64         `json:"min_throughput"`
	MaxErrorRate  float64       `json:"max_error_rate"`
}

// LoadTestResult represents the result of a load test
type LoadTestResult struct {
	ScenarioName    string           `json:"scenario_name"`
	StartTime       time.Time        `json:"start_time"`
	EndTime         time.Time        `json:"end_time"`
	Duration        time.Duration    `json:"duration"`
	Success         bool             `json:"success"`
	ErrorMessage    string           `json:"error_message,omitempty"`
	BenchmarkResult *BenchmarkResult `json:"benchmark_result,omitempty"`
	Passed          bool             `json:"passed"`
	Failures        []TestFailure    `json:"failures,omitempty"`
	LoadProfile     []LoadPoint      `json:"load_profile,omitempty"`
}

// LoadPoint represents a point in the load profile
type LoadPoint struct {
	Timestamp   time.Time     `json:"timestamp"`
	ActiveUsers int           `json:"active_users"`
	Throughput  float64       `json:"throughput"`
	ErrorRate   float64       `json:"error_rate"`
	LatencyP95  time.Duration `json:"latency_p95"`
}

// RunLoadTest runs a load test scenario
func (pt *PerformanceTester) RunLoadTest(ctx context.Context, scenario *LoadTestScenario) (*LoadTestResult, error) {
	result := &LoadTestResult{
		ScenarioName: scenario.Name,
		StartTime:    time.Now(),
		LoadProfile:  make([]LoadPoint, 0),
	}

	// Create load profile
	loadProfile := pt.generateLoadProfile(scenario.Config)

	// Run load test
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalRequests int64
	var successfulRequests int64
	var failedRequests int64
	var latencies []time.Duration

	// Start load test
	for _, point := range loadProfile {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Run concurrent users for this load point
		for i := 0; i < point.ActiveUsers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				start := time.Now()
				err := scenario.TestFunction(ctx, scenario.Data)
				duration := time.Since(start)

				mu.Lock()
				totalRequests++
				if err != nil {
					failedRequests++
				} else {
					successfulRequests++
				}
				latencies = append(latencies, duration)
				mu.Unlock()
			}()
		}

		// Wait for this load point to complete
		time.Sleep(time.Second) // 1 second per load point

		// Record load point metrics
		mu.Lock()
		throughput := float64(successfulRequests) / time.Since(result.StartTime).Seconds()
		errorRate := float64(failedRequests) / float64(totalRequests)

		// Calculate P95 latency
		var latencyP95 time.Duration
		if len(latencies) > 0 {
			sort.Slice(latencies, func(i, j int) bool {
				return latencies[i] < latencies[j]
			})
			p95Index := int(float64(len(latencies)) * 0.95)
			if p95Index < len(latencies) {
				latencyP95 = latencies[p95Index]
			}
		}

		loadPoint := LoadPoint{
			Timestamp:   time.Now(),
			ActiveUsers: point.ActiveUsers,
			Throughput:  throughput,
			ErrorRate:   errorRate,
			LatencyP95:  latencyP95,
		}
		result.LoadProfile = append(result.LoadProfile, loadPoint)
		mu.Unlock()
	}

	// Wait for all requests to complete
	wg.Wait()

	// Calculate final results
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = true

	// Create benchmark result
	benchmarkResult := &BenchmarkResult{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successfulRequests,
		FailedRequests:     failedRequests,
		ErrorRate:          float64(failedRequests) / float64(totalRequests),
		Throughput:         float64(successfulRequests) / result.Duration.Seconds(),
		LatencyStats:       pt.calculateLatencyStats(latencies),
	}

	result.BenchmarkResult = benchmarkResult
	result.Passed = pt.evaluateTestResult(benchmarkResult, &result.Failures)

	return result, nil
}

// generateLoadProfile generates a load profile based on the configuration
func (pt *PerformanceTester) generateLoadProfile(config LoadTestConfig) []LoadPoint {
	var profile []LoadPoint

	// Calculate number of load points
	totalDuration := config.Duration
	loadPoints := int(totalDuration.Seconds()) // 1 second per load point

	// Generate load profile based on pattern
	switch config.LoadPattern {
	case "linear":
		profile = pt.generateLinearLoadProfile(config, loadPoints)
	case "exponential":
		profile = pt.generateExponentialLoadProfile(config, loadPoints)
	case "step":
		profile = pt.generateStepLoadProfile(config, loadPoints)
	case "constant":
		profile = pt.generateConstantLoadProfile(config, loadPoints)
	default:
		profile = pt.generateLinearLoadProfile(config, loadPoints)
	}

	return profile
}

// generateLinearLoadProfile generates a linear load profile
func (pt *PerformanceTester) generateLinearLoadProfile(config LoadTestConfig, loadPoints int) []LoadPoint {
	var profile []LoadPoint

	for i := 0; i < loadPoints; i++ {
		progress := float64(i) / float64(loadPoints-1)
		users := int(float64(config.MaxUsers-config.MinUsers)*progress) + config.MinUsers

		profile = append(profile, LoadPoint{
			ActiveUsers: users,
		})
	}

	return profile
}

// generateExponentialLoadProfile generates an exponential load profile
func (pt *PerformanceTester) generateExponentialLoadProfile(config LoadTestConfig, loadPoints int) []LoadPoint {
	var profile []LoadPoint

	for i := 0; i < loadPoints; i++ {
		progress := float64(i) / float64(loadPoints-1)
		exponentialProgress := progress * progress
		users := int(float64(config.MaxUsers-config.MinUsers)*exponentialProgress) + config.MinUsers

		profile = append(profile, LoadPoint{
			ActiveUsers: users,
		})
	}

	return profile
}

// generateStepLoadProfile generates a step load profile
func (pt *PerformanceTester) generateStepLoadProfile(config LoadTestConfig, loadPoints int) []LoadPoint {
	var profile []LoadPoint

	stepSize := loadPoints / 5 // 5 steps
	usersPerStep := (config.MaxUsers - config.MinUsers) / 5

	for i := 0; i < loadPoints; i++ {
		step := i / stepSize
		users := config.MinUsers + (step * usersPerStep)
		if users > config.MaxUsers {
			users = config.MaxUsers
		}

		profile = append(profile, LoadPoint{
			ActiveUsers: users,
		})
	}

	return profile
}

// generateConstantLoadProfile generates a constant load profile
func (pt *PerformanceTester) generateConstantLoadProfile(config LoadTestConfig, loadPoints int) []LoadPoint {
	var profile []LoadPoint

	users := (config.MaxUsers + config.MinUsers) / 2

	for i := 0; i < loadPoints; i++ {
		profile = append(profile, LoadPoint{
			ActiveUsers: users,
		})
	}

	return profile
}

// calculateLatencyStats calculates latency statistics from a slice of latencies
func (pt *PerformanceTester) calculateLatencyStats(latencies []time.Duration) LatencyStats {
	if len(latencies) == 0 {
		return LatencyStats{}
	}

	// Sort latencies
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)
	sort.Slice(sortedLatencies, func(i, j int) bool {
		return sortedLatencies[i] < sortedLatencies[j]
	})

	// Calculate basic statistics
	min := sortedLatencies[0]
	max := sortedLatencies[len(sortedLatencies)-1]

	// Calculate mean
	var sum time.Duration
	for _, latency := range sortedLatencies {
		sum += latency
	}
	mean := sum / time.Duration(len(sortedLatencies))

	// Calculate median
	median := sortedLatencies[len(sortedLatencies)/2]

	// Calculate percentiles
	p90 := pt.calculatePercentile(sortedLatencies, 0.90)
	p95 := pt.calculatePercentile(sortedLatencies, 0.95)
	p99 := pt.calculatePercentile(sortedLatencies, 0.99)
	p999 := pt.calculatePercentile(sortedLatencies, 0.999)

	// Calculate standard deviation
	var variance float64
	for _, latency := range sortedLatencies {
		diff := float64(latency - mean)
		variance += diff * diff
	}
	variance /= float64(len(sortedLatencies))
	stdDev := time.Duration(math.Sqrt(variance))

	return LatencyStats{
		Min:    min,
		Max:    max,
		Mean:   mean,
		Median: median,
		P90:    p90,
		P95:    p95,
		P99:    p99,
		P999:   p999,
		StdDev: stdDev,
	}
}

// calculatePercentile calculates a percentile from sorted latencies
func (pt *PerformanceTester) calculatePercentile(sortedLatencies []time.Duration, percentile float64) time.Duration {
	if len(sortedLatencies) == 0 {
		return 0
	}

	index := percentile * float64(len(sortedLatencies)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sortedLatencies) {
		return sortedLatencies[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	lowerValue := float64(sortedLatencies[lower])
	upperValue := float64(sortedLatencies[upper])

	return time.Duration(lowerValue + weight*(upperValue-lowerValue))
}
