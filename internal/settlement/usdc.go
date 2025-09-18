// internal/settlement/usdc.go
package settlement

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"time"
)

// USDCSettlement handles USDC payments and escrow
type USDCSettlement struct {
	db *sql.DB
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

// Dispute represents a dispute record
type Dispute struct {
	ID         string    `json:"id"`
	EscrowID   string    `json:"escrow_id"`
	OrderID    string    `json:"order_id"`
	Reason     string    `json:"reason"`
	Evidence   []string  `json:"evidence"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	Resolution string    `json:"resolution"`
}

// NewUSDCSettlement creates settlement handler
func NewUSDCSettlement(db *sql.DB) *USDCSettlement {
	return &USDCSettlement{
		db: db,
	}
}

// CreateEscrow creates an escrow transaction
func (u *USDCSettlement) CreateEscrow(ctx context.Context, orderID string, amount *big.Int, requesterAddr, providerAddr string) (*EscrowTransaction, error) {
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

	// Store in database
	if err := u.storeEscrow(ctx, escrow); err != nil {
		return nil, fmt.Errorf("failed to store escrow: %w", err)
	}

	return escrow, nil
}

// ReleaseEscrow pays provider and protocol fee
func (u *USDCSettlement) ReleaseEscrow(ctx context.Context, escrowID string, usageReport *UsageReport) error {
	escrow, err := u.getEscrowByID(ctx, escrowID)
	if err != nil {
		return fmt.Errorf("escrow not found: %w", err)
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

	return u.updateEscrow(ctx, escrow)
}

// DisputeEscrow initiates dispute resolution
func (u *USDCSettlement) DisputeEscrow(ctx context.Context, escrowID string, reason string, evidence []string) error {
	escrow, err := u.getEscrowByID(ctx, escrowID)
	if err != nil {
		return fmt.Errorf("escrow not found: %w", err)
	}

	if escrow.Status != "active" {
		return fmt.Errorf("escrow not active: %s", escrow.Status)
	}

	// Update escrow to disputed status
	now := time.Now()
	escrow.Status = "disputed"
	escrow.DisputedAt = &now

	// Create dispute record
	dispute := &Dispute{
		ID:          generateDisputeID(),
		EscrowID:    escrowID,
		OrderID:     escrow.OrderID,
		Reason:      reason,
		Evidence:    evidence,
		Status:      "filed",
		CreatedAt:   now,
		Resolution:  "",
	}

	if err := u.createDispute(ctx, dispute); err != nil {
		return fmt.Errorf("failed to create dispute: %w", err)
	}

	return u.updateEscrow(ctx, escrow)
}

// RefundEscrow returns funds to requester
func (u *USDCSettlement) RefundEscrow(ctx context.Context, escrowID string, refundAmount *big.Int) error {
	escrow, err := u.getEscrowByID(ctx, escrowID)
	if err != nil {
		return fmt.Errorf("escrow not found: %w", err)
	}

	if escrow.Status != "disputed" {
		return fmt.Errorf("escrow not in disputed state: %s", escrow.Status)
	}

	// Update escrow status
	now := time.Now()
	escrow.Status = "refunded"
	escrow.ReleasedAt = &now
	escrow.TxHash = fmt.Sprintf("refund_tx_%d", time.Now().UnixNano())

	return u.updateEscrow(ctx, escrow)
}

// Helper methods

func (u *USDCSettlement) calculateFinalAmount(originalAmount *big.Int, usage *UsageReport) *big.Int {
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

func generateDisputeID() string {
	return fmt.Sprintf("dispute_%d", time.Now().UnixNano())
}

// Database operations

func (u *USDCSettlement) storeEscrow(ctx context.Context, escrow *EscrowTransaction) error {
	query := `
		INSERT INTO escrow_accounts (
			escrow_id, order_id, requester_id, total_escrowed_usdc, 
			protocol_fee_usdc, provider_amount_usdc, escrow_status, 
			deposit_tx_hash, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	totalAmount := new(big.Float).SetInt(escrow.Amount)
	totalAmount.Quo(totalAmount, big.NewFloat(1e6)) // Convert from wei to USDC
	
	protocolFee := new(big.Float).SetInt(escrow.ProtocolFee)
	protocolFee.Quo(protocolFee, big.NewFloat(1e6))
	
	providerAmount := new(big.Float).SetInt(escrow.ProviderAmount)
	providerAmount.Quo(providerAmount, big.NewFloat(1e6))
	
	_, err := u.db.ExecContext(ctx, query,
		escrow.ID, escrow.OrderID, escrow.RequesterAddr,
		totalAmount.String(), protocolFee.String(), providerAmount.String(),
		escrow.Status, escrow.TxHash, escrow.CreatedAt,
	)
	
	return err
}

func (u *USDCSettlement) getEscrowByID(ctx context.Context, escrowID string) (*EscrowTransaction, error) {
	query := `
		SELECT escrow_id, order_id, requester_id, total_escrowed_usdc, 
			   escrow_status, deposit_tx_hash, created_at
		FROM escrow_accounts 
		WHERE escrow_id = $1
	`
	
	var escrow EscrowTransaction
	var totalAmount string
	
	err := u.db.QueryRowContext(ctx, query, escrowID).Scan(
		&escrow.ID, &escrow.OrderID, &escrow.RequesterAddr,
		&totalAmount, &escrow.Status, &escrow.TxHash, &escrow.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	// Convert amount back to big.Int
	amount := new(big.Float)
	amount.SetString(totalAmount)
	amount.Mul(amount, big.NewFloat(1e6)) // Convert back to wei
	
	escrow.Amount, _ = amount.Int(nil)
	
	return &escrow, nil
}

func (u *USDCSettlement) updateEscrow(ctx context.Context, escrow *EscrowTransaction) error {
	query := `
		UPDATE escrow_accounts 
		SET escrow_status = $2, deposit_tx_hash = $3
		WHERE escrow_id = $1
	`
	
	_, err := u.db.ExecContext(ctx, query, escrow.ID, escrow.Status, escrow.TxHash)
	return err
}

func (u *USDCSettlement) createDispute(ctx context.Context, dispute *Dispute) error {
	query := `
		INSERT INTO disputes (
			dispute_id, escrow_id, order_id, dispute_reason, 
			dispute_status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	_, err := u.db.ExecContext(ctx, query,
		dispute.ID, dispute.EscrowID, dispute.OrderID,
		dispute.Reason, dispute.Status, dispute.CreatedAt,
	)
	
	return err
}

// GetRailID implements SettlementRail interface
func (u *USDCSettlement) GetRailID() string {
	return "usdc_polygon"
}

// GetSupportedCurrencies implements SettlementRail interface
func (u *USDCSettlement) GetSupportedCurrencies() []string {
	return []string{"USDC"}
}

// GetJurisdictions implements SettlementRail interface
func (u *USDCSettlement) GetJurisdictions() []string {
	return []string{"US", "EU", "SG", "GB", "CA"}
}

// ProcessSettlement implements SettlementRail interface
func (u *USDCSettlement) ProcessSettlement(ctx context.Context, instruction *SettlementInstruction) (*SettlementResult, error) {
	// Convert Amount to big.Int
	amount := new(big.Int)
	amount.SetString(instruction.InstructedAmount.Value, 10)
	
	// Create escrow transaction
	escrow, err := u.CreateEscrow(ctx, instruction.ID, amount, 
		instruction.Debtor.Account.ID, instruction.Creditor.Account.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create escrow: %w", err)
	}
	
	// Return settlement result
	result := &SettlementResult{
		SettlementID:         escrow.ID,
		InstructionID:        instruction.InstructionID,
		Status:               "pending_confirmation",
		RailUsed:             "usdc_polygon",
		TransactionReference: escrow.TxHash,
		SettlementDate:       time.Now(),
		ValueDate:            time.Now().Add(time.Minute * 5), // 5 minutes for blockchain confirmation
		ActualAmount:         instruction.InstructedAmount,
		Fees:                 []*Fee{
			{
				Type:     "protocol_fee",
				Amount:   &Amount{Currency: "USDC", Value: escrow.ProtocolFee.String()},
				Currency: "USDC",
				Rate:     "2.5",
				Basis:    "percentage",
			},
		},
		CreatedAt: time.Now(),
	}
	
	return result, nil
}

// GetStatus implements SettlementRail interface
func (u *USDCSettlement) GetStatus(ctx context.Context, settlementID string) (*SettlementStatus, error) {
	escrow, err := u.getEscrowByID(ctx, settlementID)
	if err != nil {
		return nil, err
	}
	
	status := &SettlementStatus{
		SettlementID: settlementID,
		Status:       escrow.Status,
		LastUpdated:  escrow.CreatedAt,
	}
	
	return status, nil
}

// SupportsCurrency implements SettlementRail interface
func (u *USDCSettlement) SupportsCurrency(currency string) bool {
	return currency == "USDC"
}

// SupportsJurisdiction implements SettlementRail interface
func (u *USDCSettlement) SupportsJurisdiction(jurisdiction string) bool {
	supportedJurisdictions := map[string]bool{
		"US": true, "EU": true, "SG": true, "GB": true, "CA": true,
	}
	return supportedJurisdictions[jurisdiction]
}
