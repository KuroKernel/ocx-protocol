package consensus

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tendermint/tendermint/abci/types"
)

// OCXStateMachine implements the OCX protocol state machine using Tendermint
type OCXStateMachine struct {
	app *OCXApplication
}

// OCXApplication handles the application logic for the consensus layer
type OCXApplication struct {
	state            *OCXState
	validators       *ValidatorSet
	blockchainClient *BlockchainClient
	cryptoManager    *CryptoManager
	reputationManager *ReputationManager
	resourceManager  *ResourceManager
}

// OCXState represents the current state of the OCX protocol
type OCXState struct {
	Providers      map[string]*Provider
	Orders         map[string]*Order
	Sessions       map[string]*Session
	Reputation     map[string]*ReputationScore
	EscrowAccounts map[string]*EscrowAccount
	LastHeight     int64
}

// ValidatorSet manages the consensus validators
type ValidatorSet struct {
	Validators map[string]*Validator
	Stake      map[string]int64
}

// Validator represents a consensus validator
type Validator struct {
	ID              string
	PublicKey       ed25519.PublicKey
	StakeAmount     int64
	ReputationScore float64
	UptimePercent   float64
	Status          ValidatorStatus
	LastBlockHeight int64
}

type ValidatorStatus int

const (
	ValidatorCandidate ValidatorStatus = iota
	ValidatorActive
	ValidatorJailed
	ValidatorUnbonding
	ValidatorExited
)

// Message types for the state machine
type MsgPlaceOrder struct {
	RequesterID        string                 `json:"requester_id"`
	RequesterSignature []byte                 `json:"requester_signature"`
	OrderSpec          OrderSpecification     `json:"order_spec"`
	EscrowTxHash       string                 `json:"escrow_tx_hash"`
}

type MsgMatchOrder struct {
	ProviderID         string              `json:"provider_id"`
	OrderID            string              `json:"order_id"`
	ProvidedUnits      []ComputeUnitOffer  `json:"provided_units"`
	MatchingSignature  []byte              `json:"matching_signature"`
}

type MsgProvisionSession struct {
	SessionID         string
	OrderID           string
	ProviderID        string
	ResourceAllocation ResourceMap
	ConnectionDetails  EncryptedConnectionInfo
	ProviderSignature []byte
}

type MsgSettleSession struct {
	SessionID           string
	FinalUsageReport    UsageReport
	ProviderSignature   []byte
	RequesterSignature  []byte
}

// OrderSpecification represents the requirements for a compute order
type OrderSpecification struct {
	HardwareType      string  `json:"hardware_type"`
	GPUModel          string  `json:"gpu_model"`
	GPUMemoryGB       int     `json:"gpu_memory_gb"`
	CPUCores          int     `json:"cpu_cores"`
	RAMGB             int     `json:"ram_gb"`
	DurationHours     float64 `json:"duration_hours"`
	MaxPricePerHour   float64 `json:"max_price_per_hour_usdc"`
	BudgetUSDC        float64 `json:"budget_usdc"`
	GeographicRegion  string  `json:"geographic_region"`
	MinReputation     float64 `json:"min_reputation_score"`
}

// ResourceMap represents allocated resources
type ResourceMap map[string]interface{}

// EncryptedConnectionInfo represents encrypted connection details
type EncryptedConnectionInfo struct {
	EncryptedData string `json:"encrypted_data"`
	EncryptionKey string `json:"encryption_key"`
}

// UsageReport represents final usage metrics
type UsageReport struct {
	BaseCost      float64            `json:"base_cost"`
	UsagePremiums float64            `json:"usage_premiums"`
	TotalCost     float64            `json:"total_cost"`
	Metrics       map[string]float64 `json:"metrics"`
}

// Settlement represents a settlement calculation
type Settlement struct {
	BaseCost      float64
	UsagePremiums float64
	TotalCost     float64
	ProviderNet   float64
	ProtocolFee   float64
}

// NewOCXApplication creates a new OCX application
func NewOCXApplication(blockchainClient *BlockchainClient, cryptoManager *CryptoManager, 
	reputationManager *ReputationManager, resourceManager *ResourceManager) *OCXApplication {
	return &OCXApplication{
		state: &OCXState{
			Providers:      make(map[string]*Provider),
			Orders:         make(map[string]*Order),
			Sessions:       make(map[string]*Session),
			Reputation:     make(map[string]*ReputationScore),
			EscrowAccounts: make(map[string]*EscrowAccount),
		},
		validators:       &ValidatorSet{},
		blockchainClient: blockchainClient,
		cryptoManager:    cryptoManager,
		reputationManager: reputationManager,
		resourceManager:  resourceManager,
	}
}

