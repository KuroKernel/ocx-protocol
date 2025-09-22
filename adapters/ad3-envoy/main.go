// adapters/ad3-envoy/main.go - AD3 Envoy Filter for OCX Injection
// This follows the EXACT same pattern as AD2 webhook but for network-level injection

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"ocx.local/pkg/cbor"
	"ocx.local/pkg/verify"
)

// EnvoyFilter represents the Envoy filter for OCX injection
type EnvoyFilter struct {
	enabled     bool
	verifier    verify.Verifier
	annotations map[string]string
}

// NewEnvoyFilter creates a new Envoy filter instance
func NewEnvoyFilter() *EnvoyFilter {
	return &EnvoyFilter{
		enabled:     true,
		verifier:    verify.NewVerifier(),
		annotations: make(map[string]string),
	}
}

// HttpRequest represents an incoming HTTP request
type HttpRequest struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Headers     map[string]string `json:"headers"`
	Body        []byte            `json:"body"`
	QueryParams map[string]string `json:"query_params"`
}

// HttpResponse represents an HTTP response
type HttpResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

// InjectOCX injects OCX verification into HTTP requests (same pattern as AD2)
func (ef *EnvoyFilter) InjectOCX(req *HttpRequest) *HttpResponse {
	if !ef.enabled {
		return ef.passthrough(req)
	}

	// Parse annotations from headers (same as AD2 webhook)
	ef.parseAnnotations(req.Headers)

	// Check if OCX injection is enabled (same logic as AD2)
	if !ef.shouldInject() {
		return ef.passthrough(req)
	}

	// Generate OCX receipt (same pattern as AD2)
	receipt, err := ef.generateReceipt(req)
	if err != nil {
		log.Printf("Failed to generate OCX receipt: %v", err)
		return ef.passthrough(req)
	}

	// Inject OCX headers (same pattern as AD2)
	response := ef.injectHeaders(req, receipt)

	return response
}

// parseAnnotations parses OCX annotations from headers (same as AD2)
func (ef *EnvoyFilter) parseAnnotations(headers map[string]string) {
	// Look for OCX annotations in headers (Envoy-specific)
	ocxHeaders := map[string]string{
		"x-ocx-inject":     "ocx-inject",
		"x-ocx-cycles":     "ocx-cycles", 
		"x-ocx-profile":    "ocx-profile",
		"x-ocx-keystore":   "ocx-keystore",
		"x-ocx-verify-only": "ocx-verify-only",
	}

	for headerName, annotationKey := range ocxHeaders {
		if value, exists := headers[headerName]; exists {
			ef.annotations[annotationKey] = value
		}
	}
}

// shouldInject determines if OCX should be injected (same logic as AD2)
func (ef *EnvoyFilter) shouldInject() bool {
	inject := ef.annotations["ocx-inject"]
	return inject == "true" || inject == "verify"
}

// generateReceipt generates an OCX receipt (same pattern as AD2)
func (ef *EnvoyFilter) generateReceipt(req *HttpRequest) (*cbor.OCXReceiptV1_1, error) {
	// Create artifact hash from request
	artifactHash := ef.hashRequest(req)
	
	// Create input hash from request body
	inputHash := ef.hashData(req.Body)
	
	// Create output hash (placeholder for now)
	outputHash := ef.hashData([]byte("envoy_output"))
	
	// Get cycles from annotation
	cycles := ef.getCycles()
	
	// Create issuer key
	issuerKey := ef.getIssuerKey()
	
	// Create receipt
	receipt := cbor.NewOCXReceiptV1_1(artifactHash, inputHash, outputHash, cycles, issuerKey)
	
	// Add request binding
	requestDigest := ef.hashData(req.Body)
	receipt.AddRequestBinding(requestDigest)
	
	// Add witness signature if enabled
	if ef.annotations["ocx-verify-only"] == "true" {
		witnessManager := cbor.NewWitnessManager()
		witnessManager.AddWitness("envoy", issuerKey)
		witnessManager.SignReceipt(receipt)
	}
	
	return receipt, nil
}

