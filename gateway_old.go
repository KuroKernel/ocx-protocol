// gateway.go — OCX HTTP Gateway with Matching Engine
// go 1.22+

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Gateway struct {
	keyManager     *KeyManager
	matchingEngine *MatchingEngine
	offers         map[ID]*Offer
	orders         map[ID]*Order
}

func NewGateway() *Gateway {
	return &Gateway{
		keyManager:     NewKeyManager(),
		matchingEngine: NewMatchingEngine(),
		offers:         make(map[ID]*Offer),
		orders:         make(map[ID]*Order),
	}
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

	// Verify the envelope signature
	if err := g.keyManager.VerifyEnvelope(&envelope); err != nil {
		http.Error(w, fmt.Sprintf("Signature verification failed: %v", err), http.StatusUnauthorized)
		return
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

	// Add offer to matching engine
	if err := g.matchingEngine.AddOffer(&offer); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add offer: %v", err), http.StatusBadRequest)
		return
	}

	// Store the offer
	g.offers[offer.OfferID] = &offer

	// Return success response
	response := map[string]interface{}{
		"status":   "success",
		"offer_id": offer.OfferID,
		"message":  "Offer published successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// POST /orders - Place an order
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

	// Verify the envelope signature
	if err := g.keyManager.VerifyEnvelope(&envelope); err != nil {
		http.Error(w, fmt.Sprintf("Signature verification failed: %v", err), http.StatusUnauthorized)
		return
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

	// Process order through matching engine
	result, err := g.matchingEngine.ProcessOrder(&order)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to process order: %v", err), http.StatusInternalServerError)
		return
	}

	// Store the order
	g.orders[order.OrderID] = &order

	// Return matching result
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// GET /offers - List all offers
func (g *Gateway) HandleListOffers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	offers := make([]*Offer, 0, len(g.offers))
	for _, offer := range g.offers {
		offers = append(offers, offer)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(offers)
}

// GET /orders - List all orders
func (g *Gateway) HandleListOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orders := make([]*Order, 0, len(g.orders))
	for _, order := range g.orders {
		orders = append(orders, order)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// GET /leases - List all leases
func (g *Gateway) HandleListLeases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	leases := g.matchingEngine.GetActiveLeases()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(leases)
}

// GET /stats - Get market statistics
func (g *Gateway) HandleMarketStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := g.matchingEngine.GetMarketStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// StartServer starts the HTTP server
func (g *Gateway) StartServer(port string) {
	mux := http.NewServeMux()
	
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
		case http.MethodGet:
			g.HandleListOrders(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/leases", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			g.HandleListLeases(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			g.HandleMarketStats(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("Starting OCX Gateway with Matching Engine on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
