package riskmanagement

import (
	"time"
)

// RiskLevel represents the risk level of a provider
type RiskLevel string

const (
	Minimal  RiskLevel = "minimal"  // 99.9%+ reliability
	Low      RiskLevel = "low"      // 99.5-99.9%
	Medium   RiskLevel = "medium"   // 98-99.5%
	High     RiskLevel = "high"     // 95-98%
	Critical RiskLevel = "critical" // <95%
)

// FailureType represents the type of failure
type FailureType string

const (
	Hardware    FailureType = "hardware"
	Network     FailureType = "network"
	API         FailureType = "api"
	Capacity    FailureType = "capacity"
	Billing     FailureType = "billing"
	Maintenance FailureType = "maintenance"
	Unknown     FailureType = "unknown"
)

// ProviderHealthMetrics represents health metrics for a provider
type ProviderHealthMetrics struct {
	ProviderID           string    `json:"provider_id"`
	Timestamp            int64     `json:"timestamp"`
	APIResponseTime      float64   `json:"api_response_time"`
	APISuccessRate       float64   `json:"api_success_rate"`
	InstanceFailureRate  float64   `json:"instance_failure_rate"`
	NetworkLatency       float64   `json:"network_latency"`
	CapacityAvailability float64   `json:"capacity_availability"`
	BillingIssues        int       `json:"billing_issues"`
	MaintenanceWindows   []TimeWindow `json:"maintenance_windows"`
	CustomerComplaints   int       `json:"customer_complaints"`
}

// TimeWindow represents a time window for maintenance
type TimeWindow struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// CalculateHealthScore calculates overall provider health score (0-100)
func (p *ProviderHealthMetrics) CalculateHealthScore() float64 {
	apiScore := minFloat(100, (p.APISuccessRate*100)-(p.APIResponseTime*10))
	reliabilityScore := (1 - p.InstanceFailureRate) * 100
	capacityScore := p.CapacityAvailability * 100
	networkScore := maxFloat(0, 100-(p.NetworkLatency*2))
	billingScore := maxFloat(0, 100-(float64(p.BillingIssues)*10))
	complaintScore := maxFloat(0, 100-(float64(p.CustomerComplaints)*5))
	
	weights := []float64{0.25, 0.25, 0.2, 0.15, 0.1, 0.05}
	scores := []float64{apiScore, reliabilityScore, capacityScore, networkScore, billingScore, complaintScore}
	
	totalScore := 0.0
	for i, weight := range weights {
		totalScore += weight * scores[i]
	}
	
	return totalScore
}

// FailoverPlan represents a failover plan for a provider
type FailoverPlan struct {
	PrimaryProvider       string            `json:"primary_provider"`
	BackupProviders       []string          `json:"backup_providers"`
	TriggerConditions     map[string]interface{} `json:"trigger_conditions"`
	FailoverLatencySeconds float64          `json:"failover_latency_seconds"`
	CostMultiplier        float64           `json:"cost_multiplier"`
	AutoFailback          bool              `json:"auto_failback"`
}

// FailureRecord represents a failure record
type FailureRecord struct {
	Timestamp   int64     `json:"timestamp"`
	FailureType FailureType `json:"failure_type"`
	HealthScore float64   `json:"health_score"`
	Metrics     ProviderHealthMetrics `json:"metrics"`
}

// FailoverRecord represents an active failover record
type FailoverRecord struct {
	PrimaryProvider   string    `json:"primary_provider"`
	BackupProviders   []string  `json:"backup_providers"`
	TriggerTime       int64     `json:"trigger_time"`
	TriggerReason     string    `json:"trigger_reason"`
	AffectedWorkloads []string  `json:"affected_workloads"`
	Status            string    `json:"status"`
}

// ProviderReliabilityReport represents a comprehensive reliability report
type ProviderReliabilityReport struct {
	Timestamp        int64                    `json:"timestamp"`
	ProviderCount    int                      `json:"provider_count"`
	ActiveFailovers  int                      `json:"active_failovers"`
	Providers        map[string]ProviderStats `json:"providers"`
}

// ProviderStats represents statistics for a provider
type ProviderStats struct {
	HealthScore         float64 `json:"health_score"`
	RiskLevel           string  `json:"risk_level"`
	APIResponseTime     float64 `json:"api_response_time"`
	UptimePercentage    float64 `json:"uptime_percentage"`
	CapacityUtilization float64 `json:"capacity_utilization"`
	RecentFailures      int     `json:"recent_failures"`
	MTBFHours           float64 `json:"mtbf_hours"`
	IsInFailover        bool    `json:"is_in_failover"`
	LastUpdated         int64   `json:"last_updated"`
}

// Helper functions
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
