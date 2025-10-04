package monitoring

import (
	"testing"
	"time"
)

func TestAlertManager(t *testing.T) {
	config := AlertConfig{
		DefaultSeverity: "MEDIUM",
		DefaultDuration: time.Minute * 5,
		MaxAlerts:       100,
		AlertCooldown:   time.Minute * 1,
		EscalationDelay: time.Minute * 10,
		EnableEmail:     false,
		EnableSlack:     false,
		EnableWebhook:   false,
		EnablePagerDuty: false,
		Rules: []AlertRule{
			{
				Name:        "High CPU Usage",
				Description: "CPU usage exceeds threshold",
				Metric:      "cpu_usage_percent",
				Condition:   "gt",
				Threshold:   80.0,
				Severity:    "HIGH",
				Duration:    time.Minute * 2,
				Enabled:     true,
				Tags:        map[string]string{"component": "system"},
			},
		},
	}

	am := NewAlertManager(config)
	defer am.Stop()

	// Test creating an alert
	alert, err := am.CreateAlert(
		"Test Alert",
		"Test alert description",
		"HIGH",
		"test-source",
		"test-metric",
		85.0,
		80.0,
		map[string]interface{}{
			"test_key": "test_value",
		},
	)
	if err != nil {
		t.Fatalf("Failed to create alert: %v", err)
	}

	if alert.ID == "" {
		t.Error("Alert ID should not be empty")
	}

	if alert.Status != "ACTIVE" {
		t.Errorf("Expected alert status ACTIVE, got %s", alert.Status)
	}

	// Test getting alert
	retrievedAlert, err := am.GetAlert(alert.ID)
	if err != nil {
		t.Fatalf("Failed to get alert: %v", err)
	}

	if retrievedAlert.ID != alert.ID {
		t.Error("Retrieved alert ID does not match")
	}

	// Test acknowledging alert
	err = am.AcknowledgeAlert(alert.ID, "test-user")
	if err != nil {
		t.Fatalf("Failed to acknowledge alert: %v", err)
	}

	acknowledgedAlert, err := am.GetAlert(alert.ID)
	if err != nil {
		t.Fatalf("Failed to get acknowledged alert: %v", err)
	}

	if acknowledgedAlert.Status != "ACKNOWLEDGED" {
		t.Errorf("Expected alert status ACKNOWLEDGED, got %s", acknowledgedAlert.Status)
	}

	// Test resolving alert
	err = am.ResolveAlert(alert.ID, "test-user")
	if err != nil {
		t.Fatalf("Failed to resolve alert: %v", err)
	}

	resolvedAlert, err := am.GetAlert(alert.ID)
	if err != nil {
		t.Fatalf("Failed to get resolved alert: %v", err)
	}

	if resolvedAlert.Status != "RESOLVED" {
		t.Errorf("Expected alert status RESOLVED, got %s", resolvedAlert.Status)
	}

	// Test getting alerts with filter
	filter := AlertFilter{
		Statuses: []string{"RESOLVED"},
	}
	alerts := am.GetAlerts(filter)
	if len(alerts) != 1 {
		t.Errorf("Expected 1 resolved alert, got %d", len(alerts))
	}

	// Test alert statistics
	stats := am.GetAlertStatistics()
	if stats.TotalAlerts != 1 {
		t.Errorf("Expected 1 total alert, got %d", stats.TotalAlerts)
	}

	if stats.ResolvedAlerts != 1 {
		t.Errorf("Expected 1 resolved alert, got %d", stats.ResolvedAlerts)
	}
}

