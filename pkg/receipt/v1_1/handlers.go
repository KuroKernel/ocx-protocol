package v1_1

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ReceiptHandlers provides HTTP handlers for the receipt v1.1 API
type ReceiptHandlers struct {
	manager *ReceiptManager
}

// NewReceiptHandlers creates new receipt handlers
func NewReceiptHandlers(manager *ReceiptManager) *ReceiptHandlers {
	return &ReceiptHandlers{manager: manager}
}

// CreateReceiptRequest represents a request to create a receipt
type CreateReceiptRequest struct {
	ProgramHash string            `json:"program_hash"` // Hex-encoded
	InputHash   string            `json:"input_hash"`   // Hex-encoded
	OutputHash  string            `json:"output_hash"`  // Hex-encoded
	GasUsed     uint64            `json:"gas_used"`
	StartedAt   int64             `json:"started_at"`  // Unix timestamp in nanoseconds
	FinishedAt  int64             `json:"finished_at"` // Unix timestamp in nanoseconds
	IssuerID    string            `json:"issuer_id"`
	KeyID       string            `json:"key_id"`
	KeyVersion  uint32            `json:"key_version"`
	HostCycles  uint64            `json:"host_cycles"`
	HostInfo    map[string]string `json:"host_info"`
}

// CreateReceiptResponse represents a response from creating a receipt
type CreateReceiptResponse struct {
	ReceiptID  string `json:"receipt_id"`
	ReceiptB64 string `json:"receipt_b64"` // Base64-encoded CBOR
	PublicKey  string `json:"public_key"`  // Hex-encoded public key
	KeyVersion uint32 `json:"key_version"`
	IssuedAt   int64  `json:"issued_at"` // Unix timestamp in nanoseconds
	Nonce      string `json:"nonce"`     // Hex-encoded nonce
}

// VerifyReceiptRequest represents a request to verify a receipt
type VerifyReceiptRequest struct {
	ReceiptB64 string `json:"receipt_b64"` // Base64-encoded CBOR
	KeyID      string `json:"key_id"`
	KeyVersion uint32 `json:"key_version"`
}

// VerifyReceiptResponse represents a response from verifying a receipt
type VerifyReceiptResponse struct {
	Verified     bool              `json:"verified"`
	Verification *VerificationInfo `json:"verification"`
	ReceiptID    string            `json:"receipt_id"`
	IssuedAt     int64             `json:"issued_at"`
}

