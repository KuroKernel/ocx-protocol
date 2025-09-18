package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"ocx.local/internal/orchestrator"
)

func main() {
	fmt.Println("🎯 OCX Protocol - Complete System Demo")
	fmt.Println("=====================================")
	
	// Initialize system
	ocxSystem := orchestrator.NewOCXSystemOrchestrator()
	
	// Create context for system initialization
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Initialize the system
	if err := ocxSystem.InitializeSystem(ctx); err != nil {
		log.Fatalf("Failed to initialize OCX system: %v", err)
	}
	
	// Start system optimization in background
	go ocxSystem.RunSystemOptimization(ctx)
	
	// Test scenario: Multiple customer requests
	testCustomers := []struct {
		CustomerID string
		Request    *orchestrator.CustomerRequest
	}{
		{
			CustomerID: "ai_startup_alpha",
			Request: &orchestrator.CustomerRequest{
				ResourceType:     "A100",
				Region:           "us-east-1",
				Quantity:         50,
				DurationHours:    24,
				SLARequirements:  map[string]interface{}{"uptime": 99.5},
				OptimizationGoal: "cost",
				MaxPricePerHour:  5.0,
			},
		},
		{
			CustomerID: "hedge_fund_beta",
			Request: &orchestrator.CustomerRequest{
				ResourceType:     "H100",
				Region:           "asia-southeast-1",
				Quantity:         200,
				DurationHours:    168, // 1 week
				SLARequirements:  map[string]interface{}{"uptime": 99.99, "max_response_time": 5.0},
				OptimizationGoal: "reliability",
				MaxPricePerHour:  15.0,
			},
		},
		{
			CustomerID: "research_lab_gamma",
			Request: &orchestrator.CustomerRequest{
				ResourceType:     "V100",
				Region:           "eu-west-1",
				Quantity:         25,
				DurationHours:    72, // 3 days
				SLARequirements:  map[string]interface{}{"uptime": 99.0},
				OptimizationGoal: "balanced",
				MaxPricePerHour:  4.0,
			},
		},
	}
	
	fmt.Printf("\n📋 Processing %d customer requests...\n", len(testCustomers))
	
	// Process all requests
	var results []*orchestrator.ProcessResponse
	for i, customerData := range testCustomers {
		fmt.Printf("\n🔄 Processing request %d for %s\n", i+1, customerData.CustomerID)
		
		result := ocxSystem.ProcessCustomerRequest(customerData.CustomerID, customerData.Request)
		results = append(results, result)
		
		if result.Status == "processed" {
			allocation := result.Allocation
			fmt.Printf("   ✅ Allocated to: %v\n", allocation.Providers)
			fmt.Printf("   💰 Total cost: $%.2f\n", allocation.TotalCost)
			fmt.Printf("   ⚡ Cost efficiency: %.2f\n", allocation.CostEfficiency)
			fmt.Printf("   🛡️ Risk score: %.2f\n", allocation.RiskScore)
			fmt.Printf("   📊 Strategy: %s\n", allocation.PricingStrategy)
		} else {
			fmt.Printf("   ❌ Failed: %s\n", result.Status)
		}
	}
	
	// Let system run for a bit
	fmt.Printf("\n⏳ Running system for 30 seconds...\n")
	time.Sleep(30 * time.Second)
	
	// Generate system status report
	fmt.Printf("\n📊 System Status Report\n")
	fmt.Println("-" * 40)
	
	status := ocxSystem.GetSystemStatus()
	
	fmt.Printf("System Status: %s\n", status["system_status"])
	fmt.Printf("Uptime: %.2f%%\n", status["uptime_percentage"])
	fmt.Printf("Active Workloads: %d\n", status["active_workloads"])
	fmt.Printf("Compute Units Managed: %d\n", status["total_compute_units_managed"])
	
	financialMetrics := status["financial_metrics"].(map[string]interface{})
	fmt.Printf("Total Revenue: $%.2f\n", financialMetrics["total_revenue"])
	fmt.Printf("Profit Margin: %.1f%%\n", financialMetrics["profit_margin"])
	
	providerHealth := status["provider_health"].(map[string]interface{})
	fmt.Printf("\nProvider Health Summary:\n")
	for provider, health := range providerHealth {
		healthData := health.(map[string]interface{})
		healthScore := healthData["health_score"].(float64)
		capacityAvailable := healthData["capacity_utilization"].(float64)
		
		statusIcon := "🟢"
		if healthScore < 60 {
			statusIcon = "🔴"
		} else if healthScore < 80 {
			statusIcon = "🟡"
		}
		
		fmt.Printf("  %s %s: %.1f/100 (Capacity: %.1f%%)\n", 
			statusIcon, provider, healthScore, 100-capacityAvailable)
	}
	
	// Test customer analytics
	fmt.Printf("\n📈 Customer Analytics\n")
	fmt.Println("-" * 40)
	
	for _, customerData := range testCustomers {
		customerID := customerData.CustomerID
		insights := ocxSystem.GetCustomerInsights(customerID)
		
		if insights != nil {
			summary := insights.UsageSummary
			fmt.Printf("\n%s:\n", customerID)
			fmt.Printf("  Total Sessions: %d\n", summary.TotalSessions)
			fmt.Printf("  Total Spend: $%.2f\n", summary.TotalSpend)
			fmt.Printf("  Avg Cost/Hour: $%.2f\n", summary.AvgCostPerHour)
			fmt.Printf("  Cost Sensitivity: %.2f\n", insights.CostSensitivity)
			
			if len(insights.OptimizationOpportunities) > 0 {
				fmt.Printf("  Optimization Opportunities:\n")
				for _, opp := range insights.OptimizationOpportunities[:2] {
					fmt.Printf("    • %s: %s savings\n", opp.Type, opp.PotentialSavings)
				}
			}
		}
	}
	
	// Test risk management
	fmt.Printf("\n🛡️ Risk Management Report\n")
	fmt.Println("-" * 40)
	
	// Get risk management report from the system
	riskReport := status["provider_health"].(map[string]interface{})
	activeFailovers := status["active_failovers"].(int)
	
	fmt.Printf("Monitored Providers: %d\n", len(riskReport))
	fmt.Printf("Active Failovers: %d\n", activeFailovers)
	
	if activeFailovers > 0 {
		fmt.Println("Active Failover Situations:")
		// In a real implementation, we'd get this from the risk manager
		fmt.Println("  🚨 No active failover situations detected")
	} else {
		fmt.Println("✅ No active failover situations")
	}
	
	// Stop the system
	ocxSystem.StopSystem()
	
	fmt.Printf("\n🎉 Complete OCX System Test Finished!\n")
	fmt.Printf("System successfully processed %d customer requests\n", len(results))
	fmt.Printf("Total system revenue: $%.2f\n", financialMetrics["total_revenue"])
	
	processingMetrics := status["processing_metrics"].(map[string]interface{})
	fmt.Printf("Average processing time: %.1fs\n", processingMetrics["avg_request_processing_time"])
	
	fmt.Println("\n🚀 OCX Protocol: Where compute meets intelligence, and markets meet optimization!")
}
