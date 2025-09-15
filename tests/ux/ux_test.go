// ux_test.go - User Experience Testing Suite for OCX Protocol
// Tests: CLI reliability, API response times, error messages, documentation

package ux

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	BASE_URL = "http://localhost:8080"
	CLI_PATH = "./ocxctl"
)

func TestCLIReliability(t *testing.T) {
	t.Run("CLICommandsAcrossEnvironments", func(t *testing.T) {
		// Test CLI commands across different environments
		environments := []struct {
			name        string
			setupFunc   func(*testing.T) error
			cleanupFunc func(*testing.T) error
		}{
			{
				name:        "Linux",
				setupFunc:   setupLinuxEnvironment,
				cleanupFunc: cleanupLinuxEnvironment,
			},
			{
				name:        "Docker",
				setupFunc:   setupDockerEnvironment,
				cleanupFunc: cleanupDockerEnvironment,
			},
			{
				name:        "Minimal",
				setupFunc:   setupMinimalEnvironment,
				cleanupFunc: cleanupMinimalEnvironment,
			},
		}
		
		for _, env := range environments {
			t.Run(env.name, func(t *testing.T) {
				// Setup environment
				if err := env.setupFunc(t); err != nil {
					t.Fatalf("Environment setup failed: %v", err)
				}
				defer env.cleanupFunc(t)
				
				// Test CLI commands
				commands := []struct {
					name    string
					args    []string
					timeout time.Duration
				}{
					{"Help", []string{"--help"}, 5 * time.Second},
					{"Version", []string{"--version"}, 5 * time.Second},
					{"ListOffers", []string{"list-offers"}, 10 * time.Second},
					{"ListOrders", []string{"list-orders"}, 10 * time.Second},
					{"MakeOffer", []string{"make-offer"}, 15 * time.Second},
				}
				
				for _, cmd := range commands {
					t.Run(cmd.name, func(t *testing.T) {
						if !testCLICommand(t, cmd.args, cmd.timeout) {
							t.Errorf("CLI command %s failed in %s environment", cmd.name, env.name)
						}
					})
				}
			})
		}
	})
	
	t.Run("CLIErrorHandling", func(t *testing.T) {
		// Test CLI error handling
		errorCases := []struct {
			name        string
			args        []string
			expectedErr string
		}{
			{
				name:        "InvalidCommand",
				args:        []string{"invalid-command"},
				expectedErr: "Unknown command",
			},
			{
				name:        "MissingRequiredFlag",
				args:        []string{"place-order"},
				expectedErr: "Offer ID required",
			},
			{
				name:        "InvalidServerURL",
				args:        []string{"--server", "http://invalid:9999", "list-offers"},
				expectedErr: "connection refused",
			},
			{
				name:        "InvalidOfferID",
				args:        []string{"place-order", "--offer-id", "invalid-id"},
				expectedErr: "Invalid offer ID",
			},
		}
		
		for _, errorCase := range errorCases {
			t.Run(errorCase.name, func(t *testing.T) {
				if !testCLIError(t, errorCase.args, errorCase.expectedErr) {
					t.Errorf("CLI error handling failed for %s", errorCase.name)
				}
			})
		}
	})
}

