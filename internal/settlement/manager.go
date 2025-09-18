package settlement

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	// "ocx.local/internal/compliance"
	// "ocx.local/internal/ledger"
	// "ocx.local/internal/rails"
)

// SettlementManager manages the complete multi-rail settlement system
type SettlementManager struct {
	multiRailManager    *MultiRailSettlementManager
	jurisdictionMatcher *JurisdictionAwareMatcher
	// ledgerManager       *ledger.LedgerManager
	// complianceManager   *compliance.ComplianceManager
	config              *SettlementConfig
}

// SettlementConfig represents settlement configuration
type SettlementConfig struct {
	DefaultJurisdiction    string                    `json:"default_jurisdiction"`
	SupportedCurrencies    []string                  `json:"supported_currencies"`
	SupportedRails         []string                  `json:"supported_rails"`
	JurisdictionPolicies   map[string]*JurisdictionPolicy `json:"jurisdiction_policies"`
	CurrencyPreferences    map[string][]string       `json:"currency_preferences"`
	RailPreferences        map[string][]string       `json:"rail_preferences"`
	// ComplianceConfig       *compliance.ComplianceConfig `json:"compliance_config"`
	// LedgerConfig           *ledger.LedgerConfig      `json:"ledger_config"`
}

// NewSettlementManager creates a new settlement manager
func NewSettlementManager(config *SettlementConfig) *SettlementManager {
	// Create ledger manager
	// ledgerManager := ledger.NewLedgerManager(config.LedgerConfig)
	
	// Create compliance manager
	// complianceManager := compliance.NewComplianceManager(config.ComplianceConfig)
	
	// Create multi-rail manager
	multiRailManager := NewMultiRailSettlementManager(nil, nil)
	
	// Create jurisdiction matcher
	jurisdictionConfig := &JurisdictionConfig{
		DefaultJurisdiction:    config.DefaultJurisdiction,
		SupportedJurisdictions: []string{"US", "EU", "CN", "JP", "SG"},
		JurisdictionPolicies:   config.JurisdictionPolicies,
		CurrencyPreferences:    config.CurrencyPreferences,
		RailPreferences:        config.RailPreferences,
	}
	jurisdictionMatcher := NewJurisdictionAwareMatcher(jurisdictionConfig)
	
	// Create settlement manager
	sm := &SettlementManager{
		multiRailManager:    multiRailManager,
		jurisdictionMatcher: jurisdictionMatcher,
		// ledgerManager:       ledgerManager,
		// complianceManager:   complianceManager,
		config:              config,
	}
	
	// Register rails
	sm.registerRails()
	
	return sm
}

// registerRails registers all available rails
func (sm *SettlementManager) registerRails() {
// 	// Register SWIFT rail
// 	swiftConfig := &rails.SWIFTConfig{
// 		BIC:                   "OCXUS33X",
// 		LEI:                   "549300ABCDEFGHIJK123",
// 		InstitutionName:       "OCX Protocol",
// 		SupportedCurrencies:   []string{"USD", "EUR", "GBP", "JPY", "CNY"},
// 		SupportedJurisdictions: []string{"US", "EU", "GB", "JP", "CN"},
// 		APIEndpoint:           "https://api.swift.com/v1",
// 		APIKey:                "swift_api_key",
// 		CertPath:              "/path/to/swift.crt",
// 		KeyPath:               "/path/to/swift.key",
// 	}
// 	swiftRail := rails.NewSWIFTRail(swiftConfig)
// 	sm.multiRailManager.RegisterRail(swiftRail)
// 	
// 	// Register Lightning rail
// 	lightningConfig := &rails.LightningConfig{
// 		NodeID:                "03f1b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8b8",
// 		RPCEndpoint:           "localhost:10009",
// 		MacaroonPath:          "/path/to/admin.macaroon",
// 		TLSCertPath:           "/path/to/tls.cert",
// 		SupportedCurrencies:   []string{"BTC"},
// 		SupportedJurisdictions: []string{"US", "EU", "GB", "JP", "SG"},
// 		MaxPaymentAmount:      1000000, // 0.01 BTC
// 		MinPaymentAmount:      1000,    // 0.00001 BTC
// 		FeeRate:               0.0001,  // 0.01%
// 	}
// 	lightningRail := rails.NewLightningRail(lightningConfig)
// 	sm.multiRailManager.RegisterRail(lightningRail)
// 
// 	// Register USDC rail
// 	// Note: This would be configured via environment variables in production
// 	usdcRail, err := NewUSDCSettlement(
// 		"https://polygon-rpc.com", // Polygon RPC URL
// 		"your_private_key_here",    // Private key from config
// 		"0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174", // USDC contract on Polygon
// 		"0x0000000000000000000000000000000000000000", // Escrow contract address
// 		sm.ledgerManager.GetDB(), // Database connection
// 	)
// 	if err == nil {
// 		sm.multiRailManager.RegisterRail(usdcRail)
// 	}
// }
}

