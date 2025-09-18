package safety

import (
	"time"
)

// SafetyConfig defines safety configuration parameters
type SafetyConfig struct {
	MaxFunctionLines     int           `json:"max_function_lines"`
	MaxLoopIterations    int           `json:"max_loop_iterations"`
	MaxRecursionDepth    int           `json:"max_recursion_depth"`
	MaxExecutionTime     time.Duration `json:"max_execution_time"`
	MaxMemoryAllocation  int64         `json:"max_memory_allocation"`
	EnablePanicRecovery  bool          `json:"enable_panic_recovery"`
	EnableTimeoutChecks  bool          `json:"enable_timeout_checks"`
	EnableMemoryChecks   bool          `json:"enable_memory_checks"`
}

// DefaultSafetyConfig returns the default safety configuration
func DefaultSafetyConfig() *SafetyConfig {
	return &SafetyConfig{
		MaxFunctionLines:     50,
		MaxLoopIterations:    1000,
		MaxRecursionDepth:    10,
		MaxExecutionTime:     30 * time.Second,
		MaxMemoryAllocation:  100 * 1024 * 1024, // 100MB
		EnablePanicRecovery:  true,
		EnableTimeoutChecks:  true,
		EnableMemoryChecks:   true,
	}
}

// SafetyReport represents the overall safety report for the codebase
type SafetyReport struct {
	Summary struct {
		TotalFiles       int `json:"total_files"`
		ValidFiles       int `json:"valid_files"`
		InvalidFiles     int `json:"invalid_files"`
		TotalViolations  int `json:"total_violations"`
		UnhandledErrors  int `json:"unhandled_errors"`
		LongFunctions    int `json:"long_functions"`
		UnsafeLoops      int `json:"unsafe_loops"`
		HeapViolations   int `json:"heap_violations"`
		ScopeViolations  int `json:"scope_violations"`
		Other            int `json:"other"`
	} `json:"summary"`
	FileReports map[string]*ValidationResult `json:"file_reports"`
}

// ValidationResult represents the validation result for a single file
type ValidationResult struct {
	IsValid         bool     `json:"is_valid"`
	Violations      []string `json:"violations"`
	Warnings        []string `json:"warnings"`
	FunctionCount   int      `json:"function_count"`
	LongFunctions   []string `json:"long_functions"`
	UnsafeLoops     []string `json:"unsafe_loops"`
	UnhandledErrors []string `json:"unhandled_errors"`
}

// SafetyLevel represents the level of safety enforcement
type SafetyLevel int

const (
	SafetyLevelDisabled SafetyLevel = iota
	SafetyLevelBasic
	SafetyLevelStrict
	SafetyLevelParanoid
)

// String returns the string representation of SafetyLevel
func (sl SafetyLevel) String() string {
	switch sl {
	case SafetyLevelDisabled:
		return "disabled"
	case SafetyLevelBasic:
		return "basic"
	case SafetyLevelStrict:
		return "strict"
	case SafetyLevelParanoid:
		return "paranoid"
	default:
		return "unknown"
	}
}

// GetSafetyConfig returns the safety configuration for a given level
func GetSafetyConfig(level SafetyLevel) *SafetyConfig {
	config := DefaultSafetyConfig()
	
	switch level {
	case SafetyLevelDisabled:
		config.EnablePanicRecovery = false
		config.EnableTimeoutChecks = false
		config.EnableMemoryChecks = false
	case SafetyLevelBasic:
		config.MaxFunctionLines = 100
		config.MaxLoopIterations = 10000
		config.MaxRecursionDepth = 20
		config.MaxExecutionTime = 60 * time.Second
		config.MaxMemoryAllocation = 500 * 1024 * 1024 // 500MB
	case SafetyLevelStrict:
		// Use defaults
	case SafetyLevelParanoid:
		config.MaxFunctionLines = 25
		config.MaxLoopIterations = 100
		config.MaxRecursionDepth = 5
		config.MaxExecutionTime = 10 * time.Second
		config.MaxMemoryAllocation = 50 * 1024 * 1024 // 50MB
	}
	
	return config
}
