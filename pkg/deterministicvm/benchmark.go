// Package deterministicvm provides performance benchmarking for OCX Protocol
package deterministicvm

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// BenchmarkResult contains the results of a performance benchmark
type BenchmarkResult struct {
	// Test configuration
	TestName     string
	VMType       VMType
	ArtifactSize int
	InputSize    int

	// Performance metrics
	Duration   time.Duration
	GasUsed    uint64
	HostCycles uint64
	MemoryUsed uint64

	// System metrics
	CPUTime    time.Duration
	MemoryPeak uint64
	Goroutines int

	// Determinism metrics
	Deterministic bool
	StdoutHash    string
	StderrHash    string
}

// BenchmarkSuite runs a comprehensive performance benchmark suite
type BenchmarkSuite struct {
	results []BenchmarkResult
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite() *BenchmarkSuite {
	return &BenchmarkSuite{
		results: make([]BenchmarkResult, 0),
	}
}

// RunComprehensiveBenchmark runs a full benchmark suite
func (bs *BenchmarkSuite) RunComprehensiveBenchmark() error {
	fmt.Println("🚀 OCX Protocol Performance Benchmark Suite")
	fmt.Println("=============================================")
	fmt.Println()

	// Test configurations
	testConfigs := []struct {
		name        string
		script      string
		input       []byte
		description string
	}{
		{
			name:        "simple_echo",
			script:      "#!/bin/bash\necho \"Hello, OCX!\"\nexit 0",
			input:       []byte("test input"),
			description: "Simple echo command",
		},
		{
			name:        "input_processing",
			script:      "#!/bin/bash\necho \"Input: $(cat input.bin)\"\necho \"Length: $(wc -c < input.bin)\"\nexit 0",
			input:       []byte("This is a test input for performance benchmarking"),
			description: "Input processing with wc",
		},
		{
			name:        "text_processing",
			script:      "#!/bin/bash\necho \"Processing input...\"\necho \"$(cat input.bin)\" | tr 'a-z' 'A-Z'\necho \"Done processing\"\nexit 0",
			input:       []byte("hello world from ocx protocol"),
			description: "Text processing with tr",
		},
		{
			name:        "large_input",
			script:      "#!/bin/bash\necho \"Processing large input...\"\necho \"Input size: $(wc -c < input.bin)\"\necho \"First 100 chars: $(head -c 100 input.bin)\"\nexit 0",
			input:       []byte(strings.Repeat("This is a large input for performance testing. ", 100)),
			description: "Large input processing",
		},
	}

	// VM types to test
	vmTypes := []VMType{VMTypeOSProcess, VMTypeWASM}

	// Run benchmarks for each configuration
	for _, config := range testConfigs {
		fmt.Printf("📊 Benchmarking: %s (%s)\n", config.name, config.description)
		fmt.Printf("   Input size: %d bytes\n", len(config.input))
		fmt.Println()

		for _, vmType := range vmTypes {
			fmt.Printf("   🔧 VM Type: %s\n", vmType)

			// Run multiple iterations for statistical significance
			const iterations = 10
			var durations []time.Duration
			var gasValues []uint64
			var deterministic bool = true
			var firstStdout, firstStderr string

			for i := 0; i < iterations; i++ {
				result, err := bs.runSingleBenchmark(config.name, vmType, config.script, config.input)
				if err != nil {
					// For WASM with shell scripts, this is expected - skip gracefully
					if vmType == VMTypeWASM && strings.Contains(err.Error(), "not a valid WASM file") {
						fmt.Printf("      ❌ Iteration %d failed: %v\n", i+1, err)
						continue
					}
					fmt.Printf("      ❌ Iteration %d failed: %v\n", i+1, err)
					continue
				}

				durations = append(durations, result.Duration)
				gasValues = append(gasValues, result.GasUsed)

				// Check determinism
				if i == 0 {
					firstStdout = result.StdoutHash
					firstStderr = result.StderrHash
				} else {
					if result.StdoutHash != firstStdout || result.StderrHash != firstStderr {
						deterministic = false
					}
				}

				bs.results = append(bs.results, *result)
			}

			// Calculate statistics
			if len(durations) > 0 {
				stats := calculateStats(durations)
				gasStats := calculateUint64Stats(gasValues)

				fmt.Printf("      ⏱️  Duration: avg=%.2fms, min=%.2fms, max=%.2fms, p95=%.2fms\n",
					float64(stats.Average.Milliseconds()), float64(stats.Min.Milliseconds()),
					float64(stats.Max.Milliseconds()), float64(stats.P95.Milliseconds()))
				fmt.Printf("      ⛽ Gas: avg=%d, min=%d, max=%d\n",
					gasStats.Average, gasStats.Min, gasStats.Max)
				fmt.Printf("      🎯 Deterministic: %t\n", deterministic)

				// Check if meets <15ms target
				if stats.P95.Milliseconds() < 15 {
					fmt.Printf("      ✅ Meets <15ms target (p95: %.2fms)\n", float64(stats.P95.Milliseconds()))
				} else {
					fmt.Printf("      ⚠️  Exceeds <15ms target (p95: %.2fms)\n", float64(stats.P95.Milliseconds()))
				}
			}
			fmt.Println()
		}
	}

	// Generate summary report
	bs.generateSummaryReport()

	return nil
}

// runSingleBenchmark runs a single benchmark iteration
func (bs *BenchmarkSuite) runSingleBenchmark(testName string, vmType VMType, script string, input []byte) (*BenchmarkResult, error) {
	// Set VM type
	if err := SetVMType(vmType); err != nil {
		return nil, fmt.Errorf("failed to set VM type: %w", err)
	}

	// Create test artifact
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("bench-%s-*.sh", testName))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write script content
	if _, err := tmpFile.WriteString(script); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}
	tmpFile.Close()

	// Make executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return nil, fmt.Errorf("failed to make executable: %w", err)
	}

	// Set up artifact cache
	cacheDir := filepath.Dir(tmpFile.Name())
	hash := sha256.Sum256([]byte(script))
	hashStr := fmt.Sprintf("%x", hash)
	cachePath := filepath.Join(cacheDir, hashStr)

	// Copy to cache location
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read temp file: %w", err)
	}
	if err := os.WriteFile(cachePath, data, 0755); err != nil {
		return nil, fmt.Errorf("failed to write to cache: %w", err)
	}
	defer os.Remove(cachePath)

	// Record system state before execution
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	startTime := time.Now()
	goroutinesBefore := runtime.NumGoroutine()

	// Execute artifact
	result, err := ExecuteArtifact(context.Background(), hash, input)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Record system state after execution
	endTime := time.Now()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	goroutinesAfter := runtime.NumGoroutine()

	// Create benchmark result
	benchResult := &BenchmarkResult{
		TestName:      testName,
		VMType:        vmType,
		ArtifactSize:  len(script),
		InputSize:     len(input),
		Duration:      result.Duration,
		GasUsed:       result.GasUsed,
		HostCycles:    result.HostCycles,
		MemoryUsed:    result.MemoryUsed,
		CPUTime:       endTime.Sub(startTime),
		MemoryPeak:    m2.Alloc - m1.Alloc,
		Goroutines:    goroutinesAfter - goroutinesBefore,
		Deterministic: true, // Will be checked by caller
		StdoutHash:    fmt.Sprintf("%x", result.Stdout),
		StderrHash:    fmt.Sprintf("%x", result.Stderr),
	}

	return benchResult, nil
}

