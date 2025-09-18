package main

import (
	"context"
	"fmt"
	"math/big"
	"time"
)

// USDCSettlementDemo demonstrates the USDC settlement system
type USDCSettlementDemo struct {
	escrows map[string]*EscrowTransaction
}

// EscrowTransaction represents an escrow transaction
type EscrowTransaction struct {
	ID              string    `json:"id"`
	OrderID         string    `json:"order_id"`
	RequesterAddr   string    `json:"requester_address"`
	ProviderAddr    string    `json:"provider_address"`
	Amount          *big.Int  `json:"amount"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	ReleasedAt      *time.Time `json:"released_at"`
	DisputedAt      *time.Time `json:"disputed_at"`
	TxHash          string    `json:"tx_hash"`
	ProtocolFee     *big.Int  `json:"protocol_fee"`
	ProviderAmount  *big.Int  `json:"provider_amount"`
}

// UsageReport contains session usage data for settlement calculation
type UsageReport struct {
	SessionID       string        `json:"session_id"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	ActualDuration  time.Duration `json:"actual_duration"`
	AvgUtilization  float64       `json:"avg_utilization"`
	PeakUtilization int           `json:"peak_utilization"`
	SLACompliant    bool          `json:"sla_compliant"`
	EarlyTermination bool         `json:"early_termination"`
}

// NewUSDCSettlementDemo creates a new demo instance
func NewUSDCSettlementDemo() *USDCSettlementDemo {
	return &USDCSettlementDemo{
		escrows: make(map[string]*EscrowTransaction),
	}
}

// CreateEscrow creates an escrow transaction
func (u *USDCSettlementDemo) CreateEscrow(ctx context.Context, orderID string, amount *big.Int, requesterAddr, providerAddr string) (*EscrowTransaction, error) {
	// Calculate protocol fee (2.5%)
	protocolFee := new(big.Int).Div(new(big.Int).Mul(amount, big.NewInt(25)), big.NewInt(1000))
	providerAmount := new(big.Int).Sub(amount, protocolFee)

	escrow := &EscrowTransaction{
		ID:             generateEscrowID(),
		OrderID:        orderID,
		RequesterAddr:  requesterAddr,
		ProviderAddr:   providerAddr,
		Amount:         amount,
		ProtocolFee:    protocolFee,
		ProviderAmount: providerAmount,
		Status:         "pending_confirmation",
		CreatedAt:      time.Now(),
		TxHash:         fmt.Sprintf("mock_tx_%d", time.Now().UnixNano()),
	}

	// Store in memory
	u.escrows[escrow.ID] = escrow

	fmt.Printf("✅ Created escrow: %s\n", escrow.ID)
	fmt.Printf("   Order ID: %s\n", escrow.OrderID)
	fmt.Printf("   Amount: %s USDC\n", formatUSDC(amount))
	fmt.Printf("   Protocol Fee: %s USDC (2.5%%)\n", formatUSDC(protocolFee))
	fmt.Printf("   Provider Amount: %s USDC\n", formatUSDC(providerAmount))
	fmt.Printf("   Status: %s\n", escrow.Status)

	return escrow, nil
}

// ReleaseEscrow pays provider and protocol fee
func (u *USDCSettlementDemo) ReleaseEscrow(ctx context.Context, escrowID string, usageReport *UsageReport) error {
	escrow, exists := u.escrows[escrowID]
	if !exists {
		return fmt.Errorf("escrow not found: %s", escrowID)
	}

	if escrow.Status != "active" {
		return fmt.Errorf("escrow not active: %s", escrow.Status)
	}

	// Calculate final amounts based on usage
	finalAmount := u.calculateFinalAmount(escrow.Amount, usageReport)
	protocolFee := new(big.Int).Div(new(big.Int).Mul(finalAmount, big.NewInt(25)), big.NewInt(1000))
	providerAmount := new(big.Int).Sub(finalAmount, protocolFee)

	// Update escrow status
	now := time.Now()
	escrow.Status = "released"
	escrow.ReleasedAt = &now
	escrow.TxHash = fmt.Sprintf("release_tx_%d", time.Now().UnixNano())

	fmt.Printf("✅ Released escrow: %s\n", escrow.ID)
	fmt.Printf("   Final Amount: %s USDC\n", formatUSDC(finalAmount))
	fmt.Printf("   Protocol Fee: %s USDC\n", formatUSDC(protocolFee))
	fmt.Printf("   Provider Payment: %s USDC\n", formatUSDC(providerAmount))
	fmt.Printf("   Usage: %.1f%% utilization, SLA: %t\n", usageReport.AvgUtilization*100, usageReport.SLACompliant)

	return nil
}

