// gateway.go — OCX HTTP Gateway with Persistence
// Enhanced version with SQLite storage

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"ocx.local/store"
)

type Gateway struct {
	keyManager     *KeyManager
	matchingEngine *MatchingEngine
	repo           *store.Repository
	feesBPS        int
	requireSig     bool
}

func NewGateway() *Gateway {
	// Get configuration from environment
	feesBPS := 50 // Default 0.5%
	if envFees := os.Getenv("OCX_FEES_BPS"); envFees != "" {
		if parsed, err := strconv.Atoi(envFees); err == nil {
			feesBPS = parsed
		}
	}

	requireSig := false
	if os.Getenv("OCX_REQUIRE_SIG") == "true" {
		requireSig = true
	}

	// Initialize repository
	dbPath := store.GetDatabasePath()
	repo, err := store.NewRepository(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}

	return &Gateway{
		keyManager:     NewKeyManager(),
		matchingEngine: NewMatchingEngine(),
		repo:           repo,
		feesBPS:        feesBPS,
		requireSig:     requireSig,
	}
}

// POST /identities - Register a new identity
func (g *Gateway) HandleRegisterIdentity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Role        string `json:"role"`
		DisplayName string `json:"display_name"`
		Email       string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Create identity in key manager
	identity, err := g.keyManager.CreateIdentity(request.Role, request.DisplayName, request.Email)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create identity: %v", err), http.StatusInternalServerError)
		return
	}

	// Store in database
	err = g.repo.CreateIdentity(
		identity.PartyID,
		identity.Role,
		identity.DisplayName,
		identity.Email,
		identity.Keys[0].KeyID,
		identity.Keys[0].PublicKey,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to store identity: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the identity (excluding private keys)
	response := map[string]interface{}{
		"party_id":     identity.PartyID,
		"role":         identity.Role,
		"display_name": identity.DisplayName,
		"email":        identity.Email,
		"active":       identity.Active,
		"created_at":   identity.CreatedAt.Format(time.RFC3339),
		"public_key":   identity.Keys[0].PublicKey,
		"key_id":       identity.Keys[0].KeyID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// POST /offers - Publish an offer
func (g *Gateway) HandlePublishOffer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var envelope Envelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Verify signature if required
	if g.requireSig && envelope.Sig.KeyID != "" {
		if err := g.keyManager.VerifyEnvelope(&envelope); err != nil {
			http.Error(w, fmt.Sprintf("Signature verification failed: %v", err), http.StatusUnauthorized)
			return
		}
	}

	// Extract the offer from the envelope
	offerData, err := json.Marshal(envelope.Payload)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal offer: %v", err), http.StatusInternalServerError)
		return
	}

	var offer Offer
	if err := json.Unmarshal(offerData, &offer); err != nil {
		http.Error(w, fmt.Sprintf("Invalid offer format: %v", err), http.StatusBadRequest)
		return
	}

	// Add to matching engine
	if err := g.matchingEngine.AddOffer(&offer); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add offer: %v", err), http.StatusBadRequest)
		return
	}

	// Store in database
	err = g.repo.CreateOffer(
		offer.OfferID,
		offer.Provider.PartyID,
		offer.FleetID,
		string(offer.Unit),
		offer.UnitPrice.Amount,
		offer.UnitPrice.Currency,
		offer.UnitPrice.Scale,
		offer.MinHours,
		offer.MaxHours,
		offer.MinGPUs,
		offer.MaxGPUs,
		offer.ValidFrom,
		offer.ValidTo,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to store offer: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"status":   "success",
		"offer_id": offer.OfferID,
		"message":  "Offer published successfully",
		"valid_until": offer.ValidTo.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// POST /orders - Place an order (with automatic matching)
func (g *Gateway) HandlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var envelope Envelope
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Verify signature if required
	if g.requireSig && envelope.Sig.KeyID != "" {
		if err := g.keyManager.VerifyEnvelope(&envelope); err != nil {
			http.Error(w, fmt.Sprintf("Signature verification failed: %v", err), http.StatusUnauthorized)
			return
		}
	}

	// Extract the order from the envelope
	orderData, err := json.Marshal(envelope.Payload)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal order: %v", err), http.StatusInternalServerError)
		return
	}

	var order Order
	if err := json.Unmarshal(orderData, &order); err != nil {
		http.Error(w, fmt.Sprintf("Invalid order format: %v", err), http.StatusBadRequest)
		return
	}

	// Store order in database
	budgetAmount := ""
	budgetCurrency := ""
	budgetScale := 0
	if order.BudgetCap != nil {
		budgetAmount = order.BudgetCap.Amount
		budgetCurrency = order.BudgetCap.Currency
		budgetScale = order.BudgetCap.Scale
	}

	err = g.repo.CreateOrder(
		order.OrderID,
		order.Buyer.PartyID,
		order.OfferID,
		order.RequestedGPUs,
		order.Hours,
		budgetAmount,
		budgetCurrency,
		budgetScale,
		string(order.State),
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to store order: %v", err), http.StatusInternalServerError)
		return
	}

	// Process order through matching engine
	result, err := g.matchingEngine.ProcessOrder(&order)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process order: %v", err), http.StatusInternalServerError)
		return
	}

	// If successful, store lease in database
	if result.Success {
		lease, exists := g.matchingEngine.GetLease(result.LeaseID)
		if exists {
			err = g.repo.CreateLease(
				lease.LeaseID,
				lease.OrderID,
				lease.FleetID,
				lease.AssignedGPUs,
				lease.StartAt,
				*lease.EndAt,
				string(lease.State),
			)
			if err != nil {
				log.Printf("Failed to store lease: %v", err)
			}
		}
	}

	// Add fee calculation
	if result.Success {
		// Calculate fee
		priceValue := g.costToFloat(result.Price)
		feeValue := priceValue * float64(g.feesBPS) / 10000.0
		
		fee := Money{
			Currency: result.Price.Currency,
			Amount:   fmt.Sprintf("%.2f", feeValue),
			Scale:    result.Price.Scale,
		}

		result.Fee = &fee
		result.PayTo = "ocx-clearing"
	}

	// Return matching result
	if result.Success {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
	}
	
	json.NewEncoder(w).Encode(result)
}