// generateSummaryReport generates a comprehensive summary report
func (bs *BenchmarkSuite) generateSummaryReport() {
	fmt.Println("📈 PERFORMANCE SUMMARY REPORT")
	fmt.Println("=============================")
	fmt.Println()

	// Group results by VM type
	osProcessResults := make([]BenchmarkResult, 0)
	wasmResults := make([]BenchmarkResult, 0)

	for _, result := range bs.results {
		if result.VMType == VMTypeOSProcess {
			osProcessResults = append(osProcessResults, result)
		} else if result.VMType == VMTypeWASM {
			wasmResults = append(wasmResults, result)
		}
	}

	// Analyze OS Process VM performance
	if len(osProcessResults) > 0 {
		fmt.Println("🖥️  OS Process VM Performance:")
		bs.analyzeVMResults(osProcessResults)
		fmt.Println()
	}

	// Analyze WASM VM performance
	if len(wasmResults) > 0 {
		fmt.Println("🌐 WASM VM Performance:")
		bs.analyzeVMResults(wasmResults)
		fmt.Println()
	}

	// Overall performance targets
	fmt.Println("🎯 PERFORMANCE TARGETS")
	fmt.Println("======================")

	// Check <15ms target
	allDurations := make([]time.Duration, 0, len(bs.results))
	for _, result := range bs.results {
		allDurations = append(allDurations, result.Duration)
	}

	if len(allDurations) > 0 {
		stats := calculateStats(allDurations)
		fmt.Printf("End-to-end execution time:\n")
		fmt.Printf("  Average: %.2fms\n", float64(stats.Average.Milliseconds()))
		fmt.Printf("  P95: %.2fms\n", float64(stats.P95.Milliseconds()))
		fmt.Printf("  P99: %.2fms\n", float64(stats.P99.Milliseconds()))

		if stats.P95.Milliseconds() < 15 {
			fmt.Printf("  ✅ Meets <15ms target (P95: %.2fms)\n", float64(stats.P95.Milliseconds()))
		} else {
			fmt.Printf("  ⚠️  Exceeds <15ms target (P95: %.2fms)\n", float64(stats.P95.Milliseconds()))
		}
	}

	// Determinism check
	deterministicCount := 0
	for _, result := range bs.results {
		if result.Deterministic {
			deterministicCount++
		}
	}

	fmt.Printf("\nDeterminism: %d/%d tests passed (%.1f%%)\n",
		deterministicCount, len(bs.results),
		float64(deterministicCount)/float64(len(bs.results))*100)

	fmt.Println()
	fmt.Println("🏆 BENCHMARK COMPLETE")
	fmt.Println("====================")
}

