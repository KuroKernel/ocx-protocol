// sla_monitor.go — Real-Time SLA Monitoring and Enforcement
// Integrates with existing telemetry system

package telemetry

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SLAMonitor provides real-time SLA monitoring and enforcement
type SLAMonitor struct {
	db        *sql.DB
	telemetry *TelemetryCollector
	configs   map[string]*SLAConfig
}

// SLAConfig defines SLA requirements for a lease
type SLAConfig struct {
	LeaseID           string            `json:"lease_id"`
	GuaranteedUptime  float64           `json:"guaranteed_uptime"`  // Percentage (0-100)
	MaxLatency        float64           `json:"max_latency_ms"`     // Maximum latency in milliseconds
	MinThroughput     float64           `json:"min_throughput_ops"` // Minimum operations per second
	MaxErrorRate      float64           `json:"max_error_rate"`     // Maximum error rate (0-1)
	AutoClawback      bool              `json:"auto_clawback_enabled"`
	PenaltyAmount     uint64            `json:"penalty_micro_units"`
	CheckInterval     time.Duration     `json:"check_interval"`
	GracePeriod       time.Duration     `json:"grace_period"`
	PerformanceMetrics map[string]float64 `json:"performance_metrics"`
}

// SLAMetrics represents current SLA performance
type SLAMetrics struct {
	LeaseID           string            `json:"lease_id"`
	ActualUptime      float64           `json:"actual_uptime"`
	AverageLatency    float64           `json:"average_latency_ms"`
	CurrentThroughput float64           `json:"current_throughput_ops"`
	ErrorRate         float64           `json:"error_rate"`
	Compliance        float64           `json:"compliance_percentage"`
	Breaches          int               `json:"breaches_count"`
	LastCheck         time.Time         `json:"last_check"`
	Status            string            `json:"status"` // "compliant", "warning", "breach"
	PerformanceMetrics map[string]float64 `json:"performance_metrics"`
}

// ClawbackTransaction represents an automatic penalty for SLA breach
type ClawbackTransaction struct {
	TransactionID string    `json:"transaction_id"`
	LeaseID       string    `json:"lease_id"`
	Amount        uint64    `json:"amount_micro_units"`
	Reason        string    `json:"reason"`
	Automatic     bool      `json:"automatic"`
	Receipt       string    `json:"receipt_hash"`
	Timestamp     time.Time `json:"timestamp"`
	Status        string    `json:"status"` // "pending", "processed", "failed"
}

// NewSLAMonitor creates a new SLA monitoring system
func NewSLAMonitor(db *sql.DB, telemetry *TelemetryCollector) *SLAMonitor {
	return &SLAMonitor{
		db:        db,
		telemetry: telemetry,
		configs:   make(map[string]*SLAConfig),
	}
}

// RegisterSLAConfig registers SLA requirements for a lease
func (sla *SLAMonitor) RegisterSLAConfig(config *SLAConfig) error {
	// Store SLA configuration
	query := `
		INSERT INTO sla_configs (lease_id, guaranteed_uptime, max_latency, min_throughput, 
		                        max_error_rate, auto_clawback, penalty_amount, check_interval, 
		                        grace_period, performance_metrics)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (lease_id) DO UPDATE SET
			guaranteed_uptime = EXCLUDED.guaranteed_uptime,
			max_latency = EXCLUDED.max_latency,
			min_throughput = EXCLUDED.min_throughput,
			max_error_rate = EXCLUDED.max_error_rate,
			auto_clawback = EXCLUDED.auto_clawback,
			penalty_amount = EXCLUDED.penalty_amount,
			check_interval = EXCLUDED.check_interval,
			grace_period = EXCLUDED.grace_period,
			performance_metrics = EXCLUDED.performance_metrics
	`
	
	_, err := sla.db.Exec(query,
		config.LeaseID,
		config.GuaranteedUptime,
		config.MaxLatency,
		config.MinThroughput,
		config.MaxErrorRate,
		config.AutoClawback,
		config.PenaltyAmount,
		config.CheckInterval,
		config.GracePeriod,
		config.PerformanceMetrics,
	)
	
	if err != nil {
		return fmt.Errorf("failed to register SLA config: %w", err)
	}
	
	sla.configs[config.LeaseID] = config
	return nil
}

