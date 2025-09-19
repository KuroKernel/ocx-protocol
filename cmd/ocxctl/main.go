// main.go — Enhanced OCX CLI Tool
// Includes matching, leasing, and market operations

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Import the OCX types (assuming they're in the same package for this demo)
// In production, these would be imported from ocx.local/pkg/ocx

type EnhancedOCXClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewEnhancedOCXClient(baseURL string) *EnhancedOCXClient {
	return &EnhancedOCXClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateProvider creates a provider identity
func (c *EnhancedOCXClient) CreateProvider(name, email string) error {
	request := map[string]interface{}{
		"role":         "provider",
		"display_name": name,
		"email":        email,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/identities", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("server error: %s", string(body))
	}

	fmt.Printf("Provider created successfully:\n%s\n", string(body))
	return nil
}

// CreateBuyer creates a buyer identity
func (c *EnhancedOCXClient) CreateBuyer(name, email string) error {
	request := map[string]interface{}{
		"role":         "buyer",
		"display_name": name,
		"email":        email,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/identities", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create buyer: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("server error: %s", string(body))
	}

	fmt.Printf("Buyer created successfully:\n%s\n", string(body))
	return nil
}

// MakeOffer creates and publishes an offer
func (c *EnhancedOCXClient) MakeOffer(gpuCount int, hours int, priceUSD float64) error {
	// Create a sample offer without signature for demo
	offer := map[string]interface{}{
		"offer_id":    generateULID(),
		"version":     map[string]int{"major": 0, "minor": 1, "patch": 0},
		"provider":    map[string]string{"party_id": "demo-provider", "role": "provider"},
		"fleet_id":    generateULID(),
		"unit":        "gpu_hour",
		"unit_price":  map[string]interface{}{"currency": "USD", "amount": fmt.Sprintf("%.2f", priceUSD), "scale": 2},
		"min_hours":   1,
		"max_hours":   hours,
		"min_gpus":    1,
		"max_gpus":    gpuCount,
		"valid_from":  time.Now().Format(time.RFC3339),
		"valid_to":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"compliance":  []string{"GDPR"},
	}

	// Create envelope
	envelope := map[string]interface{}{
		"id":        generateULID(),
		"kind":      "offer",
		"version":   map[string]int{"major": 0, "minor": 1, "patch": 0},
		"issued_at": time.Now().Format(time.RFC3339),
		"payload":   offer,
		"hash":      map[string]string{"alg": "sha256", "value": "demo-hash"},
		"sig":       map[string]string{"alg": "ed25519", "key_id": "", "sig_b64": ""}, // Empty for demo
	}

	return c.sendRequest("/offers", envelope)
}

// PlaceOrder places an order for GPU resources
func (c *EnhancedOCXClient) PlaceOrder(gpuCount int, hours int, budgetUSD float64) error {
	// Create a sample order without signature for demo
	order := map[string]interface{}{
		"order_id":       generateULID(),
		"version":        map[string]int{"major": 0, "minor": 1, "patch": 0},
		"buyer":          map[string]string{"party_id": "demo-buyer", "role": "buyer"},
		"offer_id":       "", // Will be matched by the engine
		"requested_gpus": gpuCount,
		"hours":          hours,
		"budget_cap":     map[string]interface{}{"currency": "USD", "amount": fmt.Sprintf("%.2f", budgetUSD), "scale": 2},
		"state":          "pending",
		"created_at":     time.Now().Format(time.RFC3339),
		"updated_at":     time.Now().Format(time.RFC3339),
	}

	// Create envelope
	envelope := map[string]interface{}{
		"id":        generateULID(),
		"kind":      "order",
		"version":   map[string]int{"major": 0, "minor": 1, "patch": 0},
		"issued_at": time.Now().Format(time.RFC3339),
		"payload":   order,
		"hash":      map[string]string{"alg": "sha256", "value": "demo-hash"},
		"sig":       map[string]string{"alg": "ed25519", "key_id": "", "sig_b64": ""}, // Empty for demo
	}

	return c.sendRequest("/orders", envelope)
}

// ListOffers lists all available offers
func (c *EnhancedOCXClient) ListOffers() error {
	resp, err := c.httpClient.Get(c.baseURL + "/offers")
	if err != nil {
		return fmt.Errorf("failed to get offers: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Println("Available Offers:")
	c.prettyPrint(body)
	return nil
}

// ListLeases lists all leases
func (c *EnhancedOCXClient) ListLeases() error {
	resp, err := c.httpClient.Get(c.baseURL + "/leases")
	if err != nil {
		return fmt.Errorf("failed to get leases: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Println("All Leases:")
	c.prettyPrint(body)
	return nil
}

// GetMarketStats shows market statistics
func (c *EnhancedOCXClient) GetMarketStats() error {
	resp, err := c.httpClient.Get(c.baseURL + "/market/stats")
	if err != nil {
		return fmt.Errorf("failed to get market stats: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Println("Market Statistics:")
	c.prettyPrint(body)
	return nil
}

// GetActiveLeases shows currently active leases
func (c *EnhancedOCXClient) GetActiveLeases() error {
	resp, err := c.httpClient.Get(c.baseURL + "/market/active")
	if err != nil {
		return fmt.Errorf("failed to get active leases: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Println("Active Leases:")
	c.prettyPrint(body)
	return nil
}

// UpdateLeaseState updates the state of a lease
func (c *EnhancedOCXClient) UpdateLeaseState(leaseID, state string) error {
	request := map[string]string{"state": state}
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("PUT", c.baseURL+"/leases/"+leaseID+"/state", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update lease state: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %s", string(body))
	}

	fmt.Printf("Lease state updated:\n")
	c.prettyPrint(body)
	return nil
}

// sendRequest sends a POST request with JSON data
func (c *EnhancedOCXClient) sendRequest(endpoint string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Request successful:\n")
	c.prettyPrint(body)
	return nil
}

// ExecuteCode executes code with receipt generation
func (c *EnhancedOCXClient) ExecuteCode(leaseID, artifact, input string, maxCycles int) error {
	request := map[string]interface{}{
		"lease_id":   leaseID,
		"artifact":   []byte(artifact),
		"input":      []byte(input),
		"max_cycles": maxCycles,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/v1/execute", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to execute code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("execution failed (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Println("Code execution successful:")
	c.prettyPrint(body)
	return nil
}

// VerifyReceipt verifies a cryptographic receipt
func (c *EnhancedOCXClient) VerifyReceipt(receiptBlob string) error {
	request := map[string]string{
		"receipt_blob": receiptBlob,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/v1/verify", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to verify receipt: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Println("Receipt verification result:")
	c.prettyPrint(body)
	return nil
}

// ListReceipts queries execution history
func (c *EnhancedOCXClient) ListReceipts(leaseID string) error {
	url := c.baseURL + "/api/v1/receipts"
	if leaseID != "" {
		url += "?lease_id=" + leaseID
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to get receipts: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Println("Execution History:")
	c.prettyPrint(body)
	return nil
}

// prettyPrint formats JSON output nicely
func (c *EnhancedOCXClient) prettyPrint(data []byte) {
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		fmt.Println(string(data))
		return
	}
	
	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		fmt.Println(string(data))
		return
	}
	
	fmt.Println(string(prettyJSON))
}

func generateULID() string {
	// Simplified ULID generation for demo
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func main() {
	var (
		serverURL = flag.String("server", "http://localhost:8080", "OCX server URL")
		command   = flag.String("command", "", "Command to execute")
		
		// Identity commands
		name  = flag.String("name", "", "Display name for identity")
		email = flag.String("email", "", "Email for identity")
		
		// Offer commands
		gpuCount = flag.Int("gpus", 8, "Number of GPUs")
		hours    = flag.Int("hours", 24, "Duration in hours")
		price    = flag.Float64("price", 2.50, "Price per GPU hour in USD")
		
		// Order commands
		budget = flag.Float64("budget", 100.0, "Budget in USD")
		
		// Lease commands
		leaseID = flag.String("lease-id", "", "Lease ID")
		state   = flag.String("state", "", "New lease state")
		
		// Advanced feature flags
		tenantID    = flag.String("tenant-id", "", "Tenant ID for operations")
		artifact    = flag.String("artifact", "", "Artifact data for execution")
		input       = flag.String("input", "", "Input data for execution")
		maxCycles   = flag.Uint64("max-cycles", 1000, "Maximum cycles for execution")
		receipt     = flag.String("receipt", "", "Receipt blob for verification")
		features    = flag.String("features", "", "Advanced features to enable (comma-separated)")
		regions     = flag.String("regions", "", "Regions for global execution (comma-separated)")
		scope       = flag.String("scope", "global", "Optimization scope (global, regional, sectoral)")
		modelHash   = flag.String("model-hash", "", "Model hash for AI operations")
		inputHash   = flag.String("input-hash", "", "Input hash for AI operations")
		outputHash  = flag.String("output-hash", "", "Output hash for AI operations")
		epochs      = flag.Uint("epochs", 10, "Number of training epochs")
		learningRate = flag.Float64("learning-rate", 0.001, "Learning rate for training")
	)
	flag.Parse()

	if *command == "" {
		fmt.Fprintf(os.Stderr, "OCX Control Tool - Enhanced CLI with Advanced Features\n")
		fmt.Fprintf(os.Stderr, "======================================================\n")
		fmt.Fprintf(os.Stderr, "Core Commands:\n")
		fmt.Fprintf(os.Stderr, "  create-provider  - Create a provider identity\n")
		fmt.Fprintf(os.Stderr, "  create-buyer     - Create a buyer identity\n")
		fmt.Fprintf(os.Stderr, "  make-offer       - Publish a compute offer\n")
		fmt.Fprintf(os.Stderr, "  place-order      - Place a compute order\n")
		fmt.Fprintf(os.Stderr, "  list-offers      - List available offers\n")
		fmt.Fprintf(os.Stderr, "  list-leases      - List all leases\n")
		fmt.Fprintf(os.Stderr, "  active-leases    - List active leases\n")
		fmt.Fprintf(os.Stderr, "  market-stats     - Show market statistics\n")
		fmt.Fprintf(os.Stderr, "  update-lease     - Update lease state\n")
		fmt.Fprintf(os.Stderr, "  execute          - Execute code with receipt generation\n")
		fmt.Fprintf(os.Stderr, "  verify-receipt   - Verify cryptographic receipts\n")
		fmt.Fprintf(os.Stderr, "  list-receipts    - Query execution history\n")
		fmt.Fprintf(os.Stderr, "\nEnterprise Features:\n")
		fmt.Fprintf(os.Stderr, "  compliance-dashboard - Get compliance dashboard\n")
		fmt.Fprintf(os.Stderr, "  sla-status          - Get SLA status\n")
		fmt.Fprintf(os.Stderr, "  list-tenants        - List all tenants\n")
		fmt.Fprintf(os.Stderr, "  audit-trail         - Get audit trail\n")
		fmt.Fprintf(os.Stderr, "\nFinancial Features:\n")
		fmt.Fprintf(os.Stderr, "  list-futures        - List compute futures\n")
		fmt.Fprintf(os.Stderr, "  create-future       - Create compute future\n")
		fmt.Fprintf(os.Stderr, "  list-bonds          - List compute bonds\n")
		fmt.Fprintf(os.Stderr, "  list-carbon-credits - List carbon credits\n")
		fmt.Fprintf(os.Stderr, "  market-status       - Get market status\n")
		fmt.Fprintf(os.Stderr, "\nAI Features:\n")
		fmt.Fprintf(os.Stderr, "  ai-inference        - Execute AI inference\n")
		fmt.Fprintf(os.Stderr, "  ai-training         - Execute AI training\n")
		fmt.Fprintf(os.Stderr, "  list-models         - List AI models\n")
		fmt.Fprintf(os.Stderr, "  verify-ai           - Verify AI computation\n")
		fmt.Fprintf(os.Stderr, "\nGlobal Features:\n")
		fmt.Fprintf(os.Stderr, "  global-execute      - Execute globally\n")
		fmt.Fprintf(os.Stderr, "  optimize-planetary  - Optimize planetary resources\n")
		fmt.Fprintf(os.Stderr, "  global-status       - Get global status\n")
		fmt.Fprintf(os.Stderr, "  global-metrics      - Get global metrics\n")
		fmt.Fprintf(os.Stderr, "\nAdvanced Execution:\n")
		fmt.Fprintf(os.Stderr, "  execute-advanced    - Execute with advanced features\n")
		fmt.Fprintf(os.Stderr, "  execute-batch       - Execute batch computation\n")
		fmt.Fprintf(os.Stderr, "  execute-stream      - Execute stream computation\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -command create-provider -name \"ACME GPU Farm\" -email \"contact@acme.com\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -command make-offer -gpus 8 -hours 168 -price 2.50\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -command place-order -gpus 4 -hours 8 -budget 80.0\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -command update-lease -lease-id 12345 -state running\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -command compliance-dashboard -tenant-id tenant-1\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -command ai-inference -model-hash abc123 -input-hash def456\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -command global-execute -regions us-east,eu-west\n", os.Args[0])
		os.Exit(1)
	}

	client := NewEnhancedOCXClient(*serverURL)

	switch *command {
	case "create-provider":
		if *name == "" || *email == "" {
			fmt.Fprintf(os.Stderr, "Name and email required for create-provider\n")
			os.Exit(1)
		}
		if err := client.CreateProvider(*name, *email); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create provider: %v\n", err)
			os.Exit(1)
		}
		
	case "create-buyer":
		if *name == "" || *email == "" {
			fmt.Fprintf(os.Stderr, "Name and email required for create-buyer\n")
			os.Exit(1)
		}
		if err := client.CreateBuyer(*name, *email); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create buyer: %v\n", err)
			os.Exit(1)
		}
		
	case "make-offer":
		if err := client.MakeOffer(*gpuCount, *hours, *price); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to make offer: %v\n", err)
			os.Exit(1)
		}
		
	case "place-order":
		if err := client.PlaceOrder(*gpuCount, *hours, *budget); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to place order: %v\n", err)
			os.Exit(1)
		}
		
	case "list-offers":
		if err := client.ListOffers(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list offers: %v\n", err)
			os.Exit(1)
		}
		
	case "list-leases":
		if err := client.ListLeases(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list leases: %v\n", err)
			os.Exit(1)
		}
		
	case "active-leases":
		if err := client.GetActiveLeases(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get active leases: %v\n", err)
			os.Exit(1)
		}
		
	case "market-stats":
		if err := client.GetMarketStats(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get market stats: %v\n", err)
			os.Exit(1)
		}
		
	case "update-lease":
		if *leaseID == "" || *state == "" {
			fmt.Fprintf(os.Stderr, "Lease ID and state required for update-lease\n")
			fmt.Fprintf(os.Stderr, "Valid states: provisioning, running, paused, completed, breached, cancelled\n")
			os.Exit(1)
		}
		if err := client.UpdateLeaseState(*leaseID, *state); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to update lease state: %v\n", err)
			os.Exit(1)
		}
		
	case "execute":
		if *leaseID == "" {
			fmt.Fprintf(os.Stderr, "Lease ID required for execute\n")
			os.Exit(1)
		}
		if err := client.ExecuteCode(*leaseID, "example_artifact", "example_input", 1000); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to execute code: %v\n", err)
			os.Exit(1)
		}
		
	case "verify-receipt":
		if *leaseID == "" {
			fmt.Fprintf(os.Stderr, "Receipt blob required for verify-receipt\n")
			os.Exit(1)
		}
		if err := client.VerifyReceipt(*leaseID); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to verify receipt: %v\n", err)
			os.Exit(1)
		}
		
	case "list-receipts":
		if err := client.ListReceipts(*leaseID); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list receipts: %v\n", err)
			os.Exit(1)
		}
		
	// Enterprise Features
	case "compliance-dashboard":
		if *tenantID == "" {
			fmt.Fprintf(os.Stderr, "Tenant ID required for compliance-dashboard\n")
			os.Exit(1)
		}
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.GetComplianceDashboard(*tenantID); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get compliance dashboard: %v\n", err)
			os.Exit(1)
		}
		
	case "sla-status":
		if *tenantID == "" {
			fmt.Fprintf(os.Stderr, "Tenant ID required for sla-status\n")
			os.Exit(1)
		}
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.GetSLAStatus(*tenantID); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get SLA status: %v\n", err)
			os.Exit(1)
		}
		
	case "list-tenants":
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.ListTenants(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list tenants: %v\n", err)
			os.Exit(1)
		}
		
	case "audit-trail":
		if *tenantID == "" {
			fmt.Fprintf(os.Stderr, "Tenant ID required for audit-trail\n")
			os.Exit(1)
		}
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.GetAuditTrail(*tenantID); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get audit trail: %v\n", err)
			os.Exit(1)
		}
		
	// Financial Features
	case "list-futures":
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.ListComputeFutures(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list futures: %v\n", err)
			os.Exit(1)
		}
		
	case "create-future":
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.CreateComputeFuture("buyer-1", "seller-1", "gpu_h100", 1000000, 10); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create future: %v\n", err)
			os.Exit(1)
		}
		
	case "list-bonds":
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.ListComputeBonds(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list bonds: %v\n", err)
			os.Exit(1)
		}
		
	case "list-carbon-credits":
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.ListCarbonCredits(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list carbon credits: %v\n", err)
			os.Exit(1)
		}
		
	case "market-status":
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.GetMarketStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get market status: %v\n", err)
			os.Exit(1)
		}
		
	// AI Features
	case "ai-inference":
		if *modelHash == "" || *inputHash == "" {
			fmt.Fprintf(os.Stderr, "Model hash and input hash required for ai-inference\n")
			os.Exit(1)
		}
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.ExecuteAIInference(*modelHash, *inputHash); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to execute AI inference: %v\n", err)
			os.Exit(1)
		}
		
	case "ai-training":
		if *modelHash == "" || *inputHash == "" {
			fmt.Fprintf(os.Stderr, "Model hash and input hash required for ai-training\n")
			os.Exit(1)
		}
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.ExecuteAITraining(*inputHash, *modelHash, uint32(*epochs), *learningRate); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to execute AI training: %v\n", err)
			os.Exit(1)
		}
		
	case "list-models":
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.ListModels(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list models: %v\n", err)
			os.Exit(1)
		}
		
	case "verify-ai":
		if *modelHash == "" || *inputHash == "" || *outputHash == "" {
			fmt.Fprintf(os.Stderr, "Model hash, input hash, and output hash required for verify-ai\n")
			os.Exit(1)
		}
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.VerifyAI("mock_proof", *modelHash, *inputHash, *outputHash); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to verify AI: %v\n", err)
			os.Exit(1)
		}
		
	// Global Features
	case "global-execute":
		regions := []string{"us-east", "eu-west"}
		if *regions != "" {
			regions = strings.Split(*regions, ",")
		}
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.ExecuteGlobal("job-1", regions); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to execute globally: %v\n", err)
			os.Exit(1)
		}
		
	case "optimize-planetary":
		regions := []string{"us-east", "eu-west"}
		if *regions != "" {
			regions = strings.Split(*regions, ",")
		}
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.OptimizePlanetary(*scope, regions); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to optimize planetary: %v\n", err)
			os.Exit(1)
		}
		
	case "global-status":
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.GetGlobalStatus(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get global status: %v\n", err)
			os.Exit(1)
		}
		
	case "global-metrics":
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.GetGlobalMetrics(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get global metrics: %v\n", err)
			os.Exit(1)
		}
		
	// Advanced Execution
	case "execute-advanced":
		if *tenantID == "" || *artifact == "" || *input == "" {
			fmt.Fprintf(os.Stderr, "Tenant ID, artifact, and input required for execute-advanced\n")
			os.Exit(1)
		}
		features := []string{"compliance", "futures"}
		if *features != "" {
			features = strings.Split(*features, ",")
		}
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.ExecuteAdvanced(*tenantID, *artifact, *input, *maxCycles, features); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to execute advanced: %v\n", err)
			os.Exit(1)
		}
		
	case "execute-batch":
		// Mock batch execution
		requests := []map[string]interface{}{
			{"tenant_id": "tenant-1", "artifact": "artifact-1", "input": "input-1"},
			{"tenant_id": "tenant-2", "artifact": "artifact-2", "input": "input-2"},
		}
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.ExecuteBatch(requests); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to execute batch: %v\n", err)
			os.Exit(1)
		}
		
	case "execute-stream":
		advancedClient := NewAdvancedOCXClient(*serverURL)
		if err := advancedClient.ExecuteStream(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to execute stream: %v\n", err)
			os.Exit(1)
		}
		
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		os.Exit(1)
	}
}
