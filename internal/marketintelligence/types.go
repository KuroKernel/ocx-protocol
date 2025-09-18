package marketintelligence

import (
	
)

// MarketEvent represents different types of market events
type MarketEvent string

const (
	PriceChange        MarketEvent = "price_change"
	AvailabilityChange MarketEvent = "availability_change"
	NewProvider        MarketEvent = "new_provider"
	ProviderOutage     MarketEvent = "provider_outage"
	DemandSpike        MarketEvent = "demand_spike"
	CapacityExpansion  MarketEvent = "capacity_expansion"
)

// ResourceClass represents different types of compute resources
type ResourceClass string

const (
	TrainingGPU   ResourceClass = "training_gpu"   // A100, H100 for ML training
	InferenceGPU  ResourceClass = "inference_gpu"  // T4, RTX4090 for inference
	ComputeCPU    ResourceClass = "compute_cpu"    // High-core count CPUs
	MemoryHeavy   ResourceClass = "memory_heavy"   // High memory instances
	StorageHeavy  ResourceClass = "storage_heavy"  // High IOPS instances
)

// MarketDataPoint represents a single market observation
type MarketDataPoint struct {
	Timestamp        int64   `json:"timestamp"`
	ProviderID       string  `json:"provider_id"`
	ResourceType     string  `json:"resource_type"`
	Region           string  `json:"region"`
	PricePerHour     float64 `json:"price_per_hour"`
	AvailableQuantity int    `json:"available_quantity"`
	DemandIndicator  float64 `json:"demand_indicator"`  // 0-1, how much demand we're seeing
	QualityScore     float64 `json:"quality_score"`     // 0-1, reliability/performance score
}

// ToDict converts MarketDataPoint to map for JSON serialization
func (m *MarketDataPoint) ToDict() map[string]interface{} {
	return map[string]interface{}{
		"timestamp":     m.Timestamp,
		"provider":      m.ProviderID,
		"resource":      m.ResourceType,
		"region":        m.Region,
		"price":         m.PricePerHour,
		"available":     m.AvailableQuantity,
		"demand":        m.DemandIndicator,
		"quality":       m.QualityScore,
	}
}

// PriceEvent represents a significant pricing event in the market
type PriceEvent struct {
	EventType         MarketEvent `json:"event_type"`
	Timestamp         int64       `json:"timestamp"`
	ProviderID        string      `json:"provider_id"`
	ResourceType      string      `json:"resource_type"`
	Region            string      `json:"region"`
	OldPrice          *float64    `json:"old_price,omitempty"`
	NewPrice          float64     `json:"new_price"`
	PriceChangePercent float64    `json:"price_change_percent"`
	ImpactScore       float64     `json:"impact_score"` // How significant this event is (0-1)
}

// MarketOpportunity represents an arbitrage or optimization opportunity
type MarketOpportunity struct {
	OpportunityID     string  `json:"opportunity_id"`
	OpportunityType   string  `json:"opportunity_type"` // 'arbitrage', 'bulk_discount', 'demand_shift'
	ProviderSource    string  `json:"provider_source"`
	ProviderTarget    string  `json:"provider_target"`
	ResourceType      string  `json:"resource_type"`
	Region            string  `json:"region"`
	ProfitPotential   float64 `json:"profit_potential"`
	ConfidenceScore   float64 `json:"confidence_score"`
	ExpiresAt         int64   `json:"expires_at"`
	RequiredAction    string  `json:"required_action"`
}

// PricingRecommendation represents a pricing strategy recommendation
type PricingRecommendation struct {
	Strategy          string                 `json:"strategy"`
	Provider          string                 `json:"provider,omitempty"`
	PricePerHour      float64                `json:"price_per_hour"`
	TotalCost         float64                `json:"total_cost"`
	QualityScore      float64                `json:"quality_score"`
	RiskLevel         string                 `json:"risk_level"`
	SetupComplexity   string                 `json:"setup_complexity"`
	Allocation        []ResourceAllocation   `json:"allocation,omitempty"`
	PriceTrend        *PriceTrend            `json:"price_trend,omitempty"`
	PriceForecast     *PriceForecast         `json:"price_forecast,omitempty"`
}

// ResourceAllocation represents allocation of resources across providers
type ResourceAllocation struct {
	Provider      string  `json:"provider"`
	Quantity      int     `json:"quantity"`
	PricePerHour  float64 `json:"price_per_hour"`
	QualityScore  float64 `json:"quality_score"`
}

