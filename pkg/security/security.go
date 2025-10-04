package security

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// SecurityManager provides comprehensive security management
type SecurityManager struct {
	// Components
	keyStore    *KeyStore
	auditLogger *AuditLogger
	vulnScanner *VulnerabilityScanner

	// Configuration
	config SecurityConfig

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// SecurityConfig defines configuration for the security manager
type SecurityConfig struct {
	// Key management
	KeyStoreConfig KeyStoreConfig `json:"key_store_config"`

	// Audit logging
	AuditConfig AuditConfig `json:"audit_config"`

	// Vulnerability scanning
	VulnerabilityConfig VulnerabilityConfig `json:"vulnerability_config"`

	// Security policies
	Policies SecurityPolicies `json:"policies"`

	// Integration settings
	EnableKeyRotation        bool `json:"enable_key_rotation"`
	EnableAuditLogging       bool `json:"enable_audit_logging"`
	EnableVulnScanning       bool `json:"enable_vuln_scanning"`
	EnableSecurityMonitoring bool `json:"enable_security_monitoring"`
}

// SecurityPolicies defines security policies
type SecurityPolicies struct {
	// Authentication policies
	MaxLoginAttempts   int           `json:"max_login_attempts"`
	LockoutDuration    time.Duration `json:"lockout_duration"`
	PasswordMinLength  int           `json:"password_min_length"`
	PasswordComplexity bool          `json:"password_complexity"`
	SessionTimeout     time.Duration `json:"session_timeout"`

	// Authorization policies
	DefaultPermissions []string `json:"default_permissions"`
	AdminPermissions   []string `json:"admin_permissions"`
	RequireMFA         bool     `json:"require_mfa"`

	// API security policies
	RateLimitPerMinute int      `json:"rate_limit_per_minute"`
	MaxRequestSize     int64    `json:"max_request_size"`
	AllowedOrigins     []string `json:"allowed_origins"`
	RequireHTTPS       bool     `json:"require_https"`

	// Data protection policies
	EncryptAtRest       bool          `json:"encrypt_at_rest"`
	EncryptInTransit    bool          `json:"encrypt_in_transit"`
	DataRetentionPeriod time.Duration `json:"data_retention_period"`

	// Vulnerability policies
	MaxRiskScore    float64       `json:"max_risk_score"`
	AutoRemediation bool          `json:"auto_remediation"`
	ScanFrequency   time.Duration `json:"scan_frequency"`
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(config SecurityConfig) (*SecurityManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	sm := &SecurityManager{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize key store
	if config.EnableKeyRotation {
		var err error
		sm.keyStore, err = NewKeyStore(config.KeyStoreConfig.KeyDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create key store: %w", err)
		}
		
		// Generate initial key if none exist
		keys, err := sm.keyStore.ListKeys()
		if err != nil || len(keys) == 0 {
			_, _, err = sm.keyStore.GenerateKey("initial-key")
			if err != nil {
				return nil, fmt.Errorf("failed to generate initial key: %w", err)
			}
		}
		
		sm.keyStore.StartAutoRotation()
	}

	// Initialize audit logger
	if config.EnableAuditLogging {
		auditLogger, err := NewAuditLogger(config.AuditConfig)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create audit logger: %w", err)
		}
		sm.auditLogger = auditLogger
	}

	// Initialize vulnerability scanner
	if config.EnableVulnScanning {
		sm.vulnScanner = NewVulnerabilityScanner(config.VulnerabilityConfig)
		sm.vulnScanner.StartScheduledScans()
	}

	// Start security monitoring
	if config.EnableSecurityMonitoring {
		sm.wg.Add(1)
		go sm.securityMonitoringLoop()
	}

	return sm, nil
}

// securityMonitoringLoop runs continuous security monitoring
func (sm *SecurityManager) securityMonitoringLoop() {
	defer sm.wg.Done()

	ticker := time.NewTicker(time.Minute * 5) // Check every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.performSecurityChecks()
		}
	}
}

