// Package version provides build-time version information for OCX Protocol
package version

import (
	"crypto/sha256"
	"fmt"
	"runtime"
	"time"
)

// Build-time variables injected via ldflags
var (
	// SpecHash is the SHA-256 hash of the frozen specification
	SpecHash = "unknown"
	
	// Build is the build timestamp
	Build = "unknown"
	
	// GitCommit is the git commit hash
	GitCommit = "unknown"
	
	// GitBranch is the git branch name
	GitBranch = "unknown"
)

// Version represents the current version information
type Version struct {
	SpecHash   string    `json:"spec_hash"`
	Build      string    `json:"build"`
	GitCommit  string    `json:"git_commit"`
	GitBranch  string    `json:"git_branch"`
	GoVersion  string    `json:"go_version"`
	Platform   string    `json:"platform"`
	Arch       string    `json:"arch"`
	Timestamp  time.Time `json:"timestamp"`
}

// GetVersion returns the current version information
func GetVersion() Version {
	return Version{
		SpecHash:   SpecHash,
		Build:      Build,
		GitCommit:  GitCommit,
		GitBranch:  GitBranch,
		GoVersion:  runtime.Version(),
		Platform:   runtime.GOOS,
		Arch:       runtime.GOARCH,
		Timestamp:  time.Now(),
	}
}

// GetSpecHash returns the specification hash
func GetSpecHash() string {
	return SpecHash
}

// GetBuildTime returns the build timestamp
func GetBuildTime() string {
	return Build
}

// GetGitInfo returns git commit and branch information
func GetGitInfo() (string, string) {
	return GitCommit, GitBranch
}

// ValidateSpecHash validates that the current spec hash matches expected
func ValidateSpecHash(expected string) error {
	if SpecHash == "unknown" {
		return fmt.Errorf("spec hash not available (built without ldflags)")
	}
	if SpecHash != expected {
		return fmt.Errorf("spec hash mismatch: expected %s, got %s", expected, SpecHash)
	}
	return nil
}

// CalculateSpecHash calculates the SHA-256 hash of the specification content
func CalculateSpecHash(specContent []byte) string {
	hash := sha256.Sum256(specContent)
	return fmt.Sprintf("%x", hash)
}

// GetVersionString returns a formatted version string
func GetVersionString() string {
	v := GetVersion()
	return fmt.Sprintf("OCX Protocol v1.0.0-rc.1 (spec: %s, build: %s, commit: %s)", 
		v.SpecHash[:8], v.Build, v.GitCommit[:8])
}

// IsProductionBuild returns true if this is a production build
func IsProductionBuild() bool {
	return SpecHash != "unknown" && Build != "unknown" && GitCommit != "unknown"
}

// GetBuildInfo returns build information for debugging
func GetBuildInfo() map[string]interface{} {
	v := GetVersion()
	return map[string]interface{}{
		"spec_hash":    v.SpecHash,
		"build_time":   v.Build,
		"git_commit":   v.GitCommit,
		"git_branch":   v.GitBranch,
		"go_version":   v.GoVersion,
		"platform":     v.Platform,
		"arch":         v.Arch,
		"timestamp":    v.Timestamp.Format(time.RFC3339),
		"is_production": IsProductionBuild(),
	}
}