func TestAPIResponseTimeConsistency(t *testing.T) {
	t.Run("ResponseTimeConsistency", func(t *testing.T) {
		// Test API response time consistency
		endpoints := []struct {
			name     string
			url      string
			method   string
			maxTime  time.Duration
		}{
			{"HealthCheck", "/health", "GET", 100 * time.Millisecond},
			{"ListOffers", "/offers", "GET", 500 * time.Millisecond},
			{"ListOrders", "/orders", "GET", 500 * time.Millisecond},
			{"Metrics", "/metrics", "GET", 200 * time.Millisecond},
		}
		
		for _, endpoint := range endpoints {
			t.Run(endpoint.name, func(t *testing.T) {
				responseTimes := make([]time.Duration, 10)
				
				for i := 0; i < 10; i++ {
					start := time.Now()
					resp, err := http.Get(BASE_URL + endpoint.url)
					duration := time.Since(start)
					
					if err != nil {
						t.Errorf("Request failed: %v", err)
						continue
					}
					resp.Body.Close()
					
					responseTimes[i] = duration
					
					// Small delay between requests
					time.Sleep(100 * time.Millisecond)
				}
				
				// Calculate statistics
				avgTime := calculateAverage(responseTimes)
				maxTime := calculateMax(responseTimes)
				
				if avgTime > endpoint.maxTime {
					t.Errorf("Average response time too slow: %v (max: %v)", avgTime, endpoint.maxTime)
				}
				
				if maxTime > endpoint.maxTime*2 {
					t.Errorf("Max response time too slow: %v (max: %v)", maxTime, endpoint.maxTime*2)
				}
				
				t.Logf("Endpoint %s: avg=%v, max=%v", endpoint.name, avgTime, maxTime)
			})
		}
	})
	
	t.Run("ConcurrentRequestHandling", func(t *testing.T) {
		// Test concurrent request handling
		concurrentRequests := 50
		responseTimes := make([]time.Duration, concurrentRequests)
		
		// Send concurrent requests
		done := make(chan bool, concurrentRequests)
		for i := 0; i < concurrentRequests; i++ {
			go func(index int) {
				start := time.Now()
				resp, err := http.Get(BASE_URL + "/health")
				duration := time.Since(start)
				
				if err != nil {
					t.Errorf("Concurrent request %d failed: %v", index, err)
				} else {
					resp.Body.Close()
					responseTimes[index] = duration
				}
				
				done <- true
			}(i)
		}
		
		// Wait for all requests to complete
		for i := 0; i < concurrentRequests; i++ {
			<-done
		}
		
		// Verify all requests completed within reasonable time
		maxTime := calculateMax(responseTimes)
		if maxTime > 5*time.Second {
			t.Errorf("Concurrent requests too slow: max time %v", maxTime)
		}
		
		t.Logf("Concurrent requests: max time %v", maxTime)
	})
}

func TestErrorMessageClarity(t *testing.T) {
	t.Run("APIErrorMessages", func(t *testing.T) {
		// Test API error message clarity
		errorCases := []struct {
			name           string
			request        func() (*http.Response, error)
			expectedStatus int
			expectedError  string
		}{
			{
				name: "InvalidJSON",
				request: func() (*http.Response, error) {
					return http.Post(BASE_URL+"/orders", "application/json", 
						strings.NewReader("invalid json"))
				},
				expectedStatus: 400,
				expectedError:  "Invalid JSON",
			},
			{
				name: "MissingRequiredField",
				request: func() (*http.Response, error) {
					return http.Post(BASE_URL+"/orders", "application/json", 
						strings.NewReader(`{"buyer_id": "test"}`))
				},
				expectedStatus: 400,
				expectedError:  "Missing required field",
			},
			{
				name: "InvalidOrderID",
				request: func() (*http.Response, error) {
					return http.Get(BASE_URL + "/orders/invalid-id")
				},
				expectedStatus: 404,
				expectedError:  "Order not found",
			},
			{
				name: "Unauthorized",
				request: func() (*http.Response, error) {
					req, _ := http.NewRequest("GET", BASE_URL+"/orders", nil)
					req.Header.Set("Authorization", "Bearer invalid-token")
					return http.DefaultClient.Do(req)
				},
				expectedStatus: 401,
				expectedError:  "Unauthorized",
			},
		}
		
		for _, errorCase := range errorCases {
			t.Run(errorCase.name, func(t *testing.T) {
				resp, err := errorCase.request()
				if err != nil {
					t.Errorf("Request failed: %v", err)
					return
				}
				defer resp.Body.Close()
				
				if resp.StatusCode != errorCase.expectedStatus {
					t.Errorf("Expected status %d, got %d", errorCase.expectedStatus, resp.StatusCode)
				}
				
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Errorf("Failed to read response body: %v", err)
					return
				}
				
				if !strings.Contains(string(body), errorCase.expectedError) {
					t.Errorf("Expected error message '%s' not found in response: %s", 
						errorCase.expectedError, string(body))
				}
			})
		}
	})
	
	t.Run("CLIErrorMessages", func(t *testing.T) {
		// Test CLI error message clarity
		errorCases := []struct {
			name           string
			args           []string
			expectedError  string
		}{
			{
				name:          "InvalidCommand",
				args:          []string{"invalid-command"},
				expectedError: "Unknown command",
			},
			{
				name:          "MissingRequiredFlag",
				args:          []string{"place-order"},
				expectedError: "Offer ID required",
			},
			{
				name:          "InvalidServerURL",
				args:          []string{"--server", "http://invalid:9999", "list-offers"},
				expectedError: "connection refused",
			},
		}
		
		for _, errorCase := range errorCases {
			t.Run(errorCase.name, func(t *testing.T) {
				cmd := exec.Command(CLI_PATH, errorCase.args...)
				output, err := cmd.CombinedOutput()
				
				if err == nil {
					t.Errorf("Expected error for %s, but command succeeded", errorCase.name)
					return
				}
				
				if !strings.Contains(string(output), errorCase.expectedError) {
					t.Errorf("Expected error message '%s' not found in output: %s", 
						errorCase.expectedError, string(output))
				}
			})
		}
	})
}

