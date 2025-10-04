//go:build !linux
// +build !linux

package deterministicvm

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
)

// setNoNewPrivs is a no-op on non-Linux systems
func setNoNewPrivs() error {
	// PR_SET_NO_NEW_PRIVS is Linux-specific
	return fmt.Errorf("PR_SET_NO_NEW_PRIVS not supported on %s", runtime.GOOS)
}

// isLibSeccompAvailable always returns false on non-Linux systems
func isLibSeccompAvailable() bool {
	return false
}

// fallbackSeccompFilter provides a no-op implementation when seccomp is unavailable
type fallbackSeccompFilter struct{}

// createActualSeccompFilter creates a fallback filter for non-Linux systems
func createActualSeccompFilter(workingDir string) (SeccompFilter, error) {
	return &fallbackSeccompFilter{}, nil
}

// Load is a no-op when seccomp is not available
func (f *fallbackSeccompFilter) Load() error {
	// No-op when seccomp is not available
	return nil
}

// Release is a no-op when seccomp is not available
func (f *fallbackSeccompFilter) Release() {
	// No-op when seccomp is not available
}

// TestSeccompViolation can be used in tests to verify seccomp is working
func TestSeccompViolation(forbiddenSyscall string) error {
	switch forbiddenSyscall {
	case "socket":
		// Attempt to create a socket, which should be blocked
		_, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
		return err
	case "fork":
		// Attempt to fork, which should be blocked
		_, err := syscall.ForkExec("/bin/true", []string{"/bin/true"}, nil)
		return err
	case "ptrace":
		// Attempt ptrace, which should be blocked
		return syscall.PtraceAttach(os.Getpid())
	default:
		return fmt.Errorf("unknown test syscall: %s", forbiddenSyscall)
	}
}