// DisputeEscrow initiates dispute resolution
func (u *USDCSettlementDemo) DisputeEscrow(ctx context.Context, escrowID string, reason string, evidence []string) error {
	escrow, exists := u.escrows[escrowID]
	if !exists {
		return fmt.Errorf("escrow not found: %s", escrowID)
	}

	if escrow.Status != "active" {
		return fmt.Errorf("escrow not active: %s", escrow.Status)
	}

	// Update escrow to disputed status
	now := time.Now()
	escrow.Status = "disputed"
	escrow.DisputedAt = &now

	fmt.Printf("⚠️  Disputed escrow: %s\n", escrow.ID)
	fmt.Printf("   Reason: %s\n", reason)
	fmt.Printf("   Evidence: %v\n", evidence)
	fmt.Printf("   Status: %s\n", escrow.Status)

	return nil
}

// RefundEscrow returns funds to requester
func (u *USDCSettlementDemo) RefundEscrow(ctx context.Context, escrowID string, refundAmount *big.Int) error {
	escrow, exists := u.escrows[escrowID]
	if !exists {
		return fmt.Errorf("escrow not found: %s", escrowID)
	}

	if escrow.Status != "disputed" {
		return fmt.Errorf("escrow not in disputed state: %s", escrow.Status)
	}

	// Update escrow status
	now := time.Now()
	escrow.Status = "refunded"
	escrow.ReleasedAt = &now
	escrow.TxHash = fmt.Sprintf("refund_tx_%d", time.Now().UnixNano())

	fmt.Printf("💰 Refunded escrow: %s\n", escrow.ID)
	fmt.Printf("   Refund Amount: %s USDC\n", formatUSDC(refundAmount))
	fmt.Printf("   Status: %s\n", escrow.Status)

	return nil
}

// Helper methods

func (u *USDCSettlementDemo) calculateFinalAmount(originalAmount *big.Int, usage *UsageReport) *big.Int {
	if usage.EarlyTermination {
		// Calculate prorated amount based on actual usage
		usagePercent := int64(usage.ActualDuration.Hours() / 24.0 * 100) // Assuming 24h max duration
		return new(big.Int).Div(new(big.Int).Mul(originalAmount, big.NewInt(usagePercent)), big.NewInt(100))
	}

	if !usage.SLACompliant {
		// Apply penalty for SLA violations
		penalty := new(big.Int).Div(new(big.Int).Mul(originalAmount, big.NewInt(10)), big.NewInt(100)) // 10% penalty
		return new(big.Int).Sub(originalAmount, penalty)
	}

	if usage.AvgUtilization > 0.95 {
		// Bonus for high utilization
		bonus := new(big.Int).Div(new(big.Int).Mul(originalAmount, big.NewInt(5)), big.NewInt(100)) // 5% bonus
		return new(big.Int).Add(originalAmount, bonus)
	}

	return originalAmount
}

func generateEscrowID() string {
	return fmt.Sprintf("escrow_%d", time.Now().UnixNano())
}

func formatUSDC(amount *big.Int) string {
	// Convert from wei (18 decimals) to USDC (6 decimals)
	usdc := new(big.Float).SetInt(amount)
	usdc.Quo(usdc, big.NewFloat(1e6))
	return usdc.Text('f', 6)
}

