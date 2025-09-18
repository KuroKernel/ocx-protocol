package capacity

import (
)

// ReservationType represents the type of capacity reservation
type ReservationType string

const (
	Spot      ReservationType = "spot"       // Cheapest, can be interrupted
	OnDemand  ReservationType = "on_demand"  // Standard pricing
	Reserved  ReservationType = "reserved"   // Committed usage discount
	Futures   ReservationType = "futures"    // OCX bulk purchase for resale
)

// CapacityReservation represents a capacity reservation
type CapacityReservation struct {
	ReservationID     string          `json:"reservation_id"`
	ProviderID        string          `json:"provider_id"`
	ResourceType      string          `json:"resource_type"`
	Region            string          `json:"region"`
	Quantity          int             `json:"quantity"`
	StartTime         int64           `json:"start_time"`
	EndTime           int64           `json:"end_time"`
	ReservationType   ReservationType `json:"reservation_type"`
	PurchasePrice     float64         `json:"purchase_price"`
	CurrentMarketPrice float64        `json:"current_market_price"`
	UtilizationRate   float64         `json:"utilization_rate"`
	LockedByCustomer  bool            `json:"locked_by_customer"`
	CustomerPrice     *float64        `json:"customer_price,omitempty"`
}

// ProfitMargin calculates the profit margin for the reservation
func (c *CapacityReservation) ProfitMargin() float64 {
	if c.CustomerPrice != nil {
		return (*c.CustomerPrice - c.PurchasePrice) / c.PurchasePrice
	}
	return (c.CurrentMarketPrice - c.PurchasePrice) / c.PurchasePrice
}

// DurationHours returns the duration of the reservation in hours
func (c *CapacityReservation) DurationHours() float64 {
	return float64(c.EndTime-c.StartTime) / 3600
}

// ReservationOpportunity represents a potential reservation opportunity
type ReservationOpportunity struct {
	ProviderID        string  `json:"provider_id"`
	ResourceType      string  `json:"resource_type"`
	Region            string  `json:"region"`
	CurrentPrice      float64 `json:"current_price"`
	PredictedPrice    float64 `json:"predicted_price"`
	AvailableQuantity int     `json:"available_quantity"`
	ProfitPotential   float64 `json:"profit_potential"`
	Confidence        float64 `json:"confidence"`
	RecommendedDuration int   `json:"recommended_duration"`
	RiskScore         float64 `json:"risk_score"`
}

// ReservationAnalytics represents analytics for reservations
type ReservationAnalytics struct {
	TotalReservations    int                    `json:"total_reservations"`
	ActiveReservations   int                    `json:"active_reservations"`
	SuccessRate          float64                `json:"success_rate"`
	AvgProfitMargin      float64                `json:"avg_profit_margin"`
	TotalProfitPercent   float64                `json:"total_profit_percent"`
	ResourcePerformance  map[string]ResourceStats `json:"resource_performance"`
	TopProviders         []ProviderStats        `json:"top_providers"`
	VolumeTrend          VolumeTrend            `json:"volume_trend"`
}

// ResourceStats represents performance statistics for a resource type
type ResourceStats struct {
	AvgProfitMargin float64 `json:"avg_profit_margin"`
	MaxProfit       float64 `json:"max_profit"`
	MinProfit       float64 `json:"min_profit"`
	ReservationCount int    `json:"reservation_count"`
}

// ProviderStats represents performance statistics for a provider
type ProviderStats struct {
	ProviderID string  `json:"provider_id"`
	AvgProfit  float64 `json:"avg_profit"`
}

// VolumeTrend represents volume trend analysis
type VolumeTrend struct {
	Trend              string  `json:"trend"`
	RecentWeeklyAvg    float64 `json:"recent_weekly_avg,omitempty"`
	GrowthRate         float64 `json:"growth_rate,omitempty"`
}

// CustomerRequest represents a customer request for capacity allocation
type CustomerRequest struct {
	ResourceType     string                 `json:"resource_type"`
	Region           string                 `json:"region"`
	Quantity         int                    `json:"quantity"`
	DurationHours    float64                `json:"duration_hours"`
	MaxPricePerHour  float64                `json:"max_price_per_hour"`
	SLARequirements  map[string]interface{} `json:"sla_requirements"`
}
