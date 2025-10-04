package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AlertManager manages alerts and notifications
type AlertManager struct {
	// Configuration
	config AlertConfig

	// Alert state
	alerts      map[string]*Alert
	alertsMutex sync.RWMutex

	// Notification channels
	notifiers []Notifier

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// AlertConfig defines configuration for alerting
type AlertConfig struct {
	// Alert settings
	DefaultSeverity string        `json:"default_severity"` // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	DefaultDuration time.Duration `json:"default_duration"` // How long to wait before alerting
	MaxAlerts       int           `json:"max_alerts"`       // Maximum number of active alerts
	AlertCooldown   time.Duration `json:"alert_cooldown"`   // Cooldown period between similar alerts
	EscalationDelay time.Duration `json:"escalation_delay"` // Time before escalating alerts

	// Notification settings
	EnableEmail     bool `json:"enable_email"`
	EnableSlack     bool `json:"enable_slack"`
	EnableWebhook   bool `json:"enable_webhook"`
	EnablePagerDuty bool `json:"enable_pagerduty"`

	// Email settings
	SMTPHost     string   `json:"smtp_host"`
	SMTPPort     int      `json:"smtp_port"`
	SMTPUsername string   `json:"smtp_username"`
	SMTPPassword string   `json:"smtp_password"`
	EmailFrom    string   `json:"email_from"`
	EmailTo      []string `json:"email_to"`

	// Slack settings
	SlackWebhookURL string `json:"slack_webhook_url"`
	SlackChannel    string `json:"slack_channel"`

	// Webhook settings
	WebhookURL     string            `json:"webhook_url"`
	WebhookHeaders map[string]string `json:"webhook_headers"`

	// PagerDuty settings
	PagerDutyAPIKey    string `json:"pagerduty_api_key"`
	PagerDutyServiceID string `json:"pagerduty_service_id"`

	// Alert rules
	Rules []AlertRule `json:"rules"`
}

// Alert represents an active alert
type Alert struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Severity       string                 `json:"severity"`  // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	Status         string                 `json:"status"`    // "ACTIVE", "ACKNOWLEDGED", "RESOLVED"
	Source         string                 `json:"source"`    // Component that triggered the alert
	Metric         string                 `json:"metric"`    // Metric that triggered the alert
	Value          float64                `json:"value"`     // Current value
	Threshold      float64                `json:"threshold"` // Threshold that was exceeded
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty"`
	ResolvedAt     *time.Time             `json:"resolved_at,omitempty"`
	Metadata       map[string]interface{} `json:"metadata"`
	Notifications  []Notification         `json:"notifications"`
}

// AlertRule defines when to trigger alerts
type AlertRule struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Metric      string            `json:"metric"`
	Condition   string            `json:"condition"` // "gt", "lt", "eq", "gte", "lte"
	Threshold   float64           `json:"threshold"`
	Severity    string            `json:"severity"`
	Duration    time.Duration     `json:"duration"` // How long condition must be true
	Enabled     bool              `json:"enabled"`
	Tags        map[string]string `json:"tags"`
}

// Notification represents a sent notification
type Notification struct {
	ID         string    `json:"id"`
	Channel    string    `json:"channel"` // "email", "slack", "webhook", "pagerduty"
	Status     string    `json:"status"`  // "sent", "failed", "pending"
	SentAt     time.Time `json:"sent_at"`
	Error      string    `json:"error,omitempty"`
	RetryCount int       `json:"retry_count"`
}

// Notifier interface for sending notifications
type Notifier interface {
	Send(alert *Alert) error
	GetName() string
	IsEnabled() bool
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config AlertConfig) *AlertManager {
	ctx, cancel := context.WithCancel(context.Background())

	am := &AlertManager{
		config: config,
		alerts: make(map[string]*Alert),
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize notifiers
	am.initializeNotifiers()

	// Start alert processing
	am.wg.Add(1)
	go am.alertProcessingLoop()

	return am
}

// initializeNotifiers initializes notification channels
func (am *AlertManager) initializeNotifiers() {
	if am.config.EnableEmail {
		emailNotifier := &EmailNotifier{
			config: EmailConfig{
				SMTPHost:     am.config.SMTPHost,
				SMTPPort:     am.config.SMTPPort,
				SMTPUsername: am.config.SMTPUsername,
				SMTPPassword: am.config.SMTPPassword,
				From:         am.config.EmailFrom,
				To:           am.config.EmailTo,
			},
		}
		am.notifiers = append(am.notifiers, emailNotifier)
	}

	if am.config.EnableSlack {
		slackNotifier := &SlackNotifier{
			config: SlackConfig{
				WebhookURL: am.config.SlackWebhookURL,
				Channel:    am.config.SlackChannel,
			},
		}
		am.notifiers = append(am.notifiers, slackNotifier)
	}

	if am.config.EnableWebhook {
		webhookNotifier := &WebhookNotifier{
			config: WebhookConfig{
				URL:     am.config.WebhookURL,
				Headers: am.config.WebhookHeaders,
			},
		}
		am.notifiers = append(am.notifiers, webhookNotifier)
	}

	if am.config.EnablePagerDuty {
		pagerDutyNotifier := &PagerDutyNotifier{
			config: PagerDutyConfig{
				APIKey:    am.config.PagerDutyAPIKey,
				ServiceID: am.config.PagerDutyServiceID,
			},
		}
		am.notifiers = append(am.notifiers, pagerDutyNotifier)
	}
}

// alertProcessingLoop processes alerts and sends notifications
func (am *AlertManager) alertProcessingLoop() {
	defer am.wg.Done()

	ticker := time.NewTicker(time.Second * 30) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.processAlerts()
		}
	}
}

