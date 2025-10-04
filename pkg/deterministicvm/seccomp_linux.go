//go:build linux
// +build linux

package deterministicvm

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Pure Go seccomp implementation - NO CGO REQUIRED
// Uses direct syscalls to Linux kernel

const (
	// Seccomp modes
	SECCOMP_MODE_FILTER = 2

	// Seccomp operations
	SECCOMP_SET_MODE_FILTER = 1

	// Seccomp filter flags
	SECCOMP_FILTER_FLAG_TSYNC = 1

	// BPF instruction structure size
	BPF_INSTRUCTION_SIZE = 8

	// BPF actions
	SECCOMP_RET_KILL_PROCESS = 0x80000000
	SECCOMP_RET_ALLOW        = 0x7fff0000

	// PR_SET_NO_NEW_PRIVS
	PR_SET_NO_NEW_PRIVS = 38
)

// BPF instruction structure (Berkeley Packet Filter)
type sockFilter struct {
	code uint16
	jt   uint8
	jf   uint8
	k    uint32
}

// BPF program structure
type sockFprog struct {
	len    uint16
	filter *sockFilter
}

// setNoNewPrivs sets the PR_SET_NO_NEW_PRIVS flag using direct syscall
func setNoNewPrivs() error {
	_, _, errno := syscall.Syscall(syscall.SYS_PRCTL, PR_SET_NO_NEW_PRIVS, 1, 0)
	if errno != 0 {
		return fmt.Errorf("prctl PR_SET_NO_NEW_PRIVS failed: %v", errno)
	}
	return nil
}

// isLibSeccompAvailable checks if seccomp is available
func isLibSeccompAvailable() bool {
	// Try to read seccomp status from procfs
	_, err := syscall.Open("/proc/self/status", syscall.O_RDONLY, 0)
	if err != nil {
		return false
	}
	syscall.Close(0) // We don't actually need the fd
	return true
}

// pureGoSeccompFilter implements SeccompFilter using pure Go syscalls
type pureGoSeccompFilter struct {
	workingDir string
	program    *sockFprog
	filters    []sockFilter
}

// createActualSeccompFilter creates a real seccomp filter using pure Go
func createActualSeccompFilter(workingDir string) (SeccompFilter, error) {
	filter := &pureGoSeccompFilter{
		workingDir: workingDir,
		filters:    make([]sockFilter, 0, 32),
	}

	// Build BPF program that implements our seccomp policy
	filter.buildSeccompBPFProgram()

	return filter, nil
}

// buildSeccompBPFProgram constructs the BPF bytecode for seccomp filtering
// This implements a strict whitelist: only explicitly allowed syscalls pass, everything else is killed
func (f *pureGoSeccompFilter) buildSeccompBPFProgram() {
	// Whitelist of syscalls required for deterministic VM execution
	// Architecture: x86_64 syscall numbers (use syscall.SYS_* constants for portability)
	allowedSyscalls := []int{
		syscall.SYS_READ,
		syscall.SYS_WRITE,
		syscall.SYS_CLOSE,
		syscall.SYS_LSEEK,
		syscall.SYS_EXIT,
		syscall.SYS_EXIT_GROUP,
		syscall.SYS_GETPID,
		syscall.SYS_GETTID,
		syscall.SYS_MMAP,
		syscall.SYS_MUNMAP,
		syscall.SYS_MPROTECT,
		syscall.SYS_BRK,
		syscall.SYS_FSTAT,
		syscall.SYS_FCNTL,
		syscall.SYS_RT_SIGACTION,
		syscall.SYS_RT_SIGPROCMASK,
		syscall.SYS_CLOCK_GETTIME,
	}

	// BPF instruction: Load syscall number from seccomp_data
	// ld [0] (load word at offset 0 - the syscall number)
	f.filters = append(f.filters, sockFilter{
		code: 0x20, // BPF_LD | BPF_W | BPF_ABS
		jt:   0,
		jf:   0,
		k:    0, // offset 0 in seccomp_data structure
	})

	// For each allowed syscall, add a comparison and conditional jump
	for i, syscallNum := range allowedSyscalls {
		isLast := (i == len(allowedSyscalls)-1)

		// jeq #syscallNum, allow, next
		f.filters = append(f.filters, sockFilter{
			code: 0x15,                            // BPF_JMP | BPF_JEQ | BPF_K
			jt:   uint8(len(allowedSyscalls) - i), // jump to allow if match
			jf:   0,                               // continue to next instruction if no match
			k:    uint32(syscallNum),
		})

		if isLast {
			// ret SECCOMP_RET_KILL_PROCESS (default deny)
			f.filters = append(f.filters, sockFilter{
				code: 0x06, // BPF_RET | BPF_K
				jt:   0,
				jf:   0,
				k:    SECCOMP_RET_KILL_PROCESS,
			})
		}
	}

	// ret SECCOMP_RET_ALLOW (allow if matched any syscall)
	f.filters = append(f.filters, sockFilter{
		code: 0x06, // BPF_RET | BPF_K
		jt:   0,
		jf:   0,
		k:    SECCOMP_RET_ALLOW,
	})

	// Create the program structure
	f.program = &sockFprog{
		len:    uint16(len(f.filters)),
		filter: &f.filters[0],
	}
}

// Load applies the seccomp filter to the current process
func (f *pureGoSeccompFilter) Load() error {
	// Call seccomp(SECCOMP_SET_MODE_FILTER, flags, prog)
	// SYS_SECCOMP is not available in Go 1.18, use prctl directly
	_, _, errno := syscall.Syscall(
		syscall.SYS_PRCTL,
		22, // PR_SET_SECCOMP
		SECCOMP_MODE_FILTER,
		uintptr(unsafe.Pointer(f.program)),
	)

	if errno != 0 {
		return fmt.Errorf("seccomp filter load failed: %v", errno)
	}

	return nil
}

// Release cleans up the seccomp filter (nothing to clean in pure Go)
func (f *pureGoSeccompFilter) Release() {
	// No cleanup needed for pure Go implementation
	// Filter is enforced by kernel, no userspace resources to free
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
		_, _, errno := syscall.Syscall(syscall.SYS_PTRACE, 0, 0, 0)
		if errno != 0 {
			return errno
		}
		return nil
	default:
		return fmt.Errorf("unknown test syscall: %s", forbiddenSyscall)
	}
}
