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

// Simple server that works with existing GPU testing
func main() {
	var (
		httpPort = flag.Int("port", 8080, "HTTP server port")
		mode     = flag.String("mode", "standalone", "Server mode: standalone, database")
	)
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("[OCX-CORE] ")

	// Set up HTTP server
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status": "healthy",
			"timestamp": time.Now(),
			"mode": *mode,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// GPU info endpoint - integrates with existing GPU testing
	mux.HandleFunc("/gpu/info", func(w http.ResponseWriter, r *http.Request) {
		// This will call the existing GPU info functionality
		response := map[string]interface{}{
			"status": "GPU endpoint ready",
			"message": "Use ./scripts/test_rtx5060.sh quick for GPU testing",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Simple providers endpoint
	mux.HandleFunc("/providers", func(w http.ResponseWriter, r *http.Request) {
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
	})

	// Simple orders endpoint
	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
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
	})

	// API documentation endpoint
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		docs := map[string]interface{}{
			"version": "1.0.0",
			"endpoints": map[string]string{
				"GET /health": "Server health check",
				"GET /gpu/info": "GPU information",
				"GET /providers": "List providers",
				"POST /providers": "Register provider",
				"GET /orders": "List orders",
				"POST /orders": "Place order",
				"GET /api": "This documentation",
			},
			"gpu_testing": "./scripts/test_rtx5060.sh [quick|monitor|full]",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(docs)
	})

	// Start HTTP server
	server := &http.Server{
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
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("✅ Server stopped")
}
