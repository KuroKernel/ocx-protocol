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
	)
	flag.Parse()

	if *command == "" {
		fmt.Fprintf(os.Stderr, "Available commands:\n")
		fmt.Fprintf(os.Stderr, "  create-provider  - Create a provider identity\n")
		fmt.Fprintf(os.Stderr, "  create-buyer     - Create a buyer identity\n")
		fmt.Fprintf(os.Stderr, "  make-offer       - Publish a compute offer\n")
		fmt.Fprintf(os.Stderr, "  place-order      - Place a compute order\n")
		fmt.Fprintf(os.Stderr, "  list-offers      - List available offers\n")
		fmt.Fprintf(os.Stderr, "  list-leases      - List all leases\n")
		fmt.Fprintf(os.Stderr, "  active-leases    - List active leases\n")
		fmt.Fprintf(os.Stderr, "  market-stats     - Show market statistics\n")
		fmt.Fprintf(os.Stderr, "  update-lease     - Update lease state\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -command create-provider -name \"ACME GPU Farm\" -email \"contact@acme.com\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -command make-offer -gpus 8 -hours 168 -price 2.50\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -command place-order -gpus 4 -hours 8 -budget 80.0\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -command update-lease -lease-id 12345 -state running\n", os.Args[0])
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
		
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		os.Exit(1)
	}
}