// processAlerts processes active alerts
func (am *AlertManager) processAlerts() {
	am.alertsMutex.RLock()
	activeAlerts := make([]*Alert, 0)
	for _, alert := range am.alerts {
		if alert.Status == "ACTIVE" {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	am.alertsMutex.RUnlock()

	// Process each active alert
	for _, alert := range activeAlerts {
		am.processAlert(alert)
	}
}

// processAlert processes a single alert
func (am *AlertManager) processAlert(alert *Alert) {
	// Check if alert needs escalation
	if am.shouldEscalate(alert) {
		am.escalateAlert(alert)
	}

	// Send notifications if needed
	if am.shouldNotify(alert) {
		am.sendNotifications(alert)
	}
}

// shouldEscalate checks if an alert should be escalated
func (am *AlertManager) shouldEscalate(alert *Alert) bool {
	if alert.Status != "ACTIVE" {
		return false
	}

	// Escalate if alert has been active for longer than escalation delay
	return time.Since(alert.CreatedAt) > am.config.EscalationDelay
}

// escalateAlert escalates an alert
func (am *AlertManager) escalateAlert(alert *Alert) {
	// Increase severity
	switch alert.Severity {
	case "LOW":
		alert.Severity = "MEDIUM"
	case "MEDIUM":
		alert.Severity = "HIGH"
	case "HIGH":
		alert.Severity = "CRITICAL"
	}

	alert.UpdatedAt = time.Now()
	alert.Metadata["escalated"] = true
	alert.Metadata["escalated_at"] = time.Now()

	// Send escalation notifications
	am.sendNotifications(alert)
}

// shouldNotify checks if notifications should be sent
func (am *AlertManager) shouldNotify(alert *Alert) bool {
	if alert.Status != "ACTIVE" {
		return false
	}

	// Check cooldown period
	lastNotification := am.getLastNotification(alert)
	if lastNotification != nil {
		if time.Since(lastNotification.SentAt) < am.config.AlertCooldown {
			return false
		}
	}

	return true
}

// getLastNotification gets the most recent notification for an alert
func (am *AlertManager) getLastNotification(alert *Alert) *Notification {
	if len(alert.Notifications) == 0 {
		return nil
	}

	var lastNotification *Notification
	for _, notification := range alert.Notifications {
		if lastNotification == nil || notification.SentAt.After(lastNotification.SentAt) {
			lastNotification = &notification
		}
	}

	return lastNotification
}

// sendNotifications sends notifications for an alert
func (am *AlertManager) sendNotifications(alert *Alert) {
	for _, notifier := range am.notifiers {
		if !notifier.IsEnabled() {
			continue
		}

		notification := Notification{
			ID:         fmt.Sprintf("%s-%d", alert.ID, time.Now().Unix()),
			Channel:    notifier.GetName(),
			Status:     "pending",
			SentAt:     time.Now(),
			RetryCount: 0,
		}

		// Send notification
		err := notifier.Send(alert)
		if err != nil {
			notification.Status = "failed"
			notification.Error = err.Error()
		} else {
			notification.Status = "sent"
		}

		// Add notification to alert
		alert.Notifications = append(alert.Notifications, notification)
		alert.UpdatedAt = time.Now()
	}
}

// CreateAlert creates a new alert
func (am *AlertManager) CreateAlert(name, description, severity, source, metric string, value, threshold float64, metadata map[string]interface{}) (*Alert, error) {
	alertID := fmt.Sprintf("%s-%s-%d", source, metric, time.Now().Unix())

	alert := &Alert{
		ID:            alertID,
		Name:          name,
		Description:   description,
		Severity:      severity,
		Status:        "ACTIVE",
		Source:        source,
		Metric:        metric,
		Value:         value,
		Threshold:     threshold,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Metadata:      metadata,
		Notifications: make([]Notification, 0),
	}

	// Set default severity if not provided
	if alert.Severity == "" {
		alert.Severity = am.config.DefaultSeverity
	}

	am.alertsMutex.Lock()
	am.alerts[alertID] = alert
	am.alertsMutex.Unlock()

	// Send initial notifications
	am.sendNotifications(alert)

	return alert, nil
}

// AcknowledgeAlert acknowledges an alert
func (am *AlertManager) AcknowledgeAlert(alertID string, userID string) error {
	am.alertsMutex.Lock()
	defer am.alertsMutex.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}

	if alert.Status != "ACTIVE" {
		return fmt.Errorf("alert %s is not active", alertID)
	}

	alert.Status = "ACKNOWLEDGED"
	alert.AcknowledgedAt = &time.Time{}
	*alert.AcknowledgedAt = time.Now()
	alert.UpdatedAt = time.Now()
	alert.Metadata["acknowledged_by"] = userID

	return nil
}

// ResolveAlert resolves an alert
func (am *AlertManager) ResolveAlert(alertID string, userID string) error {
	am.alertsMutex.Lock()
	defer am.alertsMutex.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}

	alert.Status = "RESOLVED"
	alert.ResolvedAt = &time.Time{}
	*alert.ResolvedAt = time.Now()
	alert.UpdatedAt = time.Now()
	alert.Metadata["resolved_by"] = userID

	return nil
}

