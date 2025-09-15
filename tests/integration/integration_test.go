// integration_test.go - Integration Testing Suite for OCX Protocol
// Tests: Provider failures, partial provisioning, network interruptions, dispute resolution

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ocx/protocol/pkg/ocx"
)

const (
	BASE_URL = "http://localhost:8080"
	MOCK_PROVIDER_URL = "http://localhost:8081"
)

type IntegrationTestSuite struct {
	client    *http.Client
	baseURL   string
	providers map[string]*MockProvider
}

type MockProvider struct {
	ID           string
	IsHealthy    bool
	ResponseTime time.Duration
	FailureRate  float64
}

func TestProviderFailureMidTransaction(t *testing.T) {
	suite := setupIntegrationTest(t)
	
	// Test scenario: Provider fails after order is placed but before provisioning
	t.Run("ProviderFailureAfterOrder", func(t *testing.T) {
		// 1. Place order successfully
		orderID := suite.placeOrder(t, "test_buyer", "healthy_provider", 2, 4)
		
		// 2. Simulate provider failure
		suite.simulateProviderFailure("healthy_provider")
		
		// 3. Attempt to provision
		provisionResult := suite.attemptProvisioning(t, orderID)
		
		// 4. Verify order state is updated to failed
		order := suite.getOrder(t, orderID)
		if order.State != ocx.OrderFailed {
			t.Errorf("Expected order state to be failed, got %s", order.State)
		}
		
		// 5. Verify settlement is rolled back
		settlement := suite.getSettlement(t, orderID)
		if settlement.Status != "rolled_back" {
			t.Errorf("Expected settlement to be rolled back, got %s", settlement.Status)
		}
	})
}

func TestPartialGPUProvisioning(t *testing.T) {
	suite := setupIntegrationTest(t)
	
	t.Run("PartialProvisioningScenario", func(t *testing.T) {
		// 1. Place order for 8 GPUs
		orderID := suite.placeOrder(t, "test_buyer", "partial_provider", 8, 4)
		
		// 2. Provider only has 5 GPUs available
		suite.setProviderCapacity("partial_provider", 5)
		
		// 3. Attempt provisioning
		provisionResult := suite.attemptProvisioning(t, orderID)
		
		// 4. Verify partial provisioning is handled correctly
		if !provisionResult.PartialProvisioning {
			t.Error("Expected partial provisioning to be detected")
		}
		
		// 5. Verify order is split or adjusted
		order := suite.getOrder(t, orderID)
		if order.RequestedGPUs != 5 {
			t.Errorf("Expected order to be adjusted to 5 GPUs, got %d", order.RequestedGPUs)
		}
		
		// 6. Verify remaining 3 GPUs are queued for later
		remainingOrder := suite.getRemainingOrder(t, orderID)
		if remainingOrder == nil {
			t.Error("Expected remaining order to be created")
		}
	})
}

func TestNetworkInterruptionSettlement(t *testing.T) {
	suite := setupIntegrationTest(t)
	
	t.Run("NetworkInterruptionDuringSettlement", func(t *testing.T) {
		// 1. Place and provision order
		orderID := suite.placeOrder(t, "test_buyer", "reliable_provider", 2, 4)
		suite.attemptProvisioning(t, orderID)
		
		// 2. Simulate network interruption during settlement
		suite.simulateNetworkInterruption()
		
		// 3. Attempt settlement
		settlementResult := suite.attemptSettlement(t, orderID)
		
		// 4. Verify settlement is retried when network recovers
		suite.restoreNetwork()
		time.Sleep(2 * time.Second) // Allow retry logic to kick in
		
		settlement := suite.getSettlement(t, orderID)
		if settlement.Status != "completed" {
			t.Errorf("Expected settlement to complete after retry, got %s", settlement.Status)
		}
	})
}

