package monitoring

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// EmailNotifier sends alerts via email
type EmailNotifier struct {
	config EmailConfig
}

// EmailConfig defines email notification configuration
type EmailConfig struct {
	SMTPHost     string   `json:"smtp_host"`
	SMTPPort     int      `json:"smtp_port"`
	SMTPUsername string   `json:"smtp_username"`
	SMTPPassword string   `json:"smtp_password"`
	From         string   `json:"from"`
	To           []string `json:"to"`
}

// Send sends an alert via email
func (en *EmailNotifier) Send(alert *Alert) error {
	if !en.IsEnabled() {
		return fmt.Errorf("email notifier is not enabled")
	}

	subject := fmt.Sprintf("[%s] %s - %s", alert.Severity, alert.Source, alert.Name)
	body := en.formatEmailBody(alert)

	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		en.config.From,
		strings.Join(en.config.To, ","),
		subject,
		body)

	auth := smtp.PlainAuth("", en.config.SMTPUsername, en.config.SMTPPassword, en.config.SMTPHost)
	addr := fmt.Sprintf("%s:%d", en.config.SMTPHost, en.config.SMTPPort)

	return smtp.SendMail(addr, auth, en.config.From, en.config.To, []byte(message))
}

// formatEmailBody formats the email body
func (en *EmailNotifier) formatEmailBody(alert *Alert) string {
	var body strings.Builder

	body.WriteString(fmt.Sprintf("Alert: %s\n", alert.Name))
	body.WriteString(fmt.Sprintf("Description: %s\n", alert.Description))
	body.WriteString(fmt.Sprintf("Severity: %s\n", alert.Severity))
	body.WriteString(fmt.Sprintf("Source: %s\n", alert.Source))
	body.WriteString(fmt.Sprintf("Metric: %s\n", alert.Metric))
	body.WriteString(fmt.Sprintf("Value: %.2f\n", alert.Value))
	body.WriteString(fmt.Sprintf("Threshold: %.2f\n", alert.Threshold))
	body.WriteString(fmt.Sprintf("Created: %s\n", alert.CreatedAt.Format(time.RFC3339)))
	body.WriteString(fmt.Sprintf("Status: %s\n", alert.Status))

	if len(alert.Metadata) > 0 {
		body.WriteString("\nMetadata:\n")
		for key, value := range alert.Metadata {
			body.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
	}

	return body.String()
}

// GetName returns the notifier name
func (en *EmailNotifier) GetName() string {
	return "email"
}

// IsEnabled checks if email notifier is enabled
func (en *EmailNotifier) IsEnabled() bool {
	return en.config.SMTPHost != "" && len(en.config.To) > 0
}

// SlackNotifier sends alerts via Slack
type SlackNotifier struct {
	config SlackConfig
}

// SlackConfig defines Slack notification configuration
type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
}