// performSecurityChecks performs various security checks
func (sm *SecurityManager) performSecurityChecks() {
	// Check key rotation status
	if sm.keyStore != nil {
		stats := sm.keyStore.GetKeyStatistics()
		if time.Until(stats.NextRotation) < time.Hour*24 {
			sm.logSecurityEvent("KEY_ROTATION_WARNING",
				fmt.Sprintf("Key rotation due in %v", time.Until(stats.NextRotation)),
				"MEDIUM", nil)
		}
	}

	// Check vulnerability scan results
	if sm.vulnScanner != nil {
		latestScan, err := sm.vulnScanner.GetLatestScan()
		if err == nil && latestScan.Status == "COMPLETED" {
			if latestScan.Summary.RiskScore > sm.config.Policies.MaxRiskScore {
				sm.logSecurityEvent("HIGH_RISK_VULNERABILITIES",
					fmt.Sprintf("Risk score %.2f exceeds threshold %.2f",
						latestScan.Summary.RiskScore, sm.config.Policies.MaxRiskScore),
					"HIGH", map[string]interface{}{
						"risk_score": latestScan.Summary.RiskScore,
						"threshold":  sm.config.Policies.MaxRiskScore,
						"scan_id":    latestScan.ID,
					})
			}
		}
	}
}

// Sign signs data with the current key
func (sm *SecurityManager) Sign(data []byte) ([]byte, error) {
	if sm.keyStore == nil {
		return nil, fmt.Errorf("key store not available")
	}

	signature, err := sm.keyStore.Sign(data)
	if err != nil {
		sm.logSecurityEvent("SIGNATURE_FAILED",
			fmt.Sprintf("Failed to sign data: %v", err),
			"HIGH", map[string]interface{}{
				"error": err.Error(),
			})
		return nil, err
	}

	// Log successful signing
	sm.logSecurityEvent("DATA_SIGNED",
		"Data successfully signed",
		"LOW", map[string]interface{}{
			"data_size": len(data),
		})

	return signature, nil
}

// Verify verifies a signature
func (sm *SecurityManager) Verify(data, signature []byte) (bool, error) {
	if sm.keyStore == nil {
		return false, fmt.Errorf("key store not available")
	}

	valid, err := sm.keyStore.VerifyWithCurrentKey(data, signature)
	if err != nil {
		sm.logSecurityEvent("VERIFICATION_FAILED",
			fmt.Sprintf("Failed to verify signature: %v", err),
			"HIGH", map[string]interface{}{
				"error": err.Error(),
			})
		return false, err
	}

	// Log verification result
	eventType := "VERIFICATION_SUCCESS"
	riskLevel := "LOW"
	if !valid {
		eventType = "VERIFICATION_FAILED"
		riskLevel = "HIGH"
	}

	sm.logSecurityEvent(eventType,
		fmt.Sprintf("Signature verification %s", map[bool]string{true: "succeeded", false: "failed"}[valid]),
		riskLevel, map[string]interface{}{
			"valid": valid,
		})

	return valid, nil
}

// LogAuthentication logs authentication events
func (sm *SecurityManager) LogAuthentication(userID, method, result string, details map[string]interface{}) {
	if sm.auditLogger == nil {
		return
	}

	sm.auditLogger.LogAuthentication(userID, method, result, details)
}

// LogAuthorization logs authorization events
func (sm *SecurityManager) LogAuthorization(userID, resource, action, result string, details map[string]interface{}) {
	if sm.auditLogger == nil {
		return
	}

	sm.auditLogger.LogAuthorization(userID, resource, action, result, details)
}

// LogDataAccess logs data access events
func (sm *SecurityManager) LogDataAccess(userID, resource, action string, details map[string]interface{}) {
	if sm.auditLogger == nil {
		return
	}

	sm.auditLogger.LogDataAccess(userID, resource, action, details)
}

// LogAPIRequest logs API request events
func (sm *SecurityManager) LogAPIRequest(requestID, method, path string, statusCode int, userID, clientIP string, details map[string]interface{}) {
	if sm.auditLogger == nil {
		return
	}

	sm.auditLogger.LogAPIRequest(requestID, method, path, statusCode, userID, clientIP, details)
}

// logSecurityEvent logs a security event
func (sm *SecurityManager) logSecurityEvent(eventType, message, riskLevel string, details map[string]interface{}) {
	if sm.auditLogger == nil {
		return
	}

	sm.auditLogger.LogSecurityEvent(eventType, message, riskLevel, details)
}