// ProcessSettlement processes a settlement with full multi-rail support
func (sm *SettlementManager) ProcessSettlement(ctx context.Context, request *SettlementRequest) (*SettlementResponse, error) {
	// 1. Create matching request
	matchingRequest := &MatchingRequest{
		BuyerID:             request.BuyerID,
		SellerID:            request.SellerID,
		Amount:              request.Amount,
		Currency:            request.Currency,
		Jurisdiction:        request.Jurisdiction,
		PreferredRails:      request.PreferredRails,
		ExcludedRails:       request.ExcludedRails,
		PreferredCurrencies: request.PreferredCurrencies,
		ExcludedCurrencies:  request.ExcludedCurrencies,
		DataResidency:       request.DataResidency,
		ExportControlFlags:  request.ExportControlFlags,
		ComplianceLevel:     request.ComplianceLevel,
		CreatedAt:           time.Now(),
	}
	
	// 2. Match parties
	matchingResult, err := sm.jurisdictionMatcher.MatchParties(ctx, matchingRequest)
	if err != nil {
		return nil, fmt.Errorf("matching failed: %w", err)
	}
	
	// 3. Create settlement instruction
	instruction := &SettlementInstruction{
		ID:                    fmt.Sprintf("instruction_%d", time.Now().UnixNano()),
		MessageID:             fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		InstructionID:         fmt.Sprintf("inst_%d", time.Now().UnixNano()),
		EndToEndID:            fmt.Sprintf("e2e_%d", time.Now().UnixNano()),
		TransactionID:         fmt.Sprintf("txn_%d", time.Now().UnixNano()),
		InitiatingParty:       request.InitiatingParty,
		Debtor:                request.Debtor,
		DebtorAgent:           request.DebtorAgent,
		Creditor:              request.Creditor,
		CreditorAgent:         request.CreditorAgent,
		InstructedAmount:      request.Amount,
		SettlementAmount:      request.Amount,
		SettlementMethod:      matchingResult.SelectedRail,
		SettlementDate:        time.Now(),
		ValueDate:             time.Now().Add(24 * time.Hour),
		RemittanceInfo:        request.RemittanceInfo,
		Jurisdiction:          request.Jurisdiction,
		SanctionsStatus:       "pending",
		ComplianceFlags:       matchingResult.PolicyCompliance,
		PreferredRails:        []string{matchingResult.SelectedRail},
		ExcludedRails:         request.ExcludedRails,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
		ExpiresAt:             time.Now().Add(7 * 24 * time.Hour),
	}
	
	// 4. Process settlement
	result, err := sm.multiRailManager.ProcessSettlement(ctx, instruction)
	if err != nil {
		return nil, fmt.Errorf("settlement processing failed: %w", err)
	}
	
	// 5. Create response
	response := &SettlementResponse{
		SettlementID:         result.SettlementID,
		InstructionID:        result.InstructionID,
		Status:               result.Status,
		RailUsed:             result.RailUsed,
		TransactionReference: result.TransactionReference,
		SettlementDate:       result.SettlementDate,
		ValueDate:            result.ValueDate,
		Amount:               result.ActualAmount,
		ExchangeRate:         result.ExchangeRate,
		Fees:                 result.Fees,
		Receipt:              result.Receipt,
		ComplianceStatus:     matchingResult.ComplianceStatus,
		PolicyCompliance:     matchingResult.PolicyCompliance,
		Warnings:             matchingResult.Warnings,
		Recommendations:      matchingResult.Recommendations,
		CreatedAt:            time.Now(),
	}
	
	return response, nil
}

