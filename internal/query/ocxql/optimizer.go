// internal/query/ocxql/optimizer.go
package ocxql

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// ComputeResource represents a compute resource in the system
type ComputeResource struct {
	ID               string    `json:"id"`
	Type             string    `json:"type"`
	Provider         string    `json:"provider"`
	Region           string    `json:"region"`
	PricePerHour     float64   `json:"price_per_hour"`
	Availability     float64   `json:"availability"`     // SLA percentage
	MemoryGB         int       `json:"memory_gb"`
	Bandwidth        string    `json:"bandwidth"`
	LatencyMS        float64   `json:"latency_ms"`
	PowerEfficiency  float64   `json:"power_efficiency"` // FLOPS/Watt
	Interconnect     string    `json:"interconnect"`
	Resilience       string    `json:"resilience"`
	MaxUnits         int       `json:"max_units"`
	LastUpdated      time.Time `json:"last_updated"`
}

// ExecutionPlan represents an optimized execution plan
type ExecutionPlan struct {
	ID              string                    `json:"id"`
	Type            string                    `json:"type"` // "cost_optimized", "performance_optimized", "reliability_optimized"
	Resources       []ResourceAllocation     `json:"resources"`
	TotalCost       float64                  `json:"total_cost_per_hour"`
	AvgLatency      float64                  `json:"avg_latency_ms"`
	SLACompliance   float64                  `json:"sla_compliance"`
	Providers       []string                 `json:"providers"`
	Regions         []string                 `json:"regions"`
	EstimatedTime   time.Duration            `json:"estimated_provision_time"`
	RiskScore       float64                  `json:"risk_score"` // 0-1, lower is better
	OptimizationScore float64                `json:"optimization_score"`
}

// ResourceAllocation represents how resources are allocated
type ResourceAllocation struct {
	Resource   *ComputeResource `json:"resource"`
	Quantity   int              `json:"quantity"`
	Cost       float64          `json:"cost_per_hour"`
	Latency    float64          `json:"latency_ms"`
	SLA        float64          `json:"sla_percentage"`
}

// OCXQLOptimizer generates optimal execution plans for OCX-QL queries
type OCXQLOptimizer struct {
	resourceDB []*ComputeResource
	cache      map[string]*ExecutionPlan
}

// NewOCXQLOptimizer creates a new optimizer
func NewOCXQLOptimizer(resourceDB []*ComputeResource) *OCXQLOptimizer {
	return &OCXQLOptimizer{
		resourceDB: resourceDB,
		cache:      make(map[string]*ExecutionPlan),
	}
}

// Optimize generates optimal execution plans for a query
func (o *OCXQLOptimizer) Optimize(query *OCXQLQuery) (*OptimizationResult, error) {
	// Check cache first
	cacheKey := o.generateCacheKey(query)
	if cached, exists := o.cache[cacheKey]; exists {
		return &OptimizationResult{
			Query:           query,
			OptimalPlan:     cached,
			AlternativePlans: []*ExecutionPlan{cached},
			CacheHit:        true,
		}, nil
	}
	
	// Filter resources based on query constraints
	eligibleResources := o.filterResources(query)
	if len(eligibleResources) == 0 {
		return nil, fmt.Errorf("no resources match the query constraints")
	}
	
	// Generate multiple execution plans
	plans := o.generateExecutionPlans(query, eligibleResources)
	if len(plans) == 0 {
		return nil, fmt.Errorf("no feasible execution plans found")
	}
	
	// Score and rank plans
	scoredPlans := o.scorePlans(plans, query)
	
	// Select optimal plan
	optimalPlan := scoredPlans[0].Plan
	
	// Cache the result
	o.cache[cacheKey] = optimalPlan
	
	return &OptimizationResult{
		Query:           query,
		OptimalPlan:     optimalPlan,
		AlternativePlans: o.getTopPlans(scoredPlans, 3),
		CacheHit:        false,
	}, nil
}

// OptimizationResult contains the optimization results
type OptimizationResult struct {
	Query           *OCXQLQuery     `json:"query"`
	OptimalPlan     *ExecutionPlan  `json:"optimal_plan"`
	AlternativePlans []*ExecutionPlan `json:"alternative_plans"`
	CacheHit        bool            `json:"cache_hit"`
}

