package connectors

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// GCPConnector implements Google Cloud Platform API integration
type GCPConnector struct {
	*BaseConnector
}

// NewGCPConnector creates a new GCP connector
func NewGCPConnector(credentials map[string]string) *GCPConnector {
	base := NewBaseConnector("gcp", credentials, 120) // 120 requests per minute
	
	return &GCPConnector{
		BaseConnector: base,
	}
}

// GetPricing gets current pricing for resource type in region
func (g *GCPConnector) GetPricing(ctx context.Context, resourceType, region string) (map[string]interface{}, error) {
	if !g.CheckRateLimit() {
		return nil, fmt.Errorf("rate limit exceeded for GCP")
	}
	
	g.MakeRequest()
	g.SimulateAPILatency()
	
	basePrices := map[string]float64{
		"A100":    3.0,
		"H100":    8.2,
		"V100":    2.0,
		"RTX4090": 0.75,
		"T4":      0.45,
	}
	
	basePrice := basePrices[resourceType]
	if basePrice == 0 {
		basePrice = 1.8 // Default price
	}
	
	// GCP regional pricing
	regionMultipliers := map[string]float64{
		"us-central1":    1.0,
		"us-west1":       1.03,
		"europe-west1":   1.08,
		"asia-southeast1": 1.18,
		"asia-northeast1": 1.12,
		"us-east1":       1.02,
	}
	
	multiplier := regionMultipliers[region]
	if multiplier == 0 {
		multiplier = 1.0
	}
	
	price := basePrice * multiplier
	
	// Preemptible pricing (GCP's equivalent to spot instances)
	preemptiblePrice := price * (0.2 + rand.Float64()*0.2) // 20-40% discount
	
	return map[string]interface{}{
		"provider":          "gcp",
		"resource_type":     resourceType,
		"region":            region,
		"on_demand_price":   price,
		"preemptible_price": preemptiblePrice,
		"currency":          "USD",
		"timestamp":         time.Now().Unix(),
	}, nil
}

// GetAvailability gets current availability for resource type in region
func (g *GCPConnector) GetAvailability(ctx context.Context, resourceType, region string) (map[string]interface{}, error) {
	if !g.CheckRateLimit() {
		return nil, fmt.Errorf("rate limit exceeded for GCP")
	}
	
	g.MakeRequest()
	g.SimulateAPILatency()
	
	baseCapacity := map[string]int{
		"A100":    300,
		"H100":    150,
		"V100":    800,
		"RTX4090": 1500,
		"T4":      4000,
	}
	
	maxCapacity := baseCapacity[resourceType]
	if maxCapacity == 0 {
		maxCapacity = 80
	}
	
	utilization := 0.5 + rand.Float64()*0.4 // 50-90% utilized
	available := int(float64(maxCapacity) * (1 - utilization))
	
	if available < 0 {
		available = 0
	}
	
	return map[string]interface{}{
		"provider":         "gcp",
		"resource_type":    resourceType,
		"region":           region,
		"available_quantity": available,
		"total_capacity":   maxCapacity,
		"utilization_rate": utilization,
	}, nil
}
