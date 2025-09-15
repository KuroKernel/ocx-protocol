// load_test.go - Load Testing Framework for OCX Protocol
// Tests: 1000+ concurrent API calls, database performance, HMAC authentication

package load

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/ocx/protocol/pkg/ocx"
)

const (
	BASE_URL = "http://localhost:8080"
	CONCURRENT_USERS = 1000
	REQUESTS_PER_USER = 10
)

type LoadTestConfig struct {
	BaseURL          string
	ConcurrentUsers  int
	RequestsPerUser  int
	TestDuration     time.Duration
	RampUpDuration   time.Duration
}

type LoadTestResult struct {
	TotalRequests    int
	SuccessfulRequests int
	FailedRequests   int
	AverageResponseTime time.Duration
	MaxResponseTime  time.Duration
	MinResponseTime  time.Duration
	RequestsPerSecond float64
	ErrorRate        float64
}

func TestMatchingEngineLoad(t *testing.T) {
	config := LoadTestConfig{
		BaseURL:         BASE_URL,
		ConcurrentUsers: CONCURRENT_USERS,
		RequestsPerUser: REQUESTS_PER_USER,
		TestDuration:    5 * time.Minute,
		RampUpDuration:  30 * time.Second,
	}

	result := runLoadTest(t, config, testMatchingEngineEndpoint)
	
	// Assertions for world-class performance
	if result.ErrorRate > 0.01 { // Less than 1% error rate
		t.Errorf("Error rate too high: %.2f%%", result.ErrorRate*100)
	}
	
	if result.AverageResponseTime > 500*time.Millisecond {
		t.Errorf("Average response time too slow: %v", result.AverageResponseTime)
	}
	
	if result.RequestsPerSecond < 100 {
		t.Errorf("Throughput too low: %.2f req/s", result.RequestsPerSecond)
	}
	
	t.Logf("Load Test Results:")
	t.Logf("  Total Requests: %d", result.TotalRequests)
	t.Logf("  Success Rate: %.2f%%", (1-result.ErrorRate)*100)
	t.Logf("  Avg Response Time: %v", result.AverageResponseTime)
	t.Logf("  Throughput: %.2f req/s", result.RequestsPerSecond)
}

func TestDatabasePerformance(t *testing.T) {
	config := LoadTestConfig{
		BaseURL:         BASE_URL,
		ConcurrentUsers: 500,
		RequestsPerUser: 20,
		TestDuration:    3 * time.Minute,
		RampUpDuration:  15 * time.Second,
	}

	result := runLoadTest(t, config, testDatabaseOperations)
	
	// Database-specific assertions
	if result.AverageResponseTime > 100*time.Millisecond {
		t.Errorf("Database operations too slow: %v", result.AverageResponseTime)
	}
	
	t.Logf("Database Performance Results:")
	t.Logf("  Avg DB Response Time: %v", result.AverageResponseTime)
	t.Logf("  DB Throughput: %.2f ops/s", result.RequestsPerSecond)
}

func TestHMACAuthenticationLoad(t *testing.T) {
	config := LoadTestConfig{
		BaseURL:         BASE_URL,
		ConcurrentUsers: 200,
		RequestsPerUser: 50,
		TestDuration:    2 * time.Minute,
		RampUpDuration:  10 * time.Second,
	}

	result := runLoadTest(t, config, testHMACAuthentication)
	
	// Security-specific assertions
	if result.ErrorRate > 0.005 { // Less than 0.5% error rate for auth
		t.Errorf("Authentication error rate too high: %.2f%%", result.ErrorRate*100)
	}
	
	t.Logf("HMAC Authentication Load Results:")
	t.Logf("  Auth Success Rate: %.2f%%", (1-result.ErrorRate)*100)
	t.Logf("  Avg Auth Time: %v", result.AverageResponseTime)
}

