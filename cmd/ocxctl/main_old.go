// main.go — OCX CLI Tool
// go 1.22+

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

	"github.com/ocx/protocol/pkg/ocx"
)

type OCXClient struct {
	baseURL    string
	keyManager *ocx.KeyManager
	identity   *ocx.Identity
}

func NewOCXClient(baseURL string) (*OCXClient, error) {
	keyManager := ocx.NewKeyManager()
	
	// Create a default identity for demo purposes
	identity, err := keyManager.CreateIdentity("provider", "Demo Provider", "demo@example.com")
	if err != nil {
		return nil, fmt.Errorf("failed to create identity: %w", err)
	}

	return &OCXClient{
		baseURL:    baseURL,
		keyManager: keyManager,
		identity:   identity,
	}, nil
}

func (c *OCXClient) MakeOffer() error {
	// Create a sample offer
	offer := &ocx.Offer{
		OfferID:   generateULID(),
		Version:   ocx.V010,
		Provider:  ocx.PartyRef{PartyID: c.identity.PartyID, Role: "provider"},
		FleetID:   generateULID(),
		Unit:      ocx.PricePerGPUHour,
		UnitPrice: ocx.Money{Currency: "USD", Amount: "2.50", Scale: 2},
		MinHours:  1,
		MaxHours:  168, // 1 week
		MinGPUs:   1,
		MaxGPUs:   8,
		ValidFrom: time.Now(),
		ValidTo:   time.Now().Add(24 * time.Hour),
		Compliance: []string{"GDPR"},
	}

	// Create envelope
	envelope := &ocx.Envelope{
		ID:        generateULID(),
		Kind:      ocx.KindOffer,
		Version:   ocx.V010,
		IssuedAt:  time.Now(),
		Payload:   offer,
		Hash:      ocx.HashMessage([]byte("")), // Simplified for demo
	}

	// Sign the envelope
	keyID := c.identity.Keys[0].KeyID
	if err := c.keyManager.SignEnvelope(keyID, envelope); err != nil {
		return fmt.Errorf("failed to sign envelope: %w", err)
	}

	// Send to server
	return c.sendEnvelope("/offers", envelope)
}

func (c *OCXClient) PlaceOrder(offerID string) error {
	// Create a sample order
	order := &ocx.Order{
		OrderID:       generateULID(),
		Version:       ocx.V010,
		Buyer:         ocx.PartyRef{PartyID: c.identity.PartyID, Role: "buyer"},
		OfferID:       ocx.ID(offerID),
		RequestedGPUs: 2,
		Hours:         4,
		BudgetCap:     &ocx.Money{Currency: "USD", Amount: "20.00", Scale: 2},
		State:         ocx.OrderPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Create envelope
	envelope := &ocx.Envelope{
		ID:        generateULID(),
		Kind:      ocx.KindOrder,
		Version:   ocx.V010,
		IssuedAt:  time.Now(),
		Payload:   order,
		Hash:      ocx.HashMessage([]byte("")), // Simplified for demo
	}

	// Sign the envelope
	keyID := c.identity.Keys[0].KeyID
	if err := c.keyManager.SignEnvelope(keyID, envelope); err != nil {
		return fmt.Errorf("failed to sign envelope: %w", err)
	}

	// Send to server
	return c.sendEnvelope("/orders", envelope)
}

func (c *OCXClient) ListOffers() error {
	resp, err := http.Get(c.baseURL + "/offers")
	if err != nil {
		return fmt.Errorf("failed to get offers: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Println("Available Offers:")
	fmt.Println(string(body))
	return nil
}

func (c *OCXClient) ListOrders() error {
	resp, err := http.Get(c.baseURL + "/orders")
	if err != nil {
		return fmt.Errorf("failed to get orders: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Println("Current Orders:")
	fmt.Println(string(body))
	return nil
}

func (c *OCXClient) sendEnvelope(endpoint string, envelope *ocx.Envelope) error {
	jsonData, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}

	resp, err := http.Post(c.baseURL+endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: %s", string(body))
	}

	fmt.Printf("Success: %s\n", string(body))
	return nil
}

func generateULID() ocx.ID {
	// Simplified ULID generation for demo
	return ocx.ID(fmt.Sprintf("%d", time.Now().UnixNano()))
}

func main() {
	var (
		serverURL = flag.String("server", "http://localhost:8080", "OCX server URL")
		command   = flag.String("command", "", "Command to execute (make-offer, place-order, list-offers, list-orders)")
		offerID   = flag.String("offer-id", "", "Offer ID for place-order command")
	)
	flag.Parse()

	client, err := NewOCXClient(*serverURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
		os.Exit(1)
	}

	switch *command {
	case "make-offer":
		if err := client.MakeOffer(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to make offer: %v\n", err)
			os.Exit(1)
		}
	case "place-order":
		if *offerID == "" {
			fmt.Fprintf(os.Stderr, "Offer ID required for place-order command\n")
			os.Exit(1)
		}
		if err := client.PlaceOrder(*offerID); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to place order: %v\n", err)
			os.Exit(1)
		}
	case "list-offers":
		if err := client.ListOffers(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list offers: %v\n", err)
			os.Exit(1)
		}
	case "list-orders":
		if err := client.ListOrders(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list orders: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		fmt.Fprintf(os.Stderr, "Available commands: make-offer, place-order, list-offers, list-orders\n")
		os.Exit(1)
	}
}