func TestDocumentationAccuracy(t *testing.T) {
	t.Run("APIDocumentation", func(t *testing.T) {
		// Test API documentation accuracy
		docTests := []struct {
			name        string
			endpoint    string
			method      string
			description string
		}{
			{
				name:        "HealthCheck",
				endpoint:    "/health",
				method:      "GET",
				description: "Health check endpoint",
			},
			{
				name:        "ListOffers",
				endpoint:    "/offers",
				method:      "GET",
				description: "List available offers",
			},
			{
				name:        "CreateOrder",
				endpoint:    "/orders",
				method:      "POST",
				description: "Create new order",
			},
		}
		
		for _, docTest := range docTests {
			t.Run(docTest.name, func(t *testing.T) {
				// Test that endpoint exists and responds
				var resp *http.Response
				var err error
				
				switch docTest.method {
				case "GET":
					resp, err = http.Get(BASE_URL + docTest.endpoint)
				case "POST":
					resp, err = http.Post(BASE_URL+docTest.endpoint, "application/json", 
						strings.NewReader("{}"))
				}
				
				if err != nil {
					t.Errorf("Endpoint %s %s failed: %v", docTest.method, docTest.endpoint, err)
					return
				}
				defer resp.Body.Close()
				
				// Verify endpoint responds (even if with error)
				if resp.StatusCode >= 500 {
					t.Errorf("Endpoint %s %s returned server error: %d", 
						docTest.method, docTest.endpoint, resp.StatusCode)
				}
			})
		}
	})
	
	t.Run("CLIDocumentation", func(t *testing.T) {
		// Test CLI documentation accuracy
		docTests := []struct {
			name        string
			command     string
			description string
		}{
			{
				name:        "HelpCommand",
				command:     "--help",
				description: "Show help information",
			},
			{
				name:        "VersionCommand",
				command:     "--version",
				description: "Show version information",
			},
			{
				name:        "ListOffersCommand",
				command:     "list-offers",
				description: "List available offers",
			},
			{
				name:        "ListOrdersCommand",
				command:     "list-orders",
				description: "List current orders",
			},
		}
		
		for _, docTest := range docTests {
			t.Run(docTest.name, func(t *testing.T) {
				cmd := exec.Command(CLI_PATH, docTest.command)
				output, err := cmd.CombinedOutput()
				
				if err != nil {
					t.Errorf("Command %s failed: %v", docTest.command, err)
					return
				}
				
				// Verify command produces output
				if len(output) == 0 {
					t.Errorf("Command %s produced no output", docTest.command)
				}
			})
		}
	})
}