func runLoadTest(t *testing.T, config LoadTestConfig, testFunc func(*testing.T, int) error) LoadTestResult {
	var wg sync.WaitGroup
	var mu sync.Mutex
	
	results := make([]time.Duration, 0, config.ConcurrentUsers*config.RequestsPerUser)
	errors := 0
	startTime := time.Now()
	
	// Ramp up users gradually
	for i := 0; i < config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			
			// Ramp up delay
			delay := time.Duration(userID) * config.RampUpDuration / time.Duration(config.ConcurrentUsers)
			time.Sleep(delay)
			
			for j := 0; j < config.RequestsPerUser; j++ {
				requestStart := time.Now()
				err := testFunc(t, userID)
				requestDuration := time.Since(requestStart)
				
				mu.Lock()
				results = append(results, requestDuration)
				if err != nil {
					errors++
				}
				mu.Unlock()
				
				// Small delay between requests
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}
	
	wg.Wait()
	totalDuration := time.Since(startTime)
	
	// Calculate statistics
	totalRequests := len(results)
	successfulRequests := totalRequests - errors
	errorRate := float64(errors) / float64(totalRequests)
	
	var totalDuration time.Duration
	var maxDuration, minDuration time.Duration
	
	for i, duration := range results {
		totalDuration += duration
		if i == 0 || duration > maxDuration {
			maxDuration = duration
		}
		if i == 0 || duration < minDuration {
			minDuration = duration
		}
	}
	
	avgDuration := totalDuration / time.Duration(len(results))
	requestsPerSecond := float64(totalRequests) / totalDuration.Seconds()
	
	return LoadTestResult{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successfulRequests,
		FailedRequests:     errors,
		AverageResponseTime: avgDuration,
		MaxResponseTime:    maxDuration,
		MinResponseTime:    minDuration,
		RequestsPerSecond:  requestsPerSecond,
		ErrorRate:         errorRate,
	}
}

func testMatchingEngineEndpoint(t *testing.T, userID int) error {
	// Create a test order
	order := &ocx.Order{
		OrderID:       ocx.ID(fmt.Sprintf("load_test_%d_%d", userID, time.Now().UnixNano())),
		Version:       ocx.V010,
		Buyer:         ocx.PartyRef{PartyID: ocx.ID(fmt.Sprintf("buyer_%d", userID)), Role: "buyer"},
		OfferID:       ocx.ID("test_offer"),
		RequestedGPUs: 1 + (userID % 8),
		Hours:         1 + (userID % 24),
		BudgetCap:     &ocx.Money{Currency: "USD", Amount: "100.00", Scale: 2},
		State:         ocx.OrderPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
	// Create envelope
	envelope := &ocx.Envelope{
		ID:        ocx.ID(fmt.Sprintf("envelope_%d_%d", userID, time.Now().UnixNano())),
		Kind:      ocx.KindOrder,
		Version:   ocx.V010,
		IssuedAt:  time.Now(),
		Payload:   order,
		Hash:      ocx.HashMessage([]byte("test")),
	}
	
	// Sign envelope (simplified for load testing)
	envelope.Signature = "load_test_signature"
	
	// Send request
	jsonData, _ := json.Marshal(envelope)
	resp, err := http.Post(BASE_URL+"/orders", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

func testDatabaseOperations(t *testing.T, userID int) error {
	// Test database read operations
	resp, err := http.Get(fmt.Sprintf("%s/orders?user_id=%d", BASE_URL, userID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	
	return nil
}

func testHMACAuthentication(t *testing.T, userID int) error {
	// Test HMAC authentication
	key := fmt.Sprintf("test_key_%d", userID)
	message := fmt.Sprintf("test_message_%d_%d", userID, time.Now().UnixNano())
	
	// Create HMAC signature
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(message))
	signature := hex.EncodeToString(h.Sum(nil))
	
	// Send authenticated request
	req, _ := http.NewRequest("GET", BASE_URL+"/auth-test", nil)
	req.Header.Set("OCX-KEY-ID", key)
	req.Header.Set("OCX-SIGNATURE", signature)
	req.Header.Set("OCX-MESSAGE", message)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Auth failed: HTTP %d", resp.StatusCode)
	}
	
	return nil
}

// Benchmark tests for performance regression detection
func BenchmarkMatchingEngine(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := testMatchingEngineEndpoint(nil, i%1000)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDatabaseOperations(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := testDatabaseOperations(nil, i%1000)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHMACAuthentication(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := testHMACAuthentication(nil, i%1000)
		if err != nil {
			b.Fatal(err)
		}
	}
}