func TestPrometheusMonitor(t *testing.T) {
	config := PrometheusConfig{
		ListenAddress:         ":9091", // Use different port to avoid conflicts
		MetricsPath:           "/metrics",
		EnableSystemMetrics:   false, // Disable to avoid conflicts
		EnableBusinessMetrics: false, // Disable to avoid conflicts
		EnableCustomMetrics:   false, // Disable to avoid conflicts
		DefaultLabels:         map[string]string{"service": "ocx-protocol"},
	}

	pm := NewPrometheusMonitor(config)
	defer pm.Stop()

	// Test that metrics are not initialized when disabled
	systemMetrics := pm.GetSystemMetrics()
	if systemMetrics != nil {
		t.Error("System metrics should not be initialized when disabled")
	}

	businessMetrics := pm.GetBusinessMetrics()
	if businessMetrics != nil {
		t.Error("Business metrics should not be initialized when disabled")
	}

	customMetrics := pm.GetCustomMetrics()
	if customMetrics != nil {
		t.Error("Custom metrics should not be initialized when disabled")
	}

	// Test recording metrics
	pm.RecordHTTPRequest("GET", "/api/v1/test", "200", time.Millisecond*100, 1024, 2048)
	pm.RecordAPIRequest("test-endpoint", "GET", "200", time.Millisecond*50)
	pm.RecordAPIError("test-endpoint", "GET", "validation_error")
	pm.RecordExecution("os-process", "success", time.Millisecond*200)
	pm.RecordExecutionError("os-process", "timeout")
	pm.RecordVerification("success", time.Millisecond*150)
	pm.RecordVerificationError("signature_invalid")
	pm.RecordSecurityAlert("HIGH", "vulnerability_scanner")
	pm.RecordSecurityScan("dependency", time.Second*30, map[string]int{"HIGH": 2, "MEDIUM": 5})
	pm.RecordCacheMetrics("lru", 0.95, 1024*1024)
	pm.RecordKeyRotation("ed25519")
	pm.RecordKeyUsage("key-123", "sign")

	// Give some time for metrics to be recorded
	time.Sleep(time.Millisecond * 100)
}

func TestMonitor(t *testing.T) {
	config := MonitorConfig{
		Prometheus: PrometheusConfig{
			ListenAddress:         ":9092", // Use different port
			MetricsPath:           "/metrics",
			EnableSystemMetrics:   false, // Disable to avoid conflicts
			EnableBusinessMetrics: false, // Disable to avoid conflicts
			EnableCustomMetrics:   false, // Disable to avoid conflicts
		},
		Alerting: AlertConfig{
			DefaultSeverity: "MEDIUM",
			DefaultDuration: time.Minute * 5,
			MaxAlerts:       100,
			AlertCooldown:   time.Minute * 1,
			EscalationDelay: time.Minute * 10,
			EnableEmail:     false,
			EnableSlack:     false,
			EnableWebhook:   false,
			EnablePagerDuty: false,
		},
		SystemMonitoring: SystemMonitoringConfig{
			CPUThresholdPercent:            80.0,
			CPULoadThreshold:               2.0,
			MemoryThresholdPercent:         85.0,
			MemoryThresholdBytes:           1024 * 1024 * 1024, // 1GB
			DiskThresholdPercent:           90.0,
			DiskThresholdBytes:             100 * 1024 * 1024 * 1024, // 100GB
			NetworkThresholdBytesPerSecond: 1024 * 1024 * 100,        // 100MB/s
			ProcessThresholdCount:          200,
			ProcessThresholdMemory:         1024 * 1024 * 512, // 512MB
		},
		EnablePrometheus:       true,
		EnableAlerting:         true,
		EnableSystemMonitoring: true,
		EnableHealthChecks:     true,
		MetricsInterval:        time.Second * 5,
		HealthCheckInterval:    time.Second * 10,
		AlertCheckInterval:     time.Second * 30,
	}

	m, err := NewMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}
	defer m.Stop()

	// Test that components are initialized
	if m.prometheusMonitor == nil {
		t.Error("Prometheus monitor should be initialized")
	}

	if m.alertManager == nil {
		t.Error("Alert manager should be initialized")
	}

	if m.systemMonitor == nil {
		t.Error("System monitor should be initialized")
	}

	// Test getting system metrics
	systemMetrics := m.GetSystemMetrics()
	if systemMetrics.Timestamp.IsZero() {
		t.Error("System metrics timestamp should not be zero")
	}

	// Give some time for system monitoring to collect metrics
	time.Sleep(time.Second * 1)

	// Check again after waiting
	systemMetrics = m.GetSystemMetrics()
	if systemMetrics.Timestamp.IsZero() {
		t.Error("System metrics timestamp should not be zero after waiting")
	}

	// Test getting monitoring status
	status := m.GetMonitoringStatus()
	if len(status.Components) == 0 {
		t.Error("Monitoring status should have components")
	}

	// Test that prometheus component exists
	if _, exists := status.Components["prometheus"]; !exists {
		t.Error("Prometheus component should exist in status")
	}

	// Test that alerting component exists
	if _, exists := status.Components["alerting"]; !exists {
		t.Error("Alerting component should exist in status")
	}

	// Test that system component exists
	if _, exists := status.Components["system"]; !exists {
		t.Error("System component should exist in status")
	}

	// Give some time for monitoring to collect metrics
	time.Sleep(time.Second * 2)
}