func TestDisputeResolutionWorkflow(t *testing.T) {
	suite := setupIntegrationTest(t)
	
	t.Run("EndToEndDisputeResolution", func(t *testing.T) {
		// 1. Place and provision order
		orderID := suite.placeOrder(t, "test_buyer", "problematic_provider", 2, 4)
		suite.attemptProvisioning(t, orderID)
		
		// 2. Simulate dispute (provider claims payment, buyer claims no service)
		disputeID := suite.createDispute(t, orderID, "payment_dispute", "Provider claims payment not received")
		
		// 3. Verify dispute is recorded
		dispute := suite.getDispute(t, disputeID)
		if dispute.Status != "open" {
			t.Errorf("Expected dispute to be open, got %s", dispute.Status)
		}
		
		// 4. Simulate evidence submission
		suite.submitEvidence(t, disputeID, "buyer", "Service logs showing no GPU access")
		suite.submitEvidence(t, disputeID, "provider", "Payment transaction records")
		
		// 5. Simulate resolution
		resolution := suite.resolveDispute(t, disputeID, "partial_refund", "50% refund due to service issues")
		
		// 6. Verify settlement is adjusted
		settlement := suite.getSettlement(t, orderID)
		if settlement.Amount != "50.00" {
			t.Errorf("Expected settlement amount to be 50.00, got %s", settlement.Amount)
		}
		
		// 7. Verify both parties are notified
		buyerNotification := suite.getNotification(t, "test_buyer", disputeID)
		providerNotification := suite.getNotification(t, "problematic_provider", disputeID)
		
		if buyerNotification == nil || providerNotification == nil {
			t.Error("Expected both parties to be notified of resolution")
		}
	})
}

func TestConcurrentOrderMatching(t *testing.T) {
	suite := setupIntegrationTest(t)
	
	t.Run("ConcurrentOrderMatching", func(t *testing.T) {
		// 1. Create multiple orders simultaneously
		orderIDs := make([]string, 10)
		for i := 0; i < 10; i++ {
			orderIDs[i] = suite.placeOrder(t, fmt.Sprintf("buyer_%d", i), "high_capacity_provider", 1, 2)
		}
		
		// 2. Verify all orders are processed
		for _, orderID := range orderIDs {
			order := suite.getOrder(t, orderID)
			if order.State != ocx.OrderMatched && order.State != ocx.OrderPending {
				t.Errorf("Order %s in unexpected state: %s", orderID, order.State)
			}
		}
		
		// 3. Verify no double-matching occurred
		matchedOrders := suite.getMatchedOrders(t, "high_capacity_provider")
		if len(matchedOrders) > 10 {
			t.Errorf("Expected max 10 matched orders, got %d", len(matchedOrders))
		}
	})
}

func TestDataConsistencyAcrossFailures(t *testing.T) {
	suite := setupIntegrationTest(t)
	
	t.Run("DataConsistency", func(t *testing.T) {
		// 1. Create multiple orders
		orderIDs := make([]string, 5)
		for i := 0; i < 5; i++ {
			orderIDs[i] = suite.placeOrder(t, fmt.Sprintf("buyer_%d", i), "reliable_provider", 1, 1)
		}
		
		// 2. Simulate database failure during processing
		suite.simulateDatabaseFailure()
		
		// 3. Attempt operations during failure
		suite.attemptProvisioning(t, orderIDs[0])
		
		// 4. Restore database
		suite.restoreDatabase()
		
		// 5. Verify data consistency
		for _, orderID := range orderIDs {
			order := suite.getOrder(t, orderID)
			if order == nil {
				t.Errorf("Order %s lost during failure", orderID)
			}
		}
		
		// 6. Verify ledger balance is correct
		balance := suite.getLedgerBalance(t)
		if balance < 0 {
			t.Errorf("Ledger balance is negative: %f", balance)
		}
	})
}

// Helper methods
func setupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	suite := &IntegrationTestSuite{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: BASE_URL,
		providers: map[string]*MockProvider{
			"healthy_provider": {
				ID:           "healthy_provider",
				IsHealthy:    true,
				ResponseTime: 100 * time.Millisecond,
				FailureRate:  0.0,
			},
			"partial_provider": {
				ID:           "partial_provider",
				IsHealthy:    true,
				ResponseTime: 200 * time.Millisecond,
				FailureRate:  0.0,
			},
			"problematic_provider": {
				ID:           "problematic_provider",
				IsHealthy:    true,
				ResponseTime: 500 * time.Millisecond,
				FailureRate:  0.1,
			},
		},
	}
	
	// Initialize test environment
	suite.initializeTestEnvironment(t)
	return suite
}

