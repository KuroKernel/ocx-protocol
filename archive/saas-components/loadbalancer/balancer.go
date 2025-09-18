// balancer.go - Global Load Balancer for OCX Protocol
package loadbalancer

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"
)

// GlobalLoadBalancer manages load balancing across providers
type GlobalLoadBalancer struct {
	providers map[string]ProviderData
	mu        sync.RWMutex
}

// ProviderData represents provider information
type ProviderData struct {
	ProviderID        string  `json:"provider_id"`
	Name              string  `json:"name"`
	Region            string  `json:"region"`
	ResourceType      string  `json:"resource_type"`
	PricePerHour      float64 `json:"price_per_hour"`
	AvailableQuantity int     `json:"available_quantity"`
	HealthScore       float64 `json:"health_score"`
	APISuccessRate    float64 `json:"api_success_rate"`
	Latency           int     `json:"latency_ms"`
}

// LoadBalanceDecision represents a load balancing decision
type LoadBalanceDecision struct {
	TargetProviders       []string          `json:"target_providers"`
	AllocationPercentages []float64         `json:"allocation_percentages"`
	ExpectedPerformance   map[string]float64 `json:"expected_performance"`
	CostEfficiency        float64           `json:"cost_efficiency"`
	RiskScore             float64           `json:"risk_score"`
	Reasoning             string            `json:"reasoning"`
}

// ProviderPerformance represents provider performance metrics
type ProviderPerformance struct {
	ProviderID    string    `json:"provider_id"`
	Uptime        float64   `json:"uptime_percent"`
	Latency       int       `json:"latency_ms"`
	Throughput    float64   `json:"throughput_mbps"`
	ErrorRate     float64   `json:"error_rate"`
	LastUpdated   time.Time `json:"last_updated"`
}

// NewGlobalLoadBalancer creates a new global load balancer
func NewGlobalLoadBalancer() *GlobalLoadBalancer {
	return &GlobalLoadBalancer{
		providers: make(map[string]ProviderData),
	}
}

// AddProvider adds a provider to the load balancer
func (g *GlobalLoadBalancer) AddProvider(provider ProviderData) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.providers[provider.ProviderID] = provider
}

// GetProvider returns a provider by ID
func (g *GlobalLoadBalancer) GetProvider(providerID string) (ProviderData, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	provider, exists := g.providers[providerID]
	return provider, exists
}

// BalanceLoad performs load balancing for a request
func (g *GlobalLoadBalancer) BalanceLoad(ctx context.Context, resourceType, region string, quantity int, slaRequirements map[string]interface{}) (*LoadBalanceDecision, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Filter suitable providers
	var suitableProviders []ProviderData
	for _, provider := range g.providers {
		if provider.ResourceType == resourceType && provider.Region == region {
			if provider.AvailableQuantity >= quantity {
				suitableProviders = append(suitableProviders, provider)
			}
		}
	}

	if len(suitableProviders) == 0 {
		return &LoadBalanceDecision{
			TargetProviders:       []string{},
			AllocationPercentages: []float64{},
			ExpectedPerformance:   map[string]float64{},
			CostEfficiency:        0,
			RiskScore:             1.0,
			Reasoning:             "No suitable providers found",
		}, nil
	}

	// Calculate optimal allocation strategy
	optimizationGoal := "balanced"
	if goal, ok := slaRequirements["optimization_goal"].(string); ok {
		optimizationGoal = goal
	}

	switch optimizationGoal {
	case "cost":
		return g.optimizeForCost(suitableProviders, quantity, slaRequirements), nil
	case "performance":
		return g.optimizeForPerformance(suitableProviders, quantity, slaRequirements), nil
	case "reliability":
		return g.optimizeForReliability(suitableProviders, quantity, slaRequirements), nil
	default: // balanced
		return g.optimizeBalanced(suitableProviders, quantity, slaRequirements), nil
	}
}