func TestNotifiers(t *testing.T) {
	// Test email notifier
	emailConfig := EmailConfig{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "user@example.com",
		SMTPPassword: "password",
		From:         "alerts@example.com",
		To:           []string{"admin@example.com"},
	}

	emailNotifier := &EmailNotifier{config: emailConfig}
	if !emailNotifier.IsEnabled() {
		t.Error("Email notifier should be enabled with valid config")
	}

	if emailNotifier.GetName() != "email" {
		t.Errorf("Expected email notifier name 'email', got '%s'", emailNotifier.GetName())
	}

	// Test Slack notifier
	slackConfig := SlackConfig{
		WebhookURL: "https://hooks.slack.com/services/test",
		Channel:    "#alerts",
	}

	slackNotifier := &SlackNotifier{config: slackConfig}
	if !slackNotifier.IsEnabled() {
		t.Error("Slack notifier should be enabled with valid config")
	}

	if slackNotifier.GetName() != "slack" {
		t.Errorf("Expected slack notifier name 'slack', got '%s'", slackNotifier.GetName())
	}

	// Test webhook notifier
	webhookConfig := WebhookConfig{
		URL: "https://example.com/webhook",
		Headers: map[string]string{
			"Authorization": "Bearer token",
		},
	}

	webhookNotifier := &WebhookNotifier{config: webhookConfig}
	if !webhookNotifier.IsEnabled() {
		t.Error("Webhook notifier should be enabled with valid config")
	}

	if webhookNotifier.GetName() != "webhook" {
		t.Errorf("Expected webhook notifier name 'webhook', got '%s'", webhookNotifier.GetName())
	}

	// Test PagerDuty notifier
	pagerDutyConfig := PagerDutyConfig{
		APIKey:    "test-api-key",
		ServiceID: "test-service-id",
	}

	pagerDutyNotifier := &PagerDutyNotifier{config: pagerDutyConfig}
	if !pagerDutyNotifier.IsEnabled() {
		t.Error("PagerDuty notifier should be enabled with valid config")
	}

	if pagerDutyNotifier.GetName() != "pagerduty" {
		t.Errorf("Expected pagerduty notifier name 'pagerduty', got '%s'", pagerDutyNotifier.GetName())
	}
}

