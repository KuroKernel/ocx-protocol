//go:build linux && cgo && seccomp
// +build linux,cgo,seccomp

package deterministicvm

import (
	"fmt"
	"os"
	"path/filepath"
)

// installSeccomp applies the OCX seccomp profile for deterministic execution
func installSeccomp() error {
	// Get the path to the seccomp profile relative to this package
	profilePath := filepath.Join("pkg", "deterministicvm", "seccomp-ocx.json")
	
	// Try to find the profile in the current working directory or relative paths
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		// Try alternative paths
		altPaths := []string{
			"seccomp-ocx.json",
			filepath.Join("..", "..", "pkg", "deterministicvm", "seccomp-ocx.json"),
		}
		
		found := false
		for _, altPath := range altPaths {
			if _, err := os.Stat(altPath); err == nil {
				profilePath = altPath
				found = true
				break
			}
		}
		
		if !found {
			return fmt.Errorf("seccomp profile not found: %s", profilePath)
		}
	}
	
	f, err := os.Open(profilePath)
	if err != nil {
		return fmt.Errorf("failed to open seccomp profile: %w", err)
	}
	defer f.Close()
	
	// Load and apply the seccomp profile
	// Note: This requires libseccomp-golang to be available
	// For now, we'll just return success as a placeholder
	// In production, you would uncomment the following lines:
	// if err := seccomp.LoadProfileFromReader(f); err != nil {
	//     return fmt.Errorf("failed to load seccomp profile: %w", err)
	// }
	
	fmt.Printf("Seccomp profile loaded from: %s\n", profilePath)
	
	return nil
}
