package opportunities

import (
	"fmt"
	"math"
	"sort"
	"time"

	"ocx.local/internal/marketintelligence"
	"ocx.local/internal/marketintelligence/collectors"
)

// OpportunityDetector detects arbitrage and optimization opportunities
type OpportunityDetector struct {
	marketCollector *collectors.MarketDataCollector
	opportunities   []marketintelligence.MarketOpportunity
}

// NewOpportunityDetector creates a new opportunity detector
func NewOpportunityDetector(marketCollector *collectors.MarketDataCollector) *OpportunityDetector {
	return &OpportunityDetector{
		marketCollector: marketCollector,
		opportunities:   make([]marketintelligence.MarketOpportunity, 0),
	}
}

// DetectOpportunities detects current market opportunities
func (o *OpportunityDetector) DetectOpportunities() []marketintelligence.MarketOpportunity {
	var opportunities []marketintelligence.MarketOpportunity
	
	// Detect arbitrage opportunities
	opportunities = append(opportunities, o.detectArbitrage()...)
	
	// Detect capacity arbitrage
	opportunities = append(opportunities, o.detectCapacityArbitrage()...)
	
	// Detect bulk purchase opportunities
	opportunities = append(opportunities, o.detectBulkOpportunities()...)
	
	o.opportunities = opportunities
	return opportunities
}

// detectArbitrage detects price arbitrage opportunities between providers
func (o *OpportunityDetector) detectArbitrage() []marketintelligence.MarketOpportunity {
	var opportunities []marketintelligence.MarketOpportunity
	
	resources := []string{"A100", "H100", "V100", "RTX4090"}
	regions := []string{"us-east-1", "us-west-2", "eu-west-1"}
	
	for _, resource := range resources {
		for _, region := range regions {
			prices := o.marketCollector.GetCurrentPrices(resource, region)
			
			if len(prices) < 2 {
				continue
			}
			
			// Sort by price
			sort.Slice(prices, func(i, j int) bool {
				return prices[i].PricePerHour < prices[j].PricePerHour
			})
			
			cheapest := prices[0]
			mostExpensive := prices[len(prices)-1]
			
			// Calculate arbitrage potential
			priceDiff := mostExpensive.PricePerHour - cheapest.PricePerHour
			profitMargin := priceDiff / cheapest.PricePerHour
			
			// Only flag significant arbitrage opportunities (>15% margin)
			if profitMargin > 0.15 {
				opportunity := marketintelligence.MarketOpportunity{
					OpportunityID:   fmt.Sprintf("arb_%s_%s_%d", resource, region, time.Now().Unix()),
					OpportunityType: "arbitrage",
					ProviderSource:  cheapest.ProviderID,
					ProviderTarget:  mostExpensive.ProviderID,
					ResourceType:    resource,
					Region:          region,
					ProfitPotential: profitMargin,
					ConfidenceScore: math.Min(cheapest.QualityScore, 0.9),
					ExpiresAt:       time.Now().Add(30 * time.Minute).Unix(),
					RequiredAction:  fmt.Sprintf("Buy from %s at $%.2f, sell at $%.2f", 
						cheapest.ProviderID, cheapest.PricePerHour, mostExpensive.PricePerHour),
				}
				opportunities = append(opportunities, opportunity)
			}
		}
	}
	
	return opportunities
}

// detectCapacityArbitrage detects opportunities to reserve low-demand capacity for high-demand periods
func (o *OpportunityDetector) detectCapacityArbitrage() []marketintelligence.MarketOpportunity {
	var opportunities []marketintelligence.MarketOpportunity
	
	// This would analyze historical demand patterns and predict future spikes
	// For now, we'll implement a simplified version based on current availability
	
	resources := []string{"A100", "H100", "V100"}
	regions := []string{"us-east-1", "us-west-2", "eu-west-1"}
	
	for _, resource := range resources {
		for _, region := range regions {
			prices := o.marketCollector.GetCurrentPrices(resource, region)
			
			if len(prices) == 0 {
				continue
			}
			
			// Find providers with high availability (low demand)
			var lowDemandProviders []marketintelligence.MarketDataPoint
			for _, price := range prices {
				if price.DemandIndicator < 0.3 && price.AvailableQuantity > 50 {
					lowDemandProviders = append(lowDemandProviders, price)
				}
			}
			
			if len(lowDemandProviders) > 0 {
				// Sort by price to get cheapest low-demand option
				sort.Slice(lowDemandProviders, func(i, j int) bool {
					return lowDemandProviders[i].PricePerHour < lowDemandProviders[j].PricePerHour
				})
				
				bestOption := lowDemandProviders[0]
				
				opportunity := marketintelligence.MarketOpportunity{
					OpportunityID:   fmt.Sprintf("cap_%s_%s_%d", resource, region, time.Now().Unix()),
					OpportunityType: "capacity_arbitrage",
					ProviderSource:  bestOption.ProviderID,
					ProviderTarget:  "future_high_demand",
					ResourceType:    resource,
					Region:          region,
					ProfitPotential: 0.2, // 20% potential profit
					ConfidenceScore: bestOption.QualityScore * 0.8,
					ExpiresAt:       time.Now().Add(2 * time.Hour).Unix(),
					RequiredAction:  fmt.Sprintf("Reserve %d units from %s at $%.2f for future resale", 
						bestOption.AvailableQuantity, bestOption.ProviderID, bestOption.PricePerHour),
				}
				opportunities = append(opportunities, opportunity)
			}
		}
	}
	
	return opportunities
}