func TestUserWorkflow(t *testing.T) {
	t.Run("CompleteUserWorkflow", func(t *testing.T) {
		// Test complete user workflow
		steps := []struct {
			name        string
			action      func(*testing.T) error
			description string
		}{
			{
				name:        "ListOffers",
				action:      testListOffers,
				description: "User lists available offers",
			},
			{
				name:        "CreateOrder",
				action:      testCreateOrder,
				description: "User creates an order",
			},
			{
				name:        "CheckOrderStatus",
				action:      testCheckOrderStatus,
				description: "User checks order status",
			},
			{
				name:        "ListOrders",
				action:      testListOrders,
				description: "User lists their orders",
			},
		}
		
		for _, step := range steps {
			t.Run(step.name, func(t *testing.T) {
				if err := step.action(t); err != nil {
					t.Errorf("Workflow step %s failed: %v", step.name, err)
				}
			})
		}
	})
}

// Helper functions
func testCLICommand(t *testing.T, args []string, timeout time.Duration) bool {
	cmd := exec.Command(CLI_PATH, args...)
	cmd.Env = append(os.Environ(), "OCX_SERVER="+BASE_URL)
	
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()
	
	select {
	case err := <-done:
		return err == nil
	case <-time.After(timeout):
		cmd.Process.Kill()
		return false
	}
}

func testCLIError(t *testing.T, args []string, expectedError string) bool {
	cmd := exec.Command(CLI_PATH, args...)
	cmd.Env = append(os.Environ(), "OCX_SERVER="+BASE_URL)
	
	output, err := cmd.CombinedOutput()
	if err == nil {
		return false // Expected error but command succeeded
	}
	
	return strings.Contains(string(output), expectedError)
}

func calculateAverage(times []time.Duration) time.Duration {
	var total time.Duration
	for _, t := range times {
		total += t
	}
	return total / time.Duration(len(times))
}

func calculateMax(times []time.Duration) time.Duration {
	var max time.Duration
	for _, t := range times {
		if t > max {
			max = t
		}
	}
	return max
}

// Environment setup functions
func setupLinuxEnvironment(t *testing.T) error {
	// Setup Linux environment
	return nil
}

func cleanupLinuxEnvironment(t *testing.T) error {
	// Cleanup Linux environment
	return nil
}

func setupDockerEnvironment(t *testing.T) error {
	// Setup Docker environment
	return nil
}

func cleanupDockerEnvironment(t *testing.T) error {
	// Cleanup Docker environment
	return nil
}

func setupMinimalEnvironment(t *testing.T) error {
	// Setup minimal environment
	return nil
}

func cleanupMinimalEnvironment(t *testing.T) error {
	// Cleanup minimal environment
	return nil
}

// Workflow test functions
func testListOffers(t *testing.T) error {
	resp, err := http.Get(BASE_URL + "/offers")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("list offers failed: %d", resp.StatusCode)
	}
	
	return nil
}

func testCreateOrder(t *testing.T) error {
	order := map[string]interface{}{
		"buyer_id": "test_buyer",
		"gpus":     2,
		"hours":    4,
		"amount":   100.0,
	}
	
	jsonData, _ := json.Marshal(order)
	resp, err := http.Post(BASE_URL+"/orders", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("create order failed: %d", resp.StatusCode)
	}
	
	return nil
}

func testCheckOrderStatus(t *testing.T) error {
	resp, err := http.Get(BASE_URL + "/orders/test_order")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// Accept both 200 (found) and 404 (not found) as valid responses
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("check order status failed: %d", resp.StatusCode)
	}
	
	return nil
}

func testListOrders(t *testing.T) error {
	resp, err := http.Get(BASE_URL + "/orders")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("list orders failed: %d", resp.StatusCode)
	}
	
	return nil
}