// optimizeForCost optimizes for cost efficiency
func (g *GlobalLoadBalancer) optimizeForCost(providers []ProviderData, quantity int, slaRequirements map[string]interface{}) *LoadBalanceDecision {
	// Sort by price
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].PricePerHour < providers[j].PricePerHour
	})

	// Allocate to cheapest provider
	allocation := []string{providers[0].ProviderID}
	percentages := []float64{1.0}

	expectedPerformance := map[string]float64{
		"total_cost_per_hour": providers[0].PricePerHour * float64(quantity),
		"avg_reliability":     providers[0].APISuccessRate * 100,
		"avg_performance_score": providers[0].HealthScore,
		"provider_diversity":   1.0,
	}

	return &LoadBalanceDecision{
		TargetProviders:       allocation,
		AllocationPercentages: percentages,
		ExpectedPerformance:   expectedPerformance,
		CostEfficiency:        1.0,
		RiskScore:             0.8,
		Reasoning:             "Cost optimization: selected cheapest provider",
	}
}

// optimizeForPerformance optimizes for performance
func (g *GlobalLoadBalancer) optimizeForPerformance(providers []ProviderData, quantity int, slaRequirements map[string]interface{}) *LoadBalanceDecision {
	// Sort by health score
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].HealthScore > providers[j].HealthScore
	})

	// Allocate to best performing provider
	allocation := []string{providers[0].ProviderID}
	percentages := []float64{1.0}

	expectedPerformance := map[string]float64{
		"total_cost_per_hour": providers[0].PricePerHour * float64(quantity),
		"avg_reliability":     providers[0].APISuccessRate * 100,
		"avg_performance_score": providers[0].HealthScore,
		"provider_diversity":   1.0,
	}

	return &LoadBalanceDecision{
		TargetProviders:       allocation,
		AllocationPercentages: percentages,
		ExpectedPerformance:   expectedPerformance,
		CostEfficiency:        0.6,
		RiskScore:             0.3,
		Reasoning:             "Performance optimization: selected highest performing provider",
	}
}

// optimizeForReliability optimizes for reliability
func (g *GlobalLoadBalancer) optimizeForReliability(providers []ProviderData, quantity int, slaRequirements map[string]interface{}) *LoadBalanceDecision {
	// Sort by API success rate
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].APISuccessRate > providers[i].APISuccessRate
	})

	// Allocate to most reliable provider
	allocation := []string{providers[0].ProviderID}
	percentages := []float64{1.0}

	expectedPerformance := map[string]float64{
		"total_cost_per_hour": providers[0].PricePerHour * float64(quantity),
		"avg_reliability":     providers[0].APISuccessRate * 100,
		"avg_performance_score": providers[0].HealthScore,
		"provider_diversity":   1.0,
	}

	return &LoadBalanceDecision{
		TargetProviders:       allocation,
		AllocationPercentages: percentages,
		ExpectedPerformance:   expectedPerformance,
		CostEfficiency:        0.7,
		RiskScore:             0.2,
		Reasoning:             "Reliability optimization: selected most reliable provider",
	}
}