// Send sends an alert via Slack
func (sn *SlackNotifier) Send(alert *Alert) error {
	if !sn.IsEnabled() {
		return fmt.Errorf("slack notifier is not enabled")
	}

	payload := sn.formatSlackPayload(alert)
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	resp, err := http.Post(sn.config.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// formatSlackPayload formats the Slack payload
func (sn *SlackNotifier) formatSlackPayload(alert *Alert) map[string]interface{} {
	// Determine color based on severity
	color := "good" // green
	switch alert.Severity {
	case "MEDIUM":
		color = "warning" // yellow
	case "HIGH":
		color = "danger" // red
	case "CRITICAL":
		color = "danger" // red
	}

	// Determine emoji based on severity
	emoji := "✅"
	switch alert.Severity {
	case "MEDIUM":
		emoji = "⚠️"
	case "HIGH":
		emoji = "🚨"
	case "CRITICAL":
		emoji = "🔥"
	}

	title := fmt.Sprintf("%s %s - %s", emoji, alert.Severity, alert.Name)
	text := fmt.Sprintf("*Description:* %s\n*Source:* %s\n*Metric:* %s\n*Value:* %.2f (threshold: %.2f)\n*Status:* %s",
		alert.Description,
		alert.Source,
		alert.Metric,
		alert.Value,
		alert.Threshold,
		alert.Status)

	payload := map[string]interface{}{
		"channel": sn.config.Channel,
		"attachments": []map[string]interface{}{
			{
				"color":     color,
				"title":     title,
				"text":      text,
				"timestamp": alert.CreatedAt.Unix(),
				"fields": []map[string]interface{}{
					{
						"title": "Alert ID",
						"value": alert.ID,
						"short": true,
					},
					{
						"title": "Created",
						"value": alert.CreatedAt.Format(time.RFC3339),
						"short": true,
					},
				},
			},
		},
	}

	return payload
}

// GetName returns the notifier name
func (sn *SlackNotifier) GetName() string {
	return "slack"
}

// IsEnabled checks if Slack notifier is enabled
func (sn *SlackNotifier) IsEnabled() bool {
	return sn.config.WebhookURL != ""
}

// WebhookNotifier sends alerts via webhook
type WebhookNotifier struct {
	config WebhookConfig
}

// WebhookConfig defines webhook notification configuration
type WebhookConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// Send sends an alert via webhook
func (wn *WebhookNotifier) Send(alert *Alert) error {
	if !wn.IsEnabled() {
		return fmt.Errorf("webhook notifier is not enabled")
	}

	jsonData, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	req, err := http.NewRequest("POST", wn.config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range wn.config.Headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: time.Second * 30}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// GetName returns the notifier name
func (wn *WebhookNotifier) GetName() string {
	return "webhook"
}

// IsEnabled checks if webhook notifier is enabled
func (wn *WebhookNotifier) IsEnabled() bool {
	return wn.config.URL != ""
}

// PagerDutyNotifier sends alerts via PagerDuty
type PagerDutyNotifier struct {
	config PagerDutyConfig
}

// PagerDutyConfig defines PagerDuty notification configuration
type PagerDutyConfig struct {
	APIKey    string `json:"api_key"`
	ServiceID string `json:"service_id"`
}

// Send sends an alert via PagerDuty
func (pn *PagerDutyNotifier) Send(alert *Alert) error {
	if !pn.IsEnabled() {
		return fmt.Errorf("pagerduty notifier is not enabled")
	}

	payload := pn.formatPagerDutyPayload(alert)
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal pagerduty payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://events.pagerduty.com/v2/enqueue", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create pagerduty request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token token="+pn.config.APIKey)

	client := &http.Client{Timeout: time.Second * 30}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send pagerduty alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pagerduty returned status %d", resp.StatusCode)
	}

	return nil
}

// formatPagerDutyPayload formats the PagerDuty payload
func (pn *PagerDutyNotifier) formatPagerDutyPayload(alert *Alert) map[string]interface{} {
	// Determine severity mapping
	severity := "info"
	switch alert.Severity {
	case "MEDIUM":
		severity = "warning"
	case "HIGH":
		severity = "error"
	case "CRITICAL":
		severity = "critical"
	}

	payload := map[string]interface{}{
		"routing_key":  pn.config.ServiceID,
		"event_action": "trigger",
		"dedup_key":    alert.ID,
		"payload": map[string]interface{}{
			"summary":   fmt.Sprintf("%s - %s", alert.Source, alert.Name),
			"source":    alert.Source,
			"severity":  severity,
			"timestamp": alert.CreatedAt.Format(time.RFC3339),
			"component": alert.Metric,
			"group":     "ocx-protocol",
			"class":     "monitoring",
			"custom_details": map[string]interface{}{
				"description": alert.Description,
				"value":       alert.Value,
				"threshold":   alert.Threshold,
				"status":      alert.Status,
				"metadata":    alert.Metadata,
			},
		},
	}

	return payload
}

// GetName returns the notifier name
func (pn *PagerDutyNotifier) GetName() string {
	return "pagerduty"
}

// IsEnabled checks if PagerDuty notifier is enabled
func (pn *PagerDutyNotifier) IsEnabled() bool {
	return pn.config.APIKey != "" && pn.config.ServiceID != ""
}
