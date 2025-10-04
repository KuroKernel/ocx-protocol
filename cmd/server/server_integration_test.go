package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ocx.local/internal/api"
	"ocx.local/pkg/keystore"
	"ocx.local/pkg/metrics"
)

// setupArtifactCache sets up the artifact cache for deterministic VM testing
func setupArtifactCache(t *testing.T, artifactBytes []byte, artifactHash [32]byte) {
	cacheDir := filepath.Join(os.TempDir(), "ocx-test-cache", "artifacts")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}
	
	// Copy artifact to cache location
	hashStr := fmt.Sprintf("%x", artifactHash)
	cachePath := filepath.Join(cacheDir, hashStr)
	if err := os.WriteFile(cachePath, artifactBytes, 0755); err != nil {
		t.Fatalf("Failed to copy artifact to cache: %v", err)
	}
}

func TestServerIntegration(t *testing.T) {
	// Create test keystore
	tempDir := t.TempDir()
	ks, err := keystore.New(tempDir)
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	err = ks.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create test server
	server, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create test HTTP server with proper routing
	mux := http.NewServeMux()
	
	// Health check endpoints (no auth required)
	mux.HandleFunc("/livez", server.handleLivez)
	mux.HandleFunc("/readyz", server.handleReadyz)
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/metrics", server.handleMetrics)
	
	// API endpoints (with security and metrics)
	securityMiddleware := api.NewSecurityMiddleware()
	metricsMiddleware := metrics.NewMetricsMiddleware(server.metrics)
	
	mux.Handle("/verify", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleVerify))))
	mux.Handle("/batch-verify", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleBatchVerify))))
	mux.Handle("/extract-fields", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleExtractFields))))
	mux.Handle("/status", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleStatus))))
	mux.Handle("/api/v1/execute", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleExecute))))
	mux.Handle("/api/v1/artifact/info", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleArtifactInfo))))
	mux.Handle("/api/v1/receipts/", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleReceipts))))
	
	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("health_endpoints", func(t *testing.T) {
		// Test /livez
		resp, err := http.Get(ts.URL + "/livez")
		if err != nil {
			t.Fatalf("Failed to get /livez: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for /livez, got %d", resp.StatusCode)
		}

		// Test /readyz
		resp, err = http.Get(ts.URL + "/readyz")
		if err != nil {
			t.Fatalf("Failed to get /readyz: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for /readyz, got %d", resp.StatusCode)
		}

		// Test /health
		resp, err = http.Get(ts.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to get /health: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for /health, got %d", resp.StatusCode)
		}

		var healthResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
			t.Fatalf("Failed to decode health response: %v", err)
		}

		if healthResp["overall"] != "healthy" {
			t.Errorf("Expected overall health to be 'healthy', got %v", healthResp["overall"])
		}
	})

	t.Run("metrics_endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/metrics")
		if err != nil {
			t.Fatalf("Failed to get /metrics: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for /metrics, got %d", resp.StatusCode)
		}

		var metricsResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&metricsResp); err != nil {
			t.Fatalf("Failed to decode metrics response: %v", err)
		}

		if metricsResp["timestamp"] == nil {
			t.Error("Expected timestamp in metrics response")
		}
		if metricsResp["metrics"] == nil {
			t.Error("Expected metrics in response")
		}
	})

	t.Run("execute_endpoint", func(t *testing.T) {
		// Create a simple test artifact
		artifactScript := `#!/bin/bash
echo "Hello from OCX Protocol!"
echo "Input: $(cat input.bin)"
exit 0`

		// Create temporary artifact file
		artifactPath := filepath.Join(tempDir, "test-artifact")
		err := os.WriteFile(artifactPath, []byte(artifactScript), 0755)
		if err != nil {
			t.Fatalf("Failed to create test artifact: %v", err)
		}

		// Calculate artifact hash
		artifactBytes, err := os.ReadFile(artifactPath)
		if err != nil {
			t.Fatalf("Failed to read artifact: %v", err)
		}
		artifactHash := sha256.Sum256(artifactBytes)
		
		// Set up artifact cache for deterministic VM
		setupArtifactCache(t, artifactBytes, artifactHash)

		// Create execute request
		executeReq := map[string]interface{}{
			"artifact_hash": hex.EncodeToString(artifactHash[:]),
			"input":         hex.EncodeToString([]byte("test input")),
		}

		reqBody, err := json.Marshal(executeReq)
		if err != nil {
			t.Fatalf("Failed to marshal execute request: %v", err)
		}

		// Make execute request
		resp, err := http.Post(ts.URL+"/api/v1/execute", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to make execute request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200 for execute, got %d: %s", resp.StatusCode, string(body))
		}

		var executeResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&executeResp); err != nil {
			t.Fatalf("Failed to decode execute response: %v", err)
		}

		// Verify response structure
		if executeResp["receipt_id"] == nil {
			t.Error("Expected receipt_id in execute response")
		}
		if executeResp["receipt"] == nil {
			t.Error("Expected receipt in execute response")
		}
		if executeResp["receipt_b64"] == nil {
			t.Error("Expected receipt_b64 in execute response")
		}
		if executeResp["execution"] == nil {
			t.Error("Expected execution in execute response")
		}

		// Verify execution details
		execution, ok := executeResp["execution"].(map[string]interface{})
		if !ok {
			t.Error("Expected execution to be a map")
		}
		if execution["gas_used"] == nil {
			t.Error("Expected gas_used in execution")
		}
		if execution["memory_used"] == nil {
			t.Error("Expected memory_used in execution")
		}
		if execution["started_at"] == nil {
			t.Error("Expected started_at in execution")
		}
		if execution["finished_at"] == nil {
			t.Error("Expected finished_at in execution")
		}

		// Verify receipt can be decoded
		receiptHex, ok := executeResp["receipt"].(string)
		if !ok {
			t.Error("Expected receipt to be a string")
		}

		receiptBytes, err := hex.DecodeString(receiptHex)
		if err != nil {
			t.Fatalf("Failed to decode receipt hex: %v", err)
		}

		// Verify receipt is valid CBOR
		// Note: We can't easily decode CBOR without the cbor package, but we can verify it's not empty
		if len(receiptBytes) == 0 {
			t.Error("Expected non-empty receipt bytes")
		}

		// Verify receipt_b64 can be decoded
		receiptB64, ok := executeResp["receipt_b64"].(string)
		if !ok {
			t.Error("Expected receipt_b64 to be a string")
		}

		receiptBytesB64, err := base64.StdEncoding.DecodeString(receiptB64)
		if err != nil {
			t.Fatalf("Failed to decode receipt base64: %v", err)
		}

		if len(receiptBytesB64) == 0 {
			t.Error("Expected non-empty receipt bytes from base64")
		}

		// Verify hex and base64 are the same
		if string(receiptBytes) != string(receiptBytesB64) {
			t.Error("Expected receipt hex and base64 to be the same")
		}
	})

	t.Run("execute_endpoint_invalid_request", func(t *testing.T) {
		// Test with invalid JSON
		resp, err := http.Post(ts.URL+"/api/v1/execute", "application/json", strings.NewReader("invalid json"))
		if err != nil {
			t.Fatalf("Failed to make invalid execute request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Error("Expected error status for invalid JSON request")
		}
	})

	t.Run("execute_endpoint_missing_fields", func(t *testing.T) {
		// Test with missing required fields
		executeReq := map[string]interface{}{
			"artifact_hash": "invalid-hex",
		}

		reqBody, err := json.Marshal(executeReq)
		if err != nil {
			t.Fatalf("Failed to marshal execute request: %v", err)
		}

		resp, err := http.Post(ts.URL+"/api/v1/execute", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to make execute request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Error("Expected error status for missing fields request")
		}
	})

	t.Run("execute_endpoint_invalid_hex", func(t *testing.T) {
		// Test with invalid hex
		executeReq := map[string]interface{}{
			"artifact_hash": "invalid-hex",
			"input_hex":    "invalid-hex",
			"max_gas":      10000,
		}

		reqBody, err := json.Marshal(executeReq)
		if err != nil {
			t.Fatalf("Failed to marshal execute request: %v", err)
		}

		resp, err := http.Post(ts.URL+"/api/v1/execute", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to make execute request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Error("Expected error status for invalid hex request")
		}
	})

	t.Run("execute_endpoint_nonexistent_artifact", func(t *testing.T) {
		// Test with nonexistent artifact
		nonexistentHash := sha256.Sum256([]byte("nonexistent artifact"))
		executeReq := map[string]interface{}{
			"artifact_hex": hex.EncodeToString(nonexistentHash[:]),
			"input_hex":    hex.EncodeToString([]byte("test input")),
			"max_gas":      10000,
		}

		reqBody, err := json.Marshal(executeReq)
		if err != nil {
			t.Fatalf("Failed to marshal execute request: %v", err)
		}

		resp, err := http.Post(ts.URL+"/api/v1/execute", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to make execute request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Error("Expected error status for nonexistent artifact request")
		}
	})

	t.Run("receipts_endpoint", func(t *testing.T) {
		// First, create a receipt by executing an artifact
		artifactScript := `#!/bin/bash
echo "Hello from OCX Protocol!"
exit 0`

		artifactPath := filepath.Join(tempDir, "receipt-test-artifact")
		err := os.WriteFile(artifactPath, []byte(artifactScript), 0755)
		if err != nil {
			t.Fatalf("Failed to create test artifact: %v", err)
		}

		artifactBytes, err := os.ReadFile(artifactPath)
		if err != nil {
			t.Fatalf("Failed to read artifact: %v", err)
		}
		artifactHash := sha256.Sum256(artifactBytes)
		
		// Set up artifact cache for deterministic VM
		setupArtifactCache(t, artifactBytes, artifactHash)

		executeReq := map[string]interface{}{
			"artifact_hash": hex.EncodeToString(artifactHash[:]),
			"input":         hex.EncodeToString([]byte("test input")),
		}

		reqBody, err := json.Marshal(executeReq)
		if err != nil {
			t.Fatalf("Failed to marshal execute request: %v", err)
		}

		resp, err := http.Post(ts.URL+"/api/v1/execute", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to make execute request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200 for execute, got %d: %s", resp.StatusCode, string(body))
		}

		var executeResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&executeResp); err != nil {
			t.Fatalf("Failed to decode execute response: %v", err)
		}

		receiptID, ok := executeResp["receipt_id"].(string)
		if !ok {
			t.Fatalf("Expected receipt_id to be a string, got %T", executeResp["receipt_id"])
		}

		// Now test the receipts endpoint
		resp, err = http.Get(ts.URL + "/api/v1/receipts/" + receiptID)
		if err != nil {
			t.Fatalf("Failed to get receipt: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200 for get receipt, got %d: %s", resp.StatusCode, string(body))
		}

		var receiptResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&receiptResp); err != nil {
			t.Fatalf("Failed to decode receipt response: %v", err)
		}

		if receiptResp["receipt_id"] != receiptID {
			t.Errorf("Expected receipt_id %s, got %v", receiptID, receiptResp["receipt_id"])
		}
		if receiptResp["receipt_cbor"] == nil {
			t.Error("Expected receipt_cbor in response")
		}
	})

	t.Run("receipts_endpoint_nonexistent", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/receipts/nonexistent-receipt-id")
		if err != nil {
			t.Fatalf("Failed to get nonexistent receipt: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Error("Expected error status for nonexistent receipt")
		}
	})

	t.Run("receipts_endpoint_delete", func(t *testing.T) {
		// First, create a receipt
		artifactScript := `#!/bin/bash
echo "Hello from OCX Protocol!"
exit 0`

		artifactPath := filepath.Join(tempDir, "delete-test-artifact")
		err := os.WriteFile(artifactPath, []byte(artifactScript), 0755)
		if err != nil {
			t.Fatalf("Failed to create test artifact: %v", err)
		}

		artifactBytes, err := os.ReadFile(artifactPath)
		if err != nil {
			t.Fatalf("Failed to read artifact: %v", err)
		}
		artifactHash := sha256.Sum256(artifactBytes)
		
		// Set up artifact cache for deterministic VM
		setupArtifactCache(t, artifactBytes, artifactHash)

		executeReq := map[string]interface{}{
			"artifact_hash": hex.EncodeToString(artifactHash[:]),
			"input":         hex.EncodeToString([]byte("test input")),
		}

		reqBody, err := json.Marshal(executeReq)
		if err != nil {
			t.Fatalf("Failed to marshal execute request: %v", err)
		}

		resp, err := http.Post(ts.URL+"/api/v1/execute", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to make execute request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 200 for execute, got %d: %s", resp.StatusCode, string(body))
		}

		var executeResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&executeResp); err != nil {
			t.Fatalf("Failed to decode execute response: %v", err)
		}

		receiptID, ok := executeResp["receipt_id"].(string)
		if !ok {
			t.Fatalf("Expected receipt_id to be a string, got %T", executeResp["receipt_id"])
		}

		// Now delete the receipt
		req, err := http.NewRequest("DELETE", ts.URL+"/api/v1/receipts/"+receiptID, nil)
		if err != nil {
			t.Fatalf("Failed to create delete request: %v", err)
		}

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to delete receipt: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotImplemented {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected status 501 for delete receipt (not implemented), got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("unsupported_endpoints", func(t *testing.T) {
		// Test unsupported endpoints return 404
		endpoints := []string{
			"/api/v1/unsupported",
			"/api/v2/execute",
			"/admin",
			"/debug",
		}

		for _, endpoint := range endpoints {
			resp, err := http.Get(ts.URL + endpoint)
			if err != nil {
				t.Fatalf("Failed to get %s: %v", endpoint, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("Expected status 404 for %s, got %d", endpoint, resp.StatusCode)
			}
		}
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		// Test unsupported methods
		req, err := http.NewRequest("PUT", ts.URL+"/api/v1/execute", nil)
		if err != nil {
			t.Fatalf("Failed to create PUT request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make PUT request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405 for PUT request, got %d", resp.StatusCode)
		}
	})
}

func TestServerConcurrency(t *testing.T) {
	// Create test server
	server, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create test HTTP server with proper routing
	mux := http.NewServeMux()
	
	// Health check endpoints (no auth required)
	mux.HandleFunc("/livez", server.handleLivez)
	mux.HandleFunc("/readyz", server.handleReadyz)
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/metrics", server.handleMetrics)
	
	// API endpoints (with security and metrics)
	securityMiddleware := api.NewSecurityMiddleware()
	metricsMiddleware := metrics.NewMetricsMiddleware(server.metrics)
	
	mux.Handle("/verify", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleVerify))))
	mux.Handle("/batch-verify", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleBatchVerify))))
	mux.Handle("/extract-fields", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleExtractFields))))
	mux.Handle("/status", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleStatus))))
	mux.Handle("/api/v1/execute", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleExecute))))
	mux.Handle("/api/v1/artifact/info", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleArtifactInfo))))
	mux.Handle("/api/v1/receipts/", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleReceipts))))
	
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Create test artifact
	tempDir := t.TempDir()
	artifactScript := `#!/bin/bash
echo "Hello from OCX Protocol!"
echo "Input: $(cat input.bin)"
exit 0`

	artifactPath := filepath.Join(tempDir, "concurrent-test-artifact")
	err = os.WriteFile(artifactPath, []byte(artifactScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create test artifact: %v", err)
	}

	artifactBytes, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("Failed to read artifact: %v", err)
	}
	artifactHash := sha256.Sum256(artifactBytes)
	
	// Set up artifact cache for deterministic VM
	setupArtifactCache(t, artifactBytes, artifactHash)

	numGoroutines := 10
	results := make(chan error, numGoroutines)

	t.Run("concurrent_execute_requests", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				executeReq := map[string]interface{}{
			"artifact_hash": hex.EncodeToString(artifactHash[:]),
			"input":         hex.EncodeToString([]byte(fmt.Sprintf("test input %d", index))),
				}

				reqBody, err := json.Marshal(executeReq)
				if err != nil {
					results <- err
					return
				}

				resp, err := http.Post(ts.URL+"/api/v1/execute", "application/json", bytes.NewBuffer(reqBody))
				if err != nil {
					results <- err
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(resp.Body)
					results <- fmt.Errorf("expected status 200, got %d: %s", resp.StatusCode, string(body))
					return
				}

				var executeResp map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&executeResp); err != nil {
					results <- err
					return
				}

				if executeResp["receipt_id"] == nil {
					results <- fmt.Errorf("expected receipt_id in response")
					return
				}

				results <- nil
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent execute request failed: %v", err)
			}
		}
	})

	t.Run("concurrent_health_requests", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			go func() {
				resp, err := http.Get(ts.URL + "/health")
				if err != nil {
					results <- err
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					results <- fmt.Errorf("expected status 200 for health, got %d", resp.StatusCode)
					return
				}

				results <- nil
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent health request failed: %v", err)
			}
		}
	})
}