// RunVulnerabilityScan runs a vulnerability scan
func (sm *SecurityManager) RunVulnerabilityScan() (string, error) {
	if sm.vulnScanner == nil {
		return "", fmt.Errorf("vulnerability scanner not available")
	}

	// Create a new scan
	scanID := sm.vulnScanner.generateScanID()
	scan := &VulnerabilityScan{
		ID:        scanID,
		Type:      "MANUAL",
		Status:    "RUNNING",
		StartTime: time.Now(),
		Target:    "project",
		Findings:  make([]VulnerabilityFinding, 0),
		Metadata:  make(map[string]interface{}),
	}

	// Add to scanner
	sm.vulnScanner.scansMutex.Lock()
	sm.vulnScanner.scans[scanID] = scan
	sm.vulnScanner.scansMutex.Unlock()

	// Run scan in background
	go sm.vulnScanner.runScan(scan)

	// Log scan initiation
	sm.logSecurityEvent("VULNERABILITY_SCAN_STARTED",
		fmt.Sprintf("Vulnerability scan %s started", scanID),
		"LOW", map[string]interface{}{
			"scan_id": scanID,
		})

	return scanID, nil
}

// GetVulnerabilityScan returns a vulnerability scan by ID
func (sm *SecurityManager) GetVulnerabilityScan(scanID string) (*VulnerabilityScan, error) {
	if sm.vulnScanner == nil {
		return nil, fmt.Errorf("vulnerability scanner not available")
	}

	return sm.vulnScanner.GetScan(scanID)
}

// GetLatestVulnerabilityScan returns the most recent vulnerability scan
func (sm *SecurityManager) GetLatestVulnerabilityScan() (*VulnerabilityScan, error) {
	if sm.vulnScanner == nil {
		return nil, fmt.Errorf("vulnerability scanner not available")
	}

	return sm.vulnScanner.GetLatestScan()
}

// GenerateVulnerabilityReport generates a vulnerability report
func (sm *SecurityManager) GenerateVulnerabilityReport(scanID string) ([]byte, error) {
	if sm.vulnScanner == nil {
		return nil, fmt.Errorf("vulnerability scanner not available")
	}

	return sm.vulnScanner.GenerateReport(scanID)
}

// GetSecurityStatus returns the current security status
func (sm *SecurityManager) GetSecurityStatus() SecurityStatus {
	status := SecurityStatus{
		Timestamp:  time.Now(),
		Components: make(map[string]ComponentStatus),
	}

	// Key store status
	if sm.keyStore != nil {
		keyStats := sm.keyStore.GetKeyStatistics()
		status.Components["keystore"] = ComponentStatus{
			Name:         "keystore",
			Status:       "healthy",
			LastRotation: keyStats.LastRotation,
			NextRotation: keyStats.NextRotation,
			TotalKeys:    keyStats.TotalKeys,
			ActiveKeys:   keyStats.ActiveKeys,
		}
	}

	// Audit logger status
	if sm.auditLogger != nil {
		auditStats := sm.auditLogger.GetLogStatistics()
		status.Components["audit"] = ComponentStatus{
			Name:             "audit",
			Status:           "healthy",
			TotalEvents:      auditStats.TotalEvents,
			SuspiciousEvents: auditStats.SuspiciousEvents,
			BufferSize:       auditStats.BufferSize,
		}
	}

	// Vulnerability scanner status
	if sm.vulnScanner != nil {
		vulnStats := sm.vulnScanner.GetStatistics()
		status.Components["vulnerability"] = ComponentStatus{
			Name:             "vulnerability",
			Status:           "healthy",
			TotalScans:       vulnStats.TotalScans,
			CompletedScans:   vulnStats.CompletedScans,
			TotalFindings:    vulnStats.TotalFindings,
			CriticalFindings: vulnStats.CriticalFindings,
		}
	}

	// Calculate overall security score
	status.SecurityScore = sm.calculateSecurityScore(status.Components)

	return status
}