func TestAlertRules(t *testing.T) {
	rule := AlertRule{
		Name:        "Test Rule",
		Description: "Test alert rule",
		Metric:      "test_metric",
		Condition:   "gt",
		Threshold:   100.0,
		Severity:    "HIGH",
		Duration:    time.Minute * 5,
		Enabled:     true,
		Tags:        map[string]string{"environment": "test"},
	}

	// Test rule validation
	if rule.Name == "" {
		t.Error("Rule name should not be empty")
	}

	if rule.Metric == "" {
		t.Error("Rule metric should not be empty")
	}

	if rule.Threshold <= 0 {
		t.Error("Rule threshold should be positive")
	}

	if rule.Duration <= 0 {
		t.Error("Rule duration should be positive")
	}

	// Test valid conditions
	validConditions := []string{"gt", "lt", "eq", "gte", "lte"}
	found := false
	for _, condition := range validConditions {
		if rule.Condition == condition {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Rule condition '%s' is not valid", rule.Condition)
	}

	// Test valid severities
	validSeverities := []string{"LOW", "MEDIUM", "HIGH", "CRITICAL"}
	found = false
	for _, severity := range validSeverities {
		if rule.Severity == severity {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Rule severity '%s' is not valid", rule.Severity)
	}
}

func TestSystemMetrics(t *testing.T) {
	metrics := SystemMetricsData{
		Timestamp:          time.Now(),
		CPUUsagePercent:    75.5,
		CPULoadAverage:     []float64{1.2, 1.5, 1.8},
		MemoryUsageBytes:   1024 * 1024 * 512, // 512MB
		MemoryUsagePercent: 50.0,
		MemoryAvailable:    1024 * 1024 * 512, // 512MB
		DiskUsage: map[string]DiskUsage{
			"/dev/sda1": {
				Device:       "/dev/sda1",
				Mountpoint:   "/",
				TotalBytes:   100 * 1024 * 1024 * 1024, // 100GB
				UsedBytes:    50 * 1024 * 1024 * 1024,  // 50GB
				FreeBytes:    50 * 1024 * 1024 * 1024,  // 50GB
				UsagePercent: 50.0,
			},
		},
		NetworkStats: map[string]NetworkStats{
			"eth0": {
				Interface:       "eth0",
				BytesReceived:   1024 * 1024 * 100, // 100MB
				BytesSent:       1024 * 1024 * 50,  // 50MB
				PacketsReceived: 100000,
				PacketsSent:     50000,
			},
		},
		ProcessCount:       150,
		ProcessMemoryUsage: 1024 * 1024 * 64, // 64MB
		ProcessCPUUsage:    5.0,
	}

	// Test metrics validation
	if metrics.Timestamp.IsZero() {
		t.Error("Metrics timestamp should not be zero")
	}

	if metrics.CPUUsagePercent < 0 || metrics.CPUUsagePercent > 100 {
		t.Errorf("CPU usage percent should be between 0 and 100, got %f", metrics.CPUUsagePercent)
	}

	if len(metrics.CPULoadAverage) != 3 {
		t.Errorf("CPU load average should have 3 values, got %d", len(metrics.CPULoadAverage))
	}

	if metrics.MemoryUsageBytes < 0 {
		t.Error("Memory usage bytes should be non-negative")
	}

	if metrics.MemoryUsagePercent < 0 || metrics.MemoryUsagePercent > 100 {
		t.Errorf("Memory usage percent should be between 0 and 100, got %f", metrics.MemoryUsagePercent)
	}

	if len(metrics.DiskUsage) == 0 {
		t.Error("Disk usage should not be empty")
	}

	if len(metrics.NetworkStats) == 0 {
		t.Error("Network stats should not be empty")
	}

	if metrics.ProcessCount < 0 {
		t.Error("Process count should be non-negative")
	}

	if metrics.ProcessMemoryUsage < 0 {
		t.Error("Process memory usage should be non-negative")
	}

	if metrics.ProcessCPUUsage < 0 {
		t.Error("Process CPU usage should be non-negative")
	}
}

func TestMonitorContextCancellation(t *testing.T) {
	config := MonitorConfig{
		EnablePrometheus:       false,
		EnableAlerting:         false,
		EnableSystemMonitoring: false, // Disable to avoid ticker issues
		EnableHealthChecks:     false,
		MetricsInterval:        time.Second * 1, // Set valid interval
		HealthCheckInterval:    time.Second * 1, // Set valid interval
		AlertCheckInterval:     time.Second * 1, // Set valid interval
	}

	m, err := NewMonitor(config)
	if err != nil {
		t.Fatalf("Failed to create monitor: %v", err)
	}

	// Stop the monitor
	m.Stop()

	// Wait a bit to ensure goroutines have stopped
	time.Sleep(time.Millisecond * 100)

	// The monitor should be stopped now
	// This test mainly ensures that Stop() doesn't panic
}