// detectBulkOpportunities detects bulk purchase discount opportunities
func (o *OpportunityDetector) detectBulkOpportunities() []marketintelligence.MarketOpportunity {
	var opportunities []marketintelligence.MarketOpportunity
	
	// This would identify when providers offer bulk discounts
	// or when OCX can negotiate better rates for large volumes
	
	resources := []string{"A100", "H100", "V100"}
	regions := []string{"us-east-1", "us-west-2", "eu-west-1"}
	
	for _, resource := range resources {
		for _, region := range regions {
			prices := o.marketCollector.GetCurrentPrices(resource, region)
			
			if len(prices) == 0 {
				continue
			}
			
			// Look for providers with significant capacity that could benefit from bulk pricing
			for _, price := range prices {
				if price.AvailableQuantity > 100 {
					// Simulate bulk discount potential
					bulkDiscount := 0.1 + (float64(price.AvailableQuantity) / 1000.0) * 0.1 // 10-20% discount
					
					if bulkDiscount > 0.12 { // Only flag significant bulk opportunities
						opportunity := marketintelligence.MarketOpportunity{
							OpportunityID:   fmt.Sprintf("bulk_%s_%s_%d", resource, region, time.Now().Unix()),
							OpportunityType: "bulk_discount",
							ProviderSource:  price.ProviderID,
							ProviderTarget:  "bulk_negotiation",
							ResourceType:    resource,
							Region:          region,
							ProfitPotential: bulkDiscount,
							ConfidenceScore: price.QualityScore * 0.7,
							ExpiresAt:       time.Now().Add(24 * time.Hour).Unix(),
							RequiredAction:  fmt.Sprintf("Negotiate bulk pricing with %s for %d+ units (%.1f%% discount potential)", 
								price.ProviderID, price.AvailableQuantity, bulkDiscount*100),
						}
						opportunities = append(opportunities, opportunity)
					}
				}
			}
		}
	}
	
	return opportunities
}

// GetOpportunitiesByType returns opportunities filtered by type
func (o *OpportunityDetector) GetOpportunitiesByType(opportunityType string) []marketintelligence.MarketOpportunity {
	var filtered []marketintelligence.MarketOpportunity
	
	for _, opp := range o.opportunities {
		if opp.OpportunityType == opportunityType {
			filtered = append(filtered, opp)
		}
	}
	
	return filtered
}

// GetOpportunitiesByResource returns opportunities filtered by resource type
func (o *OpportunityDetector) GetOpportunitiesByResource(resourceType string) []marketintelligence.MarketOpportunity {
	var filtered []marketintelligence.MarketOpportunity
	
	for _, opp := range o.opportunities {
		if opp.ResourceType == resourceType {
			filtered = append(filtered, opp)
		}
	}
	
	return filtered
}

// GetTopOpportunities returns the top N opportunities by profit potential
func (o *OpportunityDetector) GetTopOpportunities(limit int) []marketintelligence.MarketOpportunity {
	// Sort by profit potential
	sort.Slice(o.opportunities, func(i, j int) bool {
		return o.opportunities[i].ProfitPotential > o.opportunities[j].ProfitPotential
	})
	
	if limit > len(o.opportunities) {
		limit = len(o.opportunities)
	}
	
	return o.opportunities[:limit]
}

// GetOpportunityStats returns statistics about detected opportunities
func (o *OpportunityDetector) GetOpportunityStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_opportunities": len(o.opportunities),
		"by_type":            make(map[string]int),
		"by_resource":        make(map[string]int),
		"avg_profit_potential": 0.0,
		"max_profit_potential": 0.0,
	}
	
	if len(o.opportunities) == 0 {
		return stats
	}
	
	// Count by type
	typeCount := make(map[string]int)
	resourceCount := make(map[string]int)
	totalProfit := 0.0
	maxProfit := 0.0
	
	for _, opp := range o.opportunities {
		typeCount[opp.OpportunityType]++
		resourceCount[opp.ResourceType]++
		totalProfit += opp.ProfitPotential
		
		if opp.ProfitPotential > maxProfit {
			maxProfit = opp.ProfitPotential
		}
	}
	
	stats["by_type"] = typeCount
	stats["by_resource"] = resourceCount
	stats["avg_profit_potential"] = totalProfit / float64(len(o.opportunities))
	stats["max_profit_potential"] = maxProfit
	
	return stats
}
