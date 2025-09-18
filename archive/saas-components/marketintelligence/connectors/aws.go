package connectors

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// AWSConnector implements AWS EC2 API integration
type AWSConnector struct {
	*BaseConnector
	InstanceTypes map[string][]string
}

// NewAWSConnector creates a new AWS connector
func NewAWSConnector(credentials map[string]string) *AWSConnector {
	base := NewBaseConnector("aws", credentials, 100) // 100 requests per minute
	
	connector := &AWSConnector{
		BaseConnector: base,
		InstanceTypes: map[string][]string{
			"A100": {"p4d.24xlarge", "p4de.24xlarge"},
			"H100": {"p5.48xlarge"},
			"V100": {"p3.16xlarge", "p3dn.24xlarge"},
			"RTX4090": {"g5.12xlarge"},
			"T4": {"g4dn.xlarge", "g4dn.2xlarge"},
		},
	}
	
	return connector
}

// GetPricing gets current pricing for resource type in region
func (a *AWSConnector) GetPricing(ctx context.Context, resourceType, region string) (map[string]interface{}, error) {
	if !a.CheckRateLimit() {
		return nil, fmt.Errorf("rate limit exceeded for AWS")
	}
	
	a.MakeRequest()
	a.SimulateAPILatency()
	
	// Mock realistic AWS pricing with variation
	basePrices := map[string]float64{
		"A100":    3.2,
		"H100":    8.5,
		"V100":    2.1,
		"RTX4090": 0.8,
		"T4":      0.5,
	}
	
	basePrice := basePrices[resourceType]
	if basePrice == 0 {
		basePrice = 2.0 // Default price
	}
	
	// Add regional multiplier
	regionMultipliers := map[string]float64{
		"us-east-1":      1.0,
		"us-west-2":      1.05,
		"eu-west-1":      1.1,
		"ap-southeast-1": 1.2,
		"ap-northeast-1": 1.15,
		"ca-central-1":   1.08,
	}
	
	multiplier := regionMultipliers[region]
	if multiplier == 0 {
		multiplier = 1.0
	}
	
	onDemandPrice := basePrice * multiplier
	
	// Spot pricing with volatility
	spotDiscount := 0.3 + rand.Float64()*0.5 // 30-80% discount
	spotPrice := onDemandPrice * spotDiscount
	
	return map[string]interface{}{
		"provider":        "aws",
		"resource_type":   resourceType,
		"region":          region,
		"on_demand_price": onDemandPrice,
		"spot_price":      spotPrice,
		"currency":        "USD",
		"timestamp":       time.Now().Unix(),
	}, nil
}

// GetAvailability gets current availability for resource type in region
func (a *AWSConnector) GetAvailability(ctx context.Context, resourceType, region string) (map[string]interface{}, error) {
	if !a.CheckRateLimit() {
		return nil, fmt.Errorf("rate limit exceeded for AWS")
	}
	
	a.MakeRequest()
	a.SimulateAPILatency()
	
	// Realistic availability simulation
	baseCapacity := map[string]int{
		"A100":    500,
		"H100":    200,
		"V100":    1000,
		"RTX4090": 2000,
		"T4":      5000,
	}
	
	maxCapacity := baseCapacity[resourceType]
	if maxCapacity == 0 {
		maxCapacity = 100
	}
	
	// Simulate demand-based availability
	utilization := 0.6 + rand.Float64()*0.35 // 60-95% utilized
	available := int(float64(maxCapacity) * (1 - utilization))
	
	// Ensure we don't have negative availability
	if available < 0 {
		available = 0
	}
	
	estimatedWaitTime := 0.0
	if available < 10 {
		// Simulate wait time for low availability
		estimatedWaitTime = maxFloat(0, rand.NormFloat64()*2+5) // Normal distribution around 5 minutes
	}
	
	return map[string]interface{}{
		"provider":            "aws",
		"resource_type":       resourceType,
		"region":              region,
		"available_quantity":  available,
		"total_capacity":      maxCapacity,
		"utilization_rate":    utilization,
		"estimated_wait_time": estimatedWaitTime,
	}, nil
}