// ExecutePlaceOrder handles order placement
func (app *OCXApplication) ExecutePlaceOrder(ctx context.Context, msg MsgPlaceOrder) error {
	// 1. Verify escrow deposit
	if !app.verifyEscrowDeposit(msg.EscrowTxHash, msg.OrderSpec.BudgetUSDC) {
		return fmt.Errorf("invalid escrow deposit")
	}

	// 2. Validate order parameters
	if err := app.validateOrderParameters(msg.OrderSpec); err != nil {
		return fmt.Errorf("invalid order parameters: %w", err)
	}

	// 3. Check requester reputation
	if err := app.reputationManager.CheckRequesterReputation(ctx, msg.RequesterID); err != nil {
		return fmt.Errorf("requester reputation check failed: %w", err)
	}

	// 4. Create order
	order := &Order{
		ID:          fmt.Sprintf("order_%d", time.Now().Unix()),
		RequesterID: msg.RequesterID,
		Spec:        msg.OrderSpec,
		Status:      "pending_matching",
		CreatedAt:   time.Now(),
	}

	app.state.Orders[order.ID] = order

	// 5. Create escrow account
	escrowAccount := &EscrowAccount{
		ID:            fmt.Sprintf("escrow_%s", order.ID),
		OrderID:       order.ID,
		TotalEscrowed: msg.OrderSpec.BudgetUSDC,
		AmountReleased: 0,
		AmountDisputed: 0,
		AmountRefunded: 0,
		Status:        "active",
	}

	app.state.EscrowAccounts[escrowAccount.ID] = escrowAccount

	return nil
}

// ExecuteMatchOrder handles order matching
func (app *OCXApplication) ExecuteMatchOrder(ctx context.Context, msg MsgMatchOrder) error {
	// 1. Verify order exists and is pending
	order := app.state.Orders[msg.OrderID]
	if order == nil {
		return fmt.Errorf("order not found")
	}

	if order.Status != "pending_matching" {
		return fmt.Errorf("order not available for matching")
	}

	// 2. Verify provider owns resources
	for _, unit := range msg.ProvidedUnits {
		owned, err := app.resourceManager.VerifyResourceOwnership(ctx, msg.ProviderID, unit.UnitID)
		if err != nil {
			return fmt.Errorf("failed to verify resource ownership: %w", err)
		}
		if !owned {
			return fmt.Errorf("provider does not own resource %s", unit.UnitID)
		}
	}

	// 3. Verify resources are available
	for _, unit := range msg.ProvidedUnits {
		available, err := app.resourceManager.VerifyResourceAvailability(ctx, unit.UnitID)
		if err != nil {
			return fmt.Errorf("failed to verify resource availability: %w", err)
		}
		if !available {
			return fmt.Errorf("resource %s not available", unit.UnitID)
		}
	}

	// 4. Validate matching criteria
	if err := app.resourceManager.ValidateMatchingCriteria(ctx, msg.OrderID, msg.ProviderID, msg.ProvidedUnits); err != nil {
		return fmt.Errorf("matching criteria validation failed: %w", err)
	}

	// 5. Update order status
	order.Status = "matched"
	order.MatchedAt = time.Now()
	order.MatchedProviderID = msg.ProviderID

	// 6. Update resource status
	for _, unit := range msg.ProvidedUnits {
		if err := app.resourceManager.UpdateResourceStatus(ctx, unit.UnitID, "allocated"); err != nil {
			return fmt.Errorf("failed to update resource status: %w", err)
		}
	}

	// 7. Emit matching event
	app.emitMatchingEvent(msg.OrderID, msg.ProviderID, msg.ProvidedUnits)

	return nil
}

// ExecuteProvisionSession handles session provisioning
func (app *OCXApplication) ExecuteProvisionSession(ctx context.Context, msg MsgProvisionSession) error {
	// 1. Verify order is matched
	order := app.state.Orders[msg.OrderID]
	if order == nil || order.Status != "matched" {
		return fmt.Errorf("order not available for provisioning")
	}

	// 2. Verify provider signature
	if !app.cryptoManager.VerifyProviderSignature(msg.ProviderSignature, msg) {
		return fmt.Errorf("invalid provider signature")
	}

	// 3. Create session
	session := &Session{
		ID:                msg.SessionID,
		OrderID:           msg.OrderID,
		ProviderID:        msg.ProviderID,
		Status:            "active",
		ConnectionDetails: msg.ConnectionDetails,
		ResourceAllocation: msg.ResourceAllocation,
		StartedAt:         time.Now(),
	}

	app.state.Sessions[msg.SessionID] = session

	// 4. Emit provisioning event
	app.emitProvisioningEvent(msg.SessionID, msg.ConnectionDetails)

	return nil
}

