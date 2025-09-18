package analytics

import (
	"time"
)

// WorkloadPattern represents a customer usage pattern
type WorkloadPattern struct {
	CustomerID        string                 `json:"customer_id"`
	PatternID         string                 `json:"pattern_id"`
	ResourceType      string                 `json:"resource_type"`
	TypicalQuantity   int                    `json:"typical_quantity"`
	TypicalDuration   float64                `json:"typical_duration"`
	TypicalStartTimes []int                  `json:"typical_start_times"` // Hours of day
	FrequencyDays     float64                `json:"frequency_days"`      // How often (days between runs)
	Seasonality       map[string]float64     `json:"seasonality"`         // Monthly/weekly patterns
	CostSensitivity   float64                `json:"cost_sensitivity"`    // 0-1, how price-sensitive they are
	SLARequirements   map[string]interface{} `json:"sla_requirements"`
}

// UsageRecord represents a single usage record
type UsageRecord struct {
	Timestamp        int64                  `json:"timestamp"`
	ResourceType     string                 `json:"resource_type"`
	Region           string                 `json:"region"`
	Quantity         int                    `json:"quantity"`
	DurationHours    float64                `json:"duration_hours"`
	TotalCost        float64                `json:"total_cost"`
	SLARequirements  map[string]interface{} `json:"sla_requirements"`
	WorkloadType     string                 `json:"workload_type"`
	StartTimeHour    int                    `json:"start_time_hour"`
	DayOfWeek        int                    `json:"day_of_week"`
	DayOfMonth       int                    `json:"day_of_month"`
}

// UsagePrediction represents a predicted usage
type UsagePrediction struct {
	CustomerID           string                 `json:"customer_id"`
	PredictedResourceType string                `json:"predicted_resource_type"`
	PredictedQuantity    int                    `json:"predicted_quantity"`
	PredictedDuration    float64                `json:"predicted_duration"`
	PredictedStartTime   int64                  `json:"predicted_start_time"`
	Confidence           float64                `json:"confidence"`
	PatternID            string                 `json:"pattern_id"`
	CostSensitivity      float64                `json:"cost_sensitivity"`
	SLARequirements      map[string]interface{} `json:"sla_requirements"`
}

// CustomerInsights represents comprehensive customer insights
type CustomerInsights struct {
	CustomerID                string                    `json:"customer_id"`
	UsageSummary              UsageSummary              `json:"usage_summary"`
	ResourceBreakdown         map[string]ResourceUsage  `json:"resource_breakdown"`
	UsagePatterns             int                       `json:"usage_patterns"`
	PeakUsageHours            []int                     `json:"peak_usage_hours"`
	CostSensitivity           float64                   `json:"cost_sensitivity"`
	NextUsagePredictions      []UsagePrediction         `json:"next_usage_predictions"`
	OptimizationOpportunities []OptimizationOpportunity `json:"optimization_opportunities"`
}

// UsageSummary represents usage summary statistics
type UsageSummary struct {
	TotalSessions     int     `json:"total_sessions"`
	TotalComputeHours float64 `json:"total_compute_hours"`
	TotalSpend        float64 `json:"total_spend"`
	AvgCostPerHour    float64 `json:"avg_cost_per_hour"`
	AvgFrequencyDays  float64 `json:"avg_frequency_days"`
}

// ResourceUsage represents usage statistics for a resource type
type ResourceUsage struct {
	Count       int     `json:"count"`
	TotalHours  float64 `json:"total_hours"`
	TotalCost   float64 `json:"total_cost"`
}

// OptimizationOpportunity represents an optimization opportunity
type OptimizationOpportunity struct {
	Type             string `json:"type"`
	ResourceType     string `json:"resource_type"`
	PotentialSavings string `json:"potential_savings"`
	Description      string `json:"description"`
}
