package connectors

import (
	"context"
	
	"math/rand"
	"time"

	
)

// ProviderConnector defines the interface for provider API integration
type ProviderConnector interface {
	GetProviderID() string
	GetPricing(ctx context.Context, resourceType, region string) (map[string]interface{}, error)
	GetAvailability(ctx context.Context, resourceType, region string) (map[string]interface{}, error)
	GetRateLimit() int
	CheckRateLimit() bool
}

// BaseConnector provides common functionality for all provider connectors
type BaseConnector struct {
	ProviderID       string
	Credentials      map[string]string
	RateLimit        int
	LastRequestTime  time.Time
	RequestCount     int
	RequestInterval  time.Duration
}

// NewBaseConnector creates a new base connector
func NewBaseConnector(providerID string, credentials map[string]string, rateLimit int) *BaseConnector {
	return &BaseConnector{
		ProviderID:      providerID,
		Credentials:     credentials,
		RateLimit:       rateLimit,
		RequestInterval: time.Minute,
	}
}

// GetProviderID returns the provider ID
func (b *BaseConnector) GetProviderID() string {
	return b.ProviderID
}

// GetRateLimit returns the rate limit
func (b *BaseConnector) GetRateLimit() int {
	return b.RateLimit
}

// CheckRateLimit checks if we can make a request without exceeding rate limits
func (b *BaseConnector) CheckRateLimit() bool {
	now := time.Now()
	
	// Reset counter if a minute has passed
	if now.Sub(b.LastRequestTime) >= b.RequestInterval {
		b.RequestCount = 0
		b.LastRequestTime = now
	}
	
	return b.RequestCount < b.RateLimit
}

// MakeRequest increments the request counter
func (b *BaseConnector) MakeRequest() {
	b.RequestCount++
}

// SimulateAPILatency simulates realistic API latency
func (b *BaseConnector) SimulateAPILatency() {
	// Simulate realistic API latency (50-200ms)
	latency := time.Duration(50+rand.Intn(150)) * time.Millisecond
	time.Sleep(latency)
}

// CalculateQualityScore calculates a quality score for the provider/resource combination
func (b *BaseConnector) CalculateQualityScore(resourceType string) float64 {
	// Base quality scores by provider (would be learned from historical data)
	baseScores := map[string]float64{
		"aws":      0.95,
		"gcp":      0.92,
		"azure":    0.90,
		"runpod":   0.75,
		"lambdalabs": 0.70,
		"coreweave": 0.85,
	}
	
	base := baseScores[b.ProviderID]
	if base == 0 {
		base = 0.8 // Default score
	}
	
	// Add some variation based on resource type and recent performance
	variation := (rand.Float64() - 0.5) * 0.1 // ±5% variation
	return maxFloat(0.5, minFloat(1.0, base+variation))
}

// Helper functions
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