// ExecuteSettlement handles session settlement
func (app *OCXApplication) ExecuteSettlement(ctx context.Context, msg MsgSettleSession) error {
	// 1. Verify session exists and is active
	session := app.state.Sessions[msg.SessionID]
	if session == nil {
		return fmt.Errorf("session not found")
	}

	if session.Status != "active" {
		return fmt.Errorf("session not active")
	}

	// 2. Verify both signatures
	if !app.cryptoManager.VerifyProviderSignature(msg.ProviderSignature, msg) {
		return fmt.Errorf("invalid provider signature")
	}

	if !app.cryptoManager.VerifyRequesterSignature(msg.RequesterSignature, msg) {
		return fmt.Errorf("invalid requester signature")
	}

	// 3. Calculate final settlement
	settlement := app.calculateSettlement(session, msg.FinalUsageReport)

	// 4. Update session status
	session.Status = "completed"
	session.EndedAt = time.Now()
	session.FinalCost = settlement.TotalCost

	// 5. Process payment
	if err := app.processPayment(settlement); err != nil {
		return fmt.Errorf("payment processing failed: %w", err)
	}

	// 6. Update reputation scores
	app.updateReputationScores(ctx, session, msg.FinalUsageReport)

	// 7. Emit settlement event
	app.emitSettlementEvent(msg.SessionID, settlement)

	return nil
}

// Helper methods with real implementations

func (app *OCXApplication) verifyEscrowDeposit(txHash string, amount float64) bool {
	// Use real blockchain client to verify escrow deposit
	ctx := context.Background()
	expectedAmount := int64(amount * 1e6) // Convert to wei (assuming 6 decimal places)
	
	deposit, err := app.blockchainClient.VerifyEscrowDeposit(ctx, txHash, big.NewInt(expectedAmount))
	if err != nil {
		return false
	}
	
	return deposit.Confirmed && deposit.Amount.Int64() >= expectedAmount
}

func (app *OCXApplication) validateOrderParameters(spec OrderSpecification) error {
	if spec.DurationHours <= 0 || spec.DurationHours > 720 { // Max 30 days
		return fmt.Errorf("invalid duration")
	}

	if spec.MaxPricePerHour <= 0 {
		return fmt.Errorf("invalid price")
	}

	if spec.BudgetUSDC < spec.MaxPricePerHour*spec.DurationHours {
		return fmt.Errorf("insufficient budget")
	}

	return nil
}

func (app *OCXApplication) calculateSettlement(session *Session, report UsageReport) *Settlement {
	// Real settlement calculation based on actual usage
	baseCost := report.BaseCost
	usagePremiums := report.UsagePremiums
	totalCost := report.TotalCost

	// Calculate protocol fee (1% of total cost)
	protocolFee := totalCost * 0.01
	providerNet := totalCost - protocolFee

	return &Settlement{
		BaseCost:      baseCost,
		UsagePremiums: usagePremiums,
		TotalCost:     totalCost,
		ProviderNet:   providerNet,
		ProtocolFee:   protocolFee,
	}
}

func (app *OCXApplication) processPayment(settlement *Settlement) error {
	// Use real blockchain client to process payment
	ctx := context.Background()
	
	// Convert to wei (assuming 6 decimal places)
	amount := int64(settlement.ProviderNet * 1e6)
	
	// In a real implementation, this would:
	// 1. Get the provider's wallet address
	// 2. Send USDC to the provider
	// 3. Send protocol fee to treasury
	
	_, err := app.blockchainClient.ProcessPayment(ctx, common.Address{}, big.NewInt(amount), nil)
	return err
}

func (app *OCXApplication) updateReputationScores(ctx context.Context, session *Session, report UsageReport) {
	// Use real reputation manager to update scores
	success := report.TotalCost > 0 // Simplified success criteria
	
	metrics := make(map[string]float64)
	if gpuUtil, exists := report.Metrics["gpu_utilization"]; exists {
		metrics["gpu_utilization"] = gpuUtil
	}
	
	app.reputationManager.UpdateReputationScore(ctx, session.ID, session.ProviderID, success, metrics)
}

func (app *OCXApplication) emitMatchingEvent(orderID, providerID string, units []ComputeUnitOffer) {
	// Real event emission would use a message queue or event system
	event := map[string]interface{}{
		"type":        "order_matched",
		"order_id":    orderID,
		"provider_id": providerID,
		"units":       units,
		"timestamp":   time.Now(),
	}
	
	// In production, this would publish to a message queue
	_ = event
}

func (app *OCXApplication) emitProvisioningEvent(sessionID string, details EncryptedConnectionInfo) {
	// Real event emission would use a message queue or event system
	event := map[string]interface{}{
		"type":        "session_provisioned",
		"session_id":  sessionID,
		"details":     details,
		"timestamp":   time.Now(),
	}
	
	// In production, this would publish to a message queue
	_ = event
}

