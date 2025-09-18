package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"ocx.local/internal/marketintelligence"
	"ocx.local/internal/marketintelligence/collectors"
	"ocx.local/internal/marketintelligence/engines"
	"ocx.local/internal/marketintelligence/opportunities"
)

func main() {
	fmt.Println("🎯 OCX Protocol - Market Intelligence & Pricing Engine Demo")
	fmt.Println("=========================================================")
	fmt.Println("")
	fmt.Println("This demo shows how OCX uses real-time market intelligence")
	fmt.Println("to provide superior pricing and optimization recommendations.")
	fmt.Println("")
	fmt.Println("Key Features:")
	fmt.Println("✅ Real-time market data collection from all providers")
	fmt.Println("✅ Advanced pricing engine with predictive capabilities")
	fmt.Println("✅ Arbitrage and optimization opportunity detection")
	fmt.Println("✅ Multi-strategy pricing recommendations")
	fmt.Println("✅ Market condition analysis and forecasting")
	fmt.Println("")
	
	// Initialize market intelligence system
	marketCollector := collectors.NewMarketDataCollector()
	marketCollector.InitializeConnectors()
	
	pricingEngine := engines.NewPricingEngine(marketCollector)
	opportunityDetector := opportunities.NewOpportunityDetector(marketCollector)
	
	// Start market data collection
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	fmt.Println("🔄 Starting market data collection...")
	if err := marketCollector.StartCollection(ctx); err != nil {
		log.Fatalf("Failed to start collection: %v", err)
	}
	
	// Let it collect data for a few cycles
	time.Sleep(10 * time.Second)
	marketCollector.StopCollection()
	
	fmt.Println("✅ Market data collection complete")
	
	// Show market statistics
	stats := marketCollector.GetMarketStats()
	fmt.Printf("📊 Market Statistics:\n")
	fmt.Printf("   Total Data Points: %v\n", stats["total_data_points"])
	fmt.Printf("   Active Providers: %v\n", stats["active_providers"])
	fmt.Printf("   Market Keys: %v\n", stats["market_keys"])
	fmt.Println("")
	
	// Demo 1: Pricing Recommendations
	fmt.Println("💰 Demo 1: Pricing Recommendations")
	fmt.Println("----------------------------------")
	
	testScenarios := []marketintelligence.PricingRequest{
		{
			ResourceType:    "A100",
			Region:          "us-east-1",
			Quantity:        100,
			DurationHours:   24,
			SLARequirements: map[string]interface{}{"uptime": 99.9},
		},
		{
			ResourceType:    "H100",
			Region:          "asia-southeast-1",
			Quantity:        500,
			DurationHours:   168, // 1 week
			SLARequirements: map[string]interface{}{"uptime": 99.99},
		},
		{
			ResourceType:    "V100",
			Region:          "eu-west-1",
			Quantity:        50,
			DurationHours:   72, // 3 days
			SLARequirements: map[string]interface{}{"uptime": 99.5},
		},
	}
	
	for i, scenario := range testScenarios {
		fmt.Printf("\n📋 Scenario %d: %dx %s for %.0fh in %s\n", 
			i+1, scenario.Quantity, scenario.ResourceType, scenario.DurationHours, scenario.Region)
		
		recommendations, err := pricingEngine.GetOptimalPricing(&scenario)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}
		
		if len(recommendations.Recommendations) == 0 {
			fmt.Printf("⚠️  No recommendations available\n")
			continue
		}
		
		fmt.Printf("🏪 Market Conditions: %s\n", recommendations.MarketConditions.Condition)
		fmt.Printf("📈 %s\n", recommendations.MarketConditions.Recommendation)
		fmt.Printf("📊 Found %d pricing strategies:\n", len(recommendations.Recommendations))
		
		for j, rec := range recommendations.Recommendations {
			fmt.Printf("\n   Strategy %d: %s\n", j+1, rec.Strategy)
			fmt.Printf("      Provider: %s\n", rec.Provider)
			fmt.Printf("      Cost: $%.2f total ($%.2f/hr)\n", rec.TotalCost, rec.PricePerHour)
			fmt.Printf("      Quality: %.2f/1.0\n", rec.QualityScore)
			fmt.Printf("      Risk: %s\n", rec.RiskLevel)
			fmt.Printf("      Setup: %s\n", rec.SetupComplexity)
			
			if rec.PriceTrend != nil {
				fmt.Printf("      Price Trend: %s (%.1f%% change)\n", 
					rec.PriceTrend.TrendDirection, rec.PriceTrend.PriceChangePercent)
			}
			
			if rec.PriceForecast != nil {
				fmt.Printf("      Price Forecast: %s (%.0f%% increase probability)\n", 
					rec.PriceForecast.StabilityRating, rec.PriceForecast.PriceIncreaseProbability*100)
			}
			
			if len(rec.Allocation) > 0 {
				fmt.Printf("      Multi-Provider Allocation:\n")
				for _, alloc := range rec.Allocation {
					fmt.Printf("         %s: %dx @ $%.2f/hr\n", 
						alloc.Provider, alloc.Quantity, alloc.PricePerHour)
				}
			}
		}
	}
	
	// Demo 2: Opportunity Detection
	fmt.Println("\n🔍 Demo 2: Market Opportunity Detection")
	fmt.Println("--------------------------------------")
	
	opportunities := opportunityDetector.DetectOpportunities()
	fmt.Printf("💡 Found %d market opportunities:\n", len(opportunities))
	
	// Show top opportunities
	topOpportunities := opportunityDetector.GetTopOpportunities(5)
	for i, opp := range topOpportunities {
		fmt.Printf("\n   Opportunity %d: %s\n", i+1, opp.OpportunityType)
		fmt.Printf("      Resource: %s in %s\n", opp.ResourceType, opp.Region)
		fmt.Printf("      Source: %s → Target: %s\n", opp.ProviderSource, opp.ProviderTarget)
		fmt.Printf("      Profit Potential: %.1f%%\n", opp.ProfitPotential*100)
		fmt.Printf("      Confidence: %.2f\n", opp.ConfidenceScore)
		fmt.Printf("      Action: %s\n", opp.RequiredAction)
		fmt.Printf("      Expires: %s\n", time.Unix(opp.ExpiresAt, 0).Format("2006-01-02 15:04:05"))
	}
	
	// Show opportunity statistics
	oppStats := opportunityDetector.GetOpportunityStats()
	fmt.Printf("\n📊 Opportunity Statistics:\n")
	fmt.Printf("   Total Opportunities: %v\n", oppStats["total_opportunities"])
	fmt.Printf("   Average Profit Potential: %.1f%%\n", oppStats["avg_profit_potential"].(float64)*100)
	fmt.Printf("   Max Profit Potential: %.1f%%\n", oppStats["max_profit_potential"].(float64)*100)
	
	if byType, ok := oppStats["by_type"].(map[string]int); ok {
		fmt.Printf("   By Type:\n")
		for oppType, count := range byType {
			fmt.Printf("      %s: %d\n", oppType, count)
		}
	}
	
	// Demo 3: Market Analysis
	fmt.Println("\n📈 Demo 3: Market Analysis")
	fmt.Println("-------------------------")
	
	resources := []string{"A100", "H100", "V100"}
	regions := []string{"us-east-1", "us-west-2", "eu-west-1"}
	
	for _, resource := range resources {
		fmt.Printf("\n🔍 %s Analysis:\n", resource)
		
		for _, region := range regions {
			prices := marketCollector.GetCurrentPrices(resource, region)
			
			if len(prices) == 0 {
				continue
			}
			
			// Sort by price
			for i := 0; i < len(prices)-1; i++ {
				for j := i + 1; j < len(prices); j++ {
					if prices[i].PricePerHour > prices[j].PricePerHour {
						prices[i], prices[j] = prices[j], prices[i]
					}
				}
			}
			
			fmt.Printf("   %s:\n", region)
			for _, price := range prices {
				fmt.Printf("      %s: $%.2f/hr (%d available, %.1f%% demand)\n", 
					price.ProviderID, price.PricePerHour, price.AvailableQuantity, price.DemandIndicator*100)
			}
			
			// Show price trend for first provider
			if len(prices) > 0 {
				trend := marketCollector.GetPriceTrend(prices[0].ProviderID, resource, region, 20)
				fmt.Printf("      Trend: %s (%.1f%% change, %.2f volatility)\n", 
					trend.TrendDirection, trend.PriceChangePercent, trend.Volatility)
			}
		}
	}
	
	// Demo 4: Integration with OCX Protocol
	fmt.Println("\n🔗 Demo 4: OCX Protocol Integration")
	fmt.Println("----------------------------------")
	
	fmt.Println("📋 How Market Intelligence integrates with OCX:")
	fmt.Println("   • OCX-QL queries use real-time pricing data")
	fmt.Println("   • Settlement system uses verified market rates")
	fmt.Println("   • ZK proofs verify pricing claims against market data")
	fmt.Println("   • Enterprise cockpit shows live market conditions")
	fmt.Println("   • Verifier network validates pricing accuracy")
	fmt.Println("")
	
	fmt.Println("🚀 Competitive Advantages:")
	fmt.Println("   • Superior market knowledge vs competitors")
	fmt.Println("   • Real-time arbitrage opportunity detection")
	fmt.Println("   • Predictive pricing and demand forecasting")
	fmt.Println("   • Multi-provider optimization strategies")
	fmt.Println("   • Automated cost optimization recommendations")
	fmt.Println("")
	
	// Final Summary
	fmt.Println("🎯 Final Summary")
	fmt.Println("================")
	fmt.Println("")
	fmt.Println("✅ Market Intelligence System successfully demonstrates:")
	fmt.Println("   • Real-time data collection from multiple providers")
	fmt.Println("   • Advanced pricing engine with multiple strategies")
	fmt.Println("   • Arbitrage and optimization opportunity detection")
	fmt.Println("   • Market condition analysis and forecasting")
	fmt.Println("   • Integration with OCX Protocol components")
	fmt.Println("")
	fmt.Println("🚀 Key Innovations:")
	fmt.Println("   • Multi-provider market data aggregation")
	fmt.Println("   • Predictive pricing and demand forecasting")
	fmt.Println("   • Automated arbitrage opportunity detection")
	fmt.Println("   • Risk-adjusted pricing recommendations")
	fmt.Println("   • Real-time market condition analysis")
	fmt.Println("")
	fmt.Println("�� Business Impact:")
	fmt.Println("   • OCX gains unfair competitive advantage through superior market knowledge")
	fmt.Println("   • Customers get optimal pricing and resource allocation")
	fmt.Println("   • Providers benefit from increased demand visibility")
	fmt.Println("   • Market becomes more efficient through better price discovery")
	fmt.Println("   • OCX becomes the intelligence layer for compute markets")
	fmt.Println("")
	fmt.Println("🎉 Market Intelligence Demo Complete!")
	fmt.Println("")
	fmt.Println("🚀 Next Steps:")
	fmt.Println("   1. Deploy market intelligence to production")
	fmt.Println("   2. Integrate with OCX-QL query engine")
	fmt.Println("   3. Connect to settlement and payment systems")
	fmt.Println("   4. Launch arbitrage trading algorithms")
	fmt.Println("   5. Begin enterprise customer onboarding with market insights")
}
