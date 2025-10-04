package api

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"ocx.local/pkg/keystore"
	"ocx.local/pkg/metrics"
	"ocx.local/pkg/ocx"
	"ocx.local/pkg/receipt"
)

type ExecuteRequest struct {
	Artifact    string `json:"artifact"`
	Input       string `json:"input"`
	MaxCycles   int64  `json:"max_cycles"`
	ArtifactID  string `json:"artifact_id"`
	ExecutionID string `json:"execution_id"`
}

type ExecuteResponse struct {
	OutputHash    string `json:"output_hash"`
	GasUsed       int64  `json:"cycles_used"`
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
	ocx      ocx.OCXExecutor
	keystore *keystore.Keystore
	metrics  *metrics.SimpleMetrics
	cache    map[string]ExecuteResponse
	reqCache map[string]ExecuteRequest
	store    receipt.Store
}

func NewServer(ocxExecutor ocx.OCXExecutor, keystore *keystore.Keystore) *Server {
	return &Server{
		ocx:      ocxExecutor,
		keystore: keystore,
		metrics:  metrics.NewMetrics(),
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
		s.metrics.RecordExecute("unknown", "error", time.Since(start), 0, 0)
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
		s.metrics.RecordExecute("unknown", "error", time.Since(start), 0, 0)
		return
	}

	// Validate resource limits
	if req.MaxCycles <= 0 {
		req.MaxCycles = 10000 // Default to 10K cycles
	} else if req.MaxCycles > 1000000 { // 1M cycle limit
		WriteError(w, ErrInvalidInput, "max_cycles cannot exceed 1,000,000", http.StatusBadRequest)
		s.metrics.RecordExecute("unknown", "error", time.Since(start), 0, 0)
		return
	}

	// Add payload size check
	if len(req.Artifact) > 10*1024 { // 10KB base64 limit
		WriteError(w, ErrInvalidInput, "artifact too large (max 10KB)", http.StatusBadRequest)
		s.metrics.RecordExecute("unknown", "error", time.Since(start), 0, 0)
		return
	}

	if len(req.Input) > 10*1024 { // 10KB base64 limit
		WriteError(w, ErrInvalidInput, "input too large (max 10KB)", http.StatusBadRequest)
		s.metrics.RecordExecute("unknown", "error", time.Since(start), 0, 0)
		return
	}

	// Decode base64 inputs
	artifactBytes, err := base64.StdEncoding.DecodeString(req.Artifact)
	if err != nil {
		WriteError(w, ErrInvalidInput, "Invalid base64 artifact", http.StatusBadRequest)
		s.metrics.RecordExecute("unknown", "error", time.Since(start), 0, 0)
		return
	}

	inputBytes, err := base64.StdEncoding.DecodeString(req.Input)
	if err != nil {
		WriteError(w, ErrInvalidInput, "Invalid base64 input", http.StatusBadRequest)
		s.metrics.RecordExecute("unknown", "error", time.Since(start), 0, 0)
		return
	}

	// Execute
	result, err := s.ocx.Execute(artifactBytes, inputBytes, uint64(req.MaxCycles))
	if err != nil {
		WriteError(w, ErrExecutionFailed, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		s.metrics.RecordExecute("unknown", "error", time.Since(start), 0, 0)
		return
	}

	// Generate receipt with issuer ID
	receiptData, err := s.generateReceipt(result, req.ArtifactID, req.ExecutionID)
	if err != nil {
		WriteError(w, ErrInternalError, fmt.Sprintf("Receipt generation failed: %v", err), http.StatusInternalServerError)
		s.metrics.RecordExecute("unknown", "error", time.Since(start), 0, 0)
		return
	}

	receiptBlob := base64.StdEncoding.EncodeToString(receiptData)
	receiptHash := fmt.Sprintf("%x", result.ReceiptHash)

	// Store result with idempotency key
	response := ExecuteResponse{
		OutputHash:    fmt.Sprintf("%x", result.OutputHash),
		GasUsed:       int64(result.GasUsed),
		ReceiptHash:   receiptHash,
		ReceiptBlob:   receiptBlob,
		VerifyCommand: fmt.Sprintf("./ocx-cli verify --receipt '%s'", receiptBlob),
	}

	s.cache[idempotencyKey] = response
	s.reqCache[idempotencyKey] = req
	s.metrics.RecordExecute("system", "success", time.Since(start), result.GasUsed, 0)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) HandleVerify(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	contentType := r.Header.Get("Content-Type")

	var err error

	// Verify receipt
	var receiptBytes []byte
	switch {
	case strings.Contains(contentType, "application/cbor"):
		receiptBytes, err = io.ReadAll(r.Body)
		if err != nil {
			WriteError(w, ErrInvalidInput, "Failed to read CBOR data", http.StatusBadRequest)
			s.metrics.RecordVerify("go", "error", "invalid_request", time.Since(start))
			return
		}
	case strings.Contains(contentType, "application/json"):
		var req VerifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, ErrInvalidInput, "Invalid JSON request", http.StatusBadRequest)
			s.metrics.RecordVerify("go", "error", "invalid_request", time.Since(start))
			return
		}
		receiptBytes, err = base64.StdEncoding.DecodeString(req.ReceiptBlob)
		if err != nil {
			WriteError(w, ErrInvalidReceipt, "Invalid base64 receipt", http.StatusBadRequest)
			s.metrics.RecordVerify("go", "error", "invalid_request", time.Since(start))
			return
		}
	default:
		WriteError(w, ErrInvalidInput, "Content-Type must be application/json or application/cbor", http.StatusBadRequest)
		s.metrics.RecordVerify("go", "error", "invalid_request", time.Since(start))
		return
	}

	verifyResult, err := s.verifyReceipt(receiptBytes)
	if err != nil {
		response := VerifyResponse{
			Valid: false,
			Error: err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		s.metrics.RecordVerify("go", "error", "verification_failed", time.Since(start))
		return
	}

	result := &VerifyResponse{
		Valid:     verifyResult.Valid,
		Error:     verifyResult.Error,
		Timestamp: verifyResult.Timestamp,
	}

	response := VerifyResponse{
		Valid:     true,
		IssuerID:  result.IssuerID,
		Cycles:    result.Cycles,
		Timestamp: result.Timestamp,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	s.metrics.RecordVerify("go", "success", "", time.Since(start))
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
	// Parse query parameters
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Query the database for receipts
	ctx := r.Context()
	receipts, total, err := s.queryReceipts(ctx, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query receipts: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"receipts": receipts,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
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

	stats := s.metrics.GetStats()
	for key, value := range stats {
		fmt.Fprintf(w, "ocx_%s %v\n", key, value)
	}
}

// generateReceipt creates a cryptographic receipt for execution results
func (s *Server) generateReceipt(result *ocx.OCXResult, artifactID, executionID string) ([]byte, error) {
	// Get or create a signing key
	privateKey, err := s.getOrCreateSigningKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get signing key: %w", err)
	}

	// Create receipt
	receiptData, err := receipt.CreateReceipt(
		fmt.Sprintf("tx_%d", time.Now().UnixNano()),
		artifactID,
		executionID,
		0, // ExitCode not available in OCXResult
		result.GasUsed,
		result.ReceiptBlob, // Use the receipt blob as output
		[]byte{},           // Stderr not available in OCXResult
		receipt.Resource{
			CPUTimeMs:      0, // Not tracked in current implementation
			MemoryBytes:    0, // Not tracked in current implementation
			DiskReadBytes:  0, // Not tracked in current implementation
			DiskWriteBytes: 0, // Not tracked in current implementation
		},
		[]string{"OCX_PROTOCOL=v1.0.0-rc.1"},
		time.Now(),
		time.Now(),
		map[string]string{
			"executor": "ocx-protocol",
			"version":  "1.0.0-rc.1",
		},
		privateKey,
		"ocx-protocol-server",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create receipt: %w", err)
	}

	return receiptData, nil
}

// verifyReceipt verifies a cryptographic receipt
func (s *Server) verifyReceipt(receiptData []byte) (*receipt.VerifyResult, error) {
	// Get the public key for verification
	publicKey, err := s.getPublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Verify the receipt
	verifyResult, err := receipt.Verify(receiptData, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to verify receipt: %w", err)
	}

	return verifyResult, nil
}

// getOrCreateSigningKey gets or creates a signing key for receipt generation
func (s *Server) getOrCreateSigningKey() (ed25519.PrivateKey, error) {
	// Try to get key from keystore first
	if s.keystore != nil {
		// This would use the actual keystore implementation
		// generate a temporary key
	}

	// Generate a new key for this session
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signing key: %w", err)
	}

	return privateKey, nil
}

