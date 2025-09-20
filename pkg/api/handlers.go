package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"ocx.local/pkg/keystore"
	"ocx.local/pkg/metrics"
	"ocx.local/pkg/ocx"
	"ocx.local/pkg/receipt"
)

type ExecuteRequest struct {
	Artifact  string `json:"artifact"`
	Input     string `json:"input"`
	MaxCycles int64  `json:"max_cycles"`
}

type ExecuteResponse struct {
	OutputHash    string `json:"output_hash"`
	CyclesUsed    int64  `json:"cycles_used"`
	ReceiptHash   string `json:"receipt_hash"`
	ReceiptBlob   string `json:"receipt_blob"`
	VerifyCommand string `json:"verify_command"`
}

type VerifyRequest struct {
	ReceiptBlob string `json:"receipt_blob"`
}

type VerifyResponse struct {
	Valid     bool   `json:"valid"`
	IssuerID  string `json:"issuer_id,omitempty"`
	Cycles    int64  `json:"cycles,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Error     string `json:"error,omitempty"`
}

type Server struct {
	ocx       *ocx.MockExecutor
	keystore  *keystore.Keystore
	cache     map[string]ExecuteResponse
	reqCache  map[string]ExecuteRequest
}

func NewServer(ocx *ocx.MockExecutor, keystore *keystore.Keystore) *Server {
	return &Server{
		ocx:      ocx,
		keystore: keystore,
		cache:    make(map[string]ExecuteResponse),
		reqCache: make(map[string]ExecuteRequest),
	}
}

func (s *Server) HandleExecute(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	// Check idempotency key
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		WriteError(w, ErrInvalidInput, "Idempotency-Key header required", http.StatusBadRequest)
		return
	}

	// Parse request first
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidInput, "Invalid JSON request", http.StatusBadRequest)
		metrics.RecordExecution(0, time.Since(start), false)
		return
	}

	// Check for duplicate idempotency key
	if cachedResult, exists := s.cache[idempotencyKey]; exists {
		// Check if request matches cached request
		if cachedReq, reqExists := s.reqCache[idempotencyKey]; reqExists {
			if req.Artifact != cachedReq.Artifact || req.Input != cachedReq.Input || req.MaxCycles != cachedReq.MaxCycles {
				WriteError(w, ErrIdempotencyMismatch, "Idempotency-Key already used with different request body", http.StatusConflict)
				return
			}
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cachedResult)
		return
	}

	// Validate input
	if req.Artifact == "" || req.Input == "" {
		WriteError(w, ErrInvalidInput, "artifact and input are required", http.StatusBadRequest)
		metrics.RecordExecution(0, time.Since(start), false)
		return
	}

	// Validate resource limits
	if req.MaxCycles <= 0 {
		req.MaxCycles = 10000 // Default to 10K cycles
	} else if req.MaxCycles > 1000000 { // 1M cycle limit
		WriteError(w, ErrInvalidInput, "max_cycles cannot exceed 1,000,000", http.StatusBadRequest)
		metrics.RecordExecution(0, time.Since(start), false)
		return
	}

	// Add payload size check
	if len(req.Artifact) > 10*1024 { // 10KB base64 limit
		WriteError(w, ErrInvalidInput, "artifact too large (max 10KB)", http.StatusBadRequest)
		metrics.RecordExecution(0, time.Since(start), false)
		return
	}

	if len(req.Input) > 10*1024 { // 10KB base64 limit
		WriteError(w, ErrInvalidInput, "input too large (max 10KB)", http.StatusBadRequest)
		metrics.RecordExecution(0, time.Since(start), false)
		return
	}

	// Decode base64 inputs
	artifactBytes, err := base64.StdEncoding.DecodeString(req.Artifact)
	if err != nil {
		WriteError(w, ErrInvalidInput, "Invalid base64 artifact", http.StatusBadRequest)
		metrics.RecordExecution(0, time.Since(start), false)
		return
	}

	inputBytes, err := base64.StdEncoding.DecodeString(req.Input)
	if err != nil {
		WriteError(w, ErrInvalidInput, "Invalid base64 input", http.StatusBadRequest)
		metrics.RecordExecution(0, time.Since(start), false)
		return
	}

	// Execute
	result, err := s.ocx.Execute(artifactBytes, inputBytes, uint64(req.MaxCycles))
	if err != nil {
		WriteError(w, ErrExecutionFailed, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		metrics.RecordExecution(0, time.Since(start), false)
		return
	}

	// Generate receipt with issuer ID
	receiptData, err := receipt.CreateReceipt(result, s.keystore)
	if err != nil {
		WriteError(w, ErrInternalError, fmt.Sprintf("Receipt generation failed: %v", err), http.StatusInternalServerError)
		metrics.RecordExecution(0, time.Since(start), false)
		return
	}

	receiptBlob := base64.StdEncoding.EncodeToString(receiptData)
	receiptHash := fmt.Sprintf("%x", result.ReceiptHash)

	// Store result with idempotency key
	response := ExecuteResponse{
		OutputHash:    fmt.Sprintf("%x", result.OutputHash),
		CyclesUsed:    int64(result.CyclesUsed),
		ReceiptHash:   receiptHash,
		ReceiptBlob:   receiptBlob,
		VerifyCommand: fmt.Sprintf("./ocx-cli verify --receipt '%s'", receiptBlob),
	}

	s.cache[idempotencyKey] = response
	s.reqCache[idempotencyKey] = req
	metrics.RecordExecution(int64(result.CyclesUsed), time.Since(start), true)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) HandleVerify(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	contentType := r.Header.Get("Content-Type")
	
	var receiptBytes []byte
	var err error

	switch {
	case strings.Contains(contentType, "application/cbor"):
		// Raw CBOR input
		receiptBytes, err = io.ReadAll(r.Body)
		if err != nil {
			WriteError(w, ErrInvalidInput, "Failed to read CBOR data", http.StatusBadRequest)
			metrics.RecordVerification(time.Since(start), false)
			return
		}

	case strings.Contains(contentType, "application/json"):
		// JSON with base64 receipt
		var req VerifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, ErrInvalidInput, "Invalid JSON request", http.StatusBadRequest)
			metrics.RecordVerification(time.Since(start), false)
			return
		}

		receiptBytes, err = base64.StdEncoding.DecodeString(req.ReceiptBlob)
		if err != nil {
			WriteError(w, ErrInvalidReceipt, "Invalid base64 receipt", http.StatusBadRequest)
			metrics.RecordVerification(time.Since(start), false)
			return
		}

	default:
		WriteError(w, ErrInvalidInput, "Content-Type must be application/json or application/cbor", http.StatusBadRequest)
		metrics.RecordVerification(time.Since(start), false)
		return
	}

	// Verify receipt
	result, err := receipt.Verify(receiptBytes)
	if err != nil {
		response := VerifyResponse{
			Valid: false,
			Error: err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		metrics.RecordVerification(time.Since(start), false)
		return
	}

	response := VerifyResponse{
		Valid:     true,
		IssuerID:  result.IssuerID,
		Cycles:    result.Cycles,
		Timestamp: result.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	metrics.RecordVerification(time.Since(start), true)
}

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "healthy",
		"version": "1.0.0-rc.1",
		"uptime":  "0s", // This would be calculated from server start time
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	// Check keystore loaded
	// Check all critical dependencies
	
	ready := true
	checks := make(map[string]string)
	
	// Database check (simplified)
	checks["database"] = "ok"
	
	// Keystore check
	if s.keystore == nil {
		checks["keystore"] = "not_loaded"
		ready = false
	} else {
		checks["keystore"] = "ok"
	}
	
	// OCX executor check
	if s.ocx == nil {
		checks["ocx_executor"] = "not_loaded"
		ready = false
	} else {
		checks["ocx_executor"] = "ok"
	}
	
	status := "ready"
	if !ready {
		status = "not_ready"
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	response := map[string]interface{}{
		"status": status,
		"checks": checks,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	// Always return 200 if process is healthy
	response := map[string]interface{}{
		"status": "alive",
		"pid":    os.Getpid(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) HandleReceipts(w http.ResponseWriter, r *http.Request) {
	// This would query the database for receipts
	// For now, return empty list
	response := map[string]interface{}{
		"receipts": []interface{}{},
		"total":    0,
		"limit":    20,
		"offset":   0,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) HandleKeys(w http.ResponseWriter, r *http.Request) {
	// Extract key ID from URL path
	keyID := strings.TrimPrefix(r.URL.Path, "/keys/")
	
	key := s.keystore.GetKey(keyID)
	if key == nil {
		WriteError(w, ErrInvalidInput, "Key not found", http.StatusNotFound)
		return
	}
	
	response := key.Metadata
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	// Metrics endpoint
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "# OCX Protocol Metrics\n")
	fmt.Fprintf(w, "ocx_execute_total %d\n", metrics.ExecuteCounter.Value())
	fmt.Fprintf(w, "ocx_verify_total %d\n", metrics.VerifyCounter.Value())
	fmt.Fprintf(w, "ocx_active_connections %d\n", metrics.ActiveConnections.Value())
	fmt.Fprintf(w, "ocx_execute_latency_p50 %.3f\n", metrics.ExecuteLatency.P50())
	fmt.Fprintf(w, "ocx_execute_latency_p95 %.3f\n", metrics.ExecuteLatency.P95())
	fmt.Fprintf(w, "ocx_execute_latency_p99 %.3f\n", metrics.ExecuteLatency.P99())
	fmt.Fprintf(w, "ocx_verify_latency_p50 %.3f\n", metrics.VerifyLatency.P50())
	fmt.Fprintf(w, "ocx_verify_latency_p95 %.3f\n", metrics.VerifyLatency.P95())
	fmt.Fprintf(w, "ocx_verify_latency_p99 %.3f\n", metrics.VerifyLatency.P99())
}