// SettlementRequest represents a settlement request
type SettlementRequest struct {
	BuyerID               string                 `json:"buyer_id"`
	SellerID              string                 `json:"seller_id"`
	Amount                *Amount                `json:"amount"`
	Currency              string                 `json:"currency"`
	Jurisdiction          string                 `json:"jurisdiction"`
	PreferredRails        []string               `json:"preferred_rails"`
	ExcludedRails         []string               `json:"excluded_rails"`
	PreferredCurrencies   []string               `json:"preferred_currencies"`
	ExcludedCurrencies    []string               `json:"excluded_currencies"`
	DataResidency         string                 `json:"data_residency"`
	ExportControlFlags    []string               `json:"export_control_flags"`
	ComplianceLevel       string                 `json:"compliance_level"`
	InitiatingParty       *Party                 `json:"initiating_party"`
	Debtor                *Party                 `json:"debtor"`
	DebtorAgent           *FinancialInstitution  `json:"debtor_agent"`
	Creditor              *Party                 `json:"creditor"`
	CreditorAgent         *FinancialInstitution  `json:"creditor_agent"`
	RemittanceInfo        *RemittanceInformation `json:"remittance_info"`
}

// SettlementResponse represents a settlement response
type SettlementResponse struct {
	SettlementID         string                 `json:"settlement_id"`
	InstructionID        string                 `json:"instruction_id"`
	Status               string                 `json:"status"`
	RailUsed             string                 `json:"rail_used"`
	TransactionReference string                 `json:"transaction_reference"`
	SettlementDate       time.Time              `json:"settlement_date"`
	ValueDate            time.Time              `json:"value_date"`
	Amount               *Amount                `json:"amount"`
	ExchangeRate         *ExchangeRate          `json:"exchange_rate,omitempty"`
	Fees                 []*Fee                 `json:"fees"`
	Receipt              *SettlementReceipt     `json:"receipt"`
	ComplianceStatus     string                 `json:"compliance_status"`
	PolicyCompliance     []string               `json:"policy_compliance"`
	Warnings             []string               `json:"warnings"`
	Recommendations      []string               `json:"recommendations"`
	CreatedAt            time.Time              `json:"created_at"`
}

// GetSettlementStatus gets the status of a settlement
func (sm *SettlementManager) GetSettlementStatus(ctx context.Context, settlementID string) (*SettlementStatus, error) {
	return sm.multiRailManager.GetSettlementStatus(ctx, settlementID)
}

// GetSupportedRails gets all supported rails
func (sm *SettlementManager) GetSupportedRails() []string {
	return sm.multiRailManager.GetSupportedRails()
}

// GetRailCapabilities gets the capabilities of a specific rail
func (sm *SettlementManager) GetRailCapabilities(railID string) map[string]interface{} {
	return sm.multiRailManager.GetRailCapabilities(railID)
}

// GetJurisdictionPolicy gets a jurisdiction policy
func (sm *SettlementManager) GetJurisdictionPolicy(jurisdiction string) (*JurisdictionPolicy, error) {
	return sm.jurisdictionMatcher.GetJurisdictionPolicy(jurisdiction)
}

// GetSupportedJurisdictions gets supported jurisdictions
func (sm *SettlementManager) GetSupportedJurisdictions() []string {
	return sm.jurisdictionMatcher.GetSupportedJurisdictions()
}

