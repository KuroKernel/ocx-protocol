package security

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditLogger provides comprehensive audit logging capabilities
type AuditLogger struct {
	// Configuration
	config AuditConfig

	// Logging
	logFile     *os.File
	logMutex    sync.Mutex
	logBuffer   []AuditEvent
	bufferMutex sync.Mutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// AuditConfig defines configuration for audit logging
type AuditConfig struct {
	// Logging settings
	LogLevel     string        `json:"log_level"`  // "DEBUG", "INFO", "WARN", "ERROR", "CRITICAL"
	LogFormat    string        `json:"log_format"` // "JSON", "TEXT"
	LogFile      string        `json:"log_file"`
	LogRotation  bool          `json:"log_rotation"`
	MaxLogSize   int64         `json:"max_log_size"` // bytes
	MaxLogFiles  int           `json:"max_log_files"`
	LogRetention time.Duration `json:"log_retention"`

	// Buffering settings
	BufferSize    int           `json:"buffer_size"`
	FlushInterval time.Duration `json:"flush_interval"`
	AsyncLogging  bool          `json:"async_logging"`

	// Security settings
	EncryptLogs    bool `json:"encrypt_logs"`
	CompressLogs   bool `json:"compress_logs"`
	IntegrityCheck bool `json:"integrity_check"`

	// Filtering settings
	IncludeMetadata bool     `json:"include_metadata"`
	ExcludeFields   []string `json:"exclude_fields"`
	IncludeFields   []string `json:"include_fields"`
}

// AuditEvent represents an audit log event
type AuditEvent struct {
	// Event identification
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	EventLevel string    `json:"event_level"`
	Timestamp  time.Time `json:"timestamp"`

	// User/Client information
	UserID    string `json:"user_id,omitempty"`
	ClientIP  string `json:"client_ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	SessionID string `json:"session_id,omitempty"`

	// Request information
	RequestID  string `json:"request_id,omitempty"`
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`

	// Security information
	APIKey      string   `json:"api_key,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	AuthMethod  string   `json:"auth_method,omitempty"`

	// Event details
	Message  string                 `json:"message"`
	Details  map[string]interface{} `json:"details,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Security indicators
	RiskLevel    string `json:"risk_level,omitempty"`  // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	ThreatType   string `json:"threat_type,omitempty"` // "AUTHENTICATION", "AUTHORIZATION", "DATA_ACCESS", etc.
	IsSuspicious bool   `json:"is_suspicious,omitempty"`

	// Integrity
	Hash      string `json:"hash,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config AuditConfig) (*AuditLogger, error) {
	ctx, cancel := context.WithCancel(context.Background())

	al := &AuditLogger{
		config:    config,
		logBuffer: make([]AuditEvent, 0, config.BufferSize),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Open log file
	if err := al.openLogFile(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Start background processes
	if config.AsyncLogging {
		al.wg.Add(1)
		go al.flushWorker()
	}

	return al, nil
}

// Log logs an audit event
func (al *AuditLogger) Log(event AuditEvent) error {
	// Set default values
	if event.EventID == "" {
		event.EventID = al.generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.EventLevel == "" {
		event.EventLevel = "INFO"
	}

	// Calculate hash for integrity
	if al.config.IntegrityCheck {
		event.Hash = al.calculateHash(event)
	}

	// Add to buffer
	al.bufferMutex.Lock()
	al.logBuffer = append(al.logBuffer, event)
	al.bufferMutex.Unlock()

	// Flush immediately if not async or buffer is full
	if !al.config.AsyncLogging || len(al.logBuffer) >= al.config.BufferSize {
		return al.flush()
	}

	return nil
}

// LogAuthentication logs authentication events
func (al *AuditLogger) LogAuthentication(userID, method, result string, details map[string]interface{}) error {
	event := AuditEvent{
		EventType:    "AUTHENTICATION",
		EventLevel:   al.getLevelFromResult(result),
		UserID:       userID,
		AuthMethod:   method,
		Message:      fmt.Sprintf("Authentication %s for user %s using %s", result, userID, method),
		Details:      details,
		ThreatType:   "AUTHENTICATION",
		RiskLevel:    al.getRiskLevel(result),
		IsSuspicious: result == "FAILED",
	}

	return al.Log(event)
}

// LogAuthorization logs authorization events
func (al *AuditLogger) LogAuthorization(userID, resource, action, result string, details map[string]interface{}) error {
	event := AuditEvent{
		EventType:    "AUTHORIZATION",
		EventLevel:   al.getLevelFromResult(result),
		UserID:       userID,
		Message:      fmt.Sprintf("Authorization %s for user %s to %s %s", result, userID, action, resource),
		Details:      details,
		ThreatType:   "AUTHORIZATION",
		RiskLevel:    al.getRiskLevel(result),
		IsSuspicious: result == "DENIED",
	}

	return al.Log(event)
}

// LogDataAccess logs data access events
func (al *AuditLogger) LogDataAccess(userID, resource, action string, details map[string]interface{}) error {
	event := AuditEvent{
		EventType:  "DATA_ACCESS",
		EventLevel: "INFO",
		UserID:     userID,
		Message:    fmt.Sprintf("Data access: user %s %s %s", userID, action, resource),
		Details:    details,
		ThreatType: "DATA_ACCESS",
		RiskLevel:  "LOW",
	}

	return al.Log(event)
}

// LogSecurityEvent logs security-related events
func (al *AuditLogger) LogSecurityEvent(eventType, message string, riskLevel string, details map[string]interface{}) error {
	event := AuditEvent{
		EventType:    eventType,
		EventLevel:   al.getLevelFromRisk(riskLevel),
		Message:      message,
		Details:      details,
		ThreatType:   eventType,
		RiskLevel:    riskLevel,
		IsSuspicious: riskLevel == "HIGH" || riskLevel == "CRITICAL",
	}

	return al.Log(event)
}

// LogAPIRequest logs API request events
func (al *AuditLogger) LogAPIRequest(requestID, method, path string, statusCode int, userID, clientIP string, details map[string]interface{}) error {
	event := AuditEvent{
		EventType:    "API_REQUEST",
		EventLevel:   al.getLevelFromStatusCode(statusCode),
		RequestID:    requestID,
		Method:       method,
		Path:         path,
		StatusCode:   statusCode,
		UserID:       userID,
		ClientIP:     clientIP,
		Message:      fmt.Sprintf("API request: %s %s -> %d", method, path, statusCode),
		Details:      details,
		ThreatType:   "API_ACCESS",
		RiskLevel:    al.getRiskLevelFromStatusCode(statusCode),
		IsSuspicious: statusCode >= 400,
	}

	return al.Log(event)
}

// openLogFile opens the log file for writing
func (al *AuditLogger) openLogFile() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(al.config.LogFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	file, err := os.OpenFile(al.config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	al.logFile = file
	return nil
}

// flushWorker runs in the background to flush logs periodically
func (al *AuditLogger) flushWorker() {
	defer al.wg.Done()

	ticker := time.NewTicker(al.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-al.ctx.Done():
			al.flush() // Final flush
			return
		case <-ticker.C:
			al.flush()
		}
	}
}

// flush writes buffered logs to file
func (al *AuditLogger) flush() error {
	al.bufferMutex.Lock()
	if len(al.logBuffer) == 0 {
		al.bufferMutex.Unlock()
		return nil
	}

	// Copy buffer and clear it
	events := make([]AuditEvent, len(al.logBuffer))
	copy(events, al.logBuffer)
	al.logBuffer = al.logBuffer[:0]
	al.bufferMutex.Unlock()

	// Write events to file
	al.logMutex.Lock()
	defer al.logMutex.Unlock()

	for _, event := range events {
		if err := al.writeEvent(event); err != nil {
			return fmt.Errorf("failed to write audit event: %w", err)
		}
	}

	return nil
}

// writeEvent writes a single audit event to the log file
func (al *AuditLogger) writeEvent(event AuditEvent) error {
	var logLine string

	switch al.config.LogFormat {
	case "JSON":
		jsonData, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal audit event: %w", err)
		}
		logLine = string(jsonData)
	case "TEXT":
		logLine = al.formatTextEvent(event)
	default:
		return fmt.Errorf("unsupported log format: %s", al.config.LogFormat)
	}

	// Write to file
	if _, err := al.logFile.WriteString(logLine + "\n"); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}

// formatTextEvent formats an audit event as text
func (al *AuditLogger) formatTextEvent(event AuditEvent) string {
	return fmt.Sprintf("[%s] %s %s: %s (User: %s, IP: %s, Risk: %s)",
		event.Timestamp.Format(time.RFC3339),
		event.EventLevel,
		event.EventType,
		event.Message,
		event.UserID,
		event.ClientIP,
		event.RiskLevel,
	)
}

// generateEventID generates a unique event ID
func (al *AuditLogger) generateEventID() string {
	return fmt.Sprintf("audit_%d_%d", time.Now().UnixNano(), len(al.logBuffer))
}

// calculateHash calculates a hash for the audit event
func (al *AuditLogger) calculateHash(event AuditEvent) string {
	// Create a copy without hash field for hashing
	eventCopy := event
	eventCopy.Hash = ""

	jsonData, _ := json.Marshal(eventCopy)
	return fmt.Sprintf("%x", jsonData) // Simple hash for demo
}

// getLevelFromResult converts result to log level
func (al *AuditLogger) getLevelFromResult(result string) string {
	switch result {
	case "SUCCESS":
		return "INFO"
	case "FAILED", "DENIED":
		return "WARN"
	case "ERROR":
		return "ERROR"
	default:
		return "INFO"
	}
}

// getLevelFromRisk converts risk level to log level
func (al *AuditLogger) getLevelFromRisk(riskLevel string) string {
	switch riskLevel {
	case "LOW":
		return "INFO"
	case "MEDIUM":
		return "WARN"
	case "HIGH":
		return "ERROR"
	case "CRITICAL":
		return "CRITICAL"
	default:
		return "INFO"
	}
}

// getLevelFromStatusCode converts HTTP status code to log level
func (al *AuditLogger) getLevelFromStatusCode(statusCode int) string {
	switch {
	case statusCode < 300:
		return "INFO"
	case statusCode < 400:
		return "INFO"
	case statusCode < 500:
		return "WARN"
	default:
		return "ERROR"
	}
}

// getRiskLevel converts result to risk level
func (al *AuditLogger) getRiskLevel(result string) string {
	switch result {
	case "SUCCESS":
		return "LOW"
	case "FAILED", "DENIED":
		return "MEDIUM"
	case "ERROR":
		return "HIGH"
	default:
		return "LOW"
	}
}

// getRiskLevelFromStatusCode converts HTTP status code to risk level
func (al *AuditLogger) getRiskLevelFromStatusCode(statusCode int) string {
	switch {
	case statusCode < 300:
		return "LOW"
	case statusCode < 400:
		return "LOW"
	case statusCode < 500:
		return "MEDIUM"
	default:
		return "HIGH"
	}
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	al.cancel()
	al.wg.Wait()

	// Final flush
	if err := al.flush(); err != nil {
		return fmt.Errorf("failed to flush logs on close: %w", err)
	}

	// Close log file
	if al.logFile != nil {
		return al.logFile.Close()
	}

	return nil
}

// GetLogStatistics returns statistics about audit logging
func (al *AuditLogger) GetLogStatistics() AuditStatistics {
	al.bufferMutex.Lock()
	defer al.bufferMutex.Unlock()

	stats := AuditStatistics{
		BufferSize:    len(al.logBuffer),
		MaxBufferSize: al.config.BufferSize,
		TotalEvents:   len(al.logBuffer),
		LastFlush:     time.Now(), // This would be tracked in a real implementation
	}

	// Count events by level
	levelCounts := make(map[string]int)
	for _, event := range al.logBuffer {
		levelCounts[event.EventLevel]++
	}
	stats.EventsByLevel = levelCounts

	// Count events by type
	typeCounts := make(map[string]int)
	for _, event := range al.logBuffer {
		typeCounts[event.EventType]++
	}
	stats.EventsByType = typeCounts

	// Count suspicious events
	suspiciousCount := 0
	for _, event := range al.logBuffer {
		if event.IsSuspicious {
			suspiciousCount++
		}
	}
	stats.SuspiciousEvents = suspiciousCount

	return stats
}

// AuditStatistics represents statistics about audit logging
type AuditStatistics struct {
	BufferSize       int            `json:"buffer_size"`
	MaxBufferSize    int            `json:"max_buffer_size"`
	TotalEvents      int            `json:"total_events"`
	EventsByLevel    map[string]int `json:"events_by_level"`
	EventsByType     map[string]int `json:"events_by_type"`
	SuspiciousEvents int            `json:"suspicious_events"`
	LastFlush        time.Time      `json:"last_flush"`
}

// ReadAuditLogs reads audit logs from file
func (al *AuditLogger) ReadAuditLogs(limit int, filter AuditFilter) ([]AuditEvent, error) {
	file, err := os.Open(al.config.LogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file for reading: %w", err)
	}
	defer file.Close()

	var events []AuditEvent
	decoder := json.NewDecoder(file)

	for {
		var event AuditEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			continue // Skip malformed lines
		}

		// Apply filter
		if filter.Matches(event) {
			events = append(events, event)
			if len(events) >= limit {
				break
			}
		}
	}

	return events, nil
}

// AuditFilter defines filtering criteria for audit logs
type AuditFilter struct {
	EventTypes  []string   `json:"event_types,omitempty"`
	EventLevels []string   `json:"event_levels,omitempty"`
	UserIDs     []string   `json:"user_ids,omitempty"`
	ClientIPs   []string   `json:"client_ips,omitempty"`
	RiskLevels  []string   `json:"risk_levels,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Suspicious  *bool      `json:"suspicious,omitempty"`
}

// Matches checks if an audit event matches the filter criteria
func (af *AuditFilter) Matches(event AuditEvent) bool {
	// Check event type
	if len(af.EventTypes) > 0 {
		found := false
		for _, eventType := range af.EventTypes {
			if event.EventType == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check event level
	if len(af.EventLevels) > 0 {
		found := false
		for _, level := range af.EventLevels {
			if event.EventLevel == level {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check user ID
	if len(af.UserIDs) > 0 {
		found := false
		for _, userID := range af.UserIDs {
			if event.UserID == userID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check client IP
	if len(af.ClientIPs) > 0 {
		found := false
		for _, clientIP := range af.ClientIPs {
			if event.ClientIP == clientIP {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check risk level
	if len(af.RiskLevels) > 0 {
		found := false
		for _, riskLevel := range af.RiskLevels {
			if event.RiskLevel == riskLevel {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check time range
	if af.StartTime != nil && event.Timestamp.Before(*af.StartTime) {
		return false
	}
	if af.EndTime != nil && event.Timestamp.After(*af.EndTime) {
		return false
	}

	// Check suspicious flag
	if af.Suspicious != nil && event.IsSuspicious != *af.Suspicious {
		return false
	}

	return true
}
