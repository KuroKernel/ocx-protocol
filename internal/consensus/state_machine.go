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
	state        *OCXState
	validators   *ValidatorSet
	reputation   *ReputationEngine
	settlement   *SettlementEngine
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
	SessionID         string                 `json:"session_id"`
	ConnectionDetails EncryptedConnectionInfo `json:"connection_details"`
	ResourceAllocation ResourceMap           `json:"resource_allocation"`
	ProviderSignature []byte                 `json:"provider_signature"`
}

type MsgUpdateSessionMetrics struct {
	SessionID  string                `json:"session_id"`
	Metrics    SessionMetricsSnapshot `json:"metrics"`
	Timestamp  time.Time             `json:"timestamp"`
	Signature  []byte                `json:"signature"`
}

type MsgSettleSession struct {
	SessionID          string      `json:"session_id"`
	FinalUsageReport   UsageReport `json:"final_usage_report"`
	ProviderSignature  []byte      `json:"provider_signature"`
	RequesterSignature []byte      `json:"requester_signature"`
}

// Data structures
type OrderSpecification struct {
	HardwareType     string    `json:"hardware_type"`
	GPUModel         string    `json:"gpu_model"`
	GPUMemoryGB      int       `json:"gpu_memory_gb"`
	CPUCount         int       `json:"cpu_count"`
	RAMGB            int       `json:"ram_gb"`
	StorageGB        int       `json:"storage_gb"`
	DurationHours    float64   `json:"duration_hours"`
	MaxPricePerHour  float64   `json:"max_price_per_hour"`
	WorkloadType     string    `json:"workload_type"`
	ContainerImage   string    `json:"container_image"`
	StartupScript    string    `json:"startup_script"`
	EnvironmentVars  map[string]string `json:"environment_vars"`
	PreferredRegions []string  `json:"preferred_regions"`
	MinReputation    float64   `json:"min_reputation"`
	MaxProvisionTime int       `json:"max_provision_time"`
	ComplianceCerts  []string  `json:"compliance_certs"`
	BudgetUSDC       float64   `json:"budget_usdc"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type ComputeUnitOffer struct {
	UnitID           string  `json:"unit_id"`
	ProviderID       string  `json:"provider_id"`
	HardwareType     string  `json:"hardware_type"`
	GPUModel         string  `json:"gpu_model"`
	GPUMemoryGB      int     `json:"gpu_memory_gb"`
	CPUCount         int     `json:"cpu_count"`
	RAMGB            int     `json:"ram_gb"`
	StorageGB        int     `json:"storage_gb"`
	PricePerHour     float64 `json:"price_per_hour"`
	Availability     string  `json:"availability"`
	ReputationScore  float64 `json:"reputation_score"`
	ProvisionTimeSec int     `json:"provision_time_sec"`
}

type EncryptedConnectionInfo struct {
	SSHEndpoint    string `json:"ssh_endpoint"`
	SSHUser        string `json:"ssh_user"`
	SSHKey         string `json:"ssh_key"`
	APIEndpoint    string `json:"api_endpoint"`
	APIToken       string `json:"api_token"`
	EncryptedData  []byte `json:"encrypted_data"`
}

type ResourceMap struct {
	GPUDevices    []int `json:"gpu_devices"`
	CPUCores      []int `json:"cpu_cores"`
	RAMGB         int   `json:"ram_gb"`
	StoragePath   string `json:"storage_path"`
	NetworkConfig string `json:"network_config"`
}

type SessionMetricsSnapshot struct {
	GPUUtilization    int     `json:"gpu_utilization_percent"`
	GPUMemoryUsed     int     `json:"gpu_memory_used_mb"`
	GPUTemperature    int     `json:"gpu_temperature_celsius"`
	GPUPowerDraw      int     `json:"gpu_power_draw_watts"`
	CPUUtilization    int     `json:"cpu_utilization_percent"`
	RAMUsed           float64 `json:"ram_used_gb"`
	DiskIORead        float64 `json:"disk_io_read_mbps"`
	DiskIOWrite       float64 `json:"disk_io_write_mbps"`
	NetworkRX         float64 `json:"network_rx_mbps"`
	NetworkTX         float64 `json:"network_tx_mbps"`
	TrainingStepsSec  float64 `json:"training_steps_per_second"`
	InferenceTokensSec float64 `json:"inference_tokens_per_second"`
	BatchSize         int     `json:"batch_size_processed"`
	MemoryPeak        int     `json:"memory_peak_mb"`
}

type UsageReport struct {
	SessionID        string    `json:"session_id"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	TotalHours       float64   `json:"total_hours"`
	BaseCost         float64   `json:"base_cost_usdc"`
	UsagePremiums    float64   `json:"usage_premiums_usdc"`
	TotalCost        float64   `json:"total_cost_usdc"`
	AverageUtilization float64 `json:"average_utilization_percent"`
	PeakUtilization  int       `json:"peak_utilization_percent"`
	MetricsHash      string    `json:"metrics_hash"`
}