// GetAlert returns an alert by ID
func (am *AlertManager) GetAlert(alertID string) (*Alert, error) {
	am.alertsMutex.RLock()
	defer am.alertsMutex.RUnlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return nil, fmt.Errorf("alert %s not found", alertID)
	}

	return alert, nil
}

// GetAlerts returns all alerts with optional filtering
func (am *AlertManager) GetAlerts(filter AlertFilter) []*Alert {
	am.alertsMutex.RLock()
	defer am.alertsMutex.RUnlock()

	var filteredAlerts []*Alert
	for _, alert := range am.alerts {
		if am.matchesFilter(alert, filter) {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}

	return filteredAlerts
}

// matchesFilter checks if an alert matches the filter criteria
func (am *AlertManager) matchesFilter(alert *Alert, filter AlertFilter) bool {
	if len(filter.Statuses) > 0 {
		found := false
		for _, status := range filter.Statuses {
			if alert.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.Severities) > 0 {
		found := false
		for _, severity := range filter.Severities {
			if alert.Severity == severity {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.Sources) > 0 {
		found := false
		for _, source := range filter.Sources {
			if alert.Source == source {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if filter.StartTime != nil && alert.CreatedAt.Before(*filter.StartTime) {
		return false
	}

	if filter.EndTime != nil && alert.CreatedAt.After(*filter.EndTime) {
		return false
	}

	return true
}

// AlertFilter defines filtering criteria for alerts
type AlertFilter struct {
	Statuses   []string   `json:"statuses,omitempty"`
	Severities []string   `json:"severities,omitempty"`
	Sources    []string   `json:"sources,omitempty"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
}

// GetAlertStatistics returns alert statistics
func (am *AlertManager) GetAlertStatistics() AlertStatistics {
	am.alertsMutex.RLock()
	defer am.alertsMutex.RUnlock()

	stats := AlertStatistics{
		TotalAlerts:        len(am.alerts),
		ActiveAlerts:       0,
		AcknowledgedAlerts: 0,
		ResolvedAlerts:     0,
		BySeverity:         make(map[string]int),
		BySource:           make(map[string]int),
	}

	for _, alert := range am.alerts {
		switch alert.Status {
		case "ACTIVE":
			stats.ActiveAlerts++
		case "ACKNOWLEDGED":
			stats.AcknowledgedAlerts++
		case "RESOLVED":
			stats.ResolvedAlerts++
		}

		stats.BySeverity[alert.Severity]++
		stats.BySource[alert.Source]++
	}

	return stats
}

// AlertStatistics represents alert statistics
type AlertStatistics struct {
	TotalAlerts        int            `json:"total_alerts"`
	ActiveAlerts       int            `json:"active_alerts"`
	AcknowledgedAlerts int            `json:"acknowledged_alerts"`
	ResolvedAlerts     int            `json:"resolved_alerts"`
	BySeverity         map[string]int `json:"by_severity"`
	BySource           map[string]int `json:"by_source"`
}

// Stop stops the alert manager
func (am *AlertManager) Stop() {
	am.cancel()
	am.wg.Wait()
}
