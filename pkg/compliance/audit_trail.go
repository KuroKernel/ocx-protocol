package compliance

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// AuditTrailManager manages audit trails for compliance
type AuditTrailManager struct {
	// Configuration
	config AuditTrailConfig

	// Audit trail storage
	entries      []AuditEntry
	entriesMutex sync.RWMutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// AuditTrailConfig defines configuration for audit trails
type AuditTrailConfig struct {
	// Storage settings
	MaxEntries      int           `json:"max_entries"`
	RetentionPeriod time.Duration `json:"retention_period"`
	StorageType     string        `json:"storage_type"` // "memory", "file", "database"
	StoragePath     string        `json:"storage_path"`

	// Audit settings
	EnableIntegrityCheck   bool          `json:"enable_integrity_check"`
	IntegrityCheckInterval time.Duration `json:"integrity_check_interval"`
	EnableCompression      bool          `json:"enable_compression"`
	EnableEncryption       bool          `json:"enable_encryption"`
	EncryptionKey          string        `json:"encryption_key"`

	// Compliance settings
	ComplianceStandards []string          `json:"compliance_standards"` // "SOX", "GDPR", "HIPAA", "PCI-DSS", "ISO27001"
	RequiredFields      []string          `json:"required_fields"`
	CustomFields        map[string]string `json:"custom_fields"`

	// Notification settings
	EnableNotifications  bool           `json:"enable_notifications"`
	NotificationChannels []string       `json:"notification_channels"`
	AlertThresholds      map[string]int `json:"alert_thresholds"`
}

// AuditEntry represents a single audit trail entry
type AuditEntry struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	EventType       string                 `json:"event_type"`
	EventCategory   string                 `json:"event_category"`
	UserID          string                 `json:"user_id"`
	SessionID       string                 `json:"session_id"`
	IPAddress       string                 `json:"ip_address"`
	UserAgent       string                 `json:"user_agent"`
	Resource        string                 `json:"resource"`
	Action          string                 `json:"action"`
	Result          string                 `json:"result"` // "success", "failure", "error"
	Details         map[string]interface{} `json:"details"`
	Metadata        map[string]interface{} `json:"metadata"`
	Hash            string                 `json:"hash"`
	PreviousHash    string                 `json:"previous_hash"`
	ComplianceFlags []string               `json:"compliance_flags"`
	RiskLevel       string                 `json:"risk_level"` // "low", "medium", "high", "critical"
}

// AuditFilter defines filtering criteria for audit entries
type AuditFilter struct {
	EventTypes      []string   `json:"event_types,omitempty"`
	EventCategories []string   `json:"event_categories,omitempty"`
	UserIDs         []string   `json:"user_ids,omitempty"`
	SessionIDs      []string   `json:"session_ids,omitempty"`
	IPAddresses     []string   `json:"ip_addresses,omitempty"`
	Resources       []string   `json:"resources,omitempty"`
	Actions         []string   `json:"actions,omitempty"`
	Results         []string   `json:"results,omitempty"`
	RiskLevels      []string   `json:"risk_levels,omitempty"`
	ComplianceFlags []string   `json:"compliance_flags,omitempty"`
	StartTime       *time.Time `json:"start_time,omitempty"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	Limit           int        `json:"limit,omitempty"`
	Offset          int        `json:"offset,omitempty"`
}

// AuditStatistics represents audit trail statistics
type AuditStatistics struct {
	TotalEntries      int            `json:"total_entries"`
	EntriesByType     map[string]int `json:"entries_by_type"`
	EntriesByCategory map[string]int `json:"entries_by_category"`
	EntriesByResult   map[string]int `json:"entries_by_result"`
	EntriesByRisk     map[string]int `json:"entries_by_risk"`
	EntriesByUser     map[string]int `json:"entries_by_user"`
	ComplianceStats   map[string]int `json:"compliance_stats"`
	TimeRange         TimeRange      `json:"time_range"`
	LastUpdated       time.Time      `json:"last_updated"`
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ComplianceReport represents a compliance report
type ComplianceReport struct {
	ID              string                     `json:"id"`
	Standard        string                     `json:"standard"`
	ReportType      string                     `json:"report_type"`
	GeneratedAt     time.Time                  `json:"generated_at"`
	Period          TimeRange                  `json:"period"`
	Summary         ComplianceSummary          `json:"summary"`
	Findings        []ComplianceFinding        `json:"findings"`
	Recommendations []ComplianceRecommendation `json:"recommendations"`
	Metadata        map[string]interface{}     `json:"metadata"`
}

// ComplianceSummary represents a compliance summary
type ComplianceSummary struct {
	TotalChecks     int     `json:"total_checks"`
	PassedChecks    int     `json:"passed_checks"`
	FailedChecks    int     `json:"failed_checks"`
	WarningChecks   int     `json:"warning_checks"`
	ComplianceScore float64 `json:"compliance_score"`
	RiskLevel       string  `json:"risk_level"`
}

// ComplianceFinding represents a compliance finding
type ComplianceFinding struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`     // "violation", "warning", "recommendation"
	Severity    string                 `json:"severity"` // "low", "medium", "high", "critical"
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Requirement string                 `json:"requirement"`
	Evidence    []string               `json:"evidence"`
	Impact      string                 `json:"impact"`
	Remediation string                 `json:"remediation"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ComplianceRecommendation represents a compliance recommendation
type ComplianceRecommendation struct {
	ID          string                 `json:"id"`
	Priority    string                 `json:"priority"` // "low", "medium", "high", "critical"
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Action      string                 `json:"action"`
	Timeline    string                 `json:"timeline"`
	Resources   []string               `json:"resources"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewAuditTrailManager creates a new audit trail manager
func NewAuditTrailManager(config AuditTrailConfig) (*AuditTrailManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	atm := &AuditTrailManager{
		config:  config,
		entries: make([]AuditEntry, 0),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Start integrity check routine if enabled
	if config.EnableIntegrityCheck {
		atm.wg.Add(1)
		go atm.integrityCheckLoop()
	}

	// Start cleanup routine
	atm.wg.Add(1)
	go atm.cleanupLoop()

	return atm, nil
}

// LogEvent logs an audit event
func (atm *AuditTrailManager) LogEvent(event AuditEntry) error {
	// Generate ID if not provided
	if event.ID == "" {
		event.ID = generateAuditID()
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Calculate hash
	event.Hash = atm.calculateHash(event)

	// Set previous hash
	atm.entriesMutex.Lock()
	if len(atm.entries) > 0 {
		event.PreviousHash = atm.entries[len(atm.entries)-1].Hash
	}
	atm.entriesMutex.Unlock()

	// Validate compliance requirements
	if err := atm.validateCompliance(event); err != nil {
		return fmt.Errorf("compliance validation failed: %w", err)
	}

	// Add to audit trail
	atm.entriesMutex.Lock()
	atm.entries = append(atm.entries, event)

	// Maintain max entries limit
	if atm.config.MaxEntries > 0 && len(atm.entries) > atm.config.MaxEntries {
		atm.entries = atm.entries[len(atm.entries)-atm.config.MaxEntries:]
	}
	atm.entriesMutex.Unlock()

	// Check for alerts
	atm.checkAlerts(event)

	return nil
}

// GetAuditEntries returns audit entries with optional filtering
func (atm *AuditTrailManager) GetAuditEntries(filter AuditFilter) []AuditEntry {
	atm.entriesMutex.RLock()
	defer atm.entriesMutex.RUnlock()

	var filteredEntries []AuditEntry
	for _, entry := range atm.entries {
		if atm.matchesFilter(entry, filter) {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	// Apply pagination
	if filter.Limit > 0 {
		start := filter.Offset
		if start >= len(filteredEntries) {
			return []AuditEntry{}
		}
		end := start + filter.Limit
		if end > len(filteredEntries) {
			end = len(filteredEntries)
		}
		filteredEntries = filteredEntries[start:end]
	}

	return filteredEntries
}

// GetAuditStatistics returns audit trail statistics
func (atm *AuditTrailManager) GetAuditStatistics(filter AuditFilter) AuditStatistics {
	atm.entriesMutex.RLock()
	defer atm.entriesMutex.RUnlock()

	stats := AuditStatistics{
		EntriesByType:     make(map[string]int),
		EntriesByCategory: make(map[string]int),
		EntriesByResult:   make(map[string]int),
		EntriesByRisk:     make(map[string]int),
		EntriesByUser:     make(map[string]int),
		ComplianceStats:   make(map[string]int),
		LastUpdated:       time.Now(),
	}

	var startTime, endTime time.Time
	hasTimeRange := false

	for _, entry := range atm.entries {
		if !atm.matchesFilter(entry, filter) {
			continue
		}

		stats.TotalEntries++

		// Count by type
		stats.EntriesByType[entry.EventType]++

		// Count by category
		stats.EntriesByCategory[entry.EventCategory]++

		// Count by result
		stats.EntriesByResult[entry.Result]++

		// Count by risk level
		stats.EntriesByRisk[entry.RiskLevel]++

		// Count by user
		if entry.UserID != "" {
			stats.EntriesByUser[entry.UserID]++
		}

		// Count compliance flags
		for _, flag := range entry.ComplianceFlags {
			stats.ComplianceStats[flag]++
		}

		// Track time range
		if !hasTimeRange {
			startTime = entry.Timestamp
			endTime = entry.Timestamp
			hasTimeRange = true
		} else {
			if entry.Timestamp.Before(startTime) {
				startTime = entry.Timestamp
			}
			if entry.Timestamp.After(endTime) {
				endTime = entry.Timestamp
			}
		}
	}

	if hasTimeRange {
		stats.TimeRange = TimeRange{
			Start: startTime,
			End:   endTime,
		}
	}

	return stats
}

// GenerateComplianceReport generates a compliance report
func (atm *AuditTrailManager) GenerateComplianceReport(standard string, reportType string, period TimeRange) (*ComplianceReport, error) {
	report := &ComplianceReport{
		ID:              generateReportID(),
		Standard:        standard,
		ReportType:      reportType,
		GeneratedAt:     time.Now(),
		Period:          period,
		Summary:         ComplianceSummary{},
		Findings:        make([]ComplianceFinding, 0),
		Recommendations: make([]ComplianceRecommendation, 0),
		Metadata:        make(map[string]interface{}),
	}

	// Get audit entries for the period
	filter := AuditFilter{
		StartTime: &period.Start,
		EndTime:   &period.End,
	}
	entries := atm.GetAuditEntries(filter)

	// Generate compliance findings based on standard
	findings, err := atm.generateComplianceFindings(standard, entries)
	if err != nil {
		return nil, fmt.Errorf("failed to generate compliance findings: %w", err)
	}
	report.Findings = findings

	// Generate recommendations
	recommendations, err := atm.generateRecommendations(standard, findings)
	if err != nil {
		return nil, fmt.Errorf("failed to generate recommendations: %w", err)
	}
	report.Recommendations = recommendations

	// Calculate summary
	report.Summary = atm.calculateComplianceSummary(findings)

	return report, nil
}

// matchesFilter checks if an audit entry matches the filter criteria
func (atm *AuditTrailManager) matchesFilter(entry AuditEntry, filter AuditFilter) bool {
	if len(filter.EventTypes) > 0 {
		found := false
		for _, eventType := range filter.EventTypes {
			if entry.EventType == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.EventCategories) > 0 {
		found := false
		for _, category := range filter.EventCategories {
			if entry.EventCategory == category {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.UserIDs) > 0 {
		found := false
		for _, userID := range filter.UserIDs {
			if entry.UserID == userID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.Results) > 0 {
		found := false
		for _, result := range filter.Results {
			if entry.Result == result {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.RiskLevels) > 0 {
		found := false
		for _, riskLevel := range filter.RiskLevels {
			if entry.RiskLevel == riskLevel {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if filter.StartTime != nil && entry.Timestamp.Before(*filter.StartTime) {
		return false
	}

	if filter.EndTime != nil && entry.Timestamp.After(*filter.EndTime) {
		return false
	}

	return true
}

// calculateHash calculates the hash of an audit entry
func (atm *AuditTrailManager) calculateHash(entry AuditEntry) string {
	// Create a copy without the hash fields for calculation
	entryCopy := entry
	entryCopy.Hash = ""
	entryCopy.PreviousHash = ""

	// Serialize to JSON
	data, err := json.Marshal(entryCopy)
	if err != nil {
		return ""
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// validateCompliance validates compliance requirements
func (atm *AuditTrailManager) validateCompliance(entry AuditEntry) error {
	// Check required fields
	for _, field := range atm.config.RequiredFields {
		switch field {
		case "user_id":
			if entry.UserID == "" {
				return fmt.Errorf("user_id is required for compliance")
			}
		case "timestamp":
			if entry.Timestamp.IsZero() {
				return fmt.Errorf("timestamp is required for compliance")
			}
		case "event_type":
			if entry.EventType == "" {
				return fmt.Errorf("event_type is required for compliance")
			}
		case "action":
			if entry.Action == "" {
				return fmt.Errorf("action is required for compliance")
			}
		}
	}

	// Check compliance standards
	for _, standard := range atm.config.ComplianceStandards {
		switch standard {
		case "SOX":
			if err := atm.validateSOXCompliance(entry); err != nil {
				return fmt.Errorf("SOX compliance validation failed: %w", err)
			}
		case "GDPR":
			if err := atm.validateGDPRCompliance(entry); err != nil {
				return fmt.Errorf("GDPR compliance validation failed: %w", err)
			}
		case "HIPAA":
			if err := atm.validateHIPAACompliance(entry); err != nil {
				return fmt.Errorf("HIPAA compliance validation failed: %w", err)
			}
		case "PCI-DSS":
			if err := atm.validatePCIDSSCompliance(entry); err != nil {
				return fmt.Errorf("PCI-DSS compliance validation failed: %w", err)
			}
		}
	}

	return nil
}

// validateSOXCompliance validates SOX compliance requirements
func (atm *AuditTrailManager) validateSOXCompliance(entry AuditEntry) error {
	// SOX requires detailed audit trails for financial data access
	if entry.EventCategory == "financial" || entry.EventCategory == "accounting" {
		if entry.UserID == "" {
			return fmt.Errorf("user_id is required for SOX compliance on financial events")
		}
		if entry.IPAddress == "" {
			return fmt.Errorf("ip_address is required for SOX compliance on financial events")
		}
	}
	return nil
}

// validateGDPRCompliance validates GDPR compliance requirements
func (atm *AuditTrailManager) validateGDPRCompliance(entry AuditEntry) error {
	// GDPR requires detailed audit trails for personal data access
	if entry.EventCategory == "personal_data" || entry.EventCategory == "privacy" {
		if entry.UserID == "" {
			return fmt.Errorf("user_id is required for GDPR compliance on personal data events")
		}
		if entry.Details == nil {
			entry.Details = make(map[string]interface{})
		}
		entry.Details["gdpr_audit"] = true
	}
	return nil
}

// validateHIPAACompliance validates HIPAA compliance requirements
func (atm *AuditTrailManager) validateHIPAACompliance(entry AuditEntry) error {
	// HIPAA requires detailed audit trails for health information access
	if entry.EventCategory == "health_info" || entry.EventCategory == "medical" {
		if entry.UserID == "" {
			return fmt.Errorf("user_id is required for HIPAA compliance on health information events")
		}
		if entry.SessionID == "" {
			return fmt.Errorf("session_id is required for HIPAA compliance on health information events")
		}
	}
	return nil
}

// validatePCIDSSCompliance validates PCI-DSS compliance requirements
func (atm *AuditTrailManager) validatePCIDSSCompliance(entry AuditEntry) error {
	// PCI-DSS requires detailed audit trails for cardholder data access
	if entry.EventCategory == "cardholder_data" || entry.EventCategory == "payment" {
		if entry.UserID == "" {
			return fmt.Errorf("user_id is required for PCI-DSS compliance on cardholder data events")
		}
		if entry.IPAddress == "" {
			return fmt.Errorf("ip_address is required for PCI-DSS compliance on cardholder data events")
		}
	}
	return nil
}

// checkAlerts checks for alert conditions
func (atm *AuditTrailManager) checkAlerts(entry AuditEntry) {
	if !atm.config.EnableNotifications {
		return
	}

	// Check risk level alerts
	if _, exists := atm.config.AlertThresholds["high_risk"]; exists {
		if entry.RiskLevel == "high" || entry.RiskLevel == "critical" {
			// This would trigger an alert
			fmt.Printf("High risk audit event detected: %s\n", entry.ID)
		}
	}

	// Check compliance flag alerts
	for _, flag := range entry.ComplianceFlags {
		if _, exists := atm.config.AlertThresholds[flag]; exists {
			// This would trigger an alert
			fmt.Printf("Compliance alert triggered for flag: %s\n", flag)
		}
	}
}

// generateComplianceFindings generates compliance findings
func (atm *AuditTrailManager) generateComplianceFindings(standard string, entries []AuditEntry) ([]ComplianceFinding, error) {
	findings := make([]ComplianceFinding, 0)

	// Generate findings based on standard
	switch standard {
	case "SOX":
		findings = append(findings, atm.generateSOXFindings(entries)...)
	case "GDPR":
		findings = append(findings, atm.generateGDPRFindings(entries)...)
	case "HIPAA":
		findings = append(findings, atm.generateHIPAAFindings(entries)...)
	case "PCI-DSS":
		findings = append(findings, atm.generatePCIDSSFindings(entries)...)
	}

	return findings, nil
}

// generateSOXFindings generates SOX compliance findings
func (atm *AuditTrailManager) generateSOXFindings(entries []AuditEntry) []ComplianceFinding {
	findings := make([]ComplianceFinding, 0)

	// Check for missing user IDs in financial events
	missingUserCount := 0
	for _, entry := range entries {
		if entry.EventCategory == "financial" && entry.UserID == "" {
			missingUserCount++
		}
	}

	if missingUserCount > 0 {
		findings = append(findings, ComplianceFinding{
			ID:          generateFindingID(),
			Type:        "violation",
			Severity:    "high",
			Title:       "Missing User IDs in Financial Events",
			Description: fmt.Sprintf("Found %d financial events without user IDs", missingUserCount),
			Requirement: "SOX Section 404 - Internal Controls",
			Evidence:    []string{fmt.Sprintf("%d events without user_id", missingUserCount)},
			Impact:      "Non-compliance with SOX audit trail requirements",
			Remediation: "Ensure all financial events include user identification",
		})
	}

	return findings
}

// generateGDPRFindings generates GDPR compliance findings
func (atm *AuditTrailManager) generateGDPRFindings(entries []AuditEntry) []ComplianceFinding {
	findings := make([]ComplianceFinding, 0)

	// Check for personal data access without proper audit trails
	personalDataAccess := 0
	for _, entry := range entries {
		if entry.EventCategory == "personal_data" {
			personalDataAccess++
		}
	}

	if personalDataAccess > 0 {
		findings = append(findings, ComplianceFinding{
			ID:          generateFindingID(),
			Type:        "recommendation",
			Severity:    "medium",
			Title:       "Personal Data Access Audit Trail",
			Description: fmt.Sprintf("Found %d personal data access events", personalDataAccess),
			Requirement: "GDPR Article 30 - Records of Processing Activities",
			Evidence:    []string{fmt.Sprintf("%d personal data access events", personalDataAccess)},
			Impact:      "Ensure compliance with GDPR audit requirements",
			Remediation: "Review and document all personal data processing activities",
		})
	}

	return findings
}

// generateHIPAAFindings generates HIPAA compliance findings
func (atm *AuditTrailManager) generateHIPAAFindings(entries []AuditEntry) []ComplianceFinding {
	findings := make([]ComplianceFinding, 0)

	// Check for health information access without proper audit trails
	healthInfoAccess := 0
	for _, entry := range entries {
		if entry.EventCategory == "health_info" {
			healthInfoAccess++
		}
	}

	if healthInfoAccess > 0 {
		findings = append(findings, ComplianceFinding{
			ID:          generateFindingID(),
			Type:        "recommendation",
			Severity:    "high",
			Title:       "Health Information Access Audit Trail",
			Description: fmt.Sprintf("Found %d health information access events", healthInfoAccess),
			Requirement: "HIPAA Security Rule - Audit Controls",
			Evidence:    []string{fmt.Sprintf("%d health information access events", healthInfoAccess)},
			Impact:      "Ensure compliance with HIPAA audit requirements",
			Remediation: "Implement comprehensive audit controls for health information access",
		})
	}

	return findings
}

// generatePCIDSSFindings generates PCI-DSS compliance findings
func (atm *AuditTrailManager) generatePCIDSSFindings(entries []AuditEntry) []ComplianceFinding {
	findings := make([]ComplianceFinding, 0)

	// Check for cardholder data access without proper audit trails
	cardholderDataAccess := 0
	for _, entry := range entries {
		if entry.EventCategory == "cardholder_data" {
			cardholderDataAccess++
		}
	}

	if cardholderDataAccess > 0 {
		findings = append(findings, ComplianceFinding{
			ID:          generateFindingID(),
			Type:        "recommendation",
			Severity:    "high",
			Title:       "Cardholder Data Access Audit Trail",
			Description: fmt.Sprintf("Found %d cardholder data access events", cardholderDataAccess),
			Requirement: "PCI-DSS Requirement 10 - Track and Monitor Access",
			Evidence:    []string{fmt.Sprintf("%d cardholder data access events", cardholderDataAccess)},
			Impact:      "Ensure compliance with PCI-DSS audit requirements",
			Remediation: "Implement comprehensive audit trails for cardholder data access",
		})
	}

	return findings
}

// generateRecommendations generates compliance recommendations
func (atm *AuditTrailManager) generateRecommendations(standard string, findings []ComplianceFinding) ([]ComplianceRecommendation, error) {
	recommendations := make([]ComplianceRecommendation, 0)

	// Generate recommendations based on findings
	for _, finding := range findings {
		if finding.Type == "violation" || finding.Severity == "high" {
			recommendations = append(recommendations, ComplianceRecommendation{
				ID:          generateRecommendationID(),
				Priority:    finding.Severity,
				Title:       "Address " + finding.Title,
				Description: finding.Remediation,
				Action:      finding.Remediation,
				Timeline:    "30 days",
				Resources:   []string{"Compliance team", "IT security team"},
			})
		}
	}

	return recommendations, nil
}

// calculateComplianceSummary calculates compliance summary
func (atm *AuditTrailManager) calculateComplianceSummary(findings []ComplianceFinding) ComplianceSummary {
	summary := ComplianceSummary{
		TotalChecks: len(findings),
	}

	for _, finding := range findings {
		switch finding.Type {
		case "violation":
			summary.FailedChecks++
		case "warning":
			summary.WarningChecks++
		default:
			summary.PassedChecks++
		}
	}

	// Calculate compliance score
	if summary.TotalChecks > 0 {
		summary.ComplianceScore = float64(summary.PassedChecks) / float64(summary.TotalChecks) * 100
	}

	// Determine risk level
	if summary.FailedChecks > 0 {
		summary.RiskLevel = "high"
	} else if summary.WarningChecks > 0 {
		summary.RiskLevel = "medium"
	} else {
		summary.RiskLevel = "low"
	}

	return summary
}

// integrityCheckLoop runs integrity checks
func (atm *AuditTrailManager) integrityCheckLoop() {
	defer atm.wg.Done()

	ticker := time.NewTicker(atm.config.IntegrityCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-atm.ctx.Done():
			return
		case <-ticker.C:
			atm.performIntegrityCheck()
		}
	}
}

// cleanupLoop runs cleanup operations
func (atm *AuditTrailManager) cleanupLoop() {
	defer atm.wg.Done()

	ticker := time.NewTicker(time.Hour * 24) // Run daily
	defer ticker.Stop()

	for {
		select {
		case <-atm.ctx.Done():
			return
		case <-ticker.C:
			atm.cleanupOldEntries()
		}
	}
}

// performIntegrityCheck performs integrity check on audit trail
func (atm *AuditTrailManager) performIntegrityCheck() {
	atm.entriesMutex.RLock()
	defer atm.entriesMutex.RUnlock()

	for i, entry := range atm.entries {
		// Verify hash
		expectedHash := atm.calculateHash(entry)
		if entry.Hash != expectedHash {
			fmt.Printf("Integrity check failed for audit entry %s at index %d\n", entry.ID, i)
		}

		// Verify previous hash chain
		if i > 0 {
			previousEntry := atm.entries[i-1]
			if entry.PreviousHash != previousEntry.Hash {
				fmt.Printf("Hash chain broken for audit entry %s at index %d\n", entry.ID, i)
			}
		}
	}
}

// cleanupOldEntries removes old audit entries
func (atm *AuditTrailManager) cleanupOldEntries() {
	atm.entriesMutex.Lock()
	defer atm.entriesMutex.Unlock()

	cutoffTime := time.Now().Add(-atm.config.RetentionPeriod)

	var keptEntries []AuditEntry
	for _, entry := range atm.entries {
		if entry.Timestamp.After(cutoffTime) {
			keptEntries = append(keptEntries, entry)
		}
	}

	atm.entries = keptEntries
}

// generateAuditID generates a unique audit ID
func generateAuditID() string {
	return fmt.Sprintf("audit-%d", time.Now().UnixNano())
}

// generateReportID generates a unique report ID
func generateReportID() string {
	return fmt.Sprintf("report-%d", time.Now().UnixNano())
}

// generateFindingID generates a unique finding ID
func generateFindingID() string {
	return fmt.Sprintf("finding-%d", time.Now().UnixNano())
}

// generateRecommendationID generates a unique recommendation ID
func generateRecommendationID() string {
	return fmt.Sprintf("recommendation-%d", time.Now().UnixNano())
}

// Stop stops the audit trail manager
func (atm *AuditTrailManager) Stop() {
	atm.cancel()
	atm.wg.Wait()
}