func TestServerErrorHandling(t *testing.T) {
	// Create test server
	server, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create test HTTP server with proper routing
	mux := http.NewServeMux()
	
	// Health check endpoints (no auth required)
	mux.HandleFunc("/livez", server.handleLivez)
	mux.HandleFunc("/readyz", server.handleReadyz)
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/metrics", server.handleMetrics)
	
	// API endpoints (with security and metrics)
	securityMiddleware := api.NewSecurityMiddleware()
	metricsMiddleware := metrics.NewMetricsMiddleware(server.metrics)
	
	mux.Handle("/verify", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleVerify))))
	mux.Handle("/batch-verify", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleBatchVerify))))
	mux.Handle("/extract-fields", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleExtractFields))))
	mux.Handle("/status", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleStatus))))
	mux.Handle("/api/v1/execute", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleExecute))))
	mux.Handle("/api/v1/artifact/info", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleArtifactInfo))))
	mux.Handle("/api/v1/receipts/", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleReceipts))))
	
	ts := httptest.NewServer(mux)
	defer ts.Close()

	t.Run("large_request_body", func(t *testing.T) {
		// Create a very large request body
		largeBody := make([]byte, 10*1024*1024) // 10MB
		for i := range largeBody {
			largeBody[i] = byte(i % 256)
		}

		resp, err := http.Post(ts.URL+"/api/v1/execute", "application/json", bytes.NewBuffer(largeBody))
		if err != nil {
			t.Fatalf("Failed to make large request: %v", err)
		}
		defer resp.Body.Close()

		// Should handle large request gracefully (either process or reject with appropriate error)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 200 or 413 for large request, got %d", resp.StatusCode)
		}
	})

	t.Run("malformed_json", func(t *testing.T) {
		malformedJSON := `{"artifact_hash": "invalid", "input": "invalid"}`

		resp, err := http.Post(ts.URL+"/api/v1/execute", "application/json", strings.NewReader(malformedJSON))
		if err != nil {
			t.Fatalf("Failed to make malformed JSON request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Error("Expected error status for malformed JSON request")
		}
	})

	t.Run("unsupported_content_type", func(t *testing.T) {
		executeReq := map[string]interface{}{
			"artifact_hash": "a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e",
			"input":         "7465737420696e707574",
		}

		reqBody, err := json.Marshal(executeReq)
		if err != nil {
			t.Fatalf("Failed to marshal execute request: %v", err)
		}

		resp, err := http.Post(ts.URL+"/api/v1/execute", "text/plain", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to make request with unsupported content type: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			t.Error("Expected error status for unsupported content type")
		}
	})
}

