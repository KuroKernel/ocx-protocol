package engines

import (
	"fmt"
	
	"sort"
	"time"

	"ocx.local/internal/marketintelligence"
	"ocx.local/internal/marketintelligence/collectors"
	"ocx.local/internal/marketintelligence/opportunities"
)

// PricingEngine provides advanced pricing recommendations with predictive capabilities
type PricingEngine struct {
	marketCollector *collectors.MarketDataCollector
	opportunityDetector *opportunities.OpportunityDetector
}

// NewPricingEngine creates a new pricing engine
func NewPricingEngine(marketCollector *collectors.MarketDataCollector) *PricingEngine {
	return &PricingEngine{
		marketCollector: marketCollector,
		opportunityDetector: opportunities.NewOpportunityDetector(marketCollector),
	}
}

// GetOptimalPricing gets optimal pricing recommendation for customer request
func (p *PricingEngine) GetOptimalPricing(request *marketintelligence.PricingRequest) (*marketintelligence.PricingResponse, error) {
	// Get current market data
	currentPrices := p.marketCollector.GetCurrentPrices(request.ResourceType, request.Region)
	
	if len(currentPrices) == 0 {
		return &marketintelligence.PricingResponse{
			ResourceType:     request.ResourceType,
			Region:           request.Region,
			Quantity:         request.Quantity,
			DurationHours:    request.DurationHours,
			Recommendations:  []marketintelligence.PricingRecommendation{},
			MarketConditions: marketintelligence.MarketConditions{
				Condition:         "no_data",
				AveragePrice:      0,
				TotalAvailability: 0,
				DemandLevel:       0,
				Recommendation:    "No market data available",
				ProviderCount:     0,
			},
			Timestamp: time.Now().Unix(),
		}, nil
	}
	
	// Sort by price
	sort.Slice(currentPrices, func(i, j int) bool {
		return currentPrices[i].PricePerHour < currentPrices[j].PricePerHour
	})
	
	// Find providers that can satisfy quantity requirement
	suitableProviders := make([]marketintelligence.MarketDataPoint, 0)
	for _, provider := range currentPrices {
		if provider.AvailableQuantity >= request.Quantity {
			suitableProviders = append(suitableProviders, provider)
		}
	}
	
	var recommendations []marketintelligence.PricingRecommendation
	
	if len(suitableProviders) == 0 {
		// Need to split across providers
		splitRec, err := p.getSplitAllocationPricing(currentPrices, request)
		if err != nil {
			return nil, fmt.Errorf("split allocation failed: %w", err)
		}
		recommendations = append(recommendations, *splitRec)
	} else {
		// Calculate pricing with different strategies
		recommendations = p.calculatePricingStrategies(suitableProviders, request)
	}
	
	// Add market intelligence insights
	for i := range recommendations {
		rec := &recommendations[i]
		providerID := rec.Provider
		
		// Add price trend analysis
		trend := p.marketCollector.GetPriceTrend(providerID, request.ResourceType, request.Region, 20)
		rec.PriceTrend = trend
		
		// Add price forecast
		rec.PriceForecast = p.forecastPriceStability(providerID, request.ResourceType, request.Region, request.DurationHours)
	}
	
	// Assess market conditions
	marketConditions := p.assessMarketConditions(currentPrices)
	
	return &marketintelligence.PricingResponse{
		ResourceType:     request.ResourceType,
		Region:           request.Region,
		Quantity:         request.Quantity,
		DurationHours:    request.DurationHours,
		Recommendations:  recommendations,
		MarketConditions: marketConditions,
		Timestamp:        time.Now().Unix(),
	}, nil
}

