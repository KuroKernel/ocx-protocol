package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ocx.local/internal/consensus"
	"ocx.local/internal/database"
	"ocx.local/internal/query"
	"ocx.local/internal/reputation"
)

func main() {
	var (
		dbHost     = flag.String("db-host", "localhost", "Database host")
		dbPort     = flag.Int("db-port", 5432, "Database port")
		dbUser     = flag.String("db-user", "ocx_user", "Database user")
		dbPassword = flag.String("db-password", "ocx_password", "Database password")
		dbName     = flag.String("db-name", "ocx_protocol", "Database name")
		httpPort   = flag.Int("http-port", 8080, "HTTP server port")
		consensusMode = flag.String("consensus", "standalone", "Consensus mode: standalone, tendermint")
	)
	flag.Parse()

	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("[OCX-SERVER] ")

	// Initialize database connection
	dbConfig := &database.DatabaseConfig{
		Host:     *dbHost,
		Port:     *dbPort,
		User:     *dbUser,
		Password: *dbPassword,
		DBName:   *dbName,
		SSLMode:  "disable",
		MaxConns: 25,
		MaxIdle:  5,
	}

	dbConn, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	// Initialize reputation engine
	reputationEngine := reputation.NewReputationEngine(dbConn.GetDB())

	// Initialize query engine
	queryEngine := query.NewQueryOptimizer(nil, nil) // Simplified for demo

	// Initialize consensus layer
	var consensusLayer *consensus.OCXStateMachine
	if *consensusMode == "tendermint" {
		consensusLayer = consensus.NewOCXStateMachine()
		log.Println("Tendermint consensus mode enabled")
	} else {
		log.Println("Standalone mode (no consensus)")
	}

	// Set up HTTP server
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := dbConn.HealthCheck(); err != nil {
			http.Error(w, "Database unhealthy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Query endpoint
	mux.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		queryStr := r.FormValue("query")
		if queryStr == "" {
			http.Error(w, "Query parameter required", http.StatusBadRequest)
			return
		}

		// Parse and execute query
		parser := query.NewParser(queryStr)
		parsedQuery, err := parser.Parse()
		if err != nil {
			http.Error(w, fmt.Sprintf("Query parse error: %v", err), http.StatusBadRequest)
			return
		}

		// Optimize query
		plan, err := queryEngine.OptimizeSelectQuery(context.Background(), parsedQuery)
		if err != nil {
			http.Error(w, fmt.Sprintf("Query optimization error: %v", err), http.StatusInternalServerError)
			return
		}

		// Execute query (simplified)
		result := map[string]interface{}{
			"query": queryStr,
			"plan":  plan,
			"status": "success",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// Reputation endpoint
	mux.HandleFunc("/reputation/", func(w http.ResponseWriter, r *http.Request) {
		providerID := r.URL.Path[len("/reputation/"):]
		if providerID == "" {
			http.Error(w, "Provider ID required", http.StatusBadRequest)
			return
		}

		score, err := reputationEngine.CalculateReputation(context.Background(), providerID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Reputation calculation error: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(score)
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
		log.Printf("Starting OCX server on port %d", *httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
