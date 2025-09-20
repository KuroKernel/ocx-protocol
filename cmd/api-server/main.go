package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ocx.local/config"
	"ocx.local/pkg/api"
	"ocx.local/pkg/keystore"
	"ocx.local/pkg/ocx"
)

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key")
}

func main() {
	// Load configuration
	cfg := config.Load()
	
	// Initialize OCX engine
	ocxEngine := ocx.New()
	
	// Initialize keystore
	ks, err := keystore.New(cfg.KeystoreDir)
	if err != nil {
		log.Fatalf("Failed to initialize keystore: %v", err)
	}
	
	// Create API server
	server := api.NewServer(ocxEngine, ks)
	
	// Create mux with CORS and metrics middleware
	
	// Add CORS middleware
	corsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enableCORS(w)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// Route to appropriate handler
		switch r.URL.Path {
		case "/api/v1/execute":
			server.HandleExecute(w, r)
		case "/api/v1/verify":
			server.HandleVerify(w, r)
		case "/api/v1/receipts":
			server.HandleReceipts(w, r)
		case "/health":
			server.HandleHealth(w, r)
		case "/keys/":
			server.HandleKeys(w, r)
		case "/metrics":
			server.HandleMetrics(w, r)
		case "/readyz":
			server.HandleReadiness(w, r)
		case "/livez":
			server.HandleLiveness(w, r)
		default:
			http.NotFound(w, r)
		}
	})
	
	// Add metrics middleware
	handler := server.MetricsMiddleware(corsHandler)
	
	// Start server
	log.Printf("🚀 OCX API Server starting on port %s", cfg.Port)
	log.Printf("📊 Metrics enabled: %v", cfg.MetricsEnabled)
	log.Printf("🗄️  Database: %s", cfg.DatabaseType)
	log.Printf("🔑 Keystore: %s", cfg.KeystoreDir)
	log.Printf("")
	log.Printf("🌐 Endpoints available:")
	log.Printf("  POST /api/v1/execute    - Execute code with idempotency")
	log.Printf("  POST /api/v1/verify     - Verify receipts (JSON/CBOR)")
	log.Printf("  GET  /api/v1/receipts   - List receipts")
	log.Printf("  GET  /health            - Health check")
	log.Printf("  GET  /readyz            - Readiness probe")
	log.Printf("  GET  /livez             - Liveness probe")
	log.Printf("  GET  /keys/{id}         - Key metadata")
	log.Printf("  GET  /metrics           - Prometheus metrics")
	log.Printf("")
	log.Printf("📋 OpenAPI spec: /api/openapi.yaml")
	
	// Configure server with defensive timeouts
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	
	log.Printf("🚀 Starting server with defensive timeouts...")
	
	// Graceful shutdown handling
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("🛑 Shutting down server...")
	
	// Give in-flight requests up to 10 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	
	log.Println("✅ Server exited gracefully")
}