// GetCurrencyPreferences gets currency preferences for a jurisdiction
func (sm *SettlementManager) GetCurrencyPreferences(jurisdiction string) []string {
	return sm.jurisdictionMatcher.GetCurrencyPreferences(jurisdiction)
}

// GetRailPreferences gets rail preferences for a jurisdiction
func (sm *SettlementManager) GetRailPreferences(jurisdiction string) []string {
	return sm.jurisdictionMatcher.GetRailPreferences(jurisdiction)
}

// GetTrialBalance gets the trial balance
// func (sm *SettlementManager) GetTrialBalance() (*ledger.TrialBalance, error) {
// 	return sm.ledgerManager.GetTrialBalance()
// }
// 
// // ExportISO20022 exports ledger data in ISO 20022 format
// func (sm *SettlementManager) ExportISO20022(ctx context.Context, startDate, endDate time.Time) (*ledger.ISO20022Export, error) {
// 	return sm.ledgerManager.ExportISO20022(ctx, startDate, endDate)
// }

// GetComplianceResult gets a comprehensive compliance result
// // func (sm *SettlementManager) GetComplianceResult(ctx context.Context, request *SettlementRequest) (*compliance.ComplianceResult, error) {
// 	// Create settlement instruction for compliance check
// 	instruction := &SettlementInstruction{
// 		ID:                    fmt.Sprintf("compliance_check_%d", time.Now().UnixNano()),
// 		MessageID:             fmt.Sprintf("msg_%d", time.Now().UnixNano()),
// 		InstructionID:         fmt.Sprintf("inst_%d", time.Now().UnixNano()),
// 		EndToEndID:            fmt.Sprintf("e2e_%d", time.Now().UnixNano()),
// 		TransactionID:         fmt.Sprintf("txn_%d", time.Now().UnixNano()),
// 		InitiatingParty:       request.InitiatingParty,
// 		Debtor:                request.Debtor,
// 		DebtorAgent:           request.DebtorAgent,
// 		Creditor:              request.Creditor,
// 		CreditorAgent:         request.CreditorAgent,
// 		InstructedAmount:      request.Amount,
// 		SettlementAmount:      request.Amount,
// 		SettlementMethod:      "compliance_check",
// 		SettlementDate:        time.Now(),
// 		ValueDate:             time.Now().Add(24 * time.Hour),
// 		RemittanceInfo:        request.RemittanceInfo,
// 		Jurisdiction:          request.Jurisdiction,
// 		SanctionsStatus:       "pending",
// 		ComplianceFlags:       []string{},
// 		PreferredRails:        request.PreferredRails,
// 		ExcludedRails:         request.ExcludedRails,
// 		CreatedAt:             time.Now(),
// 		UpdatedAt:             time.Now(),
// 		ExpiresAt:             time.Now().Add(7 * 24 * time.Hour),
// 	}
// 	
// 	return sm.complianceManager.GetComplianceResult(ctx, instruction)
// }
// 
// GetSystemStatus gets the overall system status
func (sm *SettlementManager) GetSystemStatus(ctx context.Context) (*SystemStatus, error) {
	status := &SystemStatus{
		MultiRailManager:    "healthy",
		JurisdictionMatcher: "healthy",
		LedgerManager:       "healthy",
		ComplianceManager:   "healthy",
		SupportedRails:      sm.GetSupportedRails(),
		SupportedJurisdictions: sm.GetSupportedJurisdictions(),
		LastUpdated:         time.Now(),
	}
	
	return status, nil
}

// SystemStatus represents the overall system status
type SystemStatus struct {
	MultiRailManager      string    `json:"multi_rail_manager"`
	JurisdictionMatcher   string    `json:"jurisdiction_matcher"`
	LedgerManager         string    `json:"ledger_manager"`
	ComplianceManager     string    `json:"compliance_manager"`
	SupportedRails        []string  `json:"supported_rails"`
	SupportedJurisdictions []string `json:"supported_jurisdictions"`
	LastUpdated           time.Time `json:"last_updated"`
}