// calculateSecurityScore calculates an overall security score
func (sm *SecurityManager) calculateSecurityScore(components map[string]ComponentStatus) float64 {
	if len(components) == 0 {
		return 0.0
	}

	score := 100.0

	// Deduct points for issues
	for _, component := range components {
		// Check for suspicious events
		if component.SuspiciousEvents > 0 {
			score -= float64(component.SuspiciousEvents) * 5
		}

		// Check for critical vulnerabilities
		if component.CriticalFindings > 0 {
			score -= float64(component.CriticalFindings) * 10
		}

		// Check for expired keys
		if component.NextRotation.Before(time.Now()) {
			score -= 20
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

// SecurityStatus represents the overall security status
type SecurityStatus struct {
	Timestamp     time.Time                  `json:"timestamp"`
	SecurityScore float64                    `json:"security_score"`
	Components    map[string]ComponentStatus `json:"components"`
}

// ComponentStatus represents the status of a security component
type ComponentStatus struct {
	Name             string    `json:"name"`
	Status           string    `json:"status"`
	LastRotation     time.Time `json:"last_rotation,omitempty"`
	NextRotation     time.Time `json:"next_rotation,omitempty"`
	TotalKeys        int       `json:"total_keys,omitempty"`
	ActiveKeys       int       `json:"active_keys,omitempty"`
	TotalEvents      int       `json:"total_events,omitempty"`
	SuspiciousEvents int       `json:"suspicious_events,omitempty"`
	BufferSize       int       `json:"buffer_size,omitempty"`
	TotalScans       int       `json:"total_scans,omitempty"`
	CompletedScans   int       `json:"completed_scans,omitempty"`
	TotalFindings    int       `json:"total_findings,omitempty"`
	CriticalFindings int       `json:"critical_findings,omitempty"`
}

// GetAuditLogs retrieves audit logs with filtering
func (sm *SecurityManager) GetAuditLogs(limit int, filter AuditFilter) ([]AuditEvent, error) {
	if sm.auditLogger == nil {
		return nil, fmt.Errorf("audit logger not available")
	}

	return sm.auditLogger.ReadAuditLogs(limit, filter)
}

// GetKeyInfo returns information about the current key
func (sm *SecurityManager) GetKeyInfo() (*KeyPair, error) {
	if sm.keyStore == nil {
		return nil, fmt.Errorf("key store not available")
	}

	return sm.keyStore.GetCurrentKeyInfo()
}

// RotateKey manually rotates the current key
func (sm *SecurityManager) RotateKey() error {
	if sm.keyStore == nil {
		return fmt.Errorf("key store not available")
	}

	err := sm.keyStore.RotateKey()
	if err != nil {
		sm.logSecurityEvent("KEY_ROTATION_FAILED",
			fmt.Sprintf("Manual key rotation failed: %v", err),
			"HIGH", map[string]interface{}{
				"error": err.Error(),
			})
		return err
	}

	sm.logSecurityEvent("KEY_ROTATED",
		"Key successfully rotated",
		"LOW", nil)

	return nil
}

// ExportPublicKey exports the current public key
func (sm *SecurityManager) ExportPublicKey() (string, error) {
	if sm.keyStore == nil {
		return "", fmt.Errorf("key store not available")
	}

	keyBytes, err := sm.keyStore.ExportPublicKey()
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(keyBytes), nil
}

// Stop stops the security manager
func (sm *SecurityManager) Stop() {
	sm.cancel()
	sm.wg.Wait()

	// Stop components
	if sm.auditLogger != nil {
		sm.auditLogger.Close()
	}

	if sm.vulnScanner != nil {
		sm.vulnScanner.Stop()
	}
}

// GenerateSecurityReport generates a comprehensive security report
func (sm *SecurityManager) GenerateSecurityReport() ([]byte, error) {
	status := sm.GetSecurityStatus()

	report := map[string]interface{}{
		"timestamp":      status.Timestamp,
		"security_score": status.SecurityScore,
		"components":     status.Components,
		"policies":       sm.config.Policies,
	}

	// Add vulnerability scan results if available
	if sm.vulnScanner != nil {
		latestScan, err := sm.vulnScanner.GetLatestScan()
		if err == nil {
			report["latest_vulnerability_scan"] = latestScan
		}
	}

	// Add audit statistics if available
	if sm.auditLogger != nil {
		auditStats := sm.auditLogger.GetLogStatistics()
		report["audit_statistics"] = auditStats
	}

	// Add key statistics if available
	if sm.keyStore != nil {
		keyStats := sm.keyStore.GetKeyStatistics()
		report["key_statistics"] = keyStats
	}

	return json.MarshalIndent(report, "", "  ")
}
