package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ocx-protocol/internal/enterprise"
)

func main() {
	fmt.Println("🏢 OCX Protocol - Enterprise Cockpit Demo")
	fmt.Println("==========================================")
	fmt.Println("")
	fmt.Println("This demo shows the complete Enterprise Cockpit - the reference")
	fmt.Println("implementation of the OCX Protocol standard.")
	fmt.Println("")
	fmt.Println("Key Features:")
	fmt.Println("✅ OCX-QL DSL for compute resource management")
	fmt.Println("✅ Multi-provider resource discovery")
	fmt.Println("✅ USD-based settlement (no tokens!)")
	fmt.Println("✅ Real-time SLA monitoring")
	fmt.Println("✅ Enterprise-friendly APIs")
	fmt.Println("")
	
	// Initialize the Enterprise Cockpit
	cockpit := enterprise.NewEnterpriseCockpit()
	
	// Show system health
	fmt.Println("🔍 System Health Check:")
	fmt.Println("----------------------")
	health := cockpit.HealthCheck()
	fmt.Printf("Status: %s\n", health["status"])
	fmt.Printf("Total Reservations: %v\n", health["total_reservations"])
	fmt.Printf("Active Reservations: %v\n", health["active_reservations"])
	fmt.Printf("API Version: %s\n", health["api_version"])
	fmt.Println("")
	
	// Demo 1: Basic Resource Reservation
	fmt.Println("🚀 Demo 1: Basic Resource Reservation")
	fmt.Println("------------------------------------")
	
	ctx := context.Background()
	customerID := "quantfund_alpha"
	
	fmt.Printf("Customer: %s\n", customerID)
	fmt.Println("Request: 200x A100 GPUs for 24 hours in Asia-Pacific")
	fmt.Println("SLA: 99.99% uptime, 5ms max response time")
	fmt.Println("")
	
	// Make reservation using the primary enterprise API
	reservation, err := cockpit.Reserve(ctx, customerID, 200, "A100", "24h", "asia", map[string]interface{}{
		"sla": map[string]interface{}{
			"uptime":           99.99,
			"max_response_time": 5.0,
			"max_setup_time":    15.0,
		},
	})
	
	if err != nil {
		log.Fatalf("Reservation failed: %v", err)
	}
	
	fmt.Printf("✅ Reservation created: %s\n", reservation.ReservationID)
	fmt.Printf("📊 Status: %s\n", reservation.Status)
	fmt.Printf("💰 Estimated cost: $%.2f\n", reservation.EstimatedCost)
	fmt.Printf("🏦 Escrow: $%.2f\n", reservation.EscrowAmount)
	fmt.Println("")
	
	// Wait for provisioning to complete
	fmt.Println("⏳ Waiting for provisioning...")
	for reservation.Status == enterprise.ReservationStatusDiscovering ||
		reservation.Status == enterprise.ReservationStatusAvailable ||
		reservation.Status == enterprise.ReservationStatusReserved ||
		reservation.Status == enterprise.ReservationStatusProvisioning {
		
		time.Sleep(2 * time.Second)
		
		// Get updated reservation
		updatedReservation, exists := cockpit.GetReservation(reservation.ReservationID)
		if !exists {
			log.Fatal("Reservation not found")
		}
		reservation = updatedReservation
		
		fmt.Printf("   Status: %s\n", reservation.Status)
	}
	
	if reservation.Status == enterprise.ReservationStatusActive {
		fmt.Println("")
		fmt.Println("🟢 Compute resources are ACTIVE!")
		fmt.Println("")
		
		// Get connection information
		connection, exists := cockpit.GetConnectionInfo(reservation.ReservationID)
		if exists {
			fmt.Println("🔗 Connection Information:")
			fmt.Printf("   SSH Host: %s\n", connection["ssh_host"])
			fmt.Printf("   SSH Port: %v\n", connection["ssh_port"])
			fmt.Printf("   Jupyter URL: %s\n", connection["jupyter_url"])
			fmt.Printf("   API Endpoint: %s\n", connection["api_endpoint"])
			fmt.Printf("   API Key: %s...\n", connection["api_key"].(string)[:20])
			fmt.Println("")
		}
		
		// Simulate monitoring for a few seconds
		fmt.Println("📊 Real-time Monitoring (10 seconds):")
		fmt.Println("-------------------------------------")
		for i := 0; i < 10; i++ {
			time.Sleep(1 * time.Second)
			
			monitoring, exists := cockpit.GetMonitoring(reservation.ReservationID, 5)
			if exists {
				recentMetrics, ok := monitoring["recent_metrics"].([]enterprise.MonitoringMetrics)
				if ok && len(recentMetrics) > 0 {
					latest := recentMetrics[len(recentMetrics)-1]
					fmt.Printf("   GPU: %.1f%% | Response: %.1fms | SLA: %s\n",
						latest.GPUUtilization,
						latest.ResponseTimeMs,
						monitoring["sla_status"])
				} else {
					fmt.Printf("   Monitoring data not yet available...\n")
				}
			}
		}
		fmt.Println("")
	}
	
	// Demo 2: Multiple Reservations
	fmt.Println("📋 Demo 2: Multiple Reservations")
	fmt.Println("--------------------------------")
	
	// Create additional reservations
	reservations := []*enterprise.ComputeReservation{reservation}
	
	// Create a second reservation
	reservation2, err := cockpit.Reserve(ctx, customerID, 100, "H100", "12h", "us-east", map[string]interface{}{
		"sla": map[string]interface{}{
			"uptime":           99.9,
			"max_response_time": 10.0,
		},
	})
	if err == nil {
		reservations = append(reservations, reservation2)
		fmt.Printf("✅ Second reservation: %s\n", reservation2.ReservationID)
	}
	
	// Create a third reservation
	reservation3, err := cockpit.Reserve(ctx, "ai_lab_beta", 50, "V100", "48h", "eu-west", nil)
	if err == nil {
		reservations = append(reservations, reservation3)
		fmt.Printf("✅ Third reservation: %s\n", reservation3.ReservationID)
	}
	
	fmt.Println("")
	
	// Show all reservations for the customer
	fmt.Printf("📊 All reservations for %s:\n", customerID)
	customerReservations := cockpit.ListReservations(customerID, nil)
	for _, r := range customerReservations {
		duration := time.Since(r.CreatedAt)
		fmt.Printf("   %s: %s (%.1fh ago)\n", r.ReservationID, r.Status, duration.Hours())
	}
	fmt.Println("")
	
	// Demo 3: System Statistics
	fmt.Println("📈 Demo 3: System Statistics")
	fmt.Println("----------------------------")
	
	// Get final health check
	finalHealth := cockpit.HealthCheck()
	fmt.Printf("Total Reservations: %v\n", finalHealth["total_reservations"])
	fmt.Printf("Active Reservations: %v\n", finalHealth["active_reservations"])
	
	// Show supported resource types
	if resourceTypes, ok := finalHealth["supported_resource_types"].([]string); ok {
		fmt.Printf("Supported Resource Types: %v\n", resourceTypes)
	}
	
	// Show supported regions
	if regions, ok := finalHealth["supported_regions"].([]string); ok {
		fmt.Printf("Supported Regions: %v\n", regions)
	}
	fmt.Println("")
	
	// Demo 4: OCX-QL Examples
	fmt.Println("🔍 Demo 4: OCX-QL Examples")
	fmt.Println("--------------------------")
	
	ocxqlExamples := []string{
		"H100 200\nregion: asia-pacific\nsla: 99.99%\nmax_price: $2.50",
		"A100 500\nregion: us-east\nsla: 99.9%\nfor training\ninterconnect: nvlink",
		"V100 100\nregion: eu-west\nsla: 99.5%\nmax_price: $1.50\nbudget: $500/hour",
		"TPU_V4 50\nregion: asia-singapore\nsla: 99.99%\nfor inference",
	}
	
	for i, example := range ocxqlExamples {
		fmt.Printf("Example %d:\n%s\n\n", i+1, example)
	}
	
	// Final Summary
	fmt.Println("🎯 Final Summary")
	fmt.Println("================")
	fmt.Println("")
	fmt.Println("✅ Enterprise Cockpit successfully demonstrates:")
	fmt.Println("   • OCX-QL DSL for compute resource management")
	fmt.Println("   • Multi-provider resource discovery and selection")
	fmt.Println("   • USD-based settlement (no token sales!)")
	fmt.Println("   • Real-time SLA monitoring and compliance")
	fmt.Println("   • Enterprise-friendly APIs and interfaces")
	fmt.Println("   • Complete reservation lifecycle management")
	fmt.Println("")
	fmt.Println("🚀 Key Differentiators:")
	fmt.Println("   • Neutral protocol standard (not a competing product)")
	fmt.Println("   • Mathematical guarantees for compute delivery")
	fmt.Println("   • Enterprise-friendly USD payments")
	fmt.Println("   • Multi-provider vendor neutrality")
	fmt.Println("   • Real-time SLA enforcement")
	fmt.Println("")
	fmt.Println("💡 This is the reference implementation of the OCX Protocol")
	fmt.Println("   standard - the 'Switzerland Play' for compute infrastructure!")
	fmt.Println("")
	fmt.Println("🌍 Big players adopt OCX because it's neutral, but they're")
	fmt.Println("   still dependent on our codebase and protocol evolution.")
	fmt.Println("")
	fmt.Println("🎉 Enterprise Cockpit Demo Complete!")
}
