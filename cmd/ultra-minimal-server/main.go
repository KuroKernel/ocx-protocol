// ultra-minimal-server/main.go — Ultra Minimal OCX Server
// No database dependencies, just core functionality

package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type UltraMinimalServer struct{}

func (s *UltraMinimalServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "OCX Ultra Minimal v1.0",
	})
}

func (s *UltraMinimalServer) handleExecute(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Mock OCX_EXEC implementation
	artifactHash := sha256.Sum256([]byte(req.Artifact))
	inputHash := sha256.Sum256([]byte(req.Input))
	outputHash := sha256.Sum256([]byte("mock_output"))
	receiptHash := sha256.Sum256([]byte("mock_receipt"))

	// Create mock result
	result := map[string]interface{}{
		"output_hash":  fmt.Sprintf("%x", outputHash),
		"cycles_used":  req.MaxCycles,
		"receipt_hash": fmt.Sprintf("%x", receiptHash),
		"receipt_blob": "mock_receipt_blob",
		"artifact_hash": fmt.Sprintf("%x", artifactHash),
		"input_hash": fmt.Sprintf("%x", inputHash),
		"status": "success",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *UltraMinimalServer) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ReceiptBlob string `json:"receipt_blob"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Mock verification
	valid := req.ReceiptBlob == "mock_receipt_blob"

	response := map[string]interface{}{
		"valid":   valid,
		"status":  "success",
		"message": "Receipt verification completed",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *UltraMinimalServer) handleReceipts(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Mock receipts
	receipts := []map[string]interface{}{
		{
			"receipt_hash": "abc123",
			"lease_id":     "lease-1",
			"cycles_used":  1000,
			"created_at":   time.Now().Format(time.RFC3339),
		},
		{
			"receipt_hash": "def456",
			"lease_id":     "lease-2",
			"cycles_used":  2000,
			"created_at":   time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		},
	}

	response := map[string]interface{}{
		"receipts": receipts,
		"count":    len(receipts),
		"status":   "success",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	server := &UltraMinimalServer{}

	mux := http.NewServeMux()
	
	// Core endpoints
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/api/v1/execute", server.handleExecute)
	mux.HandleFunc("/api/v1/verify", server.handleVerify)
	mux.HandleFunc("/api/v1/receipts", server.handleReceipts)

	// API documentation
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":    "OCX Ultra Minimal Server",
			"version": "v1.0",
			"endpoints": map[string]string{
				"GET /health":         "Health check",
				"POST /api/v1/execute": "Execute code with receipt generation",
				"POST /api/v1/verify": "Verify cryptographic receipts",
				"GET /api/v1/receipts": "List all receipts",
			},
		})
	})

	port := "8080"
	log.Printf("Starting OCX Ultra Minimal Server on port %s", port)
	log.Printf("Health check: http://localhost:%s/health", port)
	log.Printf("API docs: http://localhost:%s/", port)
	
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