// Core state machine methods

// NewOCXStateMachine creates a new OCX state machine
func NewOCXStateMachine() *OCXStateMachine {
	return &OCXStateMachine{
		app: &OCXApplication{
			state: &OCXState{
				Providers:      make(map[string]*Provider),
				Orders:         make(map[string]*Order),
				Sessions:       make(map[string]*Session),
				Reputation:     make(map[string]*ReputationScore),
				EscrowAccounts: make(map[string]*EscrowAccount),
			},
			validators: &ValidatorSet{
				Validators: make(map[string]*Validator),
				Stake:      make(map[string]int64),
			},
		},
	}
}

// ValidateOrderPlacement validates order placement messages
func (app *OCXApplication) ValidateOrderPlacement(ctx context.Context, msg MsgPlaceOrder) error {
	// 1. Verify escrow deposit on connected blockchain
	if !app.verifyEscrowDeposit(msg.EscrowTxHash, msg.OrderSpec.BudgetUSDC) {
		return fmt.Errorf("escrow deposit verification failed")
	}
	
	// 2. Validate order parameters against protocol limits
	if err := app.validateOrderParameters(msg.OrderSpec); err != nil {
		return fmt.Errorf("invalid order parameters: %w", err)
	}
	
	// 3. Check requester reputation for large orders
	if msg.OrderSpec.BudgetUSDC > 10000 { // Large order threshold
		if !app.checkRequesterReputation(msg.RequesterID, msg.OrderSpec.BudgetUSDC) {
			return fmt.Errorf("insufficient requester reputation for large order")
		}
	}
	
	// 4. Ensure no duplicate orders from same requester
	if app.hasActiveOrderFromRequester(msg.RequesterID) {
		return fmt.Errorf("requester already has active order")
	}
	
	return nil
}

// ExecuteMatching executes order matching logic
func (app *OCXApplication) ExecuteMatching(ctx context.Context, msg MsgMatchOrder) error {
	// 1. Verify provider owns claimed compute units
	if !app.verifyProviderOwnsUnits(msg.ProviderID, msg.ProvidedUnits) {
		return fmt.Errorf("provider does not own claimed compute units")
	}
	
	// 2. Check unit availability in current state
	if !app.checkUnitAvailability(msg.ProvidedUnits) {
		return fmt.Errorf("claimed units are not available")
	}
	
	// 3. Validate pricing and reputation requirements
	order := app.state.Orders[msg.OrderID]
	if order == nil {
		return fmt.Errorf("order not found")
	}
	
	if !app.validatePricingAndReputation(msg.ProvidedUnits, order) {
		return fmt.Errorf("pricing or reputation requirements not met")
	}
	
	// 4. Atomically reserve units and update order status
	if err := app.reserveUnits(msg.ProvidedUnits); err != nil {
		return fmt.Errorf("failed to reserve units: %w", err)
	}
	
	// 5. Update order status to matched
	order.Status = "matched"
	order.MatchedAt = time.Now()
	order.MatchedProviderID = msg.ProviderID
	
	// 6. Emit matching event for off-chain notification
	app.emitMatchingEvent(msg.OrderID, msg.ProviderID, msg.ProvidedUnits)
	
	return nil
}

