// test_runner.go - Comprehensive test runner for OCX Protocol
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type TestConfig struct {
	CoverageThreshold float64
	RaceDetection     bool
	Benchmark         bool
	Verbose           bool
	Timeout           time.Duration
	Packages          []string
}

func main() {
	config := parseFlags()
	
	fmt.Println("🧪 OCX Protocol Test Suite")
	fmt.Println("==========================")
	
	// Run unit tests
	if err := runUnitTests(config); err != nil {
		fmt.Printf("❌ Unit tests failed: %v\n", err)
		os.Exit(1)
	}
	
	// Run integration tests
	if err := runIntegrationTests(config); err != nil {
		fmt.Printf("❌ Integration tests failed: %v\n", err)
		os.Exit(1)
	}
	
	// Run benchmarks
	if config.Benchmark {
		if err := runBenchmarks(config); err != nil {
			fmt.Printf("❌ Benchmarks failed: %v\n", err)
			os.Exit(1)
		}
	}
	
	// Generate coverage report
	if err := generateCoverageReport(config); err != nil {
		fmt.Printf("❌ Coverage report failed: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("✅ All tests passed!")
}

func parseFlags() *TestConfig {
	config := &TestConfig{
		CoverageThreshold: 80.0,
		RaceDetection:     true,
		Benchmark:         false,
		Verbose:           false,
		Timeout:           30 * time.Second,
	}
	
	flag.Float64Var(&config.CoverageThreshold, "coverage", 80.0, "Minimum coverage threshold")
	flag.BoolVar(&config.RaceDetection, "race", true, "Enable race detection")
	flag.BoolVar(&config.Benchmark, "bench", false, "Run benchmarks")
	flag.BoolVar(&config.Verbose, "v", false, "Verbose output")
	flag.DurationVar(&config.Timeout, "timeout", 30*time.Second, "Test timeout")
	
	flag.Parse()
	
	// Get packages to test
	config.Packages = flag.Args()
	if len(config.Packages) == 0 {
		config.Packages = getAllPackages()
	}
	
	return config
}

func runUnitTests(config *TestConfig) error {
	fmt.Println("\n📋 Running Unit Tests...")
	
	args := []string{"test", "-v"}
	if config.RaceDetection {
		args = append(args, "-race")
	}
	if config.Verbose {
		args = append(args, "-v")
	}
	args = append(args, "-timeout", config.Timeout.String())
	args = append(args, config.Packages...)
	
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

func runIntegrationTests(config *TestConfig) error {
	fmt.Println("\n🔗 Running Integration Tests...")
	
	// Find integration test files
	integrationTests, err := findIntegrationTests()
	if err != nil {
		return err
	}
	
	if len(integrationTests) == 0 {
		fmt.Println("No integration tests found")
		return nil
	}
	
	for _, test := range integrationTests {
		fmt.Printf("Running integration test: %s\n", test)
		
		args := []string{"test", "-v", "-timeout", config.Timeout.String(), test}
		cmd := exec.Command("go", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("integration test %s failed: %v", test, err)
		}
	}
	
	return nil
}

func runBenchmarks(config *TestConfig) error {
	fmt.Println("\n⚡ Running Benchmarks...")
	
	args := []string{"test", "-bench=.", "-benchmem", "-run=^$"}
	args = append(args, config.Packages...)
	
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

func generateCoverageReport(config *TestConfig) error {
	fmt.Println("\n📊 Generating Coverage Report...")
	
	// Create coverage directory
	if err := os.MkdirAll("tests/coverage", 0755); err != nil {
		return err
	}
	
	// Run tests with coverage
	args := []string{"test", "-coverprofile=tests/coverage/coverage.out"}
	if config.RaceDetection {
		args = append(args, "-race")
	}
	args = append(args, config.Packages...)
	
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return err
	}
	
	// Generate HTML coverage report
	cmd = exec.Command("go", "tool", "cover", "-html=tests/coverage/coverage.out", "-o=tests/coverage/coverage.html")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return err
	}
	
	// Check coverage threshold
	cmd = exec.Command("go", "tool", "cover", "-func=tests/coverage/coverage.out")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	
	coverage := parseCoverage(string(output))
	fmt.Printf("📈 Coverage: %.2f%%\n", coverage)
	
	if coverage < config.CoverageThreshold {
		return fmt.Errorf("coverage %.2f%% is below threshold %.2f%%", coverage, config.CoverageThreshold)
	}
	
	return nil
}

func getAllPackages() []string {
	cmd := exec.Command("go", "list", "./...")
	output, err := cmd.Output()
	if err != nil {
		return []string{"./..."}
	}
	
	packages := strings.Split(strings.TrimSpace(string(output)), "\n")
	var filtered []string
	
	for _, pkg := range packages {
		if !strings.Contains(pkg, "/vendor/") && !strings.Contains(pkg, "/tests/") {
			filtered = append(filtered, pkg)
		}
	}
	
	return filtered
}

func findIntegrationTests() ([]string, error) {
	var tests []string
	
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if strings.HasSuffix(path, "_integration_test.go") {
			dir := filepath.Dir(path)
			if !contains(tests, dir) {
				tests = append(tests, dir)
			}
		}
		
		return nil
	})
	
	return tests, err
}

func parseCoverage(output string) float64 {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "total:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				var coverage float64
				fmt.Sscanf(parts[2], "%f%%", &coverage)
				return coverage
			}
		}
	}
	return 0.0
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
