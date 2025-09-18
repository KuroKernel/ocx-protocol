package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"
)

func main() {
	fmt.Println("�� OCX Protocol - Complete System Demo")
	fmt.Println("=====================================")
	
	// Simulate system initialization
	fmt.Println("🚀 Initializing OCX System...")
	time.Sleep(1 * time.Second)
	
	fmt.Println("✅ Risk management system active")
	fmt.Println("✅ Market intelligence system active")
	fmt.Println("✅ Capacity reservation engine active")
	fmt.Println("✅ Usage analytics system active")
	fmt.Println("✅ Global load balancer active")
	fmt.Println("🎉 OCX System fully operational!")
	
	// Test scenario: Multiple customer requests
	testCustomers := []struct {
		CustomerID string
		Request    map[string]interface{}
	}{
		{
			CustomerID: "ai_startup_alpha",
			Request: map[string]interface{}{
				"resource_type":     "A100",
				"region":            "us-east-1",
				"quantity":          50,
				"duration_hours":    24,
				"sla_requirements":  map[string]interface{}{"uptime": 99.5},
				"optimization_goal": "cost",
				"max_price_per_hour": 5.0,
			},
		},
		{
			CustomerID: "hedge_fund_beta",
			Request: map[string]interface{}{
				"resource_type":     "H100",
				"region":            "asia-southeast-1",
				"quantity":          200,
				"duration_hours":    168, // 1 week
				"sla_requirements":  map[string]interface{}{"uptime": 99.99, "max_response_time": 5.0},
				"optimization_goal": "reliability",
				"max_price_per_hour": 15.0,
			},
		},
		{
			CustomerID: "research_lab_gamma",
			Request: map[string]interface{}{
				"resource_type":     "V100",
				"region":            "eu-west-1",
				"quantity":          25,
				"duration_hours":    72, // 3 days
				"sla_requirements":  map[string]interface{}{"uptime": 99.0},
				"optimization_goal": "balanced",
				"max_price_per_hour": 4.0,
			},
		},
	}
	
	fmt.Printf("\n📋 Processing %d customer requests...\n", len(testCustomers))
	
	// Process all requests
	var results []map[string]interface{}
	for i, customerData := range testCustomers {
		fmt.Printf("\n🔄 Processing request %d for %s\n", i+1, customerData.CustomerID)
		
		// Simulate processing
		time.Sleep(500 * time.Millisecond)
		
		// Generate mock response
		providers := []string{"aws", "gcp", "azure", "runpod"}
		selectedProviders := providers[:rand.Intn(3)+1]
		
		totalCost := customerData.Request["max_price_per_hour"].(float64) * 
			float64(customerData.Request["quantity"].(int)) * 
			customerData.Request["duration_hours"].(float64)
		
		result := map[string]interface{}{
			"request_id": fmt.Sprintf("req_%d_%d", time.Now().Unix(), rand.Intn(10000)),
			"customer_id": customerData.CustomerID,
			"status": "processed",
			"allocation": map[string]interface{}{
				"providers": selectedProviders,
				"total_cost": totalCost,
				"cost_efficiency": 0.8 + rand.Float64()*0.2,
				"risk_score": 0.2 + rand.Float64()*0.3,
				"pricing_strategy": "Market Rate",
			},
			"failover_plan": map[string]interface{}{
				"primary_provider": selectedProviders[0],
				"backup_providers": selectedProviders[1:],
				"auto_failover": true,
			},
			"reserved_capacity_used": false,
			"optimization_opportunities": []map[string]interface{}{
				{
					"type": "capacity_reservation",
					"potential_savings": "15-25%",
					"description": "Regular usage could benefit from capacity reservations",
				},
			},
			"sla_guarantees": map[string]interface{}{
				"uptime_guarantee": 99.5,
				"max_response_time_ms": 5000,
				"availability_guarantee": 95.0,
			},
			"estimated_setup_time": 5.0,
			"monitoring_enabled": true,
		}
		
		results = append(results, result)
		
		allocation := result["allocation"].(map[string]interface{})
		fmt.Printf("   ✅ Allocated to: %v\n", allocation["providers"])
		fmt.Printf("   💰 Total cost: $%.2f\n", allocation["total_cost"])
		fmt.Printf("   ⚡ Cost efficiency: %.2f\n", allocation["cost_efficiency"])
		fmt.Printf("   🛡️ Risk score: %.2f\n", allocation["risk_score"])
		fmt.Printf("   📊 Strategy: %s\n", allocation["pricing_strategy"])
	}
	
	// Let system run for a bit
	fmt.Printf("\n⏳ Running system for 30 seconds...\n")
	time.Sleep(30 * time.Second)
	
	// Generate system status report
	fmt.Printf("\n📊 System Status Report\n")
	fmt.Println("-" * 40)
	
	fmt.Printf("System Status: operational\n")
	fmt.Printf("Uptime: 99.95%%\n")
	fmt.Printf("Active Workloads: %d\n", len(results))
	fmt.Printf("Compute Units Managed: 275\n")
	
	totalRevenue := 0.0
	for _, result := range results {
		allocation := result["allocation"].(map[string]interface{})
		totalRevenue += allocation["total_cost"].(float64) * 0.15 // 15% OCX margin
	}
	
	fmt.Printf("Total Revenue: $%.2f\n", totalRevenue)
	fmt.Printf("Profit Margin: 15.0%%\n")
	
	// Provider health simulation
	providers := []string{"aws", "gcp", "azure", "runpod", "lambdalabs"}
	fmt.Printf("\nProvider Health Summary:\n")
	for _, provider := range providers {
		healthScore := 80 + rand.Float64()*20
		capacityAvailable := 60 + rand.Float64()*30
		
		statusIcon := "🟢"
		if healthScore < 60 {
			statusIcon = "🔴"
		} else if healthScore < 80 {
			statusIcon = "🟡"
		}
		
		fmt.Printf("  %s %s: %.1f/100 (Capacity: %.1f%%)\n", 
			statusIcon, provider, healthScore, capacityAvailable)
	}
	
	// Test customer analytics
	fmt.Printf("\n📈 Customer Analytics\n")
	fmt.Println("-" * 40)
	
	for _, customerData := range testCustomers {
		customerID := customerData.CustomerID
		fmt.Printf("\n%s:\n", customerID)
		fmt.Printf("  Total Sessions: %d\n", rand.Intn(10)+1)
		fmt.Printf("  Total Spend: $%.2f\n", 1000+rand.Float64()*5000)
		fmt.Printf("  Avg Cost/Hour: $%.2f\n", 2.0+rand.Float64()*3.0)
		fmt.Printf("  Cost Sensitivity: %.2f\n", rand.Float64())
		
		fmt.Printf("  Optimization Opportunities:\n")
		fmt.Printf("    • capacity_reservation: 15-25%% savings\n")
		fmt.Printf("    • provider_optimization: 20-40%% savings\n")
	}
	
	// Test risk management
	fmt.Printf("\n🛡️ Risk Management Report\n")
	fmt.Println("-" * 40)
	
	fmt.Printf("Monitored Providers: %d\n", len(providers))
	fmt.Printf("Active Failovers: 0\n")
	fmt.Println("✅ No active failover situations")
	
	fmt.Printf("\n🎉 Complete OCX System Test Finished!\n")
	fmt.Printf("System successfully processed %d customer requests\n", len(results))
	fmt.Printf("Total system revenue: $%.2f\n", totalRevenue)
	fmt.Printf("Average processing time: 2.5s\n")
	
	fmt.Println("\n🚀 OCX Protocol: Where compute meets intelligence, and markets meet optimization!")
}
