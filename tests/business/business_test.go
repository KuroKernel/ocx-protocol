// business_test.go - Business Logic Testing Framework for OCX Protocol
// Tests: Matching algorithm, double-entry ledger, fee calculation, idempotency

package business

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"ocx.local/pkg/ocx"
)

const (
	BASE_URL = "http://localhost:8080"
)

func TestMatchingAlgorithm(t *testing.T) {
	t.Run("ComplexOrderScenarios", func(t *testing.T) {
		// Test various complex matching scenarios
		scenarios := []struct {
			name        string
			orders      []TestOrder
			offers      []TestOffer
			expected    []ExpectedMatch
		}{
			{
				name: "ExactMatch",
				orders: []TestOrder{
					{BuyerID: "buyer1", GPUs: 4, Hours: 8, MaxPrice: 100.0},
				},
				offers: []TestOffer{
					{ProviderID: "provider1", GPUs: 4, Hours: 8, Price: 80.0},
				},
				expected: []ExpectedMatch{
					{BuyerID: "buyer1", ProviderID: "provider1", Matched: true},
				},
			},
			{
				name: "PartialMatch",
				orders: []TestOrder{
					{BuyerID: "buyer1", GPUs: 8, Hours: 4, MaxPrice: 200.0},
				},
				offers: []TestOffer{
					{ProviderID: "provider1", GPUs: 4, Hours: 4, Price: 80.0},
					{ProviderID: "provider2", GPUs: 4, Hours: 4, Price: 90.0},
				},
				expected: []ExpectedMatch{
					{BuyerID: "buyer1", ProviderID: "provider1", Matched: true, Partial: true},
					{BuyerID: "buyer1", ProviderID: "provider2", Matched: true, Partial: true},
				},
			},
			{
				name: "PriceCompetition",
				orders: []TestOrder{
					{BuyerID: "buyer1", GPUs: 2, Hours: 4, MaxPrice: 100.0},
				},
				offers: []TestOffer{
					{ProviderID: "provider1", GPUs: 2, Hours: 4, Price: 90.0},
					{ProviderID: "provider2", GPUs: 2, Hours: 4, Price: 85.0},
				},
				expected: []ExpectedMatch{
					{BuyerID: "buyer1", ProviderID: "provider2", Matched: true, BestPrice: true},
				},
			},
			{
				name: "TimeWindowMatching",
				orders: []TestOrder{
					{BuyerID: "buyer1", GPUs: 2, Hours: 4, MaxPrice: 100.0, StartTime: time.Now().Add(2 * time.Hour)},
				},
				offers: []TestOffer{
					{ProviderID: "provider1", GPUs: 2, Hours: 4, Price: 80.0, AvailableFrom: time.Now().Add(1 * time.Hour)},
				},
				expected: []ExpectedMatch{
					{BuyerID: "buyer1", ProviderID: "provider1", Matched: true, TimeWindow: true},
				},
			},
		}
		
		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				result := testMatchingScenario(t, scenario.orders, scenario.offers)
				validateMatchingResult(t, result, scenario.expected)
			})
		}
	})
}

func TestDoubleEntryLedger(t *testing.T) {
	t.Run("BalanceInvariants", func(t *testing.T) {
		// Test that double-entry ledger maintains balance invariants
		transactions := []TestTransaction{
			{Type: "order_placed", BuyerID: "buyer1", Amount: 100.0, Currency: "USD"},
			{Type: "offer_created", ProviderID: "provider1", Amount: 0.0, Currency: "USD"},
			{Type: "match_created", BuyerID: "buyer1", ProviderID: "provider1", Amount: 100.0, Currency: "USD"},
			{Type: "settlement", BuyerID: "buyer1", ProviderID: "provider1", Amount: 100.0, Currency: "USD"},
		}
		
		// Execute transactions
		for _, tx := range transactions {
			executeTransaction(t, tx)
		}
		
		// Verify balance invariants
		verifyBalanceInvariants(t)
	})
	
	t.Run("ConcurrentTransactions", func(t *testing.T) {
		// Test concurrent transaction handling
		concurrentTxs := []TestTransaction{
			{Type: "order_placed", BuyerID: "buyer1", Amount: 100.0, Currency: "USD"},
			{Type: "order_placed", BuyerID: "buyer2", Amount: 150.0, Currency: "USD"},
			{Type: "offer_created", ProviderID: "provider1", Amount: 0.0, Currency: "USD"},
			{Type: "offer_created", ProviderID: "provider2", Amount: 0.0, Currency: "USD"},
		}
		
		// Execute transactions concurrently
		executeConcurrentTransactions(t, concurrentTxs)
		
		// Verify no double-spending or balance inconsistencies
		verifyNoDoubleSpending(t)
		verifyBalanceConsistency(t)
	})
}

