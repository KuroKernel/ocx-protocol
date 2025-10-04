package security

import (
	"context"
	"testing"
	"time"
)

// TestSecurityHardening tests the security hardening implementation
func TestSecurityHardening(t *testing.T) {
	config := DefaultHardeningConfig()
	hardening := NewSecurityHardening(config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Test hardening application
	if err := hardening.ApplyHardening(ctx); err != nil {
		t.Fatalf("Failed to apply hardening: %v", err)
	}
	
	// Test statistics
	stats := hardening.GetStats()
	if stats.LastHardeningTime.IsZero() {
		t.Error("Last hardening time should be set")
	}
}

// TestSeccompProfiles tests different seccomp profiles
func TestSeccompProfiles(t *testing.T) {
	config := DefaultHardeningConfig()
	hardening := NewSecurityHardening(config)
	
	// Test strict profile
	config.SeccompProfile = StrictProfile
	strictSyscalls := hardening.getAllowedSyscalls(StrictProfile)
	if len(strictSyscalls) == 0 {
		t.Error("Strict profile should have allowed syscalls")
	}
	
	// Test standard profile
	config.SeccompProfile = StandardProfile
	standardSyscalls := hardening.getAllowedSyscalls(StandardProfile)
	if len(standardSyscalls) <= len(strictSyscalls) {
		t.Error("Standard profile should have more syscalls than strict")
	}
	
	// Test permissive profile
	config.SeccompProfile = PermissiveProfile
	permissiveSyscalls := hardening.getAllowedSyscalls(PermissiveProfile)
	if len(permissiveSyscalls) <= len(standardSyscalls) {
		t.Error("Permissive profile should have more syscalls than standard")
	}
	
	// Test network profile
	config.SeccompProfile = NetworkProfile
	networkSyscalls := hardening.getAllowedSyscalls(NetworkProfile)
	if len(networkSyscalls) <= len(standardSyscalls) {
		t.Error("Network profile should have more syscalls than standard")
	}
}

// TestHardeningConfig tests hardening configuration
func TestHardeningConfig(t *testing.T) {
	config := DefaultHardeningConfig()
	
	// Test default values
	if !config.EnableSeccomp {
		t.Error("Seccomp should be enabled by default")
	}
	
	if !config.EnableASLR {
		t.Error("ASLR should be enabled by default")
	}
	
	if !config.EnableStackCanary {
		t.Error("Stack canary should be enabled by default")
	}
	
	if !config.EnableNXBit {
		t.Error("NX bit should be enabled by default")
	}
	
	if !config.EnableNamespaces {
		t.Error("Namespaces should be enabled by default")
	}
	
	if !config.EnableCapabilities {
		t.Error("Capabilities should be enabled by default")
	}
	
	if !config.BlockNetwork {
		t.Error("Network should be blocked by default")
	}
	
	if !config.BlockIPC {
		t.Error("IPC should be blocked by default")
	}
	
	if !config.EnableDualVerify {
		t.Error("Dual verification should be enabled by default")
	}
	
	if !config.EnableKeyRotation {
		t.Error("Key rotation should be enabled by default")
	}
	
	if !config.EnableAuditLog {
		t.Error("Audit logging should be enabled by default")
	}
	
	if !config.EnableMetrics {
		t.Error("Metrics should be enabled by default")
	}
	
	if !config.EnableAlerts {
		t.Error("Alerts should be enabled by default")
	}
	
	// Test resource limits
	if config.MaxMemoryMB != 256 {
		t.Errorf("Expected MaxMemoryMB to be 256, got %d", config.MaxMemoryMB)
	}
	
	if config.MaxCPUSeconds != 60 {
		t.Errorf("Expected MaxCPUSeconds to be 60, got %d", config.MaxCPUSeconds)
	}
	
	if config.MaxFileDescriptors != 1024 {
		t.Errorf("Expected MaxFileDescriptors to be 1024, got %d", config.MaxFileDescriptors)
	}
	
	if config.MaxProcesses != 64 {
		t.Errorf("Expected MaxProcesses to be 64, got %d", config.MaxProcesses)
	}
	
	if config.KeyRotationHours != 24 {
		t.Errorf("Expected KeyRotationHours to be 24, got %d", config.KeyRotationHours)
	}
}

// TestHardeningStats tests hardening statistics
func TestHardeningStats(t *testing.T) {
	config := DefaultHardeningConfig()
	hardening := NewSecurityHardening(config)
	
	stats := hardening.GetStats()
	
	// Test initial values
	if stats.SeccompViolations != 0 {
		t.Error("Initial seccomp violations should be 0")
	}
	
	if stats.ResourceLimitHits != 0 {
		t.Error("Initial resource limit hits should be 0")
	}
	
	if stats.MemoryProtectionHits != 0 {
		t.Error("Initial memory protection hits should be 0")
	}
	
	if stats.NetworkBlocked != 0 {
		t.Error("Initial network blocked should be 0")
	}
	
	if stats.SecurityAlerts != 0 {
		t.Error("Initial security alerts should be 0")
	}
	
	if !stats.LastHardeningTime.IsZero() {
		t.Error("Initial last hardening time should be zero")
	}
}

// TestSecurityProfiles tests security profile configurations
func TestSecurityProfiles(t *testing.T) {
	// Test strict profile configuration
	strictConfig := HardeningConfig{
		EnableSeccomp:      true,
		SeccompProfile:     StrictProfile,
		StrictMode:         true,
		EnableASLR:         true,
		EnableStackCanary:  true,
		EnableNXBit:        true,
		EnableNamespaces:   true,
		EnableCapabilities: true,
		EnableChroot:       true,
		MaxMemoryMB:        128,
		MaxCPUSeconds:      30,
		MaxFileDescriptors: 512,
		MaxProcesses:       32,
		BlockNetwork:       true,
		BlockIPC:           true,
		EnableDualVerify:   true,
		EnableKeyRotation:  true,
		KeyRotationHours:   12,
		EnableAuditLog:     true,
		EnableMetrics:      true,
		EnableAlerts:       true,
	}
	
	strictHardening := NewSecurityHardening(strictConfig)
	strictSyscalls := strictHardening.getAllowedSyscalls(StrictProfile)
	
	if len(strictSyscalls) == 0 {
		t.Error("Strict profile should have allowed syscalls")
	}
	
	// Test permissive profile configuration
	permissiveConfig := HardeningConfig{
		EnableSeccomp:      true,
		SeccompProfile:     PermissiveProfile,
		StrictMode:         false,
		EnableASLR:         false,
		EnableStackCanary:  false,
		EnableNXBit:        false,
		EnableNamespaces:   false,
		EnableCapabilities: false,
		EnableChroot:       false,
		MaxMemoryMB:        1024,
		MaxCPUSeconds:      300,
		MaxFileDescriptors: 4096,
		MaxProcesses:       256,
		BlockNetwork:       false,
		BlockIPC:           false,
		EnableDualVerify:   false,
		EnableKeyRotation:  false,
		KeyRotationHours:   168, // 1 week
		EnableAuditLog:     false,
		EnableMetrics:      false,
		EnableAlerts:       false,
	}
	
	permissiveHardening := NewSecurityHardening(permissiveConfig)
	permissiveSyscalls := permissiveHardening.getAllowedSyscalls(PermissiveProfile)
	
	if len(permissiveSyscalls) <= len(strictSyscalls) {
		t.Error("Permissive profile should have more syscalls than strict")
	}
}

// TestHardeningIntegration tests hardening integration
func TestHardeningIntegration(t *testing.T) {
	config := DefaultHardeningConfig()
	hardening := NewSecurityHardening(config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Test hardening application
	if err := hardening.ApplyHardening(ctx); err != nil {
		t.Fatalf("Failed to apply hardening: %v", err)
	}
	
	// Test statistics after hardening
	stats := hardening.GetStats()
	if stats.LastHardeningTime.IsZero() {
		t.Error("Last hardening time should be set after hardening")
	}
	
	// Test that hardening can be applied multiple times
	if err := hardening.ApplyHardening(ctx); err != nil {
		t.Fatalf("Failed to apply hardening second time: %v", err)
	}
}

// BenchmarkHardening benchmarks hardening performance
func BenchmarkHardening(b *testing.B) {
	config := DefaultHardeningConfig()
	hardening := NewSecurityHardening(config)
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := hardening.ApplyHardening(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSeccompProfiles benchmarks seccomp profile generation
func BenchmarkSeccompProfiles(b *testing.B) {
	config := DefaultHardeningConfig()
	hardening := NewSecurityHardening(config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hardening.getAllowedSyscalls(StandardProfile)
	}
}

// TestHardeningErrorHandling tests error handling in hardening
func TestHardeningErrorHandling(t *testing.T) {
	// Test with invalid configuration
	config := HardeningConfig{
		EnableSeccomp:   true,
		SeccompProfile:  SeccompProfile(999), // Invalid profile
		StrictMode:      true,
		MaxMemoryMB:     -1, // Invalid memory limit
		MaxCPUSeconds:   -1, // Invalid CPU limit
		MaxProcesses:    -1, // Invalid process limit
	}
	
	hardening := NewSecurityHardening(config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// This should handle errors gracefully
	err := hardening.ApplyHardening(ctx)
	if err != nil {
		// Error is expected for invalid configuration
		t.Logf("Expected error for invalid configuration: %v", err)
	}
}

// TestHardeningConcurrency tests hardening under concurrent access
func TestHardeningConcurrency(t *testing.T) {
	config := DefaultHardeningConfig()
	hardening := NewSecurityHardening(config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Test concurrent hardening application
	done := make(chan error, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			err := hardening.ApplyHardening(ctx)
			done <- err
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Concurrent hardening failed: %v", err)
			}
		case <-ctx.Done():
			t.Fatal("Timeout waiting for concurrent hardening")
		}
	}
}