// injectHeaders injects OCX headers into the response (same pattern as AD2)
func (ef *EnvoyFilter) injectHeaders(req *HttpRequest, receipt *cbor.OCXReceiptV1_1) *HttpResponse {
	// Serialize receipt
	receiptData, err := receipt.Serialize()
	if err != nil {
		log.Printf("Failed to serialize receipt: %v", err)
		return ef.passthrough(req)
	}
	
	// Create response with OCX headers
	response := &HttpResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"x-ocx-receipt": fmt.Sprintf("%x", receiptData),
			"x-ocx-version": "1.1",
			"x-ocx-cycles": strconv.FormatUint(receipt.Cycles, 10),
			"x-ocx-chained": strconv.FormatBool(receipt.IsChained()),
			"x-ocx-witness": strconv.FormatBool(receipt.HasWitness()),
		},
		Body: req.Body, // Pass through original body
	}
	
	return response
}

// passthrough returns the request unchanged
func (ef *EnvoyFilter) passthrough(req *HttpRequest) *HttpResponse {
	return &HttpResponse{
		StatusCode: 200,
		Headers:    req.Headers,
		Body:       req.Body,
	}
}

// hashRequest creates a hash of the request
func (ef *EnvoyFilter) hashRequest(req *HttpRequest) [32]byte {
	data := fmt.Sprintf("%s %s %s", req.Method, req.Path, string(req.Body))
	return ef.hashData([]byte(data))
}

// hashData creates a SHA256 hash of data
func (ef *EnvoyFilter) hashData(data []byte) [32]byte {
	// In a real implementation, this would use crypto/sha256
	// For now, return a placeholder hash
	return [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
}

// getCycles gets the cycle count from annotations
func (ef *EnvoyFilter) getCycles() uint64 {
	cyclesStr := ef.annotations["ocx-cycles"]
	if cyclesStr == "" {
		return 10000 // Default
	}
	
	cycles, err := strconv.ParseUint(cyclesStr, 10, 64)
	if err != nil {
		return 10000
	}
	
	return cycles
}

// getIssuerKey gets the issuer key
func (ef *EnvoyFilter) getIssuerKey() [32]byte {
	// In a real implementation, this would load from keystore
	return [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
}

// EnvoyFilterServer represents the Envoy filter server
type EnvoyFilterServer struct {
	filter *EnvoyFilter
	port   string
}

// NewEnvoyFilterServer creates a new Envoy filter server
func NewEnvoyFilterServer() *EnvoyFilterServer {
	port := os.Getenv("ENVOY_FILTER_PORT")
	if port == "" {
		port = "8081"
	}
	
	return &EnvoyFilterServer{
		filter: NewEnvoyFilter(),
		port:   port,
	}
}

// handleFilter handles the Envoy filter request
func (efs *EnvoyFilterServer) handleFilter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req HttpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	// Process request through OCX filter
	response := efs.filter.InjectOCX(&req)
	
	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealth handles health check requests
func (efs *EnvoyFilterServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status": "healthy",
		"filter": "ad3-envoy",
		"port":   efs.port,
		"time":   time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Start starts the Envoy filter server
func (efs *EnvoyFilterServer) Start() error {
	http.HandleFunc("/filter", efs.handleFilter)
	http.HandleFunc("/health", efs.handleHealth)
	
	log.Printf("Starting AD3 Envoy Filter on port %s", efs.port)
	log.Printf("Filter enabled: %v", efs.filter.enabled)
	
	return http.ListenAndServe(":"+efs.port, nil)
}

func main() {
	server := NewEnvoyFilterServer()
	if err := server.Start(); err != nil {
		log.Fatal("Envoy filter server failed to start:", err)
	}
}