// ExecuteProvisioning handles session provisioning
func (app *OCXApplication) ExecuteProvisioning(ctx context.Context, msg MsgProvisionSession) error {
	// 1. Verify session exists and is in correct state
	session := app.state.Sessions[msg.SessionID]
	if session == nil {
		return fmt.Errorf("session not found")
	}
	
	if session.Status != "provisioning" {
		return fmt.Errorf("session not in provisioning state")
	}
	
	// 2. Verify provider signature
	if !app.verifyProviderSignature(msg.ProviderSignature, msg) {
		return fmt.Errorf("invalid provider signature")
	}
	
	// 3. Update session with connection details
	session.ConnectionDetails = msg.ConnectionDetails
	session.ResourceAllocation = msg.ResourceAllocation
	session.Status = "active"
	session.StartedAt = time.Now()
	
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
	if !app.verifyProviderSignature(msg.ProviderSignature, msg) {
		return fmt.Errorf("invalid provider signature")
	}
	
	if !app.verifyRequesterSignature(msg.RequesterSignature, msg) {
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
	app.updateReputationScores(session, msg.FinalUsageReport)
	
	// 7. Emit settlement event
	app.emitSettlementEvent(msg.SessionID, settlement)
	
	return nil
}

// Helper methods (simplified implementations)

func (app *OCXApplication) verifyEscrowDeposit(txHash string, amount float64) bool {
	// In a real implementation, this would verify the blockchain transaction
	return txHash != "" && amount > 0
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

func (app *OCXApplication) checkRequesterReputation(requesterID string, amount float64) bool {
	// Simplified reputation check
	// In real implementation, would check actual reputation scores
	return true
}

func (app *OCXApplication) hasActiveOrderFromRequester(requesterID string) bool {
	for _, order := range app.state.Orders {
		if order.RequesterID == requesterID && order.Status == "pending_matching" {
			return true
		}
	}
	return false
}

func (app *OCXApplication) verifyProviderOwnsUnits(providerID string, units []ComputeUnitOffer) bool {
	// Simplified verification
	// In real implementation, would check actual ownership
	return true
}

func (app *OCXApplication) checkUnitAvailability(units []ComputeUnitOffer) bool {
	// Simplified availability check
	// In real implementation, would check actual unit status
	return true
}

func (app *OCXApplication) validatePricingAndReputation(units []ComputeUnitOffer, order *Order) bool {
	// Simplified validation
	// In real implementation, would check actual pricing and reputation
	return true
}

func (app *OCXApplication) reserveUnits(units []ComputeUnitOffer) error {
	// Simplified unit reservation
	// In real implementation, would update unit status atomically
	return nil
}

func (app *OCXApplication) verifyProviderSignature(signature []byte, msg interface{}) bool {
	// Simplified signature verification
	// In real implementation, would verify actual Ed25519 signature
	return true
}

func (app *OCXApplication) verifyRequesterSignature(signature []byte, msg interface{}) bool {
	// Simplified signature verification
	// In real implementation, would verify actual Ed25519 signature
	return true
}

func (app *OCXApplication) calculateSettlement(session *Session, report UsageReport) *Settlement {
	// Simplified settlement calculation
	// In real implementation, would calculate actual costs and fees
	return &Settlement{
		BaseCost:      report.BaseCost,
		UsagePremiums: report.UsagePremiums,
		TotalCost:     report.TotalCost,
		ProviderNet:   report.TotalCost * 0.9, // 90% to provider
		ProtocolFee:   report.TotalCost * 0.1, // 10% to protocol
	}
}

func (app *OCXApplication) processPayment(settlement *Settlement) error {
	// Simplified payment processing
	// In real implementation, would process actual blockchain transactions
	return nil
}

func (app *OCXApplication) updateReputationScores(session *Session, report UsageReport) {
	// Simplified reputation update
	// In real implementation, would update actual reputation scores
}

func (app *OCXApplication) emitMatchingEvent(orderID, providerID string, units []ComputeUnitOffer) {
	// Emit event for off-chain notification
}

func (app *OCXApplication) emitProvisioningEvent(sessionID string, details EncryptedConnectionInfo) {
	// Emit event for off-chain notification
}

func (app *OCXApplication) emitSettlementEvent(sessionID string, settlement *Settlement) {
	// Emit event for off-chain notification
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

type ReputationScore struct {
	Overall     float64
	Reliability float64
	Performance float64
	Availability float64
	Communication float64
	Economic    float64
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

type Settlement struct {
	BaseCost      float64
	UsagePremiums float64
	TotalCost     float64
	ProviderNet   float64
	ProtocolFee   float64
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
	// Parse and execute the transaction
	var msg interface{}
	if err := json.Unmarshal(req.Tx, &msg); err != nil {
		return types.ResponseDeliverTx{Code: 1, Log: "Invalid transaction format"}
	}
	
	// Route to appropriate handler based on message type
	// This is simplified - real implementation would have proper routing
	ctx := context.Background()
	
	switch msgType := msg.(type) {
	case MsgPlaceOrder:
		if err := app.ValidateOrderPlacement(ctx, msgType); err != nil {
			return types.ResponseDeliverTx{Code: 1, Log: err.Error()}
		}
	case MsgMatchOrder:
		if err := app.ExecuteMatching(ctx, msgType); err != nil {
			return types.ResponseDeliverTx{Code: 1, Log: err.Error()}
		}
	case MsgProvisionSession:
		if err := app.ExecuteProvisioning(ctx, msgType); err != nil {
			return types.ResponseDeliverTx{Code: 1, Log: err.Error()}
		}
	case MsgSettleSession:
		if err := app.ExecuteSettlement(ctx, msgType); err != nil {
			return types.ResponseDeliverTx{Code: 1, Log: err.Error()}
		}
	default:
		return types.ResponseDeliverTx{Code: 1, Log: "Unknown message type"}
	}
	
	return types.ResponseDeliverTx{Code: 0}
}

func (app *OCXApplication) CheckTx(req types.RequestCheckTx) types.ResponseCheckTx {
	// Basic transaction validation
	return types.ResponseCheckTx{Code: 0}
}

func (app *OCXApplication) Commit() types.ResponseCommit {
	// Commit the current state
	app.state.LastHeight++
	return types.ResponseCommit{Data: []byte("ocx_state_commit")}
}

func (app *OCXApplication) Query(req types.RequestQuery) types.ResponseQuery {
	// Handle queries to the state
	return types.ResponseQuery{Code: 0, Value: []byte("query_result")}
}

func (app *OCXApplication) InitChain(req types.RequestInitChain) types.ResponseInitChain {
	// Initialize the chain
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
	return types.ResponseListSnapshots{}
}

func (app *OCXApplication) OfferSnapshot(req types.RequestOfferSnapshot) types.ResponseOfferSnapshot {
	return types.ResponseOfferSnapshot{}
}

func (app *OCXApplication) LoadSnapshotChunk(req types.RequestLoadSnapshotChunk) types.ResponseLoadSnapshotChunk {
	return types.ResponseLoadSnapshotChunk{}
}

func (app *OCXApplication) ApplySnapshotChunk(req types.RequestApplySnapshotChunk) types.ResponseApplySnapshotChunk {
	return types.ResponseApplySnapshotChunk{}
}