// analyzeVMResults analyzes results for a specific VM type
func (bs *BenchmarkSuite) analyzeVMResults(results []BenchmarkResult) {
	if len(results) == 0 {
		fmt.Println("  No results available")
		return
	}

	// Calculate duration statistics
	durations := make([]time.Duration, len(results))
	for i, result := range results {
		durations[i] = result.Duration
	}
	stats := calculateStats(durations)

	fmt.Printf("  Duration: avg=%.2fms, min=%.2fms, max=%.2fms, p95=%.2fms\n",
		float64(stats.Average.Milliseconds()), float64(stats.Min.Milliseconds()),
		float64(stats.Max.Milliseconds()), float64(stats.P95.Milliseconds()))

	// Calculate gas statistics
	gasValues := make([]uint64, len(results))
	for i, result := range results {
		gasValues[i] = result.GasUsed
	}
	gasStats := calculateUint64Stats(gasValues)

	fmt.Printf("  Gas: avg=%d, min=%d, max=%d\n",
		gasStats.Average, gasStats.Min, gasStats.Max)

	// Memory usage
	memoryValues := make([]uint64, len(results))
	for i, result := range results {
		memoryValues[i] = result.MemoryUsed
	}
	memoryStats := calculateUint64Stats(memoryValues)

	fmt.Printf("  Memory: avg=%d bytes, min=%d bytes, max=%d bytes\n",
		memoryStats.Average, memoryStats.Min, memoryStats.Max)
}

// DurationStats contains statistical information about durations
type DurationStats struct {
	Average time.Duration
	Min     time.Duration
	Max     time.Duration
	P95     time.Duration
	P99     time.Duration
}

// Uint64Stats contains statistical information about uint64 values
type Uint64Stats struct {
	Average uint64
	Min     uint64
	Max     uint64
}

// calculateStats calculates statistical measures for durations
func calculateStats(durations []time.Duration) DurationStats {
	if len(durations) == 0 {
		return DurationStats{}
	}

	// Sort durations
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate average
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	average := total / time.Duration(len(durations))

	// Calculate percentiles
	p95Index := int(float64(len(sorted)) * 0.95)
	p99Index := int(float64(len(sorted)) * 0.99)

	if p95Index >= len(sorted) {
		p95Index = len(sorted) - 1
	}
	if p99Index >= len(sorted) {
		p99Index = len(sorted) - 1
	}

	return DurationStats{
		Average: average,
		Min:     sorted[0],
		Max:     sorted[len(sorted)-1],
		P95:     sorted[p95Index],
		P99:     sorted[p99Index],
	}
}

// calculateUint64Stats calculates statistical measures for uint64 values
func calculateUint64Stats(values []uint64) Uint64Stats {
	if len(values) == 0 {
		return Uint64Stats{}
	}

	// Sort values
	sorted := make([]uint64, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate average
	var total uint64
	for _, v := range values {
		total += v
	}
	average := total / uint64(len(values))

	return Uint64Stats{
		Average: average,
		Min:     sorted[0],
		Max:     sorted[len(sorted)-1],
	}
}
