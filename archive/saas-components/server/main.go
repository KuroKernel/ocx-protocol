// main.go — OCX Protocol Server
// go 1.18+

package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"encoding/json"
	"fmt"
	"ocx.local/store"
)

// Gateway represents the HTTP gateway
type Gateway struct {
	repo *store.Repository
}

// NewGateway creates a new gateway instance
func NewGateway() *Gateway {
	// Initialize repository
	dbPath := store.GetDatabasePath()
	repo, err := store.NewRepository(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}

	return &Gateway{
		repo: repo,
	}
}

// StartServer starts the HTTP server
func (g *Gateway) StartServer(port string) {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
	})

	// API endpoints
	mux.HandleFunc("/api/v1/offers", g.handleOffers)
	mux.HandleFunc("/api/v1/orders", g.handleOrders)
	mux.HandleFunc("/api/v1/leases", g.handleLeases)
	mux.HandleFunc("/api/v1/parties", g.handleParties)

	// Start server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		server.Close()
		g.repo.Close()
		os.Exit(0)
	}()

	log.Printf("Starting OCX Protocol server on port %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// handleOffers handles offer-related requests
func (g *Gateway) handleOffers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		g.getOffers(w, r)
	case http.MethodPost:
		g.createOffer(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleOrders handles order-related requests
func (g *Gateway) handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		g.getOrders(w, r)
	case http.MethodPost:
		g.createOrder(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleLeases handles lease-related requests
func (g *Gateway) handleLeases(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		g.getLeases(w, r)
	case http.MethodPost:
		g.createLease(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleParties handles party-related requests
func (g *Gateway) handleParties(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		g.getParties(w, r)
	case http.MethodPost:
		g.createParty(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getOffers retrieves all offers
func (g *Gateway) getOffers(w http.ResponseWriter, r *http.Request) {
	offers, err := g.repo.GetOffers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(offers)
}

// createOffer creates a new offer
func (g *Gateway) createOffer(w http.ResponseWriter, r *http.Request) {
	var offer map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	validation := validateOffer(offer)
	if !validation.Valid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(validation)
		return
	}
	// Generate offer ID
	offerID := fmt.Sprintf("%d", time.Now().UnixNano())
	offer["offer_id"] = offerID
	offer["created_at"] = time.Now().Format(time.RFC3339)
	
	// Parse time values
	validFrom, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	validTo, _ := time.Parse(time.RFC3339, time.Now().Add(24 * time.Hour).Format(time.RFC3339))
	
	// Store in database
	if err := g.repo.CreateOffer(offerID, offer["provider_id"].(string), offer["fleet_id"].(string), offer["unit"].(string), offer["unit_price_amount"].(string), offer["unit_price_currency"].(string), 2, int(offer["min_hours"].(float64)), int(offer["max_hours"].(float64)), int(offer["min_gpus"].(float64)), int(offer["max_gpus"].(float64)), validFrom, validTo); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// getOrders retrieves all orders
func (g *Gateway) getOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := g.repo.GetOrders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(orders)
}

// createOrder creates a new order
func (g *Gateway) createOrder(w http.ResponseWriter, r *http.Request) {
	var order map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// TODO: Validate order structure
	// TODO: Store order in database

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// getLeases retrieves all leases
func (g *Gateway) getLeases(w http.ResponseWriter, r *http.Request) {
	leases, err := g.repo.GetLeases()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(leases)
}

// createLease creates a new lease
func (g *Gateway) createLease(w http.ResponseWriter, r *http.Request) {
	var lease map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&lease); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// TODO: Validate lease structure
	// TODO: Store lease in database

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// getParties retrieves all parties
func (g *Gateway) getParties(w http.ResponseWriter, r *http.Request) {
	parties, err := g.repo.GetParties()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(parties)
}

// createParty creates a new party
func (g *Gateway) createParty(w http.ResponseWriter, r *http.Request) {
	var party map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&party); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// TODO: Validate party structure
	// TODO: Store party in database

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func main() {
	var port = flag.String("port", "8080", "Port to listen on")
	flag.Parse()

	gateway := NewGateway()
	gateway.StartServer(*port)
}
