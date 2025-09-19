// working-server/main.go — Working Server on different port
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"ocx.local/pkg/ocx"
	"ocx.local/pkg/receipt"
	"ocx.local/store"
)

type WorkingServer struct {
	repo *store.Repository
}

func NewWorkingServer() (*WorkingServer, error) {
	log.Println("🔧 Starting database initialization...")
	start := time.Now()
	
	repo, err := store.NewRepository(":memory:")
	if err != nil {
		log.Printf("❌ Database initialization failed: %v", err)
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}
	
	elapsed := time.Since(start)
	log.Printf("✅ Database initialization completed in %v", elapsed)
	
	return &WorkingServer{repo: repo}, nil
}

func (s *WorkingServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	log.Printf("🏥 Health check requested from %s", r.RemoteAddr)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "OCX Working v1.0",
	})
}

func (s *WorkingServer) handleExecute(w http.ResponseWriter, r *http.Request) {
	log.Printf("⚡ Execute request from %s", r.RemoteAddr)
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Artifact   string `json:"artifact"`
		Input      string `json:"input"`
		MaxCycles  uint64 `json:"max_cycles"`
		LeaseID    string `json:"lease_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("📝 Processing execution: artifact=%s, input=%s, cycles=%d", 
		req.Artifact, req.Input, req.MaxCycles)

	// Mock OCX_EXEC implementation
	artifactHash := sha256.Sum256([]byte(req.Artifact))
	inputHash := sha256.Sum256([]byte(req.Input))
	
	// Create mock result
	result := &ocx.OCXResult{
		OutputHash:  sha256.Sum256([]byte("mock_output")),
		CyclesUsed:  req.MaxCycles,
		ReceiptHash: sha256.Sum256([]byte("mock_receipt")),
		ReceiptBlob: []byte("mock_receipt_blob"),
	}

	// Create receipt
	ocxReceipt := &ocx.OCXReceipt{
		Version:    ocx.OCX_VERSION,
		Artifact:   artifactHash,
		Input:      inputHash,
		Output:     result.OutputHash,
		Cycles:     result.CyclesUsed,
		Metering:   ocx.Metering{Alpha: 10, Beta: 1, Gamma: 1},
		Transcript: sha256.Sum256([]byte("mock_transcript")),
		Issuer:     [32]byte{1, 2, 3, 4, 5}, // mock issuer
		Signature:  [64]byte{1, 2, 3, 4, 5}, // mock signature
	}
	
	receipt := receipt.NewReceipt(ocxReceipt)

	// Serialize receipt
	receiptBlob, err := receipt.Serialize()
	if err != nil {
		log.Printf("❌ Failed to serialize receipt: %v", err)
		http.Error(w, "Failed to serialize receipt", http.StatusInternalServerError)
		return
	}

	// Store receipt
	receiptRecord := &store.Receipt{
		ReceiptHash:     result.ReceiptHash[:],
		ReceiptBody:     receiptBlob,
		LeaseID:         req.LeaseID,
		ArtifactHash:    artifactHash[:],
		InputHash:       inputHash[:],
		CyclesUsed:      int64(result.CyclesUsed),
		PriceMicroUnits: int64(result.CyclesUsed * 10), // 10 micro-units per cycle
		CreatedAt:       time.Now().Format(time.RFC3339),
	}

	log.Printf("💾 Storing receipt...")
	if err := s.repo.StoreReceipt(receiptRecord); err != nil {
		log.Printf("⚠️ Failed to store receipt: %v", err)
	} else {
		log.Printf("✅ Receipt stored successfully")
	}

	// Return response
	response := map[string]interface{}{
		"result":      result,
		"receipt":     receipt,
		"receipt_blob": receiptBlob,
		"status":      "success",
	}

	log.Printf("✅ Execution completed successfully")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *WorkingServer) handleVerify(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔍 Verify request from %s", r.RemoteAddr)
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ReceiptBlob string `json:"receipt_blob"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Invalid JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("🔍 Verifying receipt...")

	// Deserialize receipt
	receipt, err := receipt.Deserialize([]byte(req.ReceiptBlob))
	if err != nil {
		log.Printf("❌ Failed to deserialize receipt: %v", err)
		http.Error(w, "Failed to deserialize receipt", http.StatusBadRequest)
		return
	}

	// Verify receipt
	valid, reason := receipt.Verify()
	if !valid {
		log.Printf("❌ Receipt verification failed: %s", reason)
		http.Error(w, "Verification failed: "+reason, http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"valid":   valid,
		"status":  "success",
		"message": "Receipt verification completed",
	}

	log.Printf("✅ Receipt verification successful")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *WorkingServer) handleReceipts(w http.ResponseWriter, r *http.Request) {
	log.Printf("📋 Receipts request from %s", r.RemoteAddr)
	
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all receipts
	receipts, err := s.repo.QueryReceipts(store.ReceiptQuery{})
	if err != nil {
		log.Printf("❌ Failed to query receipts: %v", err)
		http.Error(w, "Failed to query receipts", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"receipts": receipts,
		"count":    len(receipts),
		"status":   "success",
	}

	log.Printf("✅ Listed %d receipts", len(receipts))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	log.Println("🚀 Starting OCX Working Server...")
	
	server, err := NewWorkingServer()
	if err != nil {
		log.Fatalf("❌ Failed to create server: %v", err)
	}

	mux := http.NewServeMux()
	
	// Core endpoints
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/api/v1/execute", server.handleExecute)
	mux.HandleFunc("/api/v1/verify", server.handleVerify)
	mux.HandleFunc("/api/v1/receipts", server.handleReceipts)

	// API documentation
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📖 API docs requested from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":    "OCX Working Server",
			"version": "v1.0",
			"endpoints": map[string]string{
				"GET /health":         "Health check",
				"POST /api/v1/execute": "Execute code with receipt generation",
				"POST /api/v1/verify": "Verify cryptographic receipts",
				"GET /api/v1/receipts": "List all receipts",
			},
		})
	})

	port := "8082"
	log.Printf("🌐 Starting server on port %s", port)
	log.Printf("🏥 Health check: http://localhost:%s/health", port)
	log.Printf("📖 API docs: http://localhost:%s/", port)
	
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}

