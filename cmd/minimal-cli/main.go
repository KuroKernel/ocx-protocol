// minimal-cli/main.go — Minimal OCX CLI
// Emergency simplified version with core functionality only

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

type MinimalCLI struct {
	baseURL string
	client  *http.Client
}

func NewMinimalCLI(baseURL string) *MinimalCLI {
	return &MinimalCLI{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *MinimalCLI) execute(artifact, input string, maxCycles uint64, leaseID string) error {
	req := map[string]interface{}{
		"artifact":    artifact,
		"input":       input,
		"max_cycles":  maxCycles,
		"lease_id":    leaseID,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/api/v1/execute", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to execute: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("execution failed (%d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Println("✅ Execution successful!")
	fmt.Printf("Result: %+v\n", result)
	return nil
}

func (c *MinimalCLI) verify(receiptBlob string) error {
	req := map[string]interface{}{
		"receipt_blob": receiptBlob,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/api/v1/verify", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to verify: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("verification failed (%d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Println("✅ Verification successful!")
	fmt.Printf("Result: %+v\n", result)
	return nil
}

func (c *MinimalCLI) listReceipts() error {
	resp, err := c.client.Get(c.baseURL + "/api/v1/receipts")
	if err != nil {
		return fmt.Errorf("failed to list receipts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to list receipts (%d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Println("📋 Receipts:")
	fmt.Printf("Count: %v\n", result["count"])
	if receipts, ok := result["receipts"].([]interface{}); ok {
		for i, receipt := range receipts {
			fmt.Printf("%d. %+v\n", i+1, receipt)
		}
	}
	return nil
}

func (c *MinimalCLI) health() error {
	resp, err := c.client.Get(c.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed (%d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Println("🏥 Health check:")
	fmt.Printf("Status: %v\n", result["status"])
	fmt.Printf("Version: %v\n", result["version"])
	fmt.Printf("Timestamp: %v\n", result["timestamp"])
	return nil
}

func printUsage() {
	fmt.Println("OCX Minimal CLI - Emergency Simplified Version")
	fmt.Println("==============================================")
	fmt.Println("Usage: minimal-cli <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  execute     - Execute code with receipt generation")
	fmt.Println("  verify      - Verify a cryptographic receipt")
	fmt.Println("  list        - List all receipts")
	fmt.Println("  health      - Check server health")
	fmt.Println("  help        - Show this help message")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -server <url>        Server URL (default: http://localhost:8080)")
	fmt.Println("  -artifact <data>     Artifact data for execution")
	fmt.Println("  -input <data>        Input data for execution")
	fmt.Println("  -max-cycles <count>  Maximum cycles for execution")
	fmt.Println("  -lease-id <id>       Lease ID for execution")
	fmt.Println("  -receipt <blob>      Receipt blob for verification")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  minimal-cli health")
	fmt.Println("  minimal-cli execute -artifact 'hello' -input 'world' -max-cycles 1000 -lease-id 'lease-1'")
	fmt.Println("  minimal-cli verify -receipt 'mock_receipt_blob'")
	fmt.Println("  minimal-cli list")
}

func main() {
	var (
		serverURL = flag.String("server", "http://localhost:8080", "OCX server URL")
		command   = flag.String("command", "", "Command to execute")
		
		// Execution options
		artifact   = flag.String("artifact", "hello", "Artifact data for execution")
		input      = flag.String("input", "world", "Input data for execution")
		maxCycles  = flag.Uint64("max-cycles", 1000, "Maximum cycles for execution")
		leaseID    = flag.String("lease-id", "lease-1", "Lease ID for execution")
		
		// Verification options
		receipt = flag.String("receipt", "mock_receipt_blob", "Receipt blob for verification")
	)
	flag.Parse()

	// Handle command line arguments
	if len(os.Args) > 1 && os.Args[1] != "-command" && !strings.HasPrefix(os.Args[1], "-") {
		*command = os.Args[1]
	}

	if *command == "" {
		printUsage()
		os.Exit(1)
	}

	cli := NewMinimalCLI(*serverURL)

	switch *command {
	case "execute":
		if err := cli.execute(*artifact, *input, *maxCycles, *leaseID); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Execute failed: %v\n", err)
			os.Exit(1)
		}
		
	case "verify":
		if err := cli.verify(*receipt); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Verify failed: %v\n", err)
			os.Exit(1)
		}
		
	case "list":
		if err := cli.listReceipts(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ List failed: %v\n", err)
			os.Exit(1)
		}
		
	case "health":
		if err := cli.health(); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Health check failed: %v\n", err)
			os.Exit(1)
		}
		
	case "help":
		printUsage()
		
	default:
		fmt.Fprintf(os.Stderr, "❌ Unknown command: %s\n", *command)
		printUsage()
		os.Exit(1)
	}
}
