package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test API endpoints
func TestHandleVerify_InvalidJSON(t *testing.T) {
	s := mustStartTestServer(t)

	req := httptest.NewRequest("POST", "/verify", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleVerify_ValidRequest(t *testing.T) {
	s := mustStartTestServer(t)

	payload := map[string]interface{}{
		"receipt_data": "dGVzdA==", // base64 for "test"
		"public_key":   "dGVzdA==", // base64 for "test"
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	// Should return 200 even if verification fails (due to test data)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["valid"] == nil {
		t.Fatalf("expected valid field in response")
	}
}

func TestHandleExecute_InvalidJSON(t *testing.T) {
	s := mustStartTestServer(t)

	req := httptest.NewRequest("POST", "/api/v1/execute", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleExecute_MissingArtifactHash(t *testing.T) {
	s := mustStartTestServer(t)

	payload := map[string]interface{}{
		"input": "dGVzdA==", // base64 for "test"
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleArtifactInfo_MissingHash(t *testing.T) {
	s := mustStartTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/artifact/info", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleArtifactInfo_InvalidHash(t *testing.T) {
	s := mustStartTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/artifact/info?hash=invalid", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleReceipts_MissingID(t *testing.T) {
	s := mustStartTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/receipts/", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleReceipts_NotFound(t *testing.T) {
	s := mustStartTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/receipts/nonexistent", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleDeleteReceipt_NotImplemented(t *testing.T) {
	s := mustStartTestServer(t)

	req := httptest.NewRequest("DELETE", "/api/v1/receipts/test", nil)
	rr := httptest.NewRecorder()

	s.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rr.Code)
	}
}

// Test error handling
func TestSendError(t *testing.T) {
	s := mustStartTestServer(t)

	rr := httptest.NewRecorder()
	s.server.sendError(rr, "test error", http.StatusBadRequest)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "test error" {
		t.Fatalf("expected error 'test error', got %v", response["error"])
	}
}

// Test parseHashFromHex
func TestParseHashFromHex_Valid(t *testing.T) {
	hash, err := parseHashFromHex("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := [32]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	if hash != expected {
		t.Fatalf("expected %v, got %v", expected, hash)
	}
}

func TestParseHashFromHex_Invalid(t *testing.T) {
	_, err := parseHashFromHex("invalid")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestParseHashFromHex_WrongLength(t *testing.T) {
	_, err := parseHashFromHex("0123456789abcdef")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
