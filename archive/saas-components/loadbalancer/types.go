package loadbalancer

// LoadBalanceDecision represents a load balancing decision
type LoadBalanceDecision struct {
	TargetProviders       []string          `json:"target_providers"`
	AllocationPercentages []float64         `json:"allocation_percentages"`
	ExpectedPerformance   map[string]float64 `json:"expected_performance"`
	CostEfficiency        float64           `json:"cost_efficiency"`
	RiskScore             float64           `json:"risk_score"`
	Reasoning             string            `json:"reasoning"`
}

// ProviderPerformance represents performance metrics for a provider
type ProviderPerformance struct {
	ProviderID           string  `json:"provider_id"`
	AllocationPercentage float64 `json:"allocation_percentage"`
	HealthScore          float64 `json:"health_score"`
	APIResponseTime      float64 `json:"api_response_time"`
	NetworkLatency       float64 `json:"network_latency"`
	Uptime               float64 `json:"uptime"`
	CapacityUtilization  float64 `json:"capacity_utilization"`
}

// PerformanceRecord represents a performance record for monitoring
type PerformanceRecord struct {
	Timestamp           int64                        `json:"timestamp"`
	WorkloadID          string                       `json:"workload_id"`
	Providers           map[string]ProviderPerformance `json:"providers"`
	OverallCostEfficiency float64                    `json:"overall_cost_efficiency"`
	OverallRiskScore    float64                      `json:"overall_risk_score"`
}

// WorkloadSpec represents a workload specification for load balancing
type WorkloadSpec struct {
	ResourceType      string                 `json:"resource_type"`
	Region            string                 `json:"region"`
	Quantity          int                    `json:"quantity"`
	DurationHours     float64                `json:"duration_hours"`
	SLARequirements   map[string]interface{} `json:"sla_requirements"`
	OptimizationGoal  string                 `json:"optimization_goal"`
	AllocationTime    int64                  `json:"allocation_time,omitempty"`
}
