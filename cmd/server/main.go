package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"ocx.local/pkg/deterministicvm"
	"ocx.local/pkg/verify"
)

// ExecuteRequest represents the incoming request for artifact execution
type ExecuteRequest struct {
	SpecHash     [32]byte `json:"spec_hash"`
	ArtifactHash [32]byte `json:"artifact_hash"`
	Input        []byte   `json:"input"`
}

// OCXReceipt represents the execution receipt returned to clients
type OCXReceipt struct {
	SpecHash     [32]byte  `json:"spec_hash"`
	ArtifactHash [32]byte  `json:"artifact_hash"`
	InputHash    [32]byte  `json:"input_hash"`
	OutputHash   [32]byte  `json:"output_hash"`
	CyclesUsed   uint64    `json:"cycles_used"`
	MemoryUsed   uint64    `json:"memory_used"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	ExitCode     int       `json:"exit_code"`
	// Signature and other fields would go here
}

type Server struct {
	verifier verify.Verifier
	port     string
}

func NewServer() *Server {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Server{
		verifier: verify.NewVerifier(),
		port:     port,
	}
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ReceiptData []byte `json:"receipt_data"`
		PublicKey   []byte `json:"public_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Measure verification time for performance comparison
	start := time.Now()
	
	err := s.verifier.VerifyReceipt(req.ReceiptData, req.PublicKey)
	
	duration := time.Since(start)

	if err != nil {
		response := map[string]interface{}{
			"valid":    false,
			"error":    err.Error(),
			"duration": duration.Nanoseconds(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"valid":    true,
		"duration": duration.Nanoseconds(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleBatchVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Receipts []verify.ReceiptBatch `json:"receipts"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	start := time.Now()
	
	results, err := s.verifier.BatchVerify(req.Receipts)
	
	duration := time.Since(start)

	if err != nil {
		response := map[string]interface{}{
			"error":    err.Error(),
			"duration": duration.Nanoseconds(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"results":  results,
		"duration": duration.Nanoseconds(),
		"count":    len(results),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleExtractFields(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ReceiptData []byte `json:"receipt_data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	start := time.Now()
	
	fields, err := s.verifier.ExtractReceiptFields(req.ReceiptData)
	
	duration := time.Since(start)

	if err != nil {
		response := map[string]interface{}{
			"error":    err.Error(),
			"duration": duration.Nanoseconds(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"fields":   fields,
		"duration": duration.Nanoseconds(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	version, err := s.verifier.GetVersion()
	if err != nil {
		version = "unknown"
	}

	response := map[string]interface{}{
		"status":   "healthy",
		"verifier": version,
		"port":     s.port,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleExecute is the new /api/v1/execute endpoint that uses D-MVM
func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	// Record execution start time for receipt
	startedAt := time.Now().UTC()
	
	// 1. PARSE AND VALIDATE REQUEST
	if r.Method != http.MethodPost {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if req.ArtifactHash == [32]byte{} {
		s.sendError(w, "Missing artifact_hash", http.StatusBadRequest)
		return
	}
	
	// 2. CALL D-MVM: Delegate execution to the deterministic module
	result, err := deterministicvm.ExecuteArtifact(r.Context(), req.ArtifactHash, req.Input)
	if err != nil {
		// Handle different types of errors appropriately
		statusCode := s.mapErrorToHTTPStatus(err)
		s.sendError(w, fmt.Sprintf("Execution failed: %v", err), statusCode)
		return
	}
	
	// 3. GENERATE RECEIPT: Build the execution receipt from the result
	receipt := &deterministicvm.OCXReceipt{
		SpecHash:     req.SpecHash,
		ArtifactHash: req.ArtifactHash,
		InputHash:    sha256.Sum256(req.Input),
		OutputHash:   sha256.Sum256(result.Stdout), // Use stdout as the primary output
		CyclesUsed:   result.CyclesUsed,
		StartedAt:    uint64(startedAt.Unix()),
		FinishedAt:   uint64(result.EndTime.Unix()),
		IssuerID:     "ocx-server-v1",
	}
	
	// 4. CANONICALIZE AND SIGN RECEIPT
	canonicalBytes, err := deterministicvm.CanonicalizeReceipt(receipt)
	if err != nil {
		s.sendError(w, "Failed to canonicalize receipt", http.StatusInternalServerError)
		return
	}
	
	// Sign the canonical bytes (placeholder - implement your signing logic)
	signature := s.signCanonicalReceipt(canonicalBytes)
	receipt.Signature = signature
	
	// Re-canonicalize with signature
	canonicalBytes, err = deterministicvm.CanonicalizeReceipt(receipt)
	if err != nil {
		s.sendError(w, "Failed to canonicalize signed receipt", http.StatusInternalServerError)
		return
	}
	
	// 5. RESPOND: Send the canonical CBOR receipt back to the client
	w.Header().Set("Content-Type", "application/cbor")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(canonicalBytes); err != nil {
		// Log the error but can't change response at this point
		fmt.Printf("Failed to write response: %v\n", err)
	}
}

// handleArtifactInfo provides artifact metadata without execution
func (s *Server) handleArtifactInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse artifact hash from query parameter
	hashParam := r.URL.Query().Get("hash")
	if hashParam == "" {
		s.sendError(w, "Missing hash parameter", http.StatusBadRequest)
		return
	}
	
	// Convert hex string to hash
	artifactHash, err := parseHashFromHex(hashParam)
	if err != nil {
		s.sendError(w, "Invalid hash format", http.StatusBadRequest)
		return
	}
	
	// Get artifact info
	info, err := deterministicvm.GetArtifactInfo(artifactHash)
	if err != nil {
		statusCode := s.mapErrorToHTTPStatus(err)
		s.sendError(w, fmt.Sprintf("Failed to get artifact info: %v", err), statusCode)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// mapErrorToHTTPStatus maps D-MVM errors to appropriate HTTP status codes
func (s *Server) mapErrorToHTTPStatus(err error) int {
	if execErr, ok := err.(*deterministicvm.ExecutionError); ok {
		switch execErr.Code {
		case deterministicvm.ErrorCodeArtifactNotFound:
			return http.StatusNotFound
		case deterministicvm.ErrorCodeArtifactInvalid:
			return http.StatusBadRequest
		case deterministicvm.ErrorCodeTimeout:
			return http.StatusRequestTimeout
		case deterministicvm.ErrorCodeCycleLimitExceeded,
			 deterministicvm.ErrorCodeMemoryLimitExceeded:
			return http.StatusRequestEntityTooLarge
		case deterministicvm.ErrorCodePermissionDenied:
			return http.StatusForbidden
		case deterministicvm.ErrorCodeNetworkViolation:
			return http.StatusBadRequest
		default:
			return http.StatusInternalServerError
		}
	}
	return http.StatusInternalServerError
}

// sendError sends a JSON error response
func (s *Server) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	errorResp := map[string]interface{}{
		"error":     message,
		"status":    statusCode,
		"timestamp": time.Now().UTC(),
	}
	
	json.NewEncoder(w).Encode(errorResp)
}

// signReceipt implements receipt signing (placeholder)
func (s *Server) signReceipt(receipt *OCXReceipt) error {
	// TODO: Implement your cryptographic signing logic here
	// This would typically:
	// 1. Serialize the receipt fields in a canonical format
	// 2. Hash the serialized data
	// 3. Sign the hash with your private key
	// 4. Add the signature to the receipt
	
	return nil
}

// signCanonicalReceipt signs the canonical CBOR bytes
func (s *Server) signCanonicalReceipt(canonicalBytes []byte) []byte {
	// TODO: Implement your cryptographic signing logic here
	// This would typically:
	// 1. Create the signing message with domain separator
	// 2. Sign with Ed25519 private key
	// 3. Return the 64-byte signature
	
	// For now, return a placeholder signature
	// In production, this should be a real Ed25519 signature
	placeholder := make([]byte, 64)
	for i := range placeholder {
		placeholder[i] = byte(i % 256)
	}
	return placeholder
}

// parseHashFromHex converts a hex string to a 32-byte hash
func parseHashFromHex(hexStr string) ([32]byte, error) {
	var hash [32]byte
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return hash, fmt.Errorf("invalid hex string: %w", err)
	}
	if len(decoded) != 32 {
		return hash, fmt.Errorf("hash must be 32 bytes, got %d", len(decoded))
	}
	copy(hash[:], decoded)
	return hash, nil
}

func (s *Server) Start() error {
	// Existing verification endpoints
	http.HandleFunc("/verify", s.handleVerify)
	http.HandleFunc("/batch-verify", s.handleBatchVerify)
	http.HandleFunc("/extract-fields", s.handleExtractFields)
	http.HandleFunc("/status", s.handleStatus)
	
	// New D-MVM execution endpoints
	http.HandleFunc("/api/v1/execute", s.handleExecute)
	http.HandleFunc("/api/v1/artifact/info", s.handleArtifactInfo)

	log.Printf("Starting OCX server on port %s", s.port)
	log.Printf("Using verifier: %T", s.verifier)
	log.Printf("D-MVM execution endpoints available at /api/v1/")
	
	return http.ListenAndServe(":"+s.port, nil)
}

func main() {
	server := NewServer()
	if err := server.Start(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