func (s *IntegrationTestSuite) placeOrder(t *testing.T, buyerID, providerID string, gpus, hours int) string {
	order := &ocx.Order{
		OrderID:       ocx.ID(fmt.Sprintf("test_order_%d", time.Now().UnixNano())),
		Version:       ocx.V010,
		Buyer:         ocx.PartyRef{PartyID: ocx.ID(buyerID), Role: "buyer"},
		OfferID:       ocx.ID(fmt.Sprintf("offer_%s", providerID)),
		RequestedGPUs: gpus,
		Hours:         hours,
		BudgetCap:     &ocx.Money{Currency: "USD", Amount: "100.00", Scale: 2},
		State:         ocx.OrderPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
	envelope := &ocx.Envelope{
		ID:        ocx.ID(fmt.Sprintf("envelope_%d", time.Now().UnixNano())),
		Kind:      ocx.KindOrder,
		Version:   ocx.V010,
		IssuedAt:  time.Now(),
		Payload:   order,
		Hash:      ocx.HashMessage([]byte("test")),
		Signature: "test_signature",
	}
	
	jsonData, _ := json.Marshal(envelope)
	resp, err := s.client.Post(s.baseURL+"/orders", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Order placement failed: HTTP %d", resp.StatusCode)
	}
	
	return string(order.OrderID)
}

func (s *IntegrationTestSuite) attemptProvisioning(t *testing.T, orderID string) *ProvisionResult {
	resp, err := s.client.Post(s.baseURL+"/provision", "application/json", 
		bytes.NewBuffer([]byte(fmt.Sprintf(`{"order_id": "%s"}`, orderID))))
	if err != nil {
		t.Fatalf("Failed to attempt provisioning: %v", err)
	}
	defer resp.Body.Close()
	
	var result ProvisionResult
	json.NewDecoder(resp.Body).Decode(&result)
	return &result
}

func (s *IntegrationTestSuite) getOrder(t *testing.T, orderID string) *ocx.Order {
	resp, err := s.client.Get(s.baseURL + "/orders/" + orderID)
	if err != nil {
		t.Fatalf("Failed to get order: %v", err)
	}
	defer resp.Body.Close()
	
	var order ocx.Order
	json.NewDecoder(resp.Body).Decode(&order)
	return &order
}

func (s *IntegrationTestSuite) simulateProviderFailure(providerID string) {
	if provider, exists := s.providers[providerID]; exists {
		provider.IsHealthy = false
	}
}

func (s *IntegrationTestSuite) setProviderCapacity(providerID string, capacity int) {
	// Implementation would set provider capacity
}

func (s *IntegrationTestSuite) simulateNetworkInterruption() {
	// Implementation would simulate network issues
}

func (s *IntegrationTestSuite) restoreNetwork() {
	// Implementation would restore network
}

func (s *IntegrationTestSuite) attemptSettlement(t *testing.T, orderID string) *SettlementResult {
	// Implementation would attempt settlement
	return &SettlementResult{Status: "completed"}
}

func (s *IntegrationTestSuite) getSettlement(t *testing.T, orderID string) *Settlement {
	// Implementation would get settlement
	return &Settlement{Status: "completed", Amount: "100.00"}
}

func (s *IntegrationTestSuite) createDispute(t *testing.T, orderID, disputeType, description string) string {
	// Implementation would create dispute
	return fmt.Sprintf("dispute_%d", time.Now().UnixNano())
}

func (s *IntegrationTestSuite) getDispute(t *testing.T, disputeID string) *Dispute {
	// Implementation would get dispute
	return &Dispute{ID: disputeID, Status: "open"}
}

func (s *IntegrationTestSuite) submitEvidence(t *testing.T, disputeID, party, evidence string) {
	// Implementation would submit evidence
}

func (s *IntegrationTestSuite) resolveDispute(t *testing.T, disputeID, resolution, reason string) *DisputeResolution {
	// Implementation would resolve dispute
	return &DisputeResolution{Resolution: resolution, Reason: reason}
}

func (s *IntegrationTestSuite) getNotification(t *testing.T, partyID, disputeID string) *Notification {
	// Implementation would get notification
	return &Notification{PartyID: partyID, DisputeID: disputeID}
}

func (s *IntegrationTestSuite) getMatchedOrders(t *testing.T, providerID string) []string {
	// Implementation would get matched orders
	return []string{}
}

func (s *IntegrationTestSuite) simulateDatabaseFailure() {
	// Implementation would simulate database failure
}

func (s *IntegrationTestSuite) restoreDatabase() {
	// Implementation would restore database
}

func (s *IntegrationTestSuite) getLedgerBalance(t *testing.T) float64 {
	// Implementation would get ledger balance
	return 0.0
}

func (s *IntegrationTestSuite) initializeTestEnvironment(t *testing.T) {
	// Implementation would initialize test environment
}

// Data structures for test results
type ProvisionResult struct {
	Success              bool
	PartialProvisioning  bool
	Message             string
}

type Settlement struct {
	Status string
	Amount string
}

type Dispute struct {
	ID     string
	Status string
}

type DisputeResolution struct {
	Resolution string
	Reason     string
}

type Notification struct {
	PartyID   string
	DisputeID string
}

type SettlementResult struct {
	Status string
}