// ScoredPlan represents a plan with its score
type ScoredPlan struct {
	Plan  *ExecutionPlan
	Score float64
}

// filterResources filters resources based on query constraints
func (o *OCXQLOptimizer) filterResources(query *OCXQLQuery) []*ComputeResource {
	var eligible []*ComputeResource
	
	for _, resource := range o.resourceDB {
		// Check region constraint
		if query.Region != "" && resource.Region != query.Region {
			continue
		}
		
		// Check availability constraint
		if query.Availability > 0 && resource.Availability < query.Availability {
			continue
		}
		
		// Check price constraints
		if query.MaxPrice > 0 && resource.PricePerHour > query.MaxPrice {
			continue
		}
		if query.MinPrice > 0 && resource.PricePerHour < query.MinPrice {
			continue
		}
		
		// Check memory constraint
		if query.MinMemory > 0 && resource.MemoryGB < query.MinMemory {
			continue
		}
		
		// Check latency constraint
		if query.MaxLatency > 0 && resource.LatencyMS > float64(query.MaxLatency) {
			continue
		}
		
		// Check power efficiency constraint
		if query.PowerEfficiency > 0 && resource.PowerEfficiency < query.PowerEfficiency {
			continue
		}
		
		// Check interconnect constraint
		if query.Interconnect != "" && resource.Interconnect != query.Interconnect {
			continue
		}
		
		// Check resilience constraint
		if query.Resilience != "" && resource.Resilience != query.Resilience {
			continue
		}
		
		eligible = append(eligible, resource)
	}
	
	return eligible
}

// generateExecutionPlans generates multiple execution plans
func (o *OCXQLOptimizer) generateExecutionPlans(query *OCXQLQuery, resources []*ComputeResource) []*ExecutionPlan {
	var plans []*ExecutionPlan
	
	// Group resources by type
	resourceGroups := make(map[string][]*ComputeResource)
	for _, resource := range resources {
		resourceGroups[resource.Type] = append(resourceGroups[resource.Type], resource)
	}
	
	// Generate plans for each resource type requested
	for resourceType, quantity := range query.Resources {
		typeResources, exists := resourceGroups[resourceType]
		if !exists {
			continue
		}
		
		// Generate different optimization strategies
		plans = append(plans, o.generateCostOptimizedPlan(resourceType, quantity, typeResources)...)
		plans = append(plans, o.generatePerformanceOptimizedPlan(resourceType, quantity, typeResources)...)
		plans = append(plans, o.generateReliabilityOptimizedPlan(resourceType, quantity, typeResources)...)
	}
	
	return plans
}

// generateCostOptimizedPlan generates cost-optimized execution plans
func (o *OCXQLOptimizer) generateCostOptimizedPlan(resourceType string, quantity int, resources []*ComputeResource) []*ExecutionPlan {
	// Sort by price
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].PricePerHour < resources[j].PricePerHour
	})
	
	return o.generatePlanFromResources(resourceType, quantity, resources, "cost_optimized")
}

// generatePerformanceOptimizedPlan generates performance-optimized execution plans
func (o *OCXQLOptimizer) generatePerformanceOptimizedPlan(resourceType string, quantity int, resources []*ComputeResource) []*ExecutionPlan {
	// Sort by latency (lower is better)
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].LatencyMS < resources[j].LatencyMS
	})
	
	return o.generatePlanFromResources(resourceType, quantity, resources, "performance_optimized")
}

// generateReliabilityOptimizedPlan generates reliability-optimized execution plans
func (o *OCXQLOptimizer) generateReliabilityOptimizedPlan(resourceType string, quantity int, resources []*ComputeResource) []*ExecutionPlan {
	// Sort by availability (higher is better)
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Availability > resources[j].Availability
	})
	
	return o.generatePlanFromResources(resourceType, quantity, resources, "reliability_optimized")
}

// generatePlanFromResources generates execution plans from sorted resources
func (o *OCXQLOptimizer) generatePlanFromResources(resourceType string, quantity int, resources []*ComputeResource, planType string) []*ExecutionPlan {
	var plans []*ExecutionPlan
	
	// Try different allocation strategies
	strategies := []string{"single_provider", "multi_provider", "multi_region"}
	
	for _, strategy := range strategies {
		plan := o.allocateResources(resourceType, quantity, resources, planType, strategy)
		if plan != nil {
			plans = append(plans, plan)
		}
	}
	
	return plans
}

