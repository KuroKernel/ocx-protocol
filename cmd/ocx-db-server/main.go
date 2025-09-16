package main

import (
	"database/sql"
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

	_ "github.com/lib/pq"
)

// Database-connected OCX server
func main() {
	var (
		httpPort = flag.Int("port", 8080, "HTTP server port")
		dbHost   = flag.String("db-host", "localhost", "Database host")
		dbPort   = flag.Int("db-port", 5432, "Database port")
		dbUser   = flag.String("db-user", "ocx_user", "Database user")
		dbPass   = flag.String("db-pass", "ocx_password", "Database password")
		dbName   = flag.String("db-name", "ocx_protocol", "Database name")
	)
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("[OCX-DB] ")

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		*dbHost, *dbPort, *dbUser, *dbPass, *dbName)
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("✅ Connected to PostgreSQL database")

	// Set up HTTP server
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			http.Error(w, "Database unhealthy", http.StatusServiceUnavailable)
			return
		}
		response := map[string]interface{}{
			"status": "healthy",
			"timestamp": time.Now(),
			"database": "connected",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Providers endpoint with database
	mux.HandleFunc("/providers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			rows, err := db.Query(`
				SELECT provider_id, operator_address, geographic_region, 
				       reputation_score, status, registration_timestamp
				FROM providers 
				ORDER BY registration_timestamp DESC
			`)
			if err != nil {
				http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var providers []map[string]interface{}
			for rows.Next() {
				var id, address, region, status string
				var reputation float64
				var timestamp time.Time
				
				if err := rows.Scan(&id, &address, &region, &reputation, &status, &timestamp); err != nil {
					continue
				}
				
				providers = append(providers, map[string]interface{}{
					"id": id,
					"operator_address": address,
					"geographic_region": region,
					"reputation_score": reputation,
					"status": status,
					"registered_at": timestamp,
				})
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(providers)

		case "POST":
			// Simple provider registration
			response := map[string]interface{}{
				"status": "created",
				"message": "Provider registration endpoint ready",
				"provider_id": fmt.Sprintf("provider_%d", time.Now().UnixNano()),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Orders endpoint with database
	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			rows, err := db.Query(`
				SELECT order_id, requester_id, required_hardware_type, 
				       max_price_per_hour_usdc, order_status, placed_at
				FROM compute_orders 
				ORDER BY placed_at DESC
				LIMIT 50
			`)
			if err != nil {
				http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var orders []map[string]interface{}
			for rows.Next() {
				var id, requesterID, hardwareType, status string
				var maxPrice float64
				var timestamp time.Time
				
				if err := rows.Scan(&id, &requesterID, &hardwareType, &maxPrice, &status, &timestamp); err != nil {
					continue
				}
				
				orders = append(orders, map[string]interface{}{
					"id": id,
					"requester_id": requesterID,
					"hardware_type": hardwareType,
					"max_price_per_hour_usdc": maxPrice,
					"status": status,
					"placed_at": timestamp,
				})
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(orders)

		case "POST":
			// Simple order placement
			response := map[string]interface{}{
				"status": "created",
				"message": "Order placement endpoint ready",
				"order_id": fmt.Sprintf("order_%d", time.Now().UnixNano()),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Database stats endpoint
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := make(map[string]interface{})
		
		// Count providers
		var providerCount int
		db.QueryRow("SELECT COUNT(*) FROM providers").Scan(&providerCount)
		stats["providers"] = providerCount
		
		// Count orders
		var orderCount int
		db.QueryRow("SELECT COUNT(*) FROM compute_orders").Scan(&orderCount)
		stats["orders"] = orderCount
		
		// Count sessions
		var sessionCount int
		db.QueryRow("SELECT COUNT(*) FROM compute_sessions").Scan(&sessionCount)
		stats["sessions"] = sessionCount
		
		// Count compute units
		var unitCount int
		db.QueryRow("SELECT COUNT(*) FROM compute_units").Scan(&unitCount)
		stats["compute_units"] = unitCount

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	// API documentation endpoint
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		docs := map[string]interface{}{
			"version": "1.0.0",
			"database": "PostgreSQL connected",
			"endpoints": map[string]string{
				"GET /health": "Server and database health check",
				"GET /providers": "List providers from database",
				"POST /providers": "Register provider",
				"GET /orders": "List orders from database",
				"POST /orders": "Place order",
				"GET /stats": "Database statistics",
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
		log.Printf("🚀 OCX Database Server starting on port %d", *httpPort)
		log.Printf("📊 Database: %s@%s:%d/%s", *dbUser, *dbHost, *dbPort, *dbName)
		log.Printf("📖 API Documentation: http://localhost:%d/api", *httpPort)
		log.Printf("💓 Health Check: http://localhost:%d/health", *httpPort)
		log.Printf("📈 Statistics: http://localhost:%d/stats", *httpPort)
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