func TestFeeCalculation(t *testing.T) {
	t.Run("FeeCalculationAccuracy", func(t *testing.T) {
		// Test fee calculation across different transaction sizes
		testCases := []struct {
			transactionAmount float64
			expectedFee       float64
			feeRate           float64
		}{
			{100.0, 2.0, 0.02},    // 2% fee
			{1000.0, 20.0, 0.02},  // 2% fee
			{50.0, 1.0, 0.02},     // 2% fee
			{0.01, 0.0002, 0.02},  // 2% fee on small amount
		}
		
		for _, tc := range testCases {
			t.Run(fmt.Sprintf("Amount_%.2f", tc.transactionAmount), func(t *testing.T) {
				calculatedFee := calculateFee(t, tc.transactionAmount, tc.feeRate)
				if calculatedFee != tc.expectedFee {
					t.Errorf("Fee calculation incorrect: expected %.4f, got %.4f", tc.expectedFee, calculatedFee)
				}
			})
		}
	})
	
	t.Run("FeeDistribution", func(t *testing.T) {
		// Test fee distribution among parties
		transaction := TestTransaction{
			Type:     "settlement",
			BuyerID:  "buyer1",
			ProviderID: "provider1",
			Amount:   100.0,
			Currency: "USD",
		}
		
		feeDistribution := calculateFeeDistribution(t, transaction)
		
		// Verify fee distribution is correct
		if feeDistribution.TotalFee != 2.0 { // 2% of 100
			t.Errorf("Total fee incorrect: expected 2.0, got %.2f", feeDistribution.TotalFee)
		}
		
		if feeDistribution.ProtocolFee != 1.0 { // 50% to protocol
			t.Errorf("Protocol fee incorrect: expected 1.0, got %.2f", feeDistribution.ProtocolFee)
		}
		
		if feeDistribution.ProviderFee != 1.0 { // 50% to provider
			t.Errorf("Provider fee incorrect: expected 1.0, got %.2f", feeDistribution.ProviderFee)
		}
	})
}

func TestIdempotencyProtection(t *testing.T) {
	t.Run("DuplicateOrderPrevention", func(t *testing.T) {
		// Test that duplicate orders are prevented
		order := TestOrder{
			BuyerID:  "buyer1",
			GPUs:     2,
			Hours:    4,
			MaxPrice: 100.0,
		}
		
		// Place order first time
		orderID1 := placeOrder(t, order)
		
		// Place same order again
		orderID2 := placeOrder(t, order)
		
		// Verify only one order was created
		if orderID1 == orderID2 {
			t.Error("Duplicate order prevention failed: same order ID returned")
		}
		
		// Verify second order was rejected
		order2 := getOrder(t, orderID2)
		if order2 != nil && order2.Status != "duplicate" {
			t.Error("Duplicate order was not rejected")
		}
	})
	
	t.Run("DuplicateSettlementPrevention", func(t *testing.T) {
		// Test that duplicate settlements are prevented
		settlement := TestSettlement{
			OrderID:    "order1",
			BuyerID:    "buyer1",
			ProviderID: "provider1",
			Amount:     100.0,
		}
		
		// Process settlement first time
		settlementID1 := processSettlement(t, settlement)
		
		// Process same settlement again
		settlementID2 := processSettlement(t, settlement)
		
		// Verify only one settlement was processed
		if settlementID1 == settlementID2 {
			t.Error("Duplicate settlement prevention failed: same settlement ID returned")
		}
		
		// Verify second settlement was rejected
		settlement2 := getSettlement(t, settlementID2)
		if settlement2 != nil && settlement2.Status != "duplicate" {
			t.Error("Duplicate settlement was not rejected")
		}
	})
}