// allocateResources allocates resources using a specific strategy
func (o *OCXQLOptimizer) allocateResources(resourceType string, quantity int, resources []*ComputeResource, planType, strategy string) *ExecutionPlan {
	var allocations []ResourceAllocation
	remainingQuantity := quantity
	
	// Different allocation strategies
	switch strategy {
	case "single_provider":
		// Try to allocate all from a single provider
		for _, resource := range resources {
			if remainingQuantity <= 0 {
				break
			}
			
			allocated := minFloat(remainingQuantity, resource.MaxUnits)
			if allocated > 0 {
				allocations = append(allocations, ResourceAllocation{
					Resource: resource,
					Quantity: allocated,
					Cost:     resource.PricePerHour * float64(allocated),
					Latency:  resource.LatencyMS,
					SLA:      resource.Availability,
				})
				remainingQuantity -= allocated
			}
		}
		
	case "multi_provider":
		// Distribute across multiple providers for redundancy
		providers := make(map[string]bool)
		for _, resource := range resources {
			if remainingQuantity <= 0 {
				break
			}
			
			// Limit to 2-3 providers for cost efficiency
			if len(providers) >= 3 {
				break
			}
			
			allocated := minFloat(remainingQuantity, resource.MaxUnits)
			if allocated > 0 {
				allocations = append(allocations, ResourceAllocation{
					Resource: resource,
					Quantity: allocated,
					Cost:     resource.PricePerHour * float64(allocated),
					Latency:  resource.LatencyMS,
					SLA:      resource.Availability,
				})
				remainingQuantity -= allocated
				providers[resource.Provider] = true
			}
		}
		
	case "multi_region":
		// Distribute across multiple regions for global availability
		regions := make(map[string]bool)
		for _, resource := range resources {
			if remainingQuantity <= 0 {
				break
			}
			
			// Limit to 2-3 regions for cost efficiency
			if len(regions) >= 3 {
				break
			}
			
			allocated := minFloat(remainingQuantity, resource.MaxUnits)
			if allocated > 0 {
				allocations = append(allocations, ResourceAllocation{
					Resource: resource,
					Quantity: allocated,
					Cost:     resource.PricePerHour * float64(allocated),
					Latency:  resource.LatencyMS,
					SLA:      resource.Availability,
				})
				remainingQuantity -= allocated
				regions[resource.Region] = true
			}
		}
	}
	
	// Check if we can fulfill the request
	if remainingQuantity > 0 {
		return nil // Cannot fulfill request
	}
	
	// Calculate plan metrics
	totalCost := 0.0
	totalLatency := 0.0
	minSLA := 100.0
	providers := make(map[string]bool)
	regions := make(map[string]bool)
	
	for _, allocation := range allocations {
		totalCost += allocation.Cost
		totalLatency += allocation.Latency * float64(allocation.Quantity)
		if allocation.SLA < minSLA {
			minSLA = allocation.SLA
		}
		providers[allocation.Resource.Provider] = true
		regions[allocation.Resource.Region] = true
	}
	
	avgLatency := totalLatency / float64(quantity)
	
	// Calculate risk score based on provider diversity and SLA
	riskScore := o.calculateRiskScore(allocations)
	
	// Calculate optimization score
	optimizationScore := o.calculateOptimizationScore(planType, totalCost, avgLatency, minSLA, riskScore)
	
	// Generate plan ID
	planID := fmt.Sprintf("plan_%s_%s_%d", planType, strategy, time.Now().Unix())
	
	// Convert maps to slices
	providerList := make([]string, 0, len(providers))
	for provider := range providers {
		providerList = append(providerList, provider)
	}
	
	regionList := make([]string, 0, len(regions))
	for region := range regions {
		regionList = append(regionList, region)
	}
	
	return &ExecutionPlan{
		ID:               planID,
		Type:             planType,
		Resources:        allocations,
		TotalCost:        totalCost,
		AvgLatency:       avgLatency,
		SLACompliance:    minSLA,
		Providers:        providerList,
		Regions:          regionList,
		EstimatedTime:    time.Duration(len(providers)*30) * time.Second, // 30s per provider
		RiskScore:        riskScore,
		OptimizationScore: optimizationScore,
	}
}