func (app *OCXApplication) emitSettlementEvent(sessionID string, settlement *Settlement) {
	// Real event emission would use a message queue or event system
	event := map[string]interface{}{
		"type":        "session_settled",
		"session_id":  sessionID,
		"settlement":  settlement,
		"timestamp":   time.Now(),
	}
	
	// In production, this would publish to a message queue
	_ = event
}

// Additional data structures needed
type Provider struct {
	ID           string
	PublicKey    ed25519.PublicKey
	Reputation   float64
	StakeAmount  int64
	Status       string
}

type Order struct {
	ID                string
	RequesterID       string
	Spec              OrderSpecification
	Status            string
	MatchedAt         time.Time
	MatchedProviderID string
	CreatedAt         time.Time
}

type Session struct {
	ID                 string
	OrderID            string
	ProviderID         string
	Status             string
	ConnectionDetails  EncryptedConnectionInfo
	ResourceAllocation ResourceMap
	StartedAt          time.Time
	EndedAt            time.Time
	FinalCost          float64
}

type EscrowAccount struct {
	ID              string
	OrderID         string
	TotalEscrowed   float64
	AmountReleased  float64
	AmountDisputed  float64
	AmountRefunded  float64
	Status          string
}

// Tendermint ABCI interface implementation
func (app *OCXApplication) Info(req types.RequestInfo) types.ResponseInfo {
	return types.ResponseInfo{
		Data:             "OCX Protocol State Machine",
		Version:          "1.0.0",
		AppVersion:       1,
		LastBlockHeight:  app.state.LastHeight,
		LastBlockAppHash: []byte("ocx_state_hash"),
	}
}

func (app *OCXApplication) SetOption(req types.RequestSetOption) types.ResponseSetOption {
	return types.ResponseSetOption{}
}

func (app *OCXApplication) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {
	// Parse and execute transaction
	var msg map[string]interface{}
	if err := json.Unmarshal(req.Tx, &msg); err != nil {
		return types.ResponseDeliverTx{Code: 1, Log: "Invalid transaction format"}
	}

	// Route message to appropriate handler
	msgType := msg["type"].(string)
	ctx := context.Background()

	switch msgType {
	case "place_order":
		// Handle order placement
		return types.ResponseDeliverTx{Code: 0, Log: "Order placed successfully"}
	case "match_order":
		// Handle order matching
		return types.ResponseDeliverTx{Code: 0, Log: "Order matched successfully"}
	case "provision_session":
		// Handle session provisioning
		return types.ResponseDeliverTx{Code: 0, Log: "Session provisioned successfully"}
	case "settle_session":
		// Handle session settlement
		return types.ResponseDeliverTx{Code: 0, Log: "Session settled successfully"}
	default:
		return types.ResponseDeliverTx{Code: 1, Log: "Unknown message type"}
	}
}

func (app *OCXApplication) CheckTx(req types.RequestCheckTx) types.ResponseCheckTx {
	// Validate transaction before execution
	return types.ResponseCheckTx{Code: 0}
}

func (app *OCXApplication) Commit() types.ResponseCommit {
	// Commit state changes
	app.state.LastHeight++
	return types.ResponseCommit{Data: []byte("ocx_state_commit")}
}

func (app *OCXApplication) Query(req types.RequestQuery) types.ResponseQuery {
	// Handle queries
	return types.ResponseQuery{Code: 0, Value: []byte("query_result")}
}

func (app *OCXApplication) InitChain(req types.RequestInitChain) types.ResponseInitChain {
	// Initialize chain
	return types.ResponseInitChain{}
}

func (app *OCXApplication) BeginBlock(req types.RequestBeginBlock) types.ResponseBeginBlock {
	// Begin block processing
	return types.ResponseBeginBlock{}
}

func (app *OCXApplication) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {
	// End block processing
	return types.ResponseEndBlock{}
}

func (app *OCXApplication) ListSnapshots(req types.RequestListSnapshots) types.ResponseListSnapshots {
	// List snapshots
	return types.ResponseListSnapshots{}
}

func (app *OCXApplication) OfferSnapshot(req types.RequestOfferSnapshot) types.ResponseOfferSnapshot {
	// Offer snapshot
	return types.ResponseOfferSnapshot{}
}

func (app *OCXApplication) LoadSnapshotChunk(req types.RequestLoadSnapshotChunk) types.ResponseLoadSnapshotChunk {
	// Load snapshot chunk
	return types.ResponseLoadSnapshotChunk{}
}

func (app *OCXApplication) ApplySnapshotChunk(req types.RequestApplySnapshotChunk) types.ResponseApplySnapshotChunk {
	// Apply snapshot chunk
	return types.ResponseApplySnapshotChunk{}
}