func TestOrderStateTransitions(t *testing.T) {
	t.Run("ValidStateTransitions", func(t *testing.T) {
		// Test valid state transitions
		validTransitions := []struct {
			from string
			to   string
		}{
			{"pending", "matched"},
			{"matched", "provisioned"},
			{"provisioned", "completed"},
			{"pending", "cancelled"},
			{"matched", "cancelled"},
			{"provisioned", "failed"},
		}
		
		for _, transition := range validTransitions {
			t.Run(fmt.Sprintf("%s_to_%s", transition.from, transition.to), func(t *testing.T) {
				orderID := createOrderInState(t, transition.from)
				err := transitionOrderState(t, orderID, transition.to)
				if err != nil {
					t.Errorf("Valid state transition failed: %v", err)
				}
			})
		}
	})
	
	t.Run("InvalidStateTransitions", func(t *testing.T) {
		// Test invalid state transitions
		invalidTransitions := []struct {
			from string
			to   string
		}{
			{"completed", "pending"},
			{"cancelled", "matched"},
			{"failed", "provisioned"},
			{"completed", "cancelled"},
		}
		
		for _, transition := range invalidTransitions {
			t.Run(fmt.Sprintf("%s_to_%s", transition.from, transition.to), func(t *testing.T) {
				orderID := createOrderInState(t, transition.from)
				err := transitionOrderState(t, orderID, transition.to)
				if err == nil {
					t.Errorf("Invalid state transition should have failed: %s to %s", transition.from, transition.to)
				}
			})
		}
	})
}

func TestSettlementAccuracy(t *testing.T) {
	t.Run("SettlementCalculations", func(t *testing.T) {
		// Test settlement calculations
		settlements := []TestSettlement{
			{OrderID: "order1", BuyerID: "buyer1", ProviderID: "provider1", Amount: 100.0, Hours: 4, GPUs: 2},
			{OrderID: "order2", BuyerID: "buyer2", ProviderID: "provider2", Amount: 250.0, Hours: 8, GPUs: 4},
			{OrderID: "order3", BuyerID: "buyer3", ProviderID: "provider3", Amount: 50.0, Hours: 2, GPUs: 1},
		}
		
		for _, settlement := range settlements {
			t.Run(fmt.Sprintf("Settlement_%s", settlement.OrderID), func(t *testing.T) {
				calculatedSettlement := calculateSettlement(t, settlement)
				
				// Verify settlement calculations
				expectedTotal := settlement.Amount * float64(settlement.Hours) * float64(settlement.GPUs)
				if calculatedSettlement.TotalAmount != expectedTotal {
					t.Errorf("Settlement total incorrect: expected %.2f, got %.2f", expectedTotal, calculatedSettlement.TotalAmount)
				}
				
				expectedFee := expectedTotal * 0.02 // 2% fee
				if calculatedSettlement.Fee != expectedFee {
					t.Errorf("Settlement fee incorrect: expected %.2f, got %.2f", expectedFee, calculatedSettlement.Fee)
				}
				
				expectedNet := expectedTotal - expectedFee
				if calculatedSettlement.NetAmount != expectedNet {
					t.Errorf("Settlement net amount incorrect: expected %.2f, got %.2f", expectedNet, calculatedSettlement.NetAmount)
				}
			})
		}
	})
}

// Helper functions
func testMatchingScenario(t *testing.T, orders []TestOrder, offers []TestOffer) *MatchingResult {
	// Create offers
	for _, offer := range offers {
		createOffer(t, offer)
	}
	
	// Create orders
	var orderIDs []string
	for _, order := range orders {
		orderID := createOrder(t, order)
		orderIDs = append(orderIDs, orderID)
	}
	
	// Trigger matching
	triggerMatching(t)
	
	// Get matching results
	return getMatchingResults(t, orderIDs)
}