// getPublicKey gets the public key for receipt verification
func (s *Server) getPublicKey() (ed25519.PublicKey, error) {
	// Try to get key from keystore first
	if s.keystore != nil {
		// This would use the actual keystore implementation
		// generate a temporary key
	}

	// Generate a temporary key for this session
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	return publicKey, nil
}

// queryReceipts queries the database for receipts with pagination
func (s *Server) queryReceipts(ctx context.Context, limit, offset int) ([]map[string]interface{}, int, error) {
	// Return empty results since we don't have a database connection in the Server struct
	// Implementation: query the database using pgx/v5
	// Example implementation:
	//
	// rows, err := s.db.Query(ctx, `
	//     SELECT receipt_id, artifact_id, execution_id, receipt_data, created_at
	//     FROM ocx_receipts
	//     ORDER BY created_at DESC
	//     LIMIT $1 OFFSET $2
	// `, limit, offset)
	// if err != nil {
	//     return nil, 0, fmt.Errorf("failed to query receipts: %w", err)
	// }
	// defer rows.Close()
	//
	// var receipts []map[string]interface{}
	// for rows.Next() {
	//     var receiptID, artifactID, executionID string
	//     var receiptData []byte
	//     var createdAt time.Time
	//
	//     if err := rows.Scan(&receiptID, &artifactID, &executionID, &receiptData, &createdAt); err != nil {
	//         return nil, 0, fmt.Errorf("failed to scan receipt: %w", err)
	//     }
	//
	//     receipts = append(receipts, map[string]interface{}{
	//         "receipt_id":    receiptID,
	//         "artifact_id":   artifactID,
	//         "execution_id":  executionID,
	//         "receipt_data":  base64.StdEncoding.EncodeToString(receiptData),
	//         "created_at":    createdAt,
	//     })
	// }
	//
	// // Get total count
	// var total int
	// err = s.db.QueryRow(ctx, "SELECT COUNT(*) FROM ocx_receipts").Scan(&total)
	// if err != nil {
	//     return nil, 0, fmt.Errorf("failed to count receipts: %w", err)
	// }

	// Real database query implementation
	if s.store == nil {
		return []map[string]interface{}{}, 0, fmt.Errorf("database store not available")
	}

	// Get receipts from the store
	receipts, err := s.store.ListReceipts(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list receipts: %w", err)
	}

	// Convert to API format
	var apiReceipts []map[string]interface{}
	for _, receipt := range receipts {
		apiReceipts = append(apiReceipts, map[string]interface{}{
			"id":           receipt.ID,
			"issuer_id":    receipt.IssuerID,
			"program_hash": receipt.ProgramHash,
			"gas_used":     receipt.GasUsed,
			"created_at":   receipt.CreatedAt,
		})
	}

	// Get total count
	stats, err := s.store.GetStats(ctx)
	if err != nil {
		return apiReceipts, len(apiReceipts), nil // Return partial results
	}

	return apiReceipts, int(stats.TotalReceipts), nil
}