// calculatePricingStrategies calculates different pricing strategies
func (p *PricingEngine) calculatePricingStrategies(providers []marketintelligence.MarketDataPoint, 
	request *marketintelligence.PricingRequest) []marketintelligence.PricingRecommendation {
	
	var recommendations []marketintelligence.PricingRecommendation
	
	// Strategy 1: Cheapest single provider
	cheapest := providers[0]
	recommendations = append(recommendations, marketintelligence.PricingRecommendation{
		Strategy:        "cheapest_single",
		Provider:        cheapest.ProviderID,
		PricePerHour:    cheapest.PricePerHour,
		TotalCost:       cheapest.PricePerHour * float64(request.Quantity) * request.DurationHours,
		QualityScore:    cheapest.QualityScore,
		RiskLevel:       "medium",
		SetupComplexity: "low",
	})
	
	// Strategy 2: Best quality within budget
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].QualityScore > providers[j].QualityScore
	})
	bestQuality := providers[0]
	
	if bestQuality.ProviderID != cheapest.ProviderID {
		recommendations = append(recommendations, marketintelligence.PricingRecommendation{
			Strategy:        "best_quality",
			Provider:        bestQuality.ProviderID,
			PricePerHour:    bestQuality.PricePerHour,
			TotalCost:       bestQuality.PricePerHour * float64(request.Quantity) * request.DurationHours,
			QualityScore:    bestQuality.QualityScore,
			RiskLevel:       "low",
			SetupComplexity: "low",
		})
	}
	
	// Strategy 3: Multi-provider for risk distribution
	if len(providers) >= 2 && request.Quantity >= 10 {
		multiProvider := p.calculateMultiProviderPricing(providers, request)
		recommendations = append(recommendations, multiProvider)
	}
	
	return recommendations
}

// getSplitAllocationPricing calculates pricing when splitting across multiple providers
func (p *PricingEngine) getSplitAllocationPricing(availablePrices []marketintelligence.MarketDataPoint,
	request *marketintelligence.PricingRequest) (*marketintelligence.PricingRecommendation, error) {
	
	// Greedy allocation: start with cheapest and fill capacity
	var allocation []marketintelligence.ResourceAllocation
	remainingQuantity := request.Quantity
	
	for _, provider := range availablePrices {
		if remainingQuantity <= 0 {
			break
		}
		
		allocatedQuantity := minFloat(remainingQuantity, provider.AvailableQuantity)
		if allocatedQuantity > 0 {
			allocation = append(allocation, marketintelligence.ResourceAllocation{
				Provider:     provider.ProviderID,
				Quantity:     allocatedQuantity,
				PricePerHour: provider.PricePerHour,
				QualityScore: provider.QualityScore,
			})
			remainingQuantity -= allocatedQuantity
		}
	}
	
	if remainingQuantity > 0 {
		return nil, fmt.Errorf("insufficient capacity: %d units unallocated", remainingQuantity)
	}
	
	// Calculate total cost and weighted quality score
	totalCost := 0.0
	for _, alloc := range allocation {
		totalCost += float64(alloc.Quantity) * alloc.PricePerHour * request.DurationHours
	}
	
	weightedQuality := 0.0
	for _, alloc := range allocation {
		weightedQuality += float64(alloc.Quantity) * alloc.QualityScore
	}
	weightedQuality /= float64(request.Quantity)
	
	return &marketintelligence.PricingRecommendation{
		Strategy:        "split_allocation",
		Allocation:      allocation,
		TotalCost:       totalCost,
		PricePerHour:    totalCost / (float64(request.Quantity) * request.DurationHours),
		QualityScore:    weightedQuality,
		RiskLevel:       "low", // Diversified risk
		SetupComplexity: "high",
	}, nil
}

// calculateMultiProviderPricing calculates optimal multi-provider allocation
func (p *PricingEngine) calculateMultiProviderPricing(providers []marketintelligence.MarketDataPoint,
	request *marketintelligence.PricingRequest) marketintelligence.PricingRecommendation {
	
	// Split quantity across top 2-3 providers for risk distribution
	numProviders := minFloat(3, len(providers))
	selectedProviders := providers[:numProviders]
	
	// Allocate quantities (weighted by inverse price for cost efficiency)
	totalWeight := 0.0
	for _, provider := range selectedProviders {
		totalWeight += 1.0 / provider.PricePerHour
	}
	
	var allocation []marketintelligence.ResourceAllocation
	for _, provider := range selectedProviders {
		weight := (1.0 / provider.PricePerHour) / totalWeight
		allocatedQuantity := maxFloat(1, int(float64(request.Quantity)*weight))
		
		allocation = append(allocation, marketintelligence.ResourceAllocation{
			Provider:     provider.ProviderID,
			Quantity:     allocatedQuantity,
			PricePerHour: provider.PricePerHour,
			QualityScore: provider.QualityScore,
		})
	}
	
	// Adjust for exact quantity match
	totalAllocated := 0
	for _, alloc := range allocation {
		totalAllocated += alloc.Quantity
	}
	diff := request.Quantity - totalAllocated
	
	if diff != 0 {
		// Add/remove from cheapest provider
		allocation[0].Quantity += diff
	}
	
	// Calculate metrics
	totalCost := 0.0
	for _, alloc := range allocation {
		totalCost += float64(alloc.Quantity) * alloc.PricePerHour * request.DurationHours
	}
	
	weightedQuality := 0.0
	for _, alloc := range allocation {
		weightedQuality += float64(alloc.Quantity) * alloc.QualityScore
	}
	weightedQuality /= float64(request.Quantity)
	
	return marketintelligence.PricingRecommendation{
		Strategy:        "multi_provider",
		Allocation:      allocation,
		TotalCost:       totalCost,
		PricePerHour:    totalCost / (float64(request.Quantity) * request.DurationHours),
		QualityScore:    weightedQuality,
		RiskLevel:       "very_low",
		SetupComplexity: "medium",
	}
}

