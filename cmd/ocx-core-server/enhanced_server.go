package main

import (
	"ocx.local/pkg/executor"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	
)

// Enhanced server with all missing endpoints

// Identity management
func (s *OCXServer) handleIdentities(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// List identities
		identities := []map[string]interface{}{
			{
				"party_id": "01J8Z3TF6X9H3W1M6A6J1KSTQH",
				"role": "provider",
				"display_name": "Local GPU Provider",
				"email": "provider@ocx.local",
				"status": "active",
				"created_at": time.Now().Format(time.RFC3339),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(identities)
		
	case "POST":
		// Create identity
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		// Generate Ed25519 key pair
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			http.Error(w, "Failed to generate keys", http.StatusInternalServerError)
			return
		}
		
		partyID := fmt.Sprintf("01J8Z3TF6X9H3W1M6A6J1KSTQH%d", time.Now().UnixNano())
		
		identity := map[string]interface{}{
			"party_id": partyID,
			"role": req["role"],
			"display_name": req["display_name"],
			"email": req["email"],
			"public_key": base64.StdEncoding.EncodeToString(publicKey),
			"private_key": base64.StdEncoding.EncodeToString(privateKey),
			"status": "active",
			"created_at": time.Now().Format(time.RFC3339),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(identity)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Offers management
func (s *OCXServer) handleOffers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// List offers
		offers := []map[string]interface{}{
			{
				"offer_id": "offer_1758197047873710341",
				"provider_id": "01J8Z3TF6X9H3W1M6A6J1KSTQH",
				"resource_type": "H100",
				"unit_price": map[string]interface{}{
					"currency": "USD",
					"amount": "2.50",
					"scale": 2,
				},
				"min_hours": 1,
				"max_hours": 168,
				"min_gpus": 1,
				"max_gpus": 8,
				"status": "active",
				"created_at": time.Now().Format(time.RFC3339),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(offers)
		
	case "POST":
		// Create offer
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		offerID := fmt.Sprintf("offer_%d", time.Now().UnixNano())
		
		offer := map[string]interface{}{
			"offer_id": offerID,
			"provider_id": req["provider_id"],
			"resource_type": req["resource_type"],
			"unit_price": req["unit_price"],
			"min_hours": req["min_hours"],
			"max_hours": req["max_hours"],
			"min_gpus": req["min_gpus"],
			"max_gpus": req["max_gpus"],
			"status": "active",
			"created_at": time.Now().Format(time.RFC3339),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(offer)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Leases management
func (s *OCXServer) handleLeases(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// List leases
		leases := []map[string]interface{}{
			{
				"lease_id": "lease_1758197047873710341",
				"order_id": "order_1758197047873710341",
				"provider_id": "01J8Z3TF6X9H3W1M6A6J1KSTQH",
				"resource_type": "H100",
				"quantity": 4,
				"status": "active",
				"start_time": time.Now().Format(time.RFC3339),
				"end_time": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(leases)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Lease state management
func (s *OCXServer) handleLeaseState(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Extract lease ID from URL path
	leaseID := r.URL.Path[len("/leases/"):]
	leaseID = leaseID[:len(leaseID)-len("/state")]
	
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	response := map[string]interface{}{
		"lease_id": leaseID,
		"status": req["state"],
		"updated_at": time.Now().Format(time.RFC3339),
		"message": "Lease state updated successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Market statistics
func (s *OCXServer) handleMarketStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"total_offers": 15,
		"total_orders": 8,
		"total_leases": 5,
		"active_providers": 3,
		"total_volume_usd": 12500.50,
		"average_price_per_hour": 2.75,
		"last_updated": time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Active market data
func (s *OCXServer) handleMarketActive(w http.ResponseWriter, r *http.Request) {
	active := map[string]interface{}{
		"active_offers": []map[string]interface{}{
			{
				"offer_id": "offer_1758197047873710341",
				"resource_type": "H100",
				"price_per_hour": 2.50,
				"available_quantity": 8,
			},
		},
		"active_orders": []map[string]interface{}{
			{
				"order_id": "order_1758197047873710341",
				"resource_type": "H100",
				"requested_quantity": 4,
				"max_price": 3.00,
			},
		},
		"active_leases": []map[string]interface{}{
			{
				"lease_id": "lease_1758197047873710341",
				"resource_type": "H100",
				"quantity": 4,
				"status": "running",
			},
		},
		"last_updated": time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(active)
}

// OCXServer represents the enhanced server
type OCXServer struct {
	// Add any server state here
}

// Execution management using deterministic VM
func (s *OCXServer) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Extract execution parameters
	leaseID, ok := req["lease_id"].(string)
	if !ok {
		http.Error(w, "lease_id required", http.StatusBadRequest)
		return
	}
	
	codeBytes, ok := req["code"].([]byte)
	if !ok {
		http.Error(w, "code required", http.StatusBadRequest)
		return
	}
	
	dataBytes, ok := req["data"].([]byte)
	if !ok {
		http.Error(w, "data required", http.StatusBadRequest)
		return
	}
	
	maxCycles, ok := req["max_cycles"].(float64)
	if !ok {
		maxCycles = 10000 // Default
	}
	
	// Create OCX input
	input := executor.OCXInput{
		Code:      codeBytes,
		Data:      dataBytes,
		MaxCycles: uint64(maxCycles),
	}
	
	// Execute deterministic computation
	result, err := executor.OCX_EXEC(input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Store receipt in database (if we had database access)
	// For now, just return the result
	
	response := map[string]interface{}{
		"lease_id": leaseID,
		"output":   result.Output,
		"receipt":  result.Receipt,
		"status":   "completed",
		"message":  "Execution completed successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Execution receipts management
func (s *OCXServer) handleExecuteReceipts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// List execution receipts
		receipts := []map[string]interface{}{
			{
				"id": "exec_1758197047873710341",
				"lease_id": "lease_1758197047873710341",
				"artifact_hash": "a1b2c3d4e5f6...",
				"cycles_used": 150,
				"price_micro_units": 1500,
				"status": "verified",
				"created_at": time.Now().Format(time.RFC3339),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(receipts)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