// GET /offers - List all offers
func (g *Gateway) HandleListOffers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	offers, err := g.repo.ListOffers()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list offers: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"offers": offers,
		"count":  len(offers),
	})
}

// GET /leases - List all leases
func (g *Gateway) HandleListLeases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	leases, err := g.repo.ListLeases()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list leases: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"leases": leases,
		"count":  len(leases),
	})
}

// GET /market/stats - Get market statistics
func (g *Gateway) HandleMarketStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := g.matchingEngine.GetMarketStats()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"stats":  stats,
		"fees_bps": g.feesBPS,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GET /ready - Readiness check
func (g *Gateway) HandleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check database connectivity
	if err := g.repo.Ping(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "not_ready",
			"error":  "database_unavailable",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ready",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// costToFloat converts Money to float64 for calculations
func (g *Gateway) costToFloat(money Money) float64 {
	var amount float64
	fmt.Sscanf(money.Amount, "%f", &amount)
	
	// Apply scale
	for i := 0; i < money.Scale; i++ {
		amount /= 10.0
	}
	
	return amount
}

// StartServer starts the enhanced HTTP server
func (g *Gateway) StartServer(port string) {
	mux := http.NewServeMux()
	
	// Core endpoints
	mux.HandleFunc("/offers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			g.HandlePublishOffer(w, r)
		case http.MethodGet:
			g.HandleListOffers(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			g.HandlePlaceOrder(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/leases", g.HandleListLeases)
	mux.HandleFunc("/market/stats", g.HandleMarketStats)
	mux.HandleFunc("/identities", g.HandleRegisterIdentity)

	// Health checks
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "OCX Protocol v0.1",
		})
	})

	mux.HandleFunc("/ready", g.HandleReady)

	// API documentation
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":    "OCX Protocol API",
			"version": "v0.1",
			"persistence": "sqlite",
			"fees_bps": g.feesBPS,
			"require_sig": g.requireSig,
			"endpoints": map[string]string{
				"POST /identities":    "Register new identity",
				"POST /offers":        "Publish an offer",
				"GET /offers":         "List all offers",
				"POST /orders":        "Place an order (auto-matches)",
				"GET /leases":         "List all leases",
				"GET /market/stats":   "Get market statistics",
				"GET /health":         "Health check",
				"GET /ready":          "Readiness check",
			},
		})
	})

	log.Printf("Starting OCX Gateway with SQLite persistence on port %s", port)
	log.Printf("Fees: %d bps (%.2f%%)", g.feesBPS, float64(g.feesBPS)/100.0)
	log.Printf("Require signatures: %v", g.requireSig)
	log.Printf("Database: %s", store.GetDatabasePath())
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
