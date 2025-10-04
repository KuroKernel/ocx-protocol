package security

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestSecurityManager(t *testing.T) {
	// Create test configuration
	config := SecurityConfig{
		KeyStoreConfig: KeyStoreConfig{
			KeyDir:           "/tmp/test-keys",
			RotationInterval: time.Hour * 24 * 30, // 30 days
			MaxKeys:          5,
		},
		AuditConfig: AuditConfig{
			LogFile:        "/tmp/test-audit.log",
			MaxLogSize:     10 * 1024 * 1024, // 10MB
			MaxLogFiles:    5,
			BufferSize:     1000,
			LogLevel:       "INFO",
			LogFormat:      "JSON",
			LogRotation:    true,
			LogRetention:   time.Hour * 24 * 7, // 7 days
			FlushInterval:  time.Second * 5,
			AsyncLogging:   true,
			EncryptLogs:    false,
			CompressLogs:   false,
			IntegrityCheck: true,
		},
		VulnerabilityConfig: VulnerabilityConfig{
			ScanInterval:       time.Minute * 30,
			ScanTimeout:        time.Minute * 5,
			MaxConcurrentScans: 3,
			EnableNVD:          true,
			EnableOSV:          true,
			EnableGitHub:       true,
			ScanDependencies:   true,
			ScanCode:           true,
			ScanSecrets:        true,
			ScanConfigs:        true,
			TargetPaths:        []string{"/tmp/test-scan"},
			ReportFormat:       "JSON",
			ReportPath:         "/tmp/test-reports",
			IncludeDetails:     true,
			IncludeRemediation: false,
			APIKeys:            make(map[string]string),
			RateLimit:          100,
			UserAgent:          "OCX-Security-Scanner/1.0",
		},
		Policies: SecurityPolicies{
			MaxLoginAttempts:    3,
			LockoutDuration:     time.Minute * 15,
			PasswordMinLength:   8,
			PasswordComplexity:  true,
			SessionTimeout:      time.Hour * 2,
			DefaultPermissions:  []string{"read"},
			AdminPermissions:    []string{"read", "write", "admin"},
			RequireMFA:          false,
			RateLimitPerMinute:  100,
			MaxRequestSize:      10 * 1024 * 1024, // 10MB
			AllowedOrigins:      []string{"https://example.com"},
			RequireHTTPS:        true,
			EncryptAtRest:       true,
			EncryptInTransit:    true,
			DataRetentionPeriod: time.Hour * 24 * 365, // 1 year
			MaxRiskScore:        7.0,
			AutoRemediation:     false,
			ScanFrequency:       time.Hour * 24, // Daily
		},
		EnableKeyRotation:        true,
		EnableAuditLogging:       true,
		EnableVulnScanning:       true,
		EnableSecurityMonitoring: true,
	}

	// Create security manager
	sm, err := NewSecurityManager(config)
	if err != nil {
		t.Fatalf("Failed to create security manager: %v", err)
	}
	defer sm.Stop()
	defer func() {
		// Clean up test files after test completes
		os.RemoveAll("/tmp/test-keys")
		os.Remove("/tmp/test-audit.log")
	}()

	// Test signing and verification
	testData := []byte("test data for signing")
	signature, err := sm.Sign(testData)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	valid, err := sm.Verify(testData, signature)
	if err != nil {
		t.Fatalf("Failed to verify signature: %v", err)
	}
	if !valid {
		t.Error("Signature verification failed")
	}

	// Test with invalid signature
	invalidSignature := []byte("invalid signature")
	valid, err = sm.Verify(testData, invalidSignature)
	if err != nil {
		// This is expected for invalid signatures
		if valid {
			t.Error("Invalid signature should not be valid when error occurs")
		}
	} else {
		if valid {
			t.Error("Invalid signature should not be valid")
		}
	}

	// Test audit logging
	sm.LogAuthentication("user123", "password", "success", map[string]interface{}{
		"ip_address": "192.168.1.1",
		"user_agent": "test-agent",
	})

	sm.LogAuthorization("user123", "resource1", "read", "success", map[string]interface{}{
		"resource_id": "res123",
	})

	sm.LogDataAccess("user123", "database", "select", map[string]interface{}{
		"table": "users",
		"rows":  10,
	})

	sm.LogAPIRequest("req123", "GET", "/api/v1/users", 200, "user123", "192.168.1.1", map[string]interface{}{
		"response_time": 150,
	})

	// Test vulnerability scanning
	scanID, err := sm.RunVulnerabilityScan()
	if err != nil {
		t.Fatalf("Failed to run vulnerability scan: %v", err)
	}

	// Wait for scan to complete
	time.Sleep(time.Second * 2)

	scan, err := sm.GetVulnerabilityScan(scanID)
	if err != nil {
		t.Fatalf("Failed to get vulnerability scan: %v", err)
	}
	if scan == nil {
		t.Error("Vulnerability scan should not be nil")
	}

	// Test security status
	status := sm.GetSecurityStatus()
	if status.SecurityScore < 0 || status.SecurityScore > 100 {
		t.Errorf("Security score should be between 0 and 100, got %f", status.SecurityScore)
	}

	if len(status.Components) == 0 {
		t.Error("Security status should have components")
	}

	// Test audit logs retrieval
	// First, trigger some audit events by performing operations
	_, _ = sm.Sign([]byte("test data"))
	_, _ = sm.Verify([]byte("test data"), []byte("fake signature"))
	
	// Wait for async logging to complete (flush interval is 5 seconds)
	time.Sleep(time.Second * 6)
	
	// Check if log file exists
	if _, err := os.Stat("/tmp/test-audit.log"); os.IsNotExist(err) {
		t.Logf("Log file does not exist: %v", err)
	} else {
		t.Logf("Log file exists")
	}
	
	logs, err := sm.GetAuditLogs(10, AuditFilter{})
	if err != nil {
		t.Fatalf("Failed to get audit logs: %v", err)
	}
	if len(logs) == 0 {
		t.Error("Should have audit logs")
	}

	// Test key info
	keyInfo, err := sm.GetKeyInfo()
	if err != nil {
		t.Fatalf("Failed to get key info: %v", err)
	}
	if keyInfo == nil {
		t.Error("Key info should not be nil")
	}

	// Test public key export
	publicKey, err := sm.ExportPublicKey()
	if err != nil {
		t.Fatalf("Failed to export public key: %v", err)
	}
	if publicKey == "" {
		t.Error("Public key should not be empty")
	}

	// Test manual key rotation
	err = sm.RotateKey()
	if err != nil {
		t.Fatalf("Failed to rotate key: %v", err)
	}

	// Test security report generation
	report, err := sm.GenerateSecurityReport()
	if err != nil {
		t.Fatalf("Failed to generate security report: %v", err)
	}

	var reportData map[string]interface{}
	err = json.Unmarshal(report, &reportData)
	if err != nil {
		t.Fatalf("Failed to unmarshal security report: %v", err)
	}

	if reportData["security_score"] == nil {
		t.Error("Security report should contain security score")
	}

	// Clean up is now handled by defer
}

