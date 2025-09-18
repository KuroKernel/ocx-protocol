package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"ocx.local/internal/config"
	"ocx.local/internal/services"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()
	
	// Create service manager
	serviceManager := services.NewServiceManager(cfg)
	
	// Create HTTP server
	mux := http.NewServeMux()
	
	// Configuration status endpoint
	mux.HandleFunc("/config/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		status := serviceManager.GetConfigurationStatus()
		json.NewEncoder(w).Encode(status)
	})
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		health := serviceManager.HealthCheck(r.Context())
		json.NewEncoder(w).Encode(health)
	})
	
	// Missing configuration endpoint
	mux.HandleFunc("/config/missing", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		missing := serviceManager.GetMissingConfiguration()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"missing": missing,
		})
	})
	
	// API keys needed endpoint
	mux.HandleFunc("/config/keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		keys := serviceManager.GetAPIKeysNeeded()
		json.NewEncoder(w).Encode(keys)
	})
	
	// Setup instructions endpoint
	mux.HandleFunc("/config/instructions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		instructions := serviceManager.GetServiceInstructions()
		json.NewEncoder(w).Encode(instructions)
	})
	
	// Print configuration status
	fmt.Println("🔧 OCX Protocol Configuration Status")
	fmt.Println("=====================================")
	
	// Check each service
	missing := serviceManager.GetMissingConfiguration()
	if len(missing) == 0 {
		fmt.Println("✅ All services are properly configured!")
	} else {
		fmt.Println("❌ Missing configuration:")
		for _, item := range missing {
			fmt.Printf("   - %s\n", item)
		}
	}
	
	fmt.Println("\n📋 API Keys Needed:")
	keys := serviceManager.GetAPIKeysNeeded()
	for service, keyList := range keys {
		if len(keyList) > 0 {
			fmt.Printf("   %s:\n", service)
			for _, key := range keyList {
				fmt.Printf("     - %s\n", key)
			}
		}
	}
	
	fmt.Println("\n🚀 Starting configuration server on port 8081...")
	fmt.Println("   - Configuration Status: http://localhost:8081/config/status")
	fmt.Println("   - Health Check: http://localhost:8081/health")
	fmt.Println("   - Missing Config: http://localhost:8081/config/missing")
	fmt.Println("   - API Keys Needed: http://localhost:8081/config/keys")
	fmt.Println("   - Setup Instructions: http://localhost:8081/config/instructions")
	
	// Start server
	port := os.Getenv("CONFIG_PORT")
	if port == "" {
		port = "8081"
	}
	
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
