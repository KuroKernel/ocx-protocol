//go:build !linux || !cgo || !seccomp
// +build !linux !cgo !seccomp

package deterministicvm

import "fmt"

// installSeccomp is a stub for non-Linux or non-CGO builds
func installSeccomp() error {
	return fmt.Errorf("seccomp not available on this platform")
}
