// gateway.go — OCX HTTP Gateway with Persistence
// Enhanced version with SQLite storage

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"ocx.local/store"
	"ocx.local/pkg/ocx"
	"ocx.local/pkg/receipt"
	"ocx.local/pkg/verify"
	"ocx.local/internal/api"
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

// HandleLeaseState handles lease state updates
func (g *Gateway) HandleLeaseState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract lease ID from URL path
	path := r.URL.Path
	if !strings.HasPrefix(path, "/leases/") {
		http.Error(w, "Invalid lease ID", http.StatusBadRequest)
		return
	}
	leaseID := strings.TrimPrefix(path, "/leases/")
	if strings.Contains(leaseID, "/") {
		leaseID = strings.Split(leaseID, "/")[0]
	}

	var request struct {
		State string `json:"state"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Update lease state in database
	err := g.repo.UpdateLeaseState(leaseID, request.State)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update lease state: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"lease_id": leaseID,
		"state": request.State,
		"message": "Lease state updated successfully",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// =============================================================================
// PHASE 3: NEW PRODUCTION ENDPOINTS
// =============================================================================

// POST /api/v1/execute - Deterministic execution with receipt generation
func (g *Gateway) HandleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		LeaseID    string `json:"lease_id"`
		Artifact   []byte `json:"artifact"`
		Input      []byte `json:"input"`
		MaxCycles  uint64 `json:"max_cycles"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate lease exists and is active
	lease, err := g.repo.GetLeaseByID(request.LeaseID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Lease not found: %v", err), http.StatusNotFound)
		return
	}

	if lease.State != "running" {
		http.Error(w, "Lease is not in running state", http.StatusForbidden)
		return
	}

	// Calculate hashes
	artifactHash := sha256.Sum256(request.Artifact)
	inputHash := sha256.Sum256(request.Input)

	// Execute using OCX protocol (mock implementation for now)
	// TODO: Replace with actual OCX_EXEC call
	result := &ocx.OCXResult{
		OutputHash:  sha256.Sum256([]byte("mock_output")),
		CyclesUsed:  100,
		ReceiptHash: sha256.Sum256([]byte("mock_receipt")),
		ReceiptBlob: []byte("mock_receipt_blob"),
	}

	// Create receipt
	receiptData := &ocx.OCXReceipt{
		Version:    ocx.OCX_VERSION,
		Artifact:   artifactHash,
		Input:      inputHash,
		Output:     result.OutputHash,
		Cycles:     result.CyclesUsed,
		Metering: ocx.Metering{
			Alpha: ocx.ALPHA_COST_PER_CYCLE,
			Beta:  ocx.BETA_COST_PER_IO_BYTE,
			Gamma: ocx.GAMMA_COST_PER_MEMORY_PAGE,
		},
		Transcript: sha256.Sum256([]byte("execution_transcript")),
		Issuer:     [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		Signature:  [64]byte{},
	}

	// Sign receipt (in production, use proper key management)
	receiptWrapper := receipt.NewReceipt(receiptData)
	// For demo, we'll skip actual signing
	receiptWrapper.Signature = [64]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64}

	// Serialize receipt
	receiptBlob, err := receiptWrapper.Serialize()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to serialize receipt: %v", err), http.StatusInternalServerError)
		return
	}

	// Store receipt in database
	receiptHash := receiptWrapper.Hash()
	dbReceipt := &store.Receipt{
		ReceiptHash:     receiptHash[:],
		ReceiptBody:     receiptBlob,
		LeaseID:         request.LeaseID,
		ArtifactHash:    artifactHash[:],
		InputHash:       inputHash[:],
		CyclesUsed:      int64(result.CyclesUsed),
		PriceMicroUnits: int64(result.CyclesUsed * ocx.ALPHA_COST_PER_CYCLE),
		CreatedAt:       time.Now().Format(time.RFC3339),
	}

	if err := g.repo.StoreReceipt(dbReceipt); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store receipt: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	response := map[string]interface{}{
		"success":       true,
		"receipt_hash":  fmt.Sprintf("%x", receiptHash),
		"cycles_used":   result.CyclesUsed,
		"price_micro_units": dbReceipt.PriceMicroUnits,
		"receipt_blob":  fmt.Sprintf("%x", receiptBlob),
		"verification_command": fmt.Sprintf("ocx verify receipt_%x.cbor", receiptHash[:8]),
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// POST /api/v1/verify - Receipt verification service
func (g *Gateway) HandleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		ReceiptBlob string `json:"receipt_blob"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Decode hex receipt blob
	receiptBytes, err := hex.DecodeString(request.ReceiptBlob)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid receipt format: %v", err), http.StatusBadRequest)
		return
	}

	// Use unified verifier (Go or Rust based on environment)
	valid, err := verify.VerifyReceiptUnified(receiptBytes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Verification failed: %v", err), http.StatusBadRequest)
		return
	}

	// For response, we still need to deserialize to get receipt details
	receiptWrapper, err := receipt.Deserialize(receiptBytes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to deserialize receipt: %v", err), http.StatusBadRequest)
		return
	}

	reason := "invalid"
	if valid {
		reason = "valid"
	}
	
	response := map[string]interface{}{
		"valid":   valid,
		"reason":  reason,
		"receipt_hash": fmt.Sprintf("%x", receiptWrapper.Hash()),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if valid {
		// Extract accounting information
		payer, payee, amount := receiptWrapper.ExtractAccounting()
		response["accounting"] = map[string]interface{}{
			"payer":  payer,
			"payee":  payee,
			"amount": amount,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if valid {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	json.NewEncoder(w).Encode(response)
}

// GET /api/v1/receipts - Query receipts with filtering
func (g *Gateway) HandleQueryReceipts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := store.ReceiptQuery{}
	
	if leaseID := r.URL.Query().Get("lease_id"); leaseID != "" {
		query.LeaseID = leaseID
	}
	
	if artifactHash := r.URL.Query().Get("artifact_hash"); artifactHash != "" {
		// Decode hex artifact hash
		if hash, err := hex.DecodeString(artifactHash); err == nil {
			query.ArtifactHash = hash
		}
	}
	
	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		query.StartDate = startDate
	}
	
	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		query.EndDate = endDate
	}
	
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			query.Limit = limit
		}
	}
	
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			query.Offset = offset
		}
	}

	// Query receipts
	receipts, err := g.repo.QueryReceipts(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query receipts: %v", err), http.StatusInternalServerError)
		return
	}

	// Format response
	var responseReceipts []map[string]interface{}
	for _, receipt := range receipts {
		responseReceipts = append(responseReceipts, map[string]interface{}{
			"receipt_hash":      fmt.Sprintf("%x", receipt.ReceiptHash),
			"lease_id":          receipt.LeaseID,
			"artifact_hash":     fmt.Sprintf("%x", receipt.ArtifactHash),
			"input_hash":        fmt.Sprintf("%x", receipt.InputHash),
			"cycles_used":       receipt.CyclesUsed,
			"price_micro_units": receipt.PriceMicroUnits,
			"created_at":        receipt.CreatedAt,
			"verification_command": fmt.Sprintf("ocx verify receipt_%x.cbor", receipt.ReceiptHash[:8]),
		})
	}

	response := map[string]interface{}{
		"success": true,
		"count":   len(responseReceipts),
		"receipts": responseReceipts,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
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
	mux.HandleFunc("/leases/", g.HandleLeaseState)
	mux.HandleFunc("/market/stats", g.HandleMarketStats)
	mux.HandleFunc("/identities", g.HandleRegisterIdentity)
	
	// Phase 3: New production endpoints
	mux.HandleFunc("/api/v1/execute", g.HandleExecute)
	mux.HandleFunc("/api/v1/verify", g.HandleVerify)
	mux.HandleFunc("/api/v1/receipts", g.HandleQueryReceipts)
	
	// Phase 4: Advanced features endpoints
	advancedHandler := api.NewAdvancedAPIHandler(g.repo)
	advancedHandler.RegisterAdvancedRoutes(mux)

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
				"POST /api/v1/execute": "Execute code with receipt generation",
				"POST /api/v1/verify": "Verify cryptographic receipts",
				"GET /api/v1/receipts": "Query receipts with filtering",
				"GET /api/v2/enterprise/compliance": "Get compliance dashboard",
				"GET /api/v2/enterprise/sla": "Get SLA status",
				"GET /api/v2/enterprise/tenants": "List all tenants",
				"GET /api/v2/enterprise/audit": "Get audit trail",
				"GET /api/v2/financial/futures": "List compute futures",
				"POST /api/v2/financial/futures": "Create compute future",
				"GET /api/v2/financial/bonds": "List compute bonds",
				"GET /api/v2/financial/carbon-credits": "List carbon credits",
				"GET /api/v2/financial/market-status": "Get market status",
				"POST /api/v2/ai/inference": "Execute AI inference",
				"POST /api/v2/ai/training": "Execute AI training",
				"GET /api/v2/ai/models": "List AI models",
				"POST /api/v2/ai/verify": "Verify AI computation",
				"POST /api/v2/global/execute": "Execute globally",
				"POST /api/v2/global/optimize": "Optimize planetary resources",
				"GET /api/v2/global/status": "Get global status",
				"GET /api/v2/global/metrics": "Get global metrics",
				"POST /api/v2/execute/advanced": "Execute with advanced features",
				"POST /api/v2/execute/batch": "Execute batch computation",
				"POST /api/v2/execute/stream": "Execute stream computation",
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
