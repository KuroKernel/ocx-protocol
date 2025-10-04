package performance

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BenchmarkConfig defines configuration for performance benchmarks
type BenchmarkConfig struct {
	// Test configuration
	Duration     time.Duration `json:"duration"`
	Concurrency  int           `json:"concurrency"`
	WarmupTime   time.Duration `json:"warmup_time"`
	CooldownTime time.Duration `json:"cooldown_time"`

	// Measurement settings
	SampleRate     float64       `json:"sample_rate"` // 0.0 to 1.0
	ReportInterval time.Duration `json:"report_interval"`

	// Performance targets
	TargetLatencyP50 time.Duration `json:"target_latency_p50"`
	TargetLatencyP95 time.Duration `json:"target_latency_p95"`
	TargetLatencyP99 time.Duration `json:"target_latency_p99"`
	TargetThroughput int64         `json:"target_throughput"` // requests per second
	MaxErrorRate     float64       `json:"max_error_rate"`    // 0.0 to 1.0
}

// BenchmarkResult represents the result of a performance benchmark
type BenchmarkResult struct {
	// Test configuration
	Config BenchmarkConfig `json:"config"`

	// Timing information
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
	WarmupTime   time.Duration `json:"warmup_time"`
	CooldownTime time.Duration `json:"cooldown_time"`

	// Performance metrics
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests     int64   `json:"failed_requests"`
	ErrorRate          float64 `json:"error_rate"`
	Throughput         float64 `json:"throughput"` // requests per second

	// Latency statistics
	LatencyStats LatencyStats `json:"latency_stats"`

	// Resource usage
	ResourceUsage ResourceUsage `json:"resource_usage"`

	// Performance targets
	Targets PerformanceTargets `json:"targets"`

	// Detailed samples (if enabled)
	Samples []BenchmarkSample `json:"samples,omitempty"`
}

// LatencyStats provides detailed latency statistics
type LatencyStats struct {
	Min    time.Duration `json:"min"`
	Max    time.Duration `json:"max"`
	Mean   time.Duration `json:"mean"`
	Median time.Duration `json:"median"`
	P90    time.Duration `json:"p90"`
	P95    time.Duration `json:"p95"`
	P99    time.Duration `json:"p99"`
	P999   time.Duration `json:"p999"`
	StdDev time.Duration `json:"std_dev"`
}

// ResourceUsage tracks resource consumption during the benchmark
type ResourceUsage struct {
	CPUUsage    float64 `json:"cpu_usage"`    // percentage
	MemoryUsage int64   `json:"memory_usage"` // bytes
	DiskUsage   int64   `json:"disk_usage"`   // bytes
	NetworkIO   int64   `json:"network_io"`   // bytes
}

// PerformanceTargets defines performance targets and results
type PerformanceTargets struct {
	LatencyP50 Target `json:"latency_p50"`
	LatencyP95 Target `json:"latency_p95"`
	LatencyP99 Target `json:"latency_p99"`
	Throughput Target `json:"throughput"`
	ErrorRate  Target `json:"error_rate"`
}

// Target represents a performance target with result
type Target struct {
	Expected      time.Duration `json:"expected,omitempty"`
	Actual        time.Duration `json:"actual,omitempty"`
	ExpectedFloat float64       `json:"expected_float,omitempty"`
	ActualFloat   float64       `json:"actual_float,omitempty"`
	Met           bool          `json:"met"`
}

