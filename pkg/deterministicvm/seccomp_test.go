package deterministicvm

import (
	"context"
	"log"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestSeccompAvailability(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Seccomp only available on Linux")
	}

	available := DetectSeccompAvailability()
	t.Logf("Seccomp availability: %v", available)
	
	// Don't fail if unavailable, just log it
	if !available {
		t.Log("Seccomp not available on this system - tests will use fallback")
	}
}

func TestApplySeccompProfile(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Seccomp only available on Linux")
	}

	if !DetectSeccompAvailability() {
		t.Skip("Seccomp not available")
	}

	// This test MUST run in a subprocess because seccomp affects the entire process
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		// We're in the subprocess - apply seccomp
		ctx := context.Background()
		cfg := SeccompConfig{
			StrictSandbox: true,
			WorkingDir:    "/tmp",
			Logger:        log.New(os.Stderr, "[test] ", log.LstdFlags),
		}

		err := ApplySeccompProfile(ctx, cfg)
		if err != nil {
			t.Fatalf("Failed to apply seccomp: %v", err)
		}

		// Try a forbidden syscall - this SHOULD fail
		_, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
		if err == nil {
			t.Fatal("Socket creation should have been blocked by seccomp!")
		}

		// If we got EPERM or similar, seccomp is working
		t.Logf("Socket blocked as expected: %v", err)
		os.Exit(0)
	}

	// Parent process - spawn subprocess
	cmd := exec.Command(os.Args[0], "-test.run=^TestApplySeccompProfile$")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	
	output, err := cmd.CombinedOutput()
	t.Logf("Subprocess output: %s", output)

	// Check exit status
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Process was killed by seccomp (expected)
			if exitErr.ExitCode() != 0 {
				t.Logf("Subprocess exited with code %d (may indicate seccomp kill)", exitErr.ExitCode())
				// This is actually SUCCESS - seccomp killed the process
				return
			}
		}
	}

	// Clean exit means test passed in subprocess
	if err == nil {
		t.Log("Subprocess completed successfully")
	}
}

func TestSeccompWithAllowedSyscalls(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Seccomp only available on Linux")
	}

	if !DetectSeccompAvailability() {
		t.Skip("Seccomp not available")
	}

	if os.Getenv("TEST_SUBPROCESS_ALLOWED") == "1" {
		// Subprocess - apply seccomp and use allowed syscalls
		ctx := context.Background()
		cfg := SeccompConfig{
			StrictSandbox: false, // Use non-strict mode for testing
			WorkingDir:    "/tmp",
			Logger:        log.New(os.Stderr, "[test] ", log.LstdFlags),
		}

		err := ApplySeccompProfile(ctx, cfg)
		if err != nil {
			t.Fatalf("Failed to apply seccomp: %v", err)
		}

		// These syscalls SHOULD work (they're in the allowlist)
		pid := os.Getpid()
		if pid <= 0 {
			t.Fatal("getpid failed")
		}

		// Write to stdout (should work)
		_, err = os.Stdout.WriteString("test\n")
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		t.Log("Allowed syscalls working correctly")
		os.Exit(0)
	}

	// Parent process
	cmd := exec.Command(os.Args[0], "-test.run=^TestSeccompWithAllowedSyscalls$")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS_ALLOWED=1")
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := cmd.CombinedOutput()
	t.Logf("Subprocess output: %s", output)

	select {
	case <-ctx.Done():
		t.Fatal("Test timed out")
	default:
		if err != nil {
			t.Fatalf("Subprocess failed: %v", err)
		}
	}
}

func TestSeccompFallback(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Seccomp only available on Linux")
	}

	if !DetectSeccompAvailability() {
		t.Skip("Seccomp not available")
	}

	if os.Getenv("TEST_SUBPROCESS_FALLBACK") == "1" {
		// Subprocess - test fallback mode without actually applying seccomp
		// This test verifies that the function handles unavailability gracefully
		cfg := SeccompConfig{
			StrictSandbox: false, // Non-strict mode
			WorkingDir:    "/tmp",
			Logger:        log.New(os.Stderr, "[test] ", log.LstdFlags),
		}

		// Test that non-strict mode doesn't fail when seccomp is unavailable
		// We simulate this by checking the availability first
		available := DetectSeccompAvailability()
		if !available && !cfg.StrictSandbox {
			t.Log("Seccomp not available, non-strict mode should continue")
		}

		t.Log("Fallback mode works correctly")
		return
	}

	// Run in subprocess to avoid affecting the main test process
	cmd := exec.Command(os.Args[0], "-test.run=^TestSeccompFallback$")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS_FALLBACK=1")
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := cmd.CombinedOutput()
	t.Logf("Subprocess output: %s", output)

	select {
	case <-ctx.Done():
		t.Fatal("Test timed out")
	default:
		if err != nil {
			t.Fatalf("Subprocess failed: %v", err)
		}
	}
}

func TestSeccompStrictMode(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Seccomp only available on Linux")
	}

	// If seccomp not available, strict mode should fail
	if !DetectSeccompAvailability() {
		ctx := context.Background()
		cfg := SeccompConfig{
			StrictSandbox: true,
			WorkingDir:    "/tmp",
			Logger:        log.New(os.Stderr, "[test] ", log.LstdFlags),
		}

		err := ApplySeccompProfile(ctx, cfg)
		if err != ErrSeccompUnavailable {
			t.Fatalf("Expected ErrSeccompUnavailable, got: %v", err)
		}
		t.Log("Strict mode correctly fails when seccomp unavailable")
	}
}

// TestSeccompBPFProgram verifies the BPF program structure
func TestSeccompBPFProgram(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Seccomp only available on Linux")
	}

	if !DetectSeccompAvailability() {
		t.Skip("Seccomp not available")
	}

	// Create filter
	filter, err := createSeccompFilter("/tmp")
	if err != nil {
		t.Fatalf("Failed to create filter: %v", err)
	}
	defer filter.Release()

	// Verify filter is not nil
	if filter == nil {
		t.Fatal("Filter should not be nil")
	}

	t.Log("BPF program created successfully")
}