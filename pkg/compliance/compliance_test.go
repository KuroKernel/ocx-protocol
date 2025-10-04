package compliance

import (
	"testing"
	"time"
)

func TestAuditTrailManager(t *testing.T) {
	// Create test configuration
	config := AuditTrailConfig{
		MaxEntries:             1000,
		RetentionPeriod:        time.Hour * 24 * 30, // 30 days
		StorageType:            "memory",
		StoragePath:            "/tmp/audit-trail",
		EnableIntegrityCheck:   true,
		IntegrityCheckInterval: time.Hour * 1,
		EnableCompression:      false,
		EnableEncryption:       false,
		ComplianceStandards:    []string{"SOX", "GDPR", "HIPAA"},
		RequiredFields:         []string{"user_id", "timestamp", "event_type", "action"},
		EnableNotifications:    false,
	}

	atm, err := NewAuditTrailManager(config)
	if err != nil {
		t.Fatalf("Failed to create audit trail manager: %v", err)
	}
	defer atm.Stop()

	// Test logging an audit event
	event := AuditEntry{
		ID:              "test-event-1",
		Timestamp:       time.Now(),
		EventType:       "user_login",
		EventCategory:   "authentication",
		UserID:          "user123",
		SessionID:       "session456",
		IPAddress:       "192.168.1.100",
		UserAgent:       "Mozilla/5.0",
		Resource:        "/api/v1/login",
		Action:          "login",
		Result:          "success",
		Details:         map[string]interface{}{"method": "password"},
		Metadata:        map[string]interface{}{"version": "1.0"},
		ComplianceFlags: []string{"SOX", "GDPR"},
		RiskLevel:       "low",
	}

	err = atm.LogEvent(event)
	if err != nil {
		t.Fatalf("Failed to log audit event: %v", err)
	}

	// Test getting audit entries
	filter := AuditFilter{
		EventTypes: []string{"user_login"},
		UserIDs:    []string{"user123"},
	}
	entries := atm.GetAuditEntries(filter)
	if len(entries) != 1 {
		t.Errorf("Expected 1 audit entry, got %d", len(entries))
	}

	if entries[0].ID != event.ID {
		t.Error("Retrieved audit entry ID does not match")
	}

	// Test audit statistics
	stats := atm.GetAuditStatistics(filter)
	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 total entry in statistics, got %d", stats.TotalEntries)
	}

	if stats.EntriesByType["user_login"] != 1 {
		t.Errorf("Expected 1 user_login entry in statistics, got %d", stats.EntriesByType["user_login"])
	}

	// Test compliance report generation
	period := TimeRange{
		Start: time.Now().Add(-time.Hour),
		End:   time.Now().Add(time.Hour),
	}
	report, err := atm.GenerateComplianceReport("SOX", "audit_report", period)
	if err != nil {
		t.Fatalf("Failed to generate compliance report: %v", err)
	}

	if report.Standard != "SOX" {
		t.Errorf("Expected report standard 'SOX', got '%s'", report.Standard)
	}

	if report.ReportType != "audit_report" {
		t.Errorf("Expected report type 'audit_report', got '%s'", report.ReportType)
	}
}

func TestComplianceValidator(t *testing.T) {
	// Create test configuration
	config := ComplianceValidatorConfig{
		EnableContinuousValidation: false, // Disable for testing
		ValidationInterval:         time.Hour * 1,
		EnableRealTimeValidation:   true,
		Standards:                  []string{"SOX", "GDPR", "HIPAA"},
		CustomRules:                []ValidationRule{},
		EnableReporting:            false, // Disable for testing
		ReportInterval:             time.Hour * 24,
		ReportRetentionDays:        30,
		EnableNotifications:        false,
	}

	cv, err := NewComplianceValidator(config)
	if err != nil {
		t.Fatalf("Failed to create compliance validator: %v", err)
	}
	defer cv.Stop()

	// Test validating an audit entry
	entry := AuditEntry{
		ID:              "test-event-1",
		Timestamp:       time.Now(),
		EventType:       "financial_transaction",
		EventCategory:   "financial",
		UserID:          "user123",
		SessionID:       "session456",
		IPAddress:       "192.168.1.100",
		Resource:        "/api/v1/transaction",
		Action:          "create",
		Result:          "success",
		Details:         map[string]interface{}{"amount": 1000},
		Metadata:        map[string]interface{}{"version": "1.0"},
		ComplianceFlags: []string{"SOX"},
		RiskLevel:       "medium",
	}

	results, err := cv.ValidateEntry(entry)
	if err != nil {
		t.Fatalf("Failed to validate audit entry: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected validation results, got none")
	}

	// Test getting requirements
	requirements := cv.GetRequirements()
	if len(requirements) == 0 {
		t.Error("Expected requirements, got none")
	}

	// Test generating validation report
	period := TimeRange{
		Start: time.Now().Add(-time.Hour),
		End:   time.Now().Add(time.Hour),
	}
	report, err := cv.GenerateValidationReport(period, []string{"SOX", "GDPR"})
	if err != nil {
		t.Fatalf("Failed to generate validation report: %v", err)
	}

	if len(report.Standards) != 2 {
		t.Errorf("Expected 2 standards in report, got %d", len(report.Standards))
	}

	if report.Summary.TotalValidations == 0 {
		t.Error("Expected validation summary to have validations")
	}
}

func TestSOXValidator(t *testing.T) {
	validator := &SOXValidator{}

	if validator.GetStandard() != "SOX" {
		t.Errorf("Expected standard 'SOX', got '%s'", validator.GetStandard())
	}

	// Test SOX validation
	entry := AuditEntry{
		ID:              "test-event-1",
		Timestamp:       time.Now(),
		EventType:       "financial_transaction",
		EventCategory:   "financial",
		UserID:          "user123",
		IPAddress:       "192.168.1.100",
		Resource:        "/api/v1/transaction",
		Action:          "create",
		Result:          "success",
		Details:         map[string]interface{}{"amount": 1000},
		Metadata:        map[string]interface{}{"version": "1.0"},
		ComplianceFlags: []string{"SOX"},
		RiskLevel:       "medium",
	}

	results, err := validator.Validate(entry)
	if err != nil {
		t.Fatalf("Failed to validate SOX entry: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected SOX validation results, got none")
	}

	// Test requirements
	requirements := validator.GetRequirements()
	if len(requirements) == 0 {
		t.Error("Expected SOX requirements, got none")
	}

	// Check for specific SOX requirements
	found := false
	for _, req := range requirements {
		if req.Section == "302" && req.Title == "Corporate Responsibility for Financial Reports" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find SOX Section 302 requirement")
	}
}

func TestGDPRValidator(t *testing.T) {
	validator := &GDPRValidator{}

	if validator.GetStandard() != "GDPR" {
		t.Errorf("Expected standard 'GDPR', got '%s'", validator.GetStandard())
	}

	// Test GDPR validation
	entry := AuditEntry{
		ID:              "test-event-1",
		Timestamp:       time.Now(),
		EventType:       "data_access",
		EventCategory:   "personal_data",
		UserID:          "user123",
		Resource:        "/api/v1/user-data",
		Action:          "read",
		Result:          "success",
		Details:         map[string]interface{}{"legal_basis": "consent"},
		Metadata:        map[string]interface{}{"version": "1.0"},
		ComplianceFlags: []string{"GDPR"},
		RiskLevel:       "medium",
	}

	results, err := validator.Validate(entry)
	if err != nil {
		t.Fatalf("Failed to validate GDPR entry: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected GDPR validation results, got none")
	}

	// Test requirements
	requirements := validator.GetRequirements()
	if len(requirements) == 0 {
		t.Error("Expected GDPR requirements, got none")
	}

	// Check for specific GDPR requirements
	found := false
	for _, req := range requirements {
		if req.Section == "5" && req.Title == "Principles relating to processing of personal data" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find GDPR Article 5 requirement")
	}
}

func TestHIPAAValidator(t *testing.T) {
	validator := &HIPAAValidator{}

	if validator.GetStandard() != "HIPAA" {
		t.Errorf("Expected standard 'HIPAA', got '%s'", validator.GetStandard())
	}

	// Test HIPAA validation
	entry := AuditEntry{
		ID:              "test-event-1",
		Timestamp:       time.Now(),
		EventType:       "health_data_access",
		EventCategory:   "health_info",
		UserID:          "user123",
		SessionID:       "session456",
		Resource:        "/api/v1/health-data",
		Action:          "read",
		Result:          "success",
		Details:         map[string]interface{}{"patient_id": "patient123"},
		Metadata:        map[string]interface{}{"version": "1.0"},
		ComplianceFlags: []string{"HIPAA"},
		RiskLevel:       "high",
	}

	results, err := validator.Validate(entry)
	if err != nil {
		t.Fatalf("Failed to validate HIPAA entry: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected HIPAA validation results, got none")
	}

	// Test requirements
	requirements := validator.GetRequirements()
	if len(requirements) == 0 {
		t.Error("Expected HIPAA requirements, got none")
	}

	// Check for specific HIPAA requirements
	found := false
	for _, req := range requirements {
		if req.Section == "164.308" && req.Title == "Administrative Safeguards - User Identification" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find HIPAA 164.308 requirement")
	}
}

func TestPCIDSSValidator(t *testing.T) {
	validator := &PCIDSSValidator{}

	if validator.GetStandard() != "PCI-DSS" {
		t.Errorf("Expected standard 'PCI-DSS', got '%s'", validator.GetStandard())
	}

	// Test PCI-DSS validation
	entry := AuditEntry{
		ID:              "test-event-1",
		Timestamp:       time.Now(),
		EventType:       "cardholder_data_access",
		EventCategory:   "cardholder_data",
		UserID:          "user123",
		IPAddress:       "192.168.1.100",
		Resource:        "/api/v1/payment",
		Action:          "process",
		Result:          "success",
		Details:         map[string]interface{}{"data_classification": "sensitive"},
		Metadata:        map[string]interface{}{"version": "1.0"},
		ComplianceFlags: []string{"PCI-DSS"},
		RiskLevel:       "high",
	}

	results, err := validator.Validate(entry)
	if err != nil {
		t.Fatalf("Failed to validate PCI-DSS entry: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected PCI-DSS validation results, got none")
	}

	// Test requirements
	requirements := validator.GetRequirements()
	if len(requirements) == 0 {
		t.Error("Expected PCI-DSS requirements, got none")
	}

	// Check for specific PCI-DSS requirements
	found := false
	for _, req := range requirements {
		if req.Section == "3" && req.Title == "Cardholder Data - Data Classification" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find PCI-DSS Requirement 3")
	}
}

func TestISO27001Validator(t *testing.T) {
	validator := &ISO27001Validator{}

	if validator.GetStandard() != "ISO27001" {
		t.Errorf("Expected standard 'ISO27001', got '%s'", validator.GetStandard())
	}

	// Test ISO27001 validation
	entry := AuditEntry{
		ID:              "test-event-1",
		Timestamp:       time.Now(),
		EventType:       "access_control",
		EventCategory:   "access_control",
		UserID:          "user123",
		Resource:        "/api/v1/system",
		Action:          "access",
		Result:          "success",
		Details:         map[string]interface{}{"access_request": "approved"},
		Metadata:        map[string]interface{}{"version": "1.0"},
		ComplianceFlags: []string{"ISO27001"},
		RiskLevel:       "medium",
	}

	results, err := validator.Validate(entry)
	if err != nil {
		t.Fatalf("Failed to validate ISO27001 entry: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected ISO27001 validation results, got none")
	}

	// Test requirements
	requirements := validator.GetRequirements()
	if len(requirements) == 0 {
		t.Error("Expected ISO27001 requirements, got none")
	}

	// Check for specific ISO27001 requirements
	found := false
	for _, req := range requirements {
		if req.Section == "A.9" && req.Title == "Access Control - Access Request" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find ISO27001 A.9 requirement")
	}
}

func TestAuditTrailConfig(t *testing.T) {
	config := AuditTrailConfig{
		MaxEntries:             1000,
		RetentionPeriod:        time.Hour * 24 * 30,
		StorageType:            "memory",
		StoragePath:            "/tmp/audit-trail",
		EnableIntegrityCheck:   true,
		IntegrityCheckInterval: time.Hour * 1,
		EnableCompression:      false,
		EnableEncryption:       false,
		ComplianceStandards:    []string{"SOX", "GDPR", "HIPAA"},
		RequiredFields:         []string{"user_id", "timestamp", "event_type", "action"},
		EnableNotifications:    false,
	}

	// Test configuration validation
	if config.MaxEntries <= 0 {
		t.Error("Max entries should be positive")
	}

	if config.RetentionPeriod <= 0 {
		t.Error("Retention period should be positive")
	}

	if config.StorageType == "" {
		t.Error("Storage type should not be empty")
	}

	if len(config.ComplianceStandards) == 0 {
		t.Error("Compliance standards should not be empty")
	}

	if len(config.RequiredFields) == 0 {
		t.Error("Required fields should not be empty")
	}
}

func TestComplianceValidatorConfig(t *testing.T) {
	config := ComplianceValidatorConfig{
		EnableContinuousValidation: true,
		ValidationInterval:         time.Hour * 1,
		EnableRealTimeValidation:   true,
		Standards:                  []string{"SOX", "GDPR", "HIPAA"},
		CustomRules:                []ValidationRule{},
		EnableReporting:            true,
		ReportInterval:             time.Hour * 24,
		ReportRetentionDays:        30,
		EnableNotifications:        true,
	}

	// Test configuration validation
	if config.ValidationInterval <= 0 {
		t.Error("Validation interval should be positive")
	}

	if len(config.Standards) == 0 {
		t.Error("Standards should not be empty")
	}

	if config.ReportInterval <= 0 {
		t.Error("Report interval should be positive")
	}

	if config.ReportRetentionDays <= 0 {
		t.Error("Report retention days should be positive")
	}
}