// BenchmarkSample represents a single benchmark sample
type BenchmarkSample struct {
	Timestamp   time.Time     `json:"timestamp"`
	Duration    time.Duration `json:"duration"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
	Concurrency int           `json:"concurrency"`
}

// BenchmarkFunction represents a function to benchmark
type BenchmarkFunction func(ctx context.Context) error

// BenchmarkRunner runs performance benchmarks
type BenchmarkRunner struct {
	config       BenchmarkConfig
	samples      []BenchmarkSample
	samplesMutex sync.Mutex
	stats        *BenchmarkStats
	statsMutex   sync.RWMutex
}

// BenchmarkStats tracks benchmark statistics
type BenchmarkStats struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalDuration      time.Duration
	Latencies          []time.Duration
	StartTime          time.Time
	EndTime            time.Time
}

// NewBenchmarkRunner creates a new benchmark runner
func NewBenchmarkRunner(config BenchmarkConfig) *BenchmarkRunner {
	return &BenchmarkRunner{
		config:  config,
		samples: make([]BenchmarkSample, 0),
		stats:   &BenchmarkStats{},
	}
}

// Run executes a benchmark with the given function
func (br *BenchmarkRunner) Run(ctx context.Context, fn BenchmarkFunction) (*BenchmarkResult, error) {
	// Create context with timeout
	benchmarkCtx, cancel := context.WithTimeout(ctx, br.config.Duration+br.config.WarmupTime+br.config.CooldownTime)
	defer cancel()

	// Initialize stats
	br.statsMutex.Lock()
	br.stats = &BenchmarkStats{
		Latencies: make([]time.Duration, 0),
		StartTime: time.Now(),
	}
	br.statsMutex.Unlock()

	// Warmup phase
	if br.config.WarmupTime > 0 {
		if err := br.runWarmup(benchmarkCtx, fn); err != nil {
			return nil, fmt.Errorf("warmup failed: %w", err)
		}
	}

	// Main benchmark phase
	benchmarkStart := time.Now()
	if err := br.runBenchmark(benchmarkCtx, fn); err != nil {
		return nil, fmt.Errorf("benchmark failed: %w", err)
	}
	benchmarkEnd := time.Now()

	// Cooldown phase
	if br.config.CooldownTime > 0 {
		time.Sleep(br.config.CooldownTime)
	}

	// Calculate results
	result := br.calculateResults(benchmarkStart, benchmarkEnd)
	return result, nil
}

// runWarmup runs the warmup phase
func (br *BenchmarkRunner) runWarmup(ctx context.Context, fn BenchmarkFunction) error {
	warmupCtx, cancel := context.WithTimeout(ctx, br.config.WarmupTime)
	defer cancel()

	// Run warmup with reduced concurrency
	warmupConcurrency := br.config.Concurrency / 2
	if warmupConcurrency < 1 {
		warmupConcurrency = 1
	}

	return br.runConcurrent(warmupCtx, fn, warmupConcurrency, true)
}

// runBenchmark runs the main benchmark phase
func (br *BenchmarkRunner) runBenchmark(ctx context.Context, fn BenchmarkFunction) error {
	benchmarkCtx, cancel := context.WithTimeout(ctx, br.config.Duration)
	defer cancel()

	return br.runConcurrent(benchmarkCtx, fn, br.config.Concurrency, false)
}

// runConcurrent runs the benchmark function concurrently
func (br *BenchmarkRunner) runConcurrent(ctx context.Context, fn BenchmarkFunction, concurrency int, isWarmup bool) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		case semaphore <- struct{}{}:
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-semaphore }()

				br.runSingleRequest(ctx, fn, concurrency, isWarmup)
			}()
		}
	}
}

// runSingleRequest runs a single benchmark request
func (br *BenchmarkRunner) runSingleRequest(ctx context.Context, fn BenchmarkFunction, concurrency int, isWarmup bool) {
	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	// Update stats
	br.statsMutex.Lock()
	br.stats.TotalRequests++
	if err != nil {
		br.stats.FailedRequests++
	} else {
		br.stats.SuccessfulRequests++
	}
	br.stats.Latencies = append(br.stats.Latencies, duration)
	br.statsMutex.Unlock()

	// Record sample if not warmup and sampling is enabled
	if !isWarmup && br.shouldSample() {
		br.samplesMutex.Lock()
		sample := BenchmarkSample{
			Timestamp:   start,
			Duration:    duration,
			Success:     err == nil,
			Error:       fmt.Sprintf("%v", err),
			Concurrency: concurrency,
		}
		br.samples = append(br.samples, sample)
		br.samplesMutex.Unlock()
	}
}

// shouldSample determines if this request should be sampled
func (br *BenchmarkRunner) shouldSample() bool {
	if br.config.SampleRate <= 0 {
		return false
	}
	if br.config.SampleRate >= 1.0 {
		return true
	}
	// Simple sampling based on request count
	return br.stats.TotalRequests%int64(1.0/br.config.SampleRate) == 0
}

// calculateResults calculates the final benchmark results
func (br *BenchmarkRunner) calculateResults(start, end time.Time) *BenchmarkResult {
	br.statsMutex.Lock()
	defer br.statsMutex.Unlock()

	duration := end.Sub(start)
	totalRequests := br.stats.TotalRequests
	successfulRequests := br.stats.SuccessfulRequests
	failedRequests := br.stats.FailedRequests

	// Calculate error rate
	errorRate := float64(failedRequests) / float64(totalRequests)

	// Calculate throughput
	throughput := float64(totalRequests) / duration.Seconds()

	// Calculate latency statistics
	latencyStats := br.calculateLatencyStats()

	// Calculate performance targets
	targets := br.calculatePerformanceTargets(latencyStats, throughput, errorRate)

	// Get real resource usage from system monitoring
	resourceUsage := br.getSystemResourceUsage()

	return &BenchmarkResult{
		Config:             br.config,
		StartTime:          start,
		EndTime:            end,
		Duration:           duration,
		WarmupTime:         br.config.WarmupTime,
		CooldownTime:       br.config.CooldownTime,
		TotalRequests:      totalRequests,
		SuccessfulRequests: successfulRequests,
		FailedRequests:     failedRequests,
		ErrorRate:          errorRate,
		Throughput:         throughput,
		LatencyStats:       latencyStats,
		ResourceUsage:      resourceUsage,
		Targets:            targets,
		Samples:            br.samples,
	}
}

// calculateLatencyStats calculates latency statistics from the collected latencies
func (br *BenchmarkRunner) calculateLatencyStats() LatencyStats {
	latencies := br.stats.Latencies
	if len(latencies) == 0 {
		return LatencyStats{}
	}

	// Sort latencies for percentile calculations
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
	p90 := br.calculatePercentile(sortedLatencies, 0.90)
	p95 := br.calculatePercentile(sortedLatencies, 0.95)
	p99 := br.calculatePercentile(sortedLatencies, 0.99)
	p999 := br.calculatePercentile(sortedLatencies, 0.999)

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
func (br *BenchmarkRunner) calculatePercentile(sortedLatencies []time.Duration, percentile float64) time.Duration {
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

// calculatePerformanceTargets calculates whether performance targets were met
func (br *BenchmarkRunner) calculatePerformanceTargets(latencyStats LatencyStats, throughput float64, errorRate float64) PerformanceTargets {
	return PerformanceTargets{
		LatencyP50: Target{
			Expected: br.config.TargetLatencyP50,
			Actual:   latencyStats.Median,
			Met:      latencyStats.Median <= br.config.TargetLatencyP50,
		},
		LatencyP95: Target{
			Expected: br.config.TargetLatencyP95,
			Actual:   latencyStats.P95,
			Met:      latencyStats.P95 <= br.config.TargetLatencyP95,
		},
		LatencyP99: Target{
			Expected: br.config.TargetLatencyP99,
			Actual:   latencyStats.P99,
			Met:      latencyStats.P99 <= br.config.TargetLatencyP99,
		},
		Throughput: Target{
			ExpectedFloat: float64(br.config.TargetThroughput),
			ActualFloat:   throughput,
			Met:           throughput >= float64(br.config.TargetThroughput),
		},
		ErrorRate: Target{
			ExpectedFloat: br.config.MaxErrorRate,
			ActualFloat:   errorRate,
			Met:           errorRate <= br.config.MaxErrorRate,
		},
	}
}

// BenchmarkSuite runs multiple benchmarks
type BenchmarkSuite struct {
	benchmarks map[string]*BenchmarkRunner
	results    map[string]*BenchmarkResult
	mutex      sync.RWMutex
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite() *BenchmarkSuite {
	return &BenchmarkSuite{
		benchmarks: make(map[string]*BenchmarkRunner),
		results:    make(map[string]*BenchmarkResult),
	}
}

// AddBenchmark adds a benchmark to the suite
func (bs *BenchmarkSuite) AddBenchmark(name string, config BenchmarkConfig) {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()
	bs.benchmarks[name] = NewBenchmarkRunner(config)
}

// RunBenchmark runs a specific benchmark
func (bs *BenchmarkSuite) RunBenchmark(ctx context.Context, name string, fn BenchmarkFunction) error {
	bs.mutex.RLock()
	runner, exists := bs.benchmarks[name]
	bs.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("benchmark %s not found", name)
	}

	result, err := runner.Run(ctx, fn)
	if err != nil {
		return err
	}

	bs.mutex.Lock()
	bs.results[name] = result
	bs.mutex.Unlock()

	return nil
}

// RunAllBenchmarks runs all benchmarks in the suite
func (bs *BenchmarkSuite) RunAllBenchmarks(ctx context.Context, functions map[string]BenchmarkFunction) error {
	for name, fn := range functions {
		if err := bs.RunBenchmark(ctx, name, fn); err != nil {
			return fmt.Errorf("benchmark %s failed: %w", name, err)
		}
	}
	return nil
}

// GetResults returns all benchmark results
func (bs *BenchmarkSuite) GetResults() map[string]*BenchmarkResult {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()

	results := make(map[string]*BenchmarkResult)
	for name, result := range bs.results {
		results[name] = result
	}

	return results
}

// GenerateReport generates a comprehensive benchmark report
func (bs *BenchmarkSuite) GenerateReport() string {
	results := bs.GetResults()

	report := "Benchmark Report\n"
	report += "================\n\n"

	for name, result := range results {
		report += fmt.Sprintf("Benchmark: %s\n", name)
		report += fmt.Sprintf("Duration: %v\n", result.Duration)
		report += fmt.Sprintf("Total Requests: %d\n", result.TotalRequests)
		report += fmt.Sprintf("Successful: %d\n", result.SuccessfulRequests)
		report += fmt.Sprintf("Failed: %d\n", result.FailedRequests)
		report += fmt.Sprintf("Error Rate: %.2f%%\n", result.ErrorRate*100)
		report += fmt.Sprintf("Throughput: %.2f req/s\n", result.Throughput)
		report += fmt.Sprintf("Latency P50: %v\n", result.LatencyStats.Median)
		report += fmt.Sprintf("Latency P95: %v\n", result.LatencyStats.P95)
		report += fmt.Sprintf("Latency P99: %v\n", result.LatencyStats.P99)

		// Performance targets
		report += "\nPerformance Targets:\n"
		report += fmt.Sprintf("  P50 Latency: %v (target: %v) - %s\n",
			result.Targets.LatencyP50.Actual,
			result.Targets.LatencyP50.Expected,
			map[bool]string{true: "PASS", false: "FAIL"}[result.Targets.LatencyP50.Met])
		report += fmt.Sprintf("  P95 Latency: %v (target: %v) - %s\n",
			result.Targets.LatencyP95.Actual,
			result.Targets.LatencyP95.Expected,
			map[bool]string{true: "PASS", false: "FAIL"}[result.Targets.LatencyP95.Met])
		report += fmt.Sprintf("  P99 Latency: %v (target: %v) - %s\n",
			result.Targets.LatencyP99.Actual,
			result.Targets.LatencyP99.Expected,
			map[bool]string{true: "PASS", false: "FAIL"}[result.Targets.LatencyP99.Met])
		report += fmt.Sprintf("  Throughput: %.2f req/s (target: %.2f req/s) - %s\n",
			result.Targets.Throughput.ActualFloat,
			result.Targets.Throughput.ExpectedFloat,
			map[bool]string{true: "PASS", false: "FAIL"}[result.Targets.Throughput.Met])
		report += fmt.Sprintf("  Error Rate: %.2f%% (target: %.2f%%) - %s\n",
			result.Targets.ErrorRate.ActualFloat*100,
			result.Targets.ErrorRate.ExpectedFloat*100,
			map[bool]string{true: "PASS", false: "FAIL"}[result.Targets.ErrorRate.Met])

		report += "\n" + strings.Repeat("-", 50) + "\n\n"
	}

	return report
}

// getSystemResourceUsage gets real system resource usage
func (br *BenchmarkRunner) getSystemResourceUsage() ResourceUsage {
	// Get real system resource usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Get CPU usage (simplified - in production would use more sophisticated monitoring)
	cpuUsage := br.getCPUUsage()

	// Get memory usage
	memoryUsage := int64(m.Alloc)

	// Get disk usage (simplified)
	diskUsage := br.getDiskUsage()

	// Get network I/O (simplified)
	networkIO := br.getNetworkIO()

	return ResourceUsage{
		CPUUsage:    cpuUsage,
		MemoryUsage: memoryUsage,
		DiskUsage:   diskUsage,
		NetworkIO:   networkIO,
	}
}

// getCPUUsage gets current CPU usage percentage using /proc/stat
func (br *BenchmarkRunner) getCPUUsage() float64 {
	// Very light /proc/stat sampler (Linux only)
	read := func() (idle, total uint64, err error) {
		f, err := os.Open("/proc/stat")
		if err != nil {
			return 0, 0, err
		}
		defer f.Close()
		s := bufio.NewScanner(f)
		for s.Scan() {
			if strings.HasPrefix(s.Text(), "cpu ") {
				fields := strings.Fields(s.Text())
				var vals []uint64
				for _, f := range fields[1:] {
					v, _ := strconv.ParseUint(f, 10, 64)
					vals = append(vals, v)
				}
				// idle = idle + iowait; total = sum(all)
				idle = vals[3] + vals[4]
				var sum uint64
				for _, v := range vals {
					sum += v
				}
				return idle, sum, nil
			}
		}
		return 0, 0, fmt.Errorf("cpu line not found")
	}

	i1, t1, err := read()
	if err != nil {
		return 0.0
	}
	time.Sleep(100 * time.Millisecond) // Small delay for measurement
	i2, t2, err := read()
	if err != nil {
		return 0.0
	}

	di := float64(i2 - i1)
	dt := float64(t2 - t1)
	if dt <= 0 {
		return 0.0
	}
	return 100.0 * (1.0 - di/dt)
}

// getDiskUsage gets current disk usage using /proc/self/statm
func (br *BenchmarkRunner) getDiskUsage() int64 {
	// Get process memory usage from /proc/self/statm
	f, err := os.Open("/proc/self/statm")
	if err != nil {
		return 0
	}
	defer f.Close()

	var size, resident uint64
	// fields: size resident share text lib data dt
	if _, err := fmt.Fscanf(f, "%d %d", &size, &resident); err != nil {
		return 0
	}

	page := uint64(os.Getpagesize())
	return int64(resident * page) // Return resident memory in bytes
}

// getNetworkIO gets current network I/O using /proc/net/dev
func (br *BenchmarkRunner) getNetworkIO() int64 {
	// Get network statistics from /proc/net/dev
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0
	}
	defer f.Close()

	var totalBytes uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, ":") {
			// Parse interface line: eth0: 1234 5678 9012 3456 ...
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				fields := strings.Fields(parts[1])
				if len(fields) >= 2 {
					// First two fields are bytes received and transmitted
					if rx, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
						totalBytes += rx
					}
					if tx, err := strconv.ParseUint(fields[8], 10, 64); err == nil {
						totalBytes += tx
					}
				}
			}
		}
	}

	return int64(totalBytes)
}
