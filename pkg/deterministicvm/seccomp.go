// Package deterministicvm provides secure sandboxing for VM execution
package deterministicvm

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
)

// SeccompConfig holds seccomp filtering configuration
type SeccompConfig struct {
	StrictSandbox bool   // If true, fail when seccomp is unavailable
	WorkingDir    string // Directory where read-only access is permitted
	Logger        *log.Logger
}

// SeccompError represents seccomp-related errors
type SeccompError struct {
	Op  string
	Err error
}

func (e SeccompError) Error() string {
	return fmt.Sprintf("seccomp %s: %v", e.Op, e.Err)
}

func (e SeccompError) Unwrap() error {
	return e.Err
}

var ErrSeccompUnavailable = &SeccompError{Op: "unavailable", Err: fmt.Errorf("libseccomp not available")}

// DetectSeccompAvailability checks if seccomp filtering is available
func DetectSeccompAvailability() bool {
	// Check if we're on a supported platform
	if runtime.GOOS != "linux" {
		return false
	}

	// Try to detect libseccomp availability by attempting to load it
	return isLibSeccompAvailable()
}

// ApplySeccompProfile applies a strict seccomp profile to the current process
func ApplySeccompProfile(ctx context.Context, cfg SeccompConfig) error {
	logger := cfg.Logger
	if logger == nil {
		logger = log.New(os.Stderr, "[seccomp] ", log.LstdFlags)
	}

	// Check availability first
	if !DetectSeccompAvailability() {
		if cfg.StrictSandbox {
			logger.Printf("Seccomp required but not available")
			return ErrSeccompUnavailable
		}
		logger.Printf("Seccomp not available, continuing without sandbox (strict_mode=false)")
		return nil
	}

	logger.Printf("Applying seccomp sandbox profile (working_dir=%s, strict_mode=%t)",
		cfg.WorkingDir, cfg.StrictSandbox)

	// Set PR_SET_NO_NEW_PRIVS to prevent privilege escalation
	if err := setNoNewPrivs(); err != nil {
		return &SeccompError{Op: "set_no_new_privs", Err: err}
	}

	// Create and apply seccomp filter
	filter, err := createSeccompFilter(cfg.WorkingDir)
	if err != nil {
		return &SeccompError{Op: "create_filter", Err: err}
	}
	defer filter.Release()

	if err := filter.Load(); err != nil {
		return &SeccompError{Op: "load_filter", Err: err}
	}

	logger.Printf("Seccomp sandbox profile applied successfully")
	return nil
}

// createSeccompFilter builds a strict seccomp filter
func createSeccompFilter(workingDir string) (SeccompFilter, error) {
	return createActualSeccompFilter(workingDir)
}

// SeccompFilter interface abstracts seccomp filter operations
type SeccompFilter interface {
	Load() error
	Release()
}