// calculateRiskScore calculates the risk score for a plan
func (o *OCXQLOptimizer) calculateRiskScore(allocations []ResourceAllocation) float64 {
	if len(allocations) == 0 {
		return 1.0
	}
	
	// Risk factors: single provider, low SLA, high latency
	providerCount := len(allocations)
	minSLA := 100.0
	maxLatency := 0.0
	
	for _, allocation := range allocations {
		if allocation.SLA < minSLA {
			minSLA = allocation.SLA
		}
		if allocation.Latency > maxLatency {
			maxLatency = allocation.Latency
		}
	}
	
	// Calculate risk score (0-1, lower is better)
	providerRisk := math.Max(0, 1.0-float64(providerCount)/3.0) // Lower risk with more providers
	slaRisk := math.Max(0, (99.0-minSLA)/99.0) // Higher risk with lower SLA
	latencyRisk := math.Min(1.0, maxLatency/100.0) // Higher risk with higher latency
	
	return (providerRisk + slaRisk + latencyRisk) / 3.0
}

// calculateOptimizationScore calculates the optimization score for a plan
func (o *OCXQLOptimizer) calculateOptimizationScore(planType string, cost, latency, sla, risk float64) float64 {
	// Different weights based on optimization type
	var costWeight, latencyWeight, slaWeight, riskWeight float64
	
	switch planType {
	case "cost_optimized":
		costWeight, latencyWeight, slaWeight, riskWeight = 0.5, 0.2, 0.2, 0.1
	case "performance_optimized":
		costWeight, latencyWeight, slaWeight, riskWeight = 0.2, 0.5, 0.2, 0.1
	case "reliability_optimized":
		costWeight, latencyWeight, slaWeight, riskWeight = 0.2, 0.2, 0.5, 0.1
	default:
		costWeight, latencyWeight, slaWeight, riskWeight = 0.3, 0.3, 0.3, 0.1
	}
	
	// Normalize scores (lower cost/latency/risk is better, higher SLA is better)
	costScore := 1.0 / (1.0 + cost/100.0) // Normalize around $100/hr
	latencyScore := 1.0 / (1.0 + latency/10.0) // Normalize around 10ms
	slaScore := sla / 100.0 // Convert percentage to decimal
	riskScore := 1.0 - risk // Invert risk (lower risk = higher score)
	
	return costWeight*costScore + latencyWeight*latencyScore + slaWeight*slaScore + riskWeight*riskScore
}

// scorePlans scores and ranks execution plans
func (o *OCXQLOptimizer) scorePlans(plans []*ExecutionPlan, query *OCXQLQuery) []ScoredPlan {
	var scoredPlans []ScoredPlan
	
	for _, plan := range plans {
		score := plan.OptimizationScore
		scoredPlans = append(scoredPlans, ScoredPlan{
			Plan:  plan,
			Score: score,
		})
	}
	
	// Sort by score (higher is better)
	sort.Slice(scoredPlans, func(i, j int) bool {
		return scoredPlans[i].Score > scoredPlans[j].Score
	})
	
	return scoredPlans
}

// getTopPlans returns the top N plans
func (o *OCXQLOptimizer) getTopPlans(scoredPlans []ScoredPlan, n int) []*ExecutionPlan {
	if n > len(scoredPlans) {
		n = len(scoredPlans)
	}
	
	plans := make([]*ExecutionPlan, n)
	for i := 0; i < n; i++ {
		plans[i] = scoredPlans[i].Plan
	}
	
	return plans
}

// generateCacheKey generates a cache key for the query
func (o *OCXQLOptimizer) generateCacheKey(query *OCXQLQuery) string {
	// Simple cache key based on query parameters
	key := fmt.Sprintf("%v_%s_%.2f_%.2f_%d_%s",
		query.Resources,
		query.Region,
		query.Availability,
		query.MaxPrice,
		query.MinMemory,
		query.WorkloadType,
	)
	return key
}

// Helper function
func minFloat(a, b int) int {
	if a < b {
		return a
	}
	return b
}