// MonitorSLA performs real-time SLA monitoring for a lease
func (sla *SLAMonitor) MonitorSLA(leaseID string) (*SLAMetrics, error) {
	config, exists := sla.configs[leaseID]
	if !exists {
		return nil, fmt.Errorf("SLA config not found for lease %s", leaseID)
	}

	// Get recent telemetry data
	metrics, err := sla.getRecentMetrics(leaseID, config.CheckInterval)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent metrics: %w", err)
	}

	// Calculate SLA metrics
	slaMetrics := sla.calculateSLAMetrics(leaseID, config, metrics)

	// Store SLA metrics
	err = sla.storeSLAMetrics(slaMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to store SLA metrics: %w", err)
	}

	// Check for SLA breaches
	if slaMetrics.Status == "breach" && config.AutoClawback {
		clawback, err := sla.EnforceSLA(leaseID, slaMetrics)
		if err != nil {
			return nil, fmt.Errorf("failed to enforce SLA: %w", err)
		}
		if clawback != nil {
			slaMetrics.Breaches++
		}
	}

	return slaMetrics, nil
}

// getRecentMetrics retrieves recent telemetry data for SLA calculation
func (sla *SLAMonitor) getRecentMetrics(leaseID string, interval time.Duration) ([]MetricsSnapshot, error) {
	query := `
		SELECT session_id, timestamp, gpu_utilization_percent, gpu_temperature_celsius,
		       cpu_utilization_percent, training_steps_per_second, inference_tokens_per_second,
		       compliance_score, error_count, total_operations
		FROM ocx_session_metrics 
		WHERE session_id = $1 AND timestamp >= $2
		ORDER BY timestamp DESC
	`
	
	rows, err := sla.db.Query(query, leaseID, time.Now().Add(-interval))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []MetricsSnapshot
	for rows.Next() {
		var m MetricsSnapshot
		var errorCount, totalOps int
		err := rows.Scan(
			&m.SessionID, &m.Timestamp, &m.GPUUtilization, &m.GPUTemperature,
			&m.CPUUtilization, &m.TrainingStepsSec, &m.InferenceTokensSec,
			&m.ComplianceScore, &errorCount, &totalOps,
		)
		if err != nil {
			return nil, err
		}
		
		// Calculate error rate
		if totalOps > 0 {
			m.ErrorRate = float64(errorCount) / float64(totalOps)
		}
		
		metrics = append(metrics, m)
	}

	return metrics, nil
}

// calculateSLAMetrics calculates SLA compliance metrics
func (sla *SLAMonitor) calculateSLAMetrics(leaseID string, config *SLAConfig, metrics []MetricsSnapshot) *SLAMetrics {
	if len(metrics) == 0 {
		return &SLAMetrics{
			LeaseID:           leaseID,
			Status:            "warning",
			Compliance:        0.0,
			LastCheck:         time.Now(),
			PerformanceMetrics: make(map[string]float64),
		}
	}

	// Calculate uptime (based on GPU utilization > 0)
	uptimeCount := 0
	for _, m := range metrics {
		if m.GPUUtilization > 0 {
			uptimeCount++
		}
	}
	actualUptime := float64(uptimeCount) / float64(len(metrics)) * 100

	// Calculate average latency (simplified - using GPU utilization as proxy)
	var totalLatency float64
	for _, m := range metrics {
		// Convert GPU utilization to latency proxy (higher utilization = lower latency)
		latency := 100.0 - float64(m.GPUUtilization)
		totalLatency += latency
	}
	avgLatency := totalLatency / float64(len(metrics))

	// Calculate throughput
	var totalThroughput float64
	for _, m := range metrics {
		throughput := m.TrainingStepsSec + m.InferenceTokensSec
		totalThroughput += throughput
	}
	avgThroughput := totalThroughput / float64(len(metrics))

	// Calculate error rate
	var totalErrorRate float64
	for _, m := range metrics {
		totalErrorRate += m.ErrorRate
	}
	avgErrorRate := totalErrorRate / float64(len(metrics)) * 100

	// Calculate compliance score
	compliance := 100.0
	if actualUptime < config.GuaranteedUptime {
		compliance -= (config.GuaranteedUptime - actualUptime) * 2
	}
	if avgLatency > config.MaxLatency {
		compliance -= (avgLatency - config.MaxLatency) * 0.1
	}
	if avgThroughput < config.MinThroughput {
		compliance -= (config.MinThroughput - avgThroughput) * 0.01
	}
	if avgErrorRate > config.MaxErrorRate*100 {
		compliance -= (avgErrorRate - config.MaxErrorRate*100) * 2
	}

	if compliance < 0 {
		compliance = 0
	}

	// Determine status
	status := "compliant"
	if compliance < 95 {
		status = "warning"
	}
	if compliance < 90 {
		status = "breach"
	}

	// Count breaches
	breachCount := 0
	if compliance < 90 {
		breachCount = 1
	}

	// Performance metrics
	perfMetrics := make(map[string]float64)
	perfMetrics["gpu_utilization"] = float64(metrics[len(metrics)-1].GPUUtilization)
	perfMetrics["cpu_utilization"] = float64(metrics[len(metrics)-1].CPUUtilization)
	perfMetrics["gpu_temperature"] = float64(metrics[len(metrics)-1].GPUTemperature)
	perfMetrics["compliance_score"] = metrics[len(metrics)-1].ComplianceScore

	return &SLAMetrics{
		LeaseID:            leaseID,
		ActualUptime:       actualUptime,
		AverageLatency:     avgLatency,
		CurrentThroughput:  avgThroughput,
		ErrorRate:          avgErrorRate,
		Compliance:         compliance,
		Breaches:           breachCount,
		LastCheck:          time.Now(),
		Status:             status,
		PerformanceMetrics: perfMetrics,
	}
}