// HandleCreateReceipt handles receipt creation requests
func (rh *ReceiptHandlers) HandleCreateReceipt(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req CreateReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ProgramHash == "" || req.InputHash == "" || req.OutputHash == "" {
		http.Error(w, "Missing required hash fields", http.StatusBadRequest)
		return
	}

	if req.IssuerID == "" || req.KeyID == "" {
		http.Error(w, "Missing issuer_id or key_id", http.StatusBadRequest)
		return
	}

	// Decode hex hashes
	programHash, err := hex.DecodeString(req.ProgramHash)
	if err != nil || len(programHash) != 32 {
		http.Error(w, "Invalid program_hash format", http.StatusBadRequest)
		return
	}

	inputHash, err := hex.DecodeString(req.InputHash)
	if err != nil || len(inputHash) != 32 {
		http.Error(w, "Invalid input_hash format", http.StatusBadRequest)
		return
	}

	outputHash, err := hex.DecodeString(req.OutputHash)
	if err != nil || len(outputHash) != 32 {
		http.Error(w, "Invalid output_hash format", http.StatusBadRequest)
		return
	}

	// Convert to fixed-size arrays
	var programHashArray, inputHashArray, outputHashArray [32]byte
	copy(programHashArray[:], programHash)
	copy(inputHashArray[:], inputHash)
	copy(outputHashArray[:], outputHash)

	// Parse timestamps
	startedAt := time.Unix(0, req.StartedAt)
	finishedAt := time.Unix(0, req.FinishedAt)

	// Create the receipt
	receipt, err := rh.manager.CreateReceipt(
		ctx,
		programHashArray, inputHashArray, outputHashArray,
		req.GasUsed, startedAt, finishedAt,
		req.IssuerID, req.KeyID, req.KeyVersion,
		req.HostCycles, req.HostInfo,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create receipt: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the public key for response
	provider, err := rh.manager.GetKMSManager().GetProvider("local")
	if err != nil {
		http.Error(w, "Failed to get KMS provider", http.StatusInternalServerError)
		return
	}

	publicKey, err := provider.GetPublicKey(ctx, req.KeyID, req.KeyVersion)
	if err != nil {
		http.Error(w, "Failed to get public key", http.StatusInternalServerError)
		return
	}

	// Encode receipt to CBOR
	encoder, err := NewCanonicalEncoder()
	if err != nil {
		http.Error(w, "Failed to create encoder", http.StatusInternalServerError)
		return
	}

	receiptData, err := encoder.EncodeFull(receipt)
	if err != nil {
		http.Error(w, "Failed to encode receipt", http.StatusInternalServerError)
		return
	}

	// Create response
	response := CreateReceiptResponse{
		ReceiptID:  rh.manager.generateReceiptID(receipt),
		ReceiptB64: base64.StdEncoding.EncodeToString(receiptData),
		PublicKey:  hex.EncodeToString(publicKey),
		KeyVersion: req.KeyVersion,
		IssuedAt:   int64(receipt.Core.IssuedAt),
		Nonce:      hex.EncodeToString(receipt.Core.Nonce[:]),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleVerifyReceipt handles receipt verification requests
func (rh *ReceiptHandlers) HandleVerifyReceipt(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req VerifyReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ReceiptB64 == "" || req.KeyID == "" {
		http.Error(w, "Missing receipt_b64 or key_id", http.StatusBadRequest)
		return
	}

	// Decode receipt
	receiptData, err := base64.StdEncoding.DecodeString(req.ReceiptB64)
	if err != nil {
		http.Error(w, "Invalid base64 receipt data", http.StatusBadRequest)
		return
	}

	// Parse CBOR receipt
	encoder, err := NewCanonicalEncoder()
	if err != nil {
		http.Error(w, "Failed to create encoder", http.StatusInternalServerError)
		return
	}

	receipt, err := encoder.DecodeFull(receiptData)
	if err != nil {
		http.Error(w, "Failed to decode receipt", http.StatusBadRequest)
		return
	}

	// Verify the receipt
	verification, err := rh.manager.VerifyReceipt(ctx, receipt, req.KeyID, req.KeyVersion)
	if err != nil {
		// Return verification failure details
		response := VerifyReceiptResponse{
			Verified:     false,
			Verification: verification,
			ReceiptID:    rh.manager.generateReceiptID(receipt),
			IssuedAt:     int64(receipt.Core.IssuedAt),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // 200 OK even for verification failures
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create successful response
	response := VerifyReceiptResponse{
		Verified:     true,
		Verification: verification,
		ReceiptID:    rh.manager.generateReceiptID(receipt),
		IssuedAt:     int64(receipt.Core.IssuedAt),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetReceipt handles requests to retrieve a receipt by ID
func (rh *ReceiptHandlers) HandleGetReceipt(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract receipt ID from URL path
	receiptID := r.URL.Query().Get("id")
	if receiptID == "" {
		http.Error(w, "Missing receipt ID", http.StatusBadRequest)
		return
	}

	// Get the receipt
	receipt, err := rh.manager.GetReceipt(ctx, receiptID)
	if err != nil {
		http.Error(w, "Receipt not found", http.StatusNotFound)
		return
	}

	// Encode receipt to CBOR
	encoder, err := NewCanonicalEncoder()
	if err != nil {
		http.Error(w, "Failed to create encoder", http.StatusInternalServerError)
		return
	}

	receiptData, err := encoder.EncodeFull(receipt)
	if err != nil {
		http.Error(w, "Failed to encode receipt", http.StatusInternalServerError)
		return
	}

	// Return the receipt
	w.Header().Set("Content-Type", "application/cbor")
	w.Write(receiptData)
}

// HandleGetDashboard handles requests for the dashboard
func (rh *ReceiptHandlers) HandleGetDashboard(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")

	if format == "html" {
		rh.manager.GetDashboard().ServeHTML(w, r)
	} else {
		rh.manager.GetDashboard().ServeHTTP(w, r)
	}
}

// HandleExportSIEM handles SIEM export requests
func (rh *ReceiptHandlers) HandleExportSIEM(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	format := r.URL.Query().Get("format")
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")

	// Default to last 24 hours if not specified
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = t
		}
	}

	if endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = t
		}
	}

	// Set appropriate content type and filename
	switch format {
	case "splunk":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=ocx_receipts_splunk_%s.jsonl", time.Now().Format("20060102_150405")))
		err := rh.manager.GetSIEMExporter().ExportSplunkHEC(ctx, w, startTime, endTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Export failed: %v", err), http.StatusInternalServerError)
			return
		}
	case "audit":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=ocx_audit_%s.jsonl", time.Now().Format("20060102_150405")))
		err := rh.manager.GetSIEMExporter().ExportAuditLog(ctx, w, startTime, endTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Export failed: %v", err), http.StatusInternalServerError)
			return
		}
	default: // jsonl
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=ocx_receipts_%s.jsonl", time.Now().Format("20060102_150405")))
		err := rh.manager.GetSIEMExporter().ExportJSONL(ctx, w, startTime, endTime)
		if err != nil {
			http.Error(w, fmt.Sprintf("Export failed: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

// HandleGetStats handles requests for system statistics
func (rh *ReceiptHandlers) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get export stats
	exportStats, err := rh.manager.GetSIEMExporter().GetExportStats(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get export stats: %v", err), http.StatusInternalServerError)
		return
	}

	// Get replay protection stats
	replayStats, err := rh.manager.GetReplayProtection().GetStats(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get replay stats: %v", err), http.StatusInternalServerError)
		return
	}

	// Get dashboard stats
	dashboardStats := rh.manager.GetDashboard().GetStats()

	// Combine all stats
	combinedStats := map[string]interface{}{
		"export_stats":    exportStats,
		"replay_stats":    replayStats,
		"dashboard_stats": dashboardStats,
		"timestamp":       time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(combinedStats)
}

// RegisterRoutes registers all receipt v1.1 routes
func (rh *ReceiptHandlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1.1/receipts/create", rh.HandleCreateReceipt)
	mux.HandleFunc("/api/v1.1/receipts/verify", rh.HandleVerifyReceipt)
	mux.HandleFunc("/api/v1.1/receipts/get", rh.HandleGetReceipt)
	mux.HandleFunc("/api/v1.1/dashboard", rh.HandleGetDashboard)
	mux.HandleFunc("/api/v1.1/export/siem", rh.HandleExportSIEM)
	mux.HandleFunc("/api/v1.1/stats", rh.HandleGetStats)
}
