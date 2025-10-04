package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ocx.local/pkg/keystore"
	"ocx.local/pkg/receipt"
	"ocx.local/pkg/verify"
)

// testServer wraps the server for testing
type testServer struct {
	server *Server
	router http.Handler
}

// mustStartTestServer creates a test server with in-memory components
func mustStartTestServer(t *testing.T) *testServer {
	t.Helper()

	// Create in-memory keystore
	ks, err := keystore.New("./test-keys")
	if err != nil {
		t.Fatalf("Failed to create keystore: %v", err)
	}

	// Create in-memory store
	store := receipt.NewMemoryStore()

	// Create metrics
	metrics := &mockMetrics{}

	// Create health checker
	healthChecker := &mockHealthChecker{}

	// Create server
	server := &Server{
		verifier:      verify.NewVerifier(),
		signer:        keystore.NewLocalSigner(ks),
		keystore:      ks,
		store:         store,
		metrics:       metrics,
		healthChecker: healthChecker,
		port:          "8080",
	}

	// Create router
	router := http.NewServeMux()

	// Health endpoints
	router.HandleFunc("/livez", server.handleLivez)
	router.HandleFunc("/readyz", server.handleReadyz)
	router.HandleFunc("/health", server.handleHealth)
	router.HandleFunc("/metrics", server.handleMetrics)

	// Core endpoints
	router.HandleFunc("/verify", server.handleVerify)
	router.HandleFunc("/status", server.handleStatus)

	return &testServer{
		server: server,
		router: router,
	}
}

// Mock implementations for testing
type mockMetrics struct{}

func (m *mockMetrics) RecordExecute(issuerID, status string, duration time.Duration, gasUsed, memoryUsed uint64) {
}
func (m *mockMetrics) RecordReceipt(issuerID, operation, status string, size int64) {}
func (m *mockMetrics) GetStats() map[string]interface{}                             { return map[string]interface{}{} }
func (m *mockMetrics) UpdateSystemMetrics(cpu, memory, disk float64)                {}

type mockHealthChecker struct{}

func (m *mockHealthChecker) RunChecks(ctx context.Context) *mockHealthStatus {
	return &mockHealthStatus{
		Overall: "healthy",
		Checks:  map[string]string{"keystore": "healthy", "verifier": "healthy"},
	}
}

type mockHealthStatus struct {
	Overall string            `json:"overall"`
	Checks  map[string]string `json:"checks"`
}

// Test cases
func TestHandleLivez_OK(t *testing.T) {
	s := mustStartTestServer(t)

	req := httptest.NewRequest("GET", "/livez", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "alive" {
		t.Fatalf("expected status 'alive', got %v", response["status"])
	}
}

func TestHandleReadyz_OK(t *testing.T) {
	s := mustStartTestServer(t)

	req := httptest.NewRequest("GET", "/readyz", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "ready" {
		t.Fatalf("expected status 'ready', got %v", response["status"])
	}
}

func TestHandleHealth_OK(t *testing.T) {
	s := mustStartTestServer(t)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["overall"] != "healthy" {
		t.Fatalf("expected overall 'healthy', got %v", response["overall"])
	}
}

func TestHandleStatus_OK(t *testing.T) {
	s := mustStartTestServer(t)

	req := httptest.NewRequest("GET", "/status", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Fatalf("expected status 'healthy', got %v", response["status"])
	}
}

func TestHandleMetrics_OK(t *testing.T) {
	s := mustStartTestServer(t)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["metrics"] == nil {
		t.Fatalf("expected metrics field, got nil")
	}
}