// optimizeBalanced optimizes for balanced cost, performance, and reliability
func (g *GlobalLoadBalancer) optimizeBalanced(providers []ProviderData, quantity int, slaRequirements map[string]interface{}) *LoadBalanceDecision {
	// Calculate combined scores
	type providerScore struct {
		provider ProviderData
		score    float64
	}

	var providerScores []providerScore
	for _, provider := range providers {
		// Normalize scores to 0-1 range
		costScore := 1.0 / (1.0 + provider.PricePerHour/5.0) // Lower cost = higher score
		performanceScore := math.Min(1.0, provider.HealthScore/100)
		reliabilityScore := provider.APISuccessRate
		availabilityScore := math.Min(1.0, float64(provider.AvailableQuantity)/float64(quantity))

		// Weighted combination (balanced)
		combinedScore := costScore*0.3 + performanceScore*0.25 + reliabilityScore*0.25 + availabilityScore*0.2

		providerScores = append(providerScores, providerScore{
			provider: provider,
			score:    combinedScore,
		})
	}

	// Sort by combined score
	sort.Slice(providerScores, func(i, j int) bool {
		return providerScores[i].score > providerScores[j].score
	})

	// Allocate across top providers
	var allocation []string
	var percentages []float64
	remainingQuantity := quantity

	for _, ps := range providerScores {
		if remainingQuantity <= 0 {
			break
		}

		allocation = append(allocation, ps.provider.ProviderID)
		
		// Calculate percentage for this provider
		providerQuantity := math.Min(float64(remainingQuantity), float64(ps.provider.AvailableQuantity))
		percentage := providerQuantity / float64(quantity)
		percentages = append(percentages, percentage)
		
		remainingQuantity -= int(providerQuantity)
	}

	// Calculate expected performance
	totalCost := 0.0
	avgReliability := 0.0
	avgPerformance := 0.0

	for i, providerID := range allocation {
		for _, ps := range providerScores {
			if ps.provider.ProviderID == providerID {
				weight := percentages[i]
				totalCost += ps.provider.PricePerHour * weight * float64(quantity)
				avgReliability += ps.provider.APISuccessRate * weight
				avgPerformance += ps.provider.HealthScore * weight
				break
			}
		}
	}

	expectedPerformance := map[string]float64{
		"total_cost_per_hour":  totalCost,
		"avg_reliability":      avgReliability * 100,
		"avg_performance_score": avgPerformance,
		"provider_diversity":   float64(len(allocation)),
	}

	return &LoadBalanceDecision{
		TargetProviders:       allocation,
		AllocationPercentages: percentages,
		ExpectedPerformance:   expectedPerformance,
		CostEfficiency:        0.8, // Good balance
		RiskScore:             0.4, // Moderate risk due to diversification
		Reasoning:             "Balanced optimization: optimal mix of cost, performance, and reliability",
	}
}

// MonitorLoadPerformance monitors performance of load-balanced workload
func (g *GlobalLoadBalancer) MonitorLoadPerformance(workloadID string, allocation *LoadBalanceDecision) map[string]ProviderPerformance {
	g.mu.Lock()
	defer g.mu.Unlock()

	performanceMetrics := make(map[string]ProviderPerformance)

	for i, providerID := range allocation.TargetProviders {
		provider, exists := g.providers[providerID]
		if !exists {
			continue
		}

		// Simulate performance metrics (in real implementation, this would query actual metrics)
		performance := ProviderPerformance{
			ProviderID:  providerID,
			Uptime:      95.0 + float64(i)*2.0, // Simulate different uptimes
			Latency:     provider.Latency + i*10, // Simulate different latencies
			Throughput:  100.0 - float64(i)*5.0, // Simulate different throughputs
			ErrorRate:   0.1 + float64(i)*0.05, // Simulate different error rates
			LastUpdated: time.Now(),
		}

		performanceMetrics[providerID] = performance
	}

	return performanceMetrics
}

// GetLoadBalancerStats returns load balancer statistics
func (g *GlobalLoadBalancer) GetLoadBalancerStats() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()

	totalProviders := len(g.providers)
	if totalProviders == 0 {
		return map[string]interface{}{
			"total_providers": 0,
			"avg_health":      0.0,
			"avg_latency":     0.0,
			"total_capacity":  0,
		}
	}

	var totalHealth, totalLatency float64
	var totalCapacity int

	for _, provider := range g.providers {
		totalHealth += provider.HealthScore
		totalLatency += float64(provider.Latency)
		totalCapacity += provider.AvailableQuantity
	}

	return map[string]interface{}{
		"total_providers": totalProviders,
		"avg_health":      totalHealth / float64(totalProviders),
		"avg_latency":     totalLatency / float64(totalProviders),
		"total_capacity":  totalCapacity,
	}
}