// forecastPriceStability forecasts price stability during workload duration
func (p *PricingEngine) forecastPriceStability(providerID, resourceType, region string, 
	durationHours float64) *marketintelligence.PriceForecast {
	
	trend := p.marketCollector.GetPriceTrend(providerID, resourceType, region, 20)
	
	// Simple forecasting based on recent trends and volatility
	volatility := trend.Volatility
	trendDirection := trend.TrendDirection
	
	var stabilityRating string
	var riskFactor float64
	
	if volatility > 0.15 { // High volatility
		stabilityRating = "unstable"
		riskFactor = 1.2
	} else if volatility > 0.08 { // Medium volatility
		stabilityRating = "moderate"
		riskFactor = 1.1
	} else {
		stabilityRating = "stable"
		riskFactor = 1.0
	}
	
	// Predict price change probability
	var priceIncreaseProbability float64
	if trendDirection == "rising" && volatility > 0.1 {
		priceIncreaseProbability = 0.7
	} else if trendDirection == "falling" {
		priceIncreaseProbability = 0.2
	} else {
		priceIncreaseProbability = 0.4
	}
	
	return &marketintelligence.PriceForecast{
		StabilityRating:          stabilityRating,
		PriceIncreaseProbability: priceIncreaseProbability,
		RiskFactor:               riskFactor,
		RecommendedHedge:         volatility > 0.12,
	}
}

// assessMarketConditions assesses overall market conditions
func (p *PricingEngine) assessMarketConditions(currentPrices []marketintelligence.MarketDataPoint) marketintelligence.MarketConditions {
	if len(currentPrices) == 0 {
		return marketintelligence.MarketConditions{
			Condition:         "no_data",
			AveragePrice:      0,
			TotalAvailability: 0,
			DemandLevel:       0,
			Recommendation:    "No market data available",
			ProviderCount:     0,
		}
	}
	
	// Calculate market metrics
	prices := make([]float64, len(currentPrices))
	availabilities := make([]int, len(currentPrices))
	demandIndicators := make([]float64, len(currentPrices))
	
	for i, price := range currentPrices {
		prices[i] = price.PricePerHour
		availabilities[i] = price.AvailableQuantity
		demandIndicators[i] = price.DemandIndicator
	}
	
	avgPrice := calculateAverage(prices)
	totalAvailability := sum(availabilities)
	avgDemand := calculateAverage(demandIndicators)
	
	// Determine market conditions
	var marketCondition string
	var recommendation string
	
	if avgDemand > 0.8 {
		marketCondition = "high_demand"
		recommendation = "Reserve capacity quickly, prices may rise"
	} else if avgDemand < 0.3 {
		marketCondition = "low_demand"
		recommendation = "Good time for cost optimization"
	} else {
		marketCondition = "normal"
		recommendation = "Standard market conditions"
	}
	
	return marketintelligence.MarketConditions{
		Condition:         marketCondition,
		AveragePrice:      avgPrice,
		TotalAvailability: totalAvailability,
		DemandLevel:       avgDemand,
		Recommendation:    recommendation,
		ProviderCount:     len(currentPrices),
	}
}

// Helper functions
func minFloat(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func sum(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}
