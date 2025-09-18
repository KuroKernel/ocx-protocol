package connectors

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// RunPodConnector implements RunPod API integration
type RunPodConnector struct {
	*BaseConnector
}

// NewRunPodConnector creates a new RunPod connector
func NewRunPodConnector(credentials map[string]string) *RunPodConnector {
	base := NewBaseConnector("runpod", credentials, 200) // 200 requests per minute (faster API)
	
	return &RunPodConnector{
		BaseConnector: base,
	}
}

// GetPricing gets current pricing for resource type in region
func (r *RunPodConnector) GetPricing(ctx context.Context, resourceType, region string) (map[string]interface{}, error) {
	if !r.CheckRateLimit() {
		return nil, fmt.Errorf("rate limit exceeded for RunPod")
	}
	
	r.MakeRequest()
	r.SimulateAPILatency()
	
	// RunPod typically 40-60% cheaper than major clouds
	basePrices := map[string]float64{
		"A100":    1.8,
		"H100":    4.9,
		"V100":    1.2,
		"RTX4090": 0.4,
		"RTX3090": 0.3,
		"T4":      0.25,
	}
	
	basePrice := basePrices[resourceType]
	if basePrice == 0 {
		basePrice = 1.0 // Default price
	}
	
	// Less regional variation for RunPod
	price := basePrice * (0.95 + rand.Float64()*0.1) // ±5% variation
	
	return map[string]interface{}{
		"provider":      "runpod",
		"resource_type": resourceType,
		"region":        region,
		"on_demand_price": price,
		"currency":      "USD",
		"timestamp":     time.Now().Unix(),
	}, nil
}

// GetAvailability gets current availability for resource type in region
func (r *RunPodConnector) GetAvailability(ctx context.Context, resourceType, region string) (map[string]interface{}, error) {
	if !r.CheckRateLimit() {
		return nil, fmt.Errorf("rate limit exceeded for RunPod")
	}
	
	r.MakeRequest()
	r.SimulateAPILatency()
	
	// Smaller but more volatile availability
	baseCapacity := map[string]int{
		"A100":    50,
		"H100":    20,
		"V100":    100,
		"RTX4090": 200,
		"RTX3090": 500,
		"T4":      1000,
	}
	
	maxCapacity := baseCapacity[resourceType]
	if maxCapacity == 0 {
		maxCapacity = 10
	}
	
	utilization := 0.3 + rand.Float64()*0.65 // 30-95% utilized (more volatile)
	available := int(float64(maxCapacity) * (1 - utilization))
	
	if available < 0 {
		available = 0
	}
	
	return map[string]interface{}{
		"provider":         "runpod",
		"resource_type":    resourceType,
		"region":           region,
		"available_quantity": available,
		"total_capacity":   maxCapacity,
		"utilization_rate": utilization,
	}, nil
}