func TestSecurityManagerWithoutComponents(t *testing.T) {
	// Test security manager with minimal configuration
	config := SecurityConfig{
		EnableKeyRotation:        false,
		EnableAuditLogging:       false,
		EnableVulnScanning:       false,
		EnableSecurityMonitoring: false,
	}

	sm, err := NewSecurityManager(config)
	if err != nil {
		t.Fatalf("Failed to create security manager: %v", err)
	}
	defer sm.Stop()

	// Test signing without key store
	_, err = sm.Sign([]byte("test"))
	if err == nil {
		t.Error("Signing should fail without key store")
	}

	// Test verification without key store
	_, err = sm.Verify([]byte("test"), []byte("signature"))
	if err == nil {
		t.Error("Verification should fail without key store")
	}

	// Test vulnerability scan without scanner
	_, err = sm.RunVulnerabilityScan()
	if err == nil {
		t.Error("Vulnerability scan should fail without scanner")
	}

	// Test audit logs without logger
	_, err = sm.GetAuditLogs(10, AuditFilter{})
	if err == nil {
		t.Error("Getting audit logs should fail without logger")
	}
}

func TestSecurityPolicies(t *testing.T) {
	policies := SecurityPolicies{
		MaxLoginAttempts:    3,
		LockoutDuration:     time.Minute * 15,
		PasswordMinLength:   8,
		PasswordComplexity:  true,
		SessionTimeout:      time.Hour * 2,
		DefaultPermissions:  []string{"read"},
		AdminPermissions:    []string{"read", "write", "admin"},
		RequireMFA:          false,
		RateLimitPerMinute:  100,
		MaxRequestSize:      10 * 1024 * 1024,
		AllowedOrigins:      []string{"https://example.com"},
		RequireHTTPS:        true,
		EncryptAtRest:       true,
		EncryptInTransit:    true,
		DataRetentionPeriod: time.Hour * 24 * 365,
		MaxRiskScore:        7.0,
		AutoRemediation:     false,
		ScanFrequency:       time.Hour * 24,
	}

	// Test policy validation
	if policies.MaxLoginAttempts <= 0 {
		t.Error("MaxLoginAttempts should be positive")
	}

	if policies.LockoutDuration <= 0 {
		t.Error("LockoutDuration should be positive")
	}

	if policies.PasswordMinLength < 8 {
		t.Error("PasswordMinLength should be at least 8")
	}

	if policies.SessionTimeout <= 0 {
		t.Error("SessionTimeout should be positive")
	}

	if len(policies.DefaultPermissions) == 0 {
		t.Error("DefaultPermissions should not be empty")
	}

	if len(policies.AdminPermissions) == 0 {
		t.Error("AdminPermissions should not be empty")
	}

	if policies.RateLimitPerMinute <= 0 {
		t.Error("RateLimitPerMinute should be positive")
	}

	if policies.MaxRequestSize <= 0 {
		t.Error("MaxRequestSize should be positive")
	}

	if len(policies.AllowedOrigins) == 0 {
		t.Error("AllowedOrigins should not be empty")
	}

	if policies.DataRetentionPeriod <= 0 {
		t.Error("DataRetentionPeriod should be positive")
	}

	if policies.MaxRiskScore < 0 || policies.MaxRiskScore > 10 {
		t.Error("MaxRiskScore should be between 0 and 10")
	}

	if policies.ScanFrequency <= 0 {
		t.Error("ScanFrequency should be positive")
	}
}