func main() {
	fmt.Println("🚀 OCX Protocol - USDC Settlement Demo")
	fmt.Println("=====================================")

	// Create settlement demo
	settlement := NewUSDCSettlementDemo()
	ctx := context.Background()

	// Demo 1: Create escrow for compute order
	fmt.Println("\n📋 Demo 1: Creating Escrow for Compute Order")
	fmt.Println("--------------------------------------------")
	
	orderID := "order_12345"
	amount := new(big.Int).Mul(big.NewInt(100), big.NewInt(1e6)) // 100 USDC
	requesterAddr := "0x1234567890123456789012345678901234567890"
	providerAddr := "0x0987654321098765432109876543210987654321"

	escrow, err := settlement.CreateEscrow(ctx, orderID, amount, requesterAddr, providerAddr)
	if err != nil {
		fmt.Printf("❌ Error creating escrow: %v\n", err)
		return
	}

	// Demo 2: Simulate successful completion
	fmt.Println("\n✅ Demo 2: Successful Session Completion")
	fmt.Println("---------------------------------------")
	
	usageReport := &UsageReport{
		SessionID:       "session_67890",
		StartTime:       time.Now().Add(-2 * time.Hour),
		EndTime:         time.Now(),
		ActualDuration:  2 * time.Hour,
		AvgUtilization:  0.92, // 92% utilization
		PeakUtilization: 98,
		SLACompliant:    true,
		EarlyTermination: false,
	}

	// Activate escrow first
	escrow.Status = "active"

	err = settlement.ReleaseEscrow(ctx, escrow.ID, usageReport)
	if err != nil {
		fmt.Printf("❌ Error releasing escrow: %v\n", err)
		return
	}

	// Demo 3: Simulate dispute scenario
	fmt.Println("\n⚠️  Demo 3: Dispute Scenario")
	fmt.Println("---------------------------")
	
	// Create another escrow for dispute demo
	escrow2, err := settlement.CreateEscrow(ctx, "order_54321", amount, requesterAddr, providerAddr)
	if err != nil {
		fmt.Printf("❌ Error creating escrow: %v\n", err)
		return
	}
	escrow2.Status = "active"

	// Simulate dispute
	err = settlement.DisputeEscrow(ctx, escrow2.ID, "GPU performance below SLA", []string{"evidence1.json", "logs.txt"})
	if err != nil {
		fmt.Printf("❌ Error disputing escrow: %v\n", err)
		return
	}

	// Demo 4: Refund scenario
	fmt.Println("\n💰 Demo 4: Refund Scenario")
	fmt.Println("-------------------------")
	
	refundAmount := new(big.Int).Div(amount, big.NewInt(2)) // 50% refund
	err = settlement.RefundEscrow(ctx, escrow2.ID, refundAmount)
	if err != nil {
		fmt.Printf("❌ Error refunding escrow: %v\n", err)
		return
	}

	// Summary
	fmt.Println("\n📊 Settlement Summary")
	fmt.Println("====================")
	fmt.Printf("Total Escrows Created: %d\n", len(settlement.escrows))
	fmt.Printf("Successful Settlements: 1\n")
	fmt.Printf("Disputes Resolved: 1\n")
	fmt.Printf("Total Protocol Fees: %s USDC\n", formatUSDC(new(big.Int).Add(escrow.ProtocolFee, escrow2.ProtocolFee)))
	
	fmt.Println("\n🎯 Key Features Demonstrated:")
	fmt.Println("• USDC escrow creation with 2.5% protocol fee")
	fmt.Println("• Usage-based settlement calculation")
	fmt.Println("• SLA compliance monitoring")
	fmt.Println("• Dispute resolution workflow")
	fmt.Println("• Automated refund processing")
	fmt.Println("• Real-time settlement status tracking")
	
	fmt.Println("\n✨ This demonstrates the 'SWIFT for Compute' settlement system!")
	fmt.Println("   Every transaction generates protocol revenue while ensuring trust.")
}