// PriceTrend represents price trend analysis
type PriceTrend struct {
	CurrentPrice        float64 `json:"current_price"`
	TrendDirection      string  `json:"trend_direction"`
	PriceChangePercent  float64 `json:"price_change_percent"`
	Volatility          float64 `json:"volatility"`
	MinPrice            float64 `json:"min_price"`
	MaxPrice            float64 `json:"max_price"`
	AvgPrice            float64 `json:"avg_price"`
}

// PriceForecast represents price stability forecast
type PriceForecast struct {
	StabilityRating           string  `json:"stability_rating"`
	PriceIncreaseProbability  float64 `json:"price_increase_probability"`
	RiskFactor                float64 `json:"risk_factor"`
	RecommendedHedge          bool    `json:"recommended_hedge"`
}

// MarketConditions represents overall market conditions
type MarketConditions struct {
	Condition         string  `json:"condition"`
	AveragePrice      float64 `json:"average_price"`
	TotalAvailability int     `json:"total_availability"`
	DemandLevel       float64 `json:"demand_level"`
	Recommendation    string  `json:"recommendation"`
	ProviderCount      int     `json:"provider_count"`
}

// PricingRequest represents a request for pricing recommendations
type PricingRequest struct {
	ResourceType     string                 `json:"resource_type"`
	Region           string                 `json:"region"`
	Quantity         int                    `json:"quantity"`
	DurationHours    float64                `json:"duration_hours"`
	SLARequirements  map[string]interface{} `json:"sla_requirements"`
}

// PricingResponse represents the response to a pricing request
type PricingResponse struct {
	ResourceType      string                   `json:"resource_type"`
	Region            string                   `json:"region"`
	Quantity          int                      `json:"quantity"`
	DurationHours     float64                  `json:"duration_hours"`
	Recommendations   []PricingRecommendation  `json:"recommendations"`
	MarketConditions  MarketConditions         `json:"market_conditions"`
	Timestamp         int64                    `json:"timestamp"`
}

// ProviderCredentials represents API credentials for a provider
type ProviderCredentials struct {
	ProviderID string            `json:"provider_id"`
	Credentials map[string]string `json:"credentials"`
}

// MarketDataBuffer represents a circular buffer for market data
type MarketDataBuffer struct {
	Data      []MarketDataPoint `json:"data"`
	MaxSize   int               `json:"max_size"`
	CurrentIndex int            `json:"current_index"`
}

// NewMarketDataBuffer creates a new market data buffer
func NewMarketDataBuffer(maxSize int) *MarketDataBuffer {
	return &MarketDataBuffer{
		Data:      make([]MarketDataPoint, 0, maxSize),
		MaxSize:   maxSize,
		CurrentIndex: 0,
	}
}

// Add adds a new data point to the buffer
func (b *MarketDataBuffer) Add(dataPoint MarketDataPoint) {
	if len(b.Data) < b.MaxSize {
		b.Data = append(b.Data, dataPoint)
	} else {
		b.Data[b.CurrentIndex] = dataPoint
		b.CurrentIndex = (b.CurrentIndex + 1) % b.MaxSize
	}
}

// GetLatest returns the most recent data point
func (b *MarketDataBuffer) GetLatest() *MarketDataPoint {
	if len(b.Data) == 0 {
		return nil
	}
	
	if len(b.Data) < b.MaxSize {
		return &b.Data[len(b.Data)-1]
	}
	
	prevIndex := (b.CurrentIndex - 1 + b.MaxSize) % b.MaxSize
	return &b.Data[prevIndex]
}

// GetAll returns all data points in chronological order
func (b *MarketDataBuffer) GetAll() []MarketDataPoint {
	if len(b.Data) < b.MaxSize {
		return b.Data
	}
	
	result := make([]MarketDataPoint, b.MaxSize)
	for i := 0; i < b.MaxSize; i++ {
		index := (b.CurrentIndex + i) % b.MaxSize
		result[i] = b.Data[index]
	}
	return result
}

// GetRecent returns the most recent N data points
func (b *MarketDataBuffer) GetRecent(count int) []MarketDataPoint {
	all := b.GetAll()
	if count >= len(all) {
		return all
	}
	return all[len(all)-count:]
}