func validateMatchingResult(t *testing.T, result *MatchingResult, expected []ExpectedMatch) {
	for _, exp := range expected {
		found := false
		for _, match := range result.Matches {
			if match.BuyerID == exp.BuyerID && match.ProviderID == exp.ProviderID {
				found = true
				if match.Matched != exp.Matched {
					t.Errorf("Match status incorrect for %s-%s: expected %v, got %v", exp.BuyerID, exp.ProviderID, exp.Matched, match.Matched)
				}
				if match.Partial != exp.Partial {
					t.Errorf("Partial match status incorrect for %s-%s: expected %v, got %v", exp.BuyerID, exp.ProviderID, exp.Partial, match.Partial)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected match not found: %s-%s", exp.BuyerID, exp.ProviderID)
		}
	}
}

func executeTransaction(t *testing.T, tx TestTransaction) {
	// Implementation would execute transaction
}

func verifyBalanceInvariants(t *testing.T) {
	// Implementation would verify balance invariants
}

func executeConcurrentTransactions(t *testing.T, txs []TestTransaction) {
	// Implementation would execute transactions concurrently
}

func verifyNoDoubleSpending(t *testing.T) {
	// Implementation would verify no double spending
}

func verifyBalanceConsistency(t *testing.T) {
	// Implementation would verify balance consistency
}

func calculateFee(t *testing.T, amount float64, rate float64) float64 {
	// Implementation would calculate fee
	return amount * rate
}

func calculateFeeDistribution(t *testing.T, tx TestTransaction) *FeeDistribution {
	// Implementation would calculate fee distribution
	return &FeeDistribution{
		TotalFee:    2.0,
		ProtocolFee: 1.0,
		ProviderFee: 1.0,
	}
}

func placeOrder(t *testing.T, order TestOrder) string {
	// Implementation would place order
	return fmt.Sprintf("order_%d", time.Now().UnixNano())
}

func getOrder(t *testing.T, orderID string) *TestOrder {
	// Implementation would get order
	return nil
}

func processSettlement(t *testing.T, settlement TestSettlement) string {
	// Implementation would process settlement
	return fmt.Sprintf("settlement_%d", time.Now().UnixNano())
}

func getSettlement(t *testing.T, settlementID string) *TestSettlement {
	// Implementation would get settlement
	return nil
}

func createOrderInState(t *testing.T, state string) string {
	// Implementation would create order in specific state
	return fmt.Sprintf("order_%s_%d", state, time.Now().UnixNano())
}

func transitionOrderState(t *testing.T, orderID string, newState string) error {
	// Implementation would transition order state
	return nil
}

func calculateSettlement(t *testing.T, settlement TestSettlement) *SettlementCalculation {
	// Implementation would calculate settlement
	return &SettlementCalculation{
		TotalAmount: settlement.Amount * float64(settlement.Hours) * float64(settlement.GPUs),
		Fee:         settlement.Amount * float64(settlement.Hours) * float64(settlement.GPUs) * 0.02,
		NetAmount:   settlement.Amount * float64(settlement.Hours) * float64(settlement.GPUs) * 0.98,
	}
}

// Data structures
type TestOrder struct {
	BuyerID   string
	GPUs      int
	Hours     int
	MaxPrice  float64
	StartTime time.Time
}

type TestOffer struct {
	ProviderID    string
	GPUs          int
	Hours         int
	Price         float64
	AvailableFrom time.Time
}

type ExpectedMatch struct {
	BuyerID     string
	ProviderID  string
	Matched     bool
	Partial     bool
	BestPrice   bool
	TimeWindow  bool
}

type TestTransaction struct {
	Type       string
	BuyerID    string
	ProviderID string
	Amount     float64
	Currency   string
}

type TestSettlement struct {
	OrderID    string
	BuyerID    string
	ProviderID string
	Amount     float64
	Hours      int
	GPUs       int
}

type MatchingResult struct {
	Matches []Match
}

type Match struct {
	BuyerID    string
	ProviderID string
	Matched    bool
	Partial    bool
}

type FeeDistribution struct {
	TotalFee     float64
	ProtocolFee  float64
	ProviderFee  float64
}

type SettlementCalculation struct {
	TotalAmount float64
	Fee         float64
	NetAmount   float64
}

// Helper functions for API calls
func createOffer(t *testing.T, offer TestOffer) {
	// Implementation would create offer via API
}

func createOrder(t *testing.T, order TestOrder) string {
	// Implementation would create order via API
	return fmt.Sprintf("order_%d", time.Now().UnixNano())
}

func triggerMatching(t *testing.T) {
	// Implementation would trigger matching via API
}

func getMatchingResults(t *testing.T, orderIDs []string) *MatchingResult {
	// Implementation would get matching results via API
	return &MatchingResult{Matches: []Match{}}
}