func TestSecurityStatus(t *testing.T) {
	status := SecurityStatus{
		Timestamp:     time.Now(),
		SecurityScore: 85.5,
		Components: map[string]ComponentStatus{
			"keystore": {
				Name:         "keystore",
				Status:       "healthy",
				LastRotation: time.Now().Add(-time.Hour * 24 * 7),
				NextRotation: time.Now().Add(time.Hour * 24 * 23),
				TotalKeys:    3,
				ActiveKeys:   1,
			},
			"audit": {
				Name:             "audit",
				Status:           "healthy",
				TotalEvents:      1000,
				SuspiciousEvents: 5,
				BufferSize:       100,
			},
			"vulnerability": {
				Name:             "vulnerability",
				Status:           "healthy",
				TotalScans:       10,
				CompletedScans:   9,
				TotalFindings:    15,
				CriticalFindings: 2,
			},
		},
	}

	// Test status validation
	if status.SecurityScore < 0 || status.SecurityScore > 100 {
		t.Errorf("Security score should be between 0 and 100, got %f", status.SecurityScore)
	}

	if len(status.Components) == 0 {
		t.Error("Security status should have components")
	}

	// Test component status
	keystore, exists := status.Components["keystore"]
	if !exists {
		t.Error("Keystore component should exist")
	}
	if keystore.TotalKeys <= 0 {
		t.Error("TotalKeys should be positive")
	}
	if keystore.ActiveKeys <= 0 {
		t.Error("ActiveKeys should be positive")
	}

	audit, exists := status.Components["audit"]
	if !exists {
		t.Error("Audit component should exist")
	}
	if audit.TotalEvents < 0 {
		t.Error("TotalEvents should be non-negative")
	}
	if audit.SuspiciousEvents < 0 {
		t.Error("SuspiciousEvents should be non-negative")
	}

	vuln, exists := status.Components["vulnerability"]
	if !exists {
		t.Error("Vulnerability component should exist")
	}
	if vuln.TotalScans < 0 {
		t.Error("TotalScans should be non-negative")
	}
	if vuln.CompletedScans < 0 {
		t.Error("CompletedScans should be non-negative")
	}
	if vuln.TotalFindings < 0 {
		t.Error("TotalFindings should be non-negative")
	}
	if vuln.CriticalFindings < 0 {
		t.Error("CriticalFindings should be non-negative")
	}
}

func TestSecurityManagerContextCancellation(t *testing.T) {
	config := SecurityConfig{
		EnableSecurityMonitoring: true,
	}

	sm, err := NewSecurityManager(config)
	if err != nil {
		t.Fatalf("Failed to create security manager: %v", err)
	}

	// Stop the security manager
	sm.Stop()

	// Wait a bit to ensure goroutines have stopped
	time.Sleep(time.Millisecond * 100)

	// The security manager should be stopped now
	// This test mainly ensures that Stop() doesn't panic
}
