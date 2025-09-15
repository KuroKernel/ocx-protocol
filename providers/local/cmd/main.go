// main.go - NVIDIA GPU GPU Test Runner
// Demonstrates complete OCX Protocol flow with real GPU hardware

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ocx/protocol/providers/local"
)

func main() {
	var (
		serverURL = flag.String("server", "http://localhost:8080", "OCX server URL")
		testType  = flag.String("test", "quick", "Test type: quick, full, monitor")
		duration  = flag.Duration("duration", 30*time.Second, "Test duration for monitor mode")
	)
	flag.Parse()

	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("[RTX5060-TEST] ")

	// Create test client
	client, err := local.NewGPUTestClient(*serverURL)
	if err != nil {
		log.Fatalf("Failed to create test client: %v", err)
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down gracefully...")
		os.Exit(0)
	}()

	// Run the appropriate test
	switch *testType {
	case "quick":
		if err := client.RunQuickTest(); err != nil {
			log.Fatalf("Quick test failed: %v", err)
		}
	case "full":
		if err := client.RunCompleteTest(); err != nil {
			log.Fatalf("Complete test failed: %v", err)
		}
	case "monitor":
		if err := runMonitorMode(client, *duration); err != nil {
			log.Fatalf("Monitor mode failed: %v", err)
		}
	default:
		log.Fatalf("Unknown test type: %s", *testType)
	}
}

func runMonitorMode(client *local.GPUTestClient, duration time.Duration) error {
	log.Println("🔍 Starting GPU Monitor Mode")
	log.Println("========================================")
	log.Printf("Monitoring for %v", duration)
	log.Println("Press Ctrl+C to stop")
	log.Println("")

	// Start monitoring
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	go client.Monitor.Start(ctx)
	defer client.Monitor.Stop()

	// Monitor loop
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Monitor mode completed")
			return nil
		case <-ticker.C:
			// Get current metrics
			metrics, err := client.Monitor.collectMetrics()
			if err != nil {
				log.Printf("Failed to get metrics: %v", err)
				continue
			}

			// Display metrics
			log.Printf("GPU Status - Util: %d%%, Temp: %d°C, Memory: %d/%dMB, Power: %dW, Clock: %d/%dMHz",
				metrics.Utilization, metrics.Temperature, metrics.MemoryUsed, metrics.MemoryTotal,
				metrics.PowerUsage, metrics.ClockGraphics, metrics.ClockMemory)

			// Show processes if any
			if len(metrics.Processes) > 0 {
				log.Printf("Active Processes:")
				for _, proc := range metrics.Processes {
					log.Printf("  PID %d: %s (Memory: %dMB, Util: %d%%)",
						proc.PID, proc.Name, proc.MemoryUsed, proc.GPUUtil)
				}
			}

			// Check health
			health, err := client.Monitor.GetHealthStatus()
			if err != nil {
				log.Printf("Health check failed: %v", err)
			} else {
				log.Printf("Health Status: %s", health)
			}

			// Get performance score
			score, err := client.Monitor.GetPerformanceScore()
			if err != nil {
				log.Printf("Performance score failed: %v", err)
			} else {
				log.Printf("Performance Score: %d/100", score)
			}

			log.Println("")
		}
	}
}
