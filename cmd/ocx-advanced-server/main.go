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
	
	"ocx.local/internal/matching"
	"ocx.local/internal/reputation"
	"ocx.local/internal/query"
	"ocx.local/internal/sessions"
)

// Advanced OCX server with all integrated features
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
	log.SetPrefix("[OCX-ADVANCED] ")

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

	// Initialize components
	reputationEngine := reputation.NewDatabaseReputationEngine(db)
	queryEngine := query.NewDatabaseQueryEngine(db)
	matchingEngine := matching.NewMatchingEngine(db)
	sessionManager := sessions.NewSessionManager(db)

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
			"features": []string{"reputation", "query", "matching", "sessions"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Reputation endpoints
	mux.HandleFunc("/reputation/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			// Get reputation scores
			scores, err := reputationEngine.GetTopProviders(10)
			if err != nil {
				http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(scores)
		}
	})

	// Query endpoints
	mux.HandleFunc("/query/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			// Execute OCX-QL query
			var queryReq query.OCXQuery
			if err := json.NewDecoder(r.Body).Decode(&queryReq); err != nil {
				http.Error(w, "Invalid query format", http.StatusBadRequest)
				return
			}
			
			result, err := queryEngine.ExecuteQuery(&queryReq)
			if err != nil {
				http.Error(w, fmt.Sprintf("Query error: %v", err), http.StatusInternalServerError)
				return
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		}
	})

	// Matching endpoints
	mux.HandleFunc("/matching/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			// Process pending orders
			count, err := matchingEngine.ProcessPendingOrders()
			if err != nil {
				http.Error(w, fmt.Sprintf("Matching error: %v", err), http.StatusInternalServerError)
				return
			}
			
			response := map[string]interface{}{
				"matched_orders": count,
				"status": "success",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	})

	// Session endpoints
	mux.HandleFunc("/sessions/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			// Get active sessions
			sessions, err := sessionManager.GetActiveSessions()
			if err != nil {
				http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(sessions)
		}
	})

	// Enhanced providers endpoint with reputation
	mux.HandleFunc("/providers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			rows, err := db.Query(`
				SELECT p.provider_id, p.operator_address, p.geographic_region, 
				       p.reputation_score, p.status, p.registration_timestamp,
				       COUNT(cu.unit_id) as unit_count
				FROM providers p
				LEFT JOIN compute_units cu ON p.provider_id = cu.provider_id
				GROUP BY p.provider_id, p.operator_address, p.geographic_region, 
				         p.reputation_score, p.status, p.registration_timestamp
				ORDER BY p.reputation_score DESC
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
				var unitCount int
				
				if err := rows.Scan(&id, &address, &region, &reputation, &status, &timestamp, &unitCount); err != nil {
					continue
				}
				
				providers = append(providers, map[string]interface{}{
					"id": id,
					"operator_address": address,
					"geographic_region": region,
					"reputation_score": reputation,
					"status": status,
					"unit_count": unitCount,
					"registered_at": timestamp,
				})
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(providers)
		}
	})

	// Enhanced orders endpoint with matching
	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			rows, err := db.Query(`
				SELECT co.order_id, co.requester_id, co.required_hardware_type, 
				       co.max_price_per_hour_usdc, co.order_status, co.placed_at,
				       COUNT(om.match_id) as match_count
				FROM compute_orders co
				LEFT JOIN order_matches om ON co.order_id = om.order_id
				GROUP BY co.order_id, co.requester_id, co.required_hardware_type, 
				         co.max_price_per_hour_usdc, co.order_status, co.placed_at
				ORDER BY co.placed_at DESC
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
				var matchCount int
				
				if err := rows.Scan(&id, &requesterID, &hardwareType, &maxPrice, &status, &timestamp, &matchCount); err != nil {
					continue
				}
				
				orders = append(orders, map[string]interface{}{
					"id": id,
					"requester_id": requesterID,
					"hardware_type": hardwareType,
					"max_price_per_hour_usdc": maxPrice,
					"status": status,
					"match_count": matchCount,
					"placed_at": timestamp,
				})
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(orders)
		}
	})

	// Statistics endpoint with all features
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := make(map[string]interface{})
		
		// Basic counts
		var providerCount, orderCount, sessionCount, unitCount int
		db.QueryRow("SELECT COUNT(*) FROM providers").Scan(&providerCount)
		db.QueryRow("SELECT COUNT(*) FROM compute_orders").Scan(&orderCount)
		db.QueryRow("SELECT COUNT(*) FROM compute_sessions").Scan(&sessionCount)
		db.QueryRow("SELECT COUNT(*) FROM compute_units").Scan(&unitCount)
		
		stats["providers"] = providerCount
		stats["orders"] = orderCount
		stats["sessions"] = sessionCount
		stats["compute_units"] = unitCount
		
		// Active sessions
		activeSessions, _ := sessionManager.GetActiveSessions()
		stats["active_sessions"] = len(activeSessions)
		
		// Top providers by reputation
		topProviders, _ := reputationEngine.GetTopProviders(5)
		stats["top_providers"] = topProviders

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	// API documentation endpoint
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		docs := map[string]interface{}{
			"version": "2.0.0",
			"database": "PostgreSQL connected",
			"features": []string{"reputation", "query", "matching", "sessions"},
			"endpoints": map[string]string{
				"GET /health": "Server and database health check",
				"GET /providers": "List providers with reputation scores",
				"GET /orders": "List orders with match counts",
				"GET /sessions/": "List active compute sessions",
				"GET /reputation/": "Get reputation scores",
				"POST /query/": "Execute OCX-QL queries",
				"POST /matching/": "Process pending order matching",
				"GET /stats": "Advanced statistics with all features",
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
		log.Printf("🚀 OCX Advanced Server starting on port %d", *httpPort)
		log.Printf("📊 Database: %s@%s:%d/%s", *dbUser, *dbHost, *dbPort, *dbName)
		log.Printf("🧠 Features: Reputation, Query, Matching, Sessions")
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