func TestServerTimeout(t *testing.T) {
	// Create test server with short timeout
	server, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create test HTTP server with short timeout
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow request
		time.Sleep(2 * time.Second)
		// Use a simple handler for timeout testing
		server.handleHealth(w, r)
	}))
	defer ts.Close()

	// Create HTTP client with short timeout
	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	t.Run("request_timeout", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/health")
		if err != nil {
			// Expected timeout error
			if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline exceeded") {
				t.Errorf("Expected timeout error, got: %v", err)
			}
			return
		}
		defer resp.Body.Close()

		// If we get here, the request didn't timeout as expected
		t.Error("Expected request to timeout")
	})
}

func BenchmarkServerHealthEndpoint(b *testing.B) {
	// Create test server
	server, err := NewServer()
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	// Create test HTTP server with proper routing
	mux := http.NewServeMux()
	
	// Health check endpoints (no auth required)
	mux.HandleFunc("/livez", server.handleLivez)
	mux.HandleFunc("/readyz", server.handleReadyz)
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/metrics", server.handleMetrics)
	
	// API endpoints (with security and metrics)
	securityMiddleware := api.NewSecurityMiddleware()
	metricsMiddleware := metrics.NewMetricsMiddleware(server.metrics)
	
	mux.Handle("/verify", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleVerify))))
	mux.Handle("/batch-verify", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleBatchVerify))))
	mux.Handle("/extract-fields", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleExtractFields))))
	mux.Handle("/status", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleStatus))))
	mux.Handle("/api/v1/execute", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleExecute))))
	mux.Handle("/api/v1/artifact/info", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleArtifactInfo))))
	mux.Handle("/api/v1/receipts/", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleReceipts))))
	
	ts := httptest.NewServer(mux)
	defer ts.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Get(ts.URL + "/health")
		if err != nil {
			b.Fatalf("Failed to get health: %v", err)
		}
		resp.Body.Close()
	}
}