// storeSLAMetrics stores SLA metrics in the database
func (sla *SLAMonitor) storeSLAMetrics(metrics *SLAMetrics) error {
	query := `
		INSERT INTO sla_metrics (lease_id, actual_uptime, average_latency, current_throughput,
		                        error_rate, compliance, breaches_count, last_check, status,
		                        performance_metrics)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (lease_id) DO UPDATE SET
			actual_uptime = EXCLUDED.actual_uptime,
			average_latency = EXCLUDED.average_latency,
			current_throughput = EXCLUDED.current_throughput,
			error_rate = EXCLUDED.error_rate,
			compliance = EXCLUDED.compliance,
			breaches_count = EXCLUDED.breaches_count,
			last_check = EXCLUDED.last_check,
			status = EXCLUDED.status,
			performance_metrics = EXCLUDED.performance_metrics
	`
	
	_, err := sla.db.Exec(query,
		metrics.LeaseID,
		metrics.ActualUptime,
		metrics.AverageLatency,
		metrics.CurrentThroughput,
		metrics.ErrorRate,
		metrics.Compliance,
		metrics.Breaches,
		metrics.LastCheck,
		metrics.Status,
		metrics.PerformanceMetrics,
	)
	
	return err
}

// EnforceSLA enforces SLA penalties and clawbacks
func (sla *SLAMonitor) EnforceSLA(leaseID string, metrics *SLAMetrics) (*ClawbackTransaction, error) {
	config, exists := sla.configs[leaseID]
	if !exists {
		return nil, fmt.Errorf("SLA config not found for lease %s", leaseID)
	}

	if !config.AutoClawback {
		return nil, nil
	}

	// Calculate penalty based on compliance score
	penalty := uint64(0)
	if metrics.Compliance < 90 {
		penalty = config.PenaltyAmount
	} else if metrics.Compliance < 95 {
		penalty = config.PenaltyAmount / 2
	}

	if penalty == 0 {
		return nil, nil
	}

	// Create clawback transaction
	clawback := &ClawbackTransaction{
		TransactionID: generateTransactionID(),
		LeaseID:       leaseID,
		Amount:        penalty,
		Reason:        "SLA_BREACH",
		Automatic:     true,
		Receipt:       generateClawbackReceipt(penalty),
		Timestamp:     time.Now(),
		Status:        "pending",
	}

	// Store clawback transaction
	err := sla.storeClawbackTransaction(clawback)
	if err != nil {
		return nil, fmt.Errorf("failed to store clawback transaction: %w", err)
	}

	return clawback, nil
}

// storeClawbackTransaction stores a clawback transaction
func (sla *SLAMonitor) storeClawbackTransaction(clawback *ClawbackTransaction) error {
	query := `
		INSERT INTO clawback_transactions (transaction_id, lease_id, amount_micro_units,
		                                 reason, automatic, receipt_hash, timestamp, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	_, err := sla.db.Exec(query,
		clawback.TransactionID,
		clawback.LeaseID,
		clawback.Amount,
		clawback.Reason,
		clawback.Automatic,
		clawback.Receipt,
		clawback.Timestamp,
		clawback.Status,
	)
	
	return err
}

// Helper functions
func generateTransactionID() string {
	return fmt.Sprintf("clawback_%d", time.Now().UnixNano())
}

func generateClawbackReceipt(amount uint64) string {
	// Generate a receipt hash for the clawback transaction
	return fmt.Sprintf("receipt_%d_%d", amount, time.Now().UnixNano())
}
