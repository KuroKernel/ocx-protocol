package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"context"
)

// Enhanced OCX Core Server with all protocol endpoints
func main() {
	var (
		httpPort = flag.Int("port", 8080, "HTTP server port")
		// mode     = flag.String("mode", "standalone", "Server mode: standalone, database")
	)
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("[OCX-CORE] ")

	// Create server instance
	server := &OCXServer{}

	// Set up HTTP server
	mux := http.NewServeMux()

	// Core protocol endpoints
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/gpu/info", server.handleGPUInfo)
	
	// Identity management
	mux.HandleFunc("/identities", server.handleIdentities)
	
	// Offers management
	mux.HandleFunc("/offers", server.handleOffers)
	
	// Orders management
	mux.HandleFunc("/orders", server.handleOrders)
	
	// Leases management
	mux.HandleFunc("/leases", server.handleLeases)
	mux.HandleFunc("/leases/", server.handleLeaseState)
	
	// Market data
	mux.HandleFunc("/market/stats", server.handleMarketStats)
	mux.HandleFunc("/market/active", server.handleMarketActive)
	
	// Providers (legacy endpoint)
	mux.HandleFunc("/providers", server.handleProviders)
	
	// API documentation
	mux.HandleFunc("/api", server.handleAPI)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", *httpPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("🚀 OCX Core Server starting on port %d", *httpPort)
		log.Printf("📖 API Documentation: http://localhost:%d/api", *httpPort)
		log.Printf("💓 Health Check: http://localhost:%d/health", *httpPort)
		log.Printf("🖥️  GPU Testing: ./scripts/test_rtx5060.sh quick")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("🛑 Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("✅ Server stopped")
}

// Health check endpoint
func (s *OCXServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now(),
		"mode": "standalone",
		"version": "0.1.0",
		"protocol": "OCX",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GPU info endpoint
func (s *OCXServer) handleGPUInfo(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status": "GPU endpoint ready",
		"message": "Use ./scripts/test_rtx5060.sh quick for GPU testing",
		"gpu_testing": "./scripts/test_rtx5060.sh [quick|monitor|full]",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Providers endpoint (legacy)
func (s *OCXServer) handleProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		providers := []map[string]interface{}{
			{
				"id": "local-gpu-provider",
				"name": "Local NVIDIA Provider",
				"gpu_model": "NVIDIA Graphics Device",
				"status": "active",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(providers)
	case "POST":
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"status": "created",
			"message": "Provider registration endpoint ready",
		}
		json.NewEncoder(w).Encode(response)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Orders endpoint
func (s *OCXServer) handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		orders := []map[string]interface{}{
			{
				"id": "sample-order-1",
				"status": "pending",
				"gpu_requirement": "NVIDIA",
				"created_at": time.Now(),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(orders)
	case "POST":
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"status": "created",
			"order_id": fmt.Sprintf("order_%d", time.Now().UnixNano()),
			"message": "Order placement endpoint ready",
		}
		json.NewEncoder(w).Encode(response)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// API documentation endpoint
func (s *OCXServer) handleAPI(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"version": "0.1.0",
		"protocol": "OCX",
		"endpoints": map[string]string{
			"GET /health": "Server health check",
			"GET /gpu/info": "GPU information",
			"GET /identities": "List identities",
			"POST /identities": "Create identity",
			"GET /offers": "List offers",
			"POST /offers": "Create offer",
			"GET /orders": "List orders",
			"POST /orders": "Place order",
			"GET /leases": "List leases",
			"PUT /leases/{id}/state": "Update lease state",
			"GET /market/stats": "Market statistics",
			"GET /market/active": "Active market data",
			"GET /providers": "List providers (legacy)",
			"POST /providers": "Register provider (legacy)",
			"GET /api": "This documentation",
			"POST /execute": "Execute deterministic computation",
			"GET /execute/": "List execution receipts",
		},
		"gpu_testing": "./scripts/test_rtx5060.sh [quick|monitor|full]",
		"cli_usage": "./ocxctl --help",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}