func BenchmarkServerExecuteEndpoint(b *testing.B) {
	// Create test server
	server, err := NewServer()
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}

	// Create test HTTP server with proper routing
	mux := http.NewServeMux()
	
	// Health check endpoints (no auth required)
	mux.HandleFunc("/livez", server.handleLivez)
	mux.HandleFunc("/readyz", server.handleReadyz)
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/metrics", server.handleMetrics)
	
	// API endpoints (with security and metrics)
	securityMiddleware := api.NewSecurityMiddleware()
	metricsMiddleware := metrics.NewMetricsMiddleware(server.metrics)
	
	mux.Handle("/verify", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleVerify))))
	mux.Handle("/batch-verify", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleBatchVerify))))
	mux.Handle("/extract-fields", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleExtractFields))))
	mux.Handle("/status", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleStatus))))
	mux.Handle("/api/v1/execute", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleExecute))))
	mux.Handle("/api/v1/artifact/info", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleArtifactInfo))))
	mux.Handle("/api/v1/receipts/", metricsMiddleware.Middleware(securityMiddleware.Middleware(http.HandlerFunc(server.handleReceipts))))
	
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Create test artifact
	tempDir := b.TempDir()
	artifactScript := `#!/bin/bash
echo "Hello from OCX Protocol!"
exit 0`

	artifactPath := filepath.Join(tempDir, "benchmark-test-artifact")
	err = os.WriteFile(artifactPath, []byte(artifactScript), 0755)
	if err != nil {
		b.Fatalf("Failed to create test artifact: %v", err)
	}

	artifactBytes, err := os.ReadFile(artifactPath)
	if err != nil {
		b.Fatalf("Failed to read artifact: %v", err)
	}
	artifactHash := sha256.Sum256(artifactBytes)

	executeReq := map[string]interface{}{
		"artifact_hex": hex.EncodeToString(artifactHash[:]),
		"input_hex":    hex.EncodeToString([]byte("test input")),
		"max_gas":      10000,
	}

	reqBody, err := json.Marshal(executeReq)
	if err != nil {
		b.Fatalf("Failed to marshal execute request: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Post(ts.URL+"/api/v1/execute", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			b.Fatalf("Failed to make execute request: %v", err)
		}
		resp.Body.Close()
	}
}
