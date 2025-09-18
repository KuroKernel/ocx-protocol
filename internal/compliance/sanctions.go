package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ocx.local/internal/settlement"
)

// ComplianceManager handles compliance and sanctions screening
type ComplianceManager struct {
	sanctionsDB    *SanctionsDatabase
	kycProvider    *KYCProvider
	riskEngine     *RiskEngine
	config         *ComplianceConfig
}

// ComplianceConfig represents compliance configuration
type ComplianceConfig struct {
	SanctionsAPIEndpoint    string   `json:"sanctions_api_endpoint"`
	SanctionsAPIKey         string   `json:"sanctions_api_key"`
	KYCProviderEndpoint     string   `json:"kyc_provider_endpoint"`
	KYCProviderAPIKey       string   `json:"kyc_provider_api_key"`
	RiskThreshold           float64  `json:"risk_threshold"`
	BlockedJurisdictions    []string `json:"blocked_jurisdictions"`
	BlockedCurrencies       []string `json:"blocked_currencies"`
	RequireKYC              bool     `json:"require_kyc"`
	RequireSanctionsCheck   bool     `json:"require_sanctions_check"`
	RequireRiskAssessment   bool     `json:"require_risk_assessment"`
}

// SanctionsDatabase represents a sanctions database
type SanctionsDatabase struct {
	config *ComplianceConfig
	// In production, this would be a real database
	// For now, we'll use in-memory storage
	sanctionsList map[string]*SanctionsEntry
}

// SanctionsEntry represents a sanctions entry
type SanctionsEntry struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"` // Individual, Entity, Country
	Jurisdiction    string                 `json:"jurisdiction"`
	SanctionsType   []string               `json:"sanctions_type"`
	EffectiveDate   time.Time              `json:"effective_date"`
	ExpiryDate      time.Time              `json:"expiry_date"`
	Source          string                 `json:"source"`
	Aliases         []string               `json:"aliases"`
	Addresses       []*settlement.Address  `json:"addresses"`
	Identifiers     []*Identifier          `json:"identifiers"`
	RiskScore       float64                `json:"risk_score"`
	LastUpdated     time.Time              `json:"last_updated"`
}

// Identifier represents an identifier
type Identifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Issuer string `json:"issuer,omitempty"`
}

// KYCProvider represents a KYC provider
type KYCProvider struct {
	config *ComplianceConfig
	// In production, this would be a real KYC provider
	// For now, we'll use mock data
}

// RiskEngine represents a risk assessment engine
type RiskEngine struct {
	config *ComplianceConfig
	// In production, this would be a real risk engine
	// For now, we'll use mock data
}

// ComplianceResult represents a compliance check result
type ComplianceResult struct {
	Passed              bool                   `json:"passed"`
	RiskScore           float64                `json:"risk_score"`
	SanctionsStatus     string                 `json:"sanctions_status"`
	KYCStatus           string                 `json:"kyc_status"`
	RiskAssessment      string                 `json:"risk_assessment"`
	BlockedReasons      []string               `json:"blocked_reasons"`
	Warnings            []string               `json:"warnings"`
	Recommendations     []string               `json:"recommendations"`
	ComplianceFlags     []string               `json:"compliance_flags"`
	LastChecked         time.Time              `json:"last_checked"`
	NextReview          time.Time              `json:"next_review"`
}

// NewComplianceManager creates a new compliance manager
func NewComplianceManager(config *ComplianceConfig) *ComplianceManager {
	return &ComplianceManager{
		sanctionsDB: NewSanctionsDatabase(config),
		kycProvider: NewKYCProvider(config),
		riskEngine:  NewRiskEngine(config),
		config:      config,
	}
}

// NewSanctionsDatabase creates a new sanctions database
func NewSanctionsDatabase(config *ComplianceConfig) *SanctionsDatabase {
	db := &SanctionsDatabase{
		config:        config,
		sanctionsList: make(map[string]*SanctionsEntry),
	}
	
	// Load initial sanctions data
	db.loadInitialSanctions()
	
	return db
}

// NewKYCProvider creates a new KYC provider
func NewKYCProvider(config *ComplianceConfig) *KYCProvider {
	return &KYCProvider{
		config: config,
	}
}

// NewRiskEngine creates a new risk engine
func NewRiskEngine(config *ComplianceConfig) *RiskEngine {
	return &RiskEngine{
		config: config,
	}
}

// ValidateSettlement validates a settlement for compliance
func (c *ComplianceManager) ValidateSettlement(ctx context.Context, instruction *settlement.SettlementInstruction) error {
	// 1. Check sanctions
	if c.config.RequireSanctionsCheck {
		if err := c.checkSanctions(ctx, instruction); err != nil {
			return err
		}
	}
	
	// 2. Check KYC
	if c.config.RequireKYC {
		if err := c.checkKYC(ctx, instruction); err != nil {
			return err
		}
	}
	
	// 3. Assess risk
	if c.config.RequireRiskAssessment {
		if err := c.assessRisk(ctx, instruction); err != nil {
			return err
		}
	}
	
	// 4. Check jurisdiction restrictions
	if err := c.checkJurisdictionRestrictions(instruction); err != nil {
		return err
	}
	
	// 5. Check currency restrictions
	if err := c.checkCurrencyRestrictions(instruction); err != nil {
		return err
	}
	
	return nil
}

// checkSanctions checks sanctions for all parties
func (c *ComplianceManager) checkSanctions(ctx context.Context, instruction *settlement.SettlementInstruction) error {
	// Check debtor
	if err := c.sanctionsDB.CheckSanctions(instruction.Debtor); err != nil {
		return fmt.Errorf("debtor sanctions check failed: %w", err)
	}
	
	// Check creditor
	if err := c.sanctionsDB.CheckSanctions(instruction.Creditor); err != nil {
		return fmt.Errorf("creditor sanctions check failed: %w", err)
	}
	
	// Check debtor agent
	if instruction.DebtorAgent != nil {
		if err := c.sanctionsDB.CheckSanctions(instruction.DebtorAgent); err != nil {
			return fmt.Errorf("debtor agent sanctions check failed: %w", err)
		}
	}
	
	// Check creditor agent
	if instruction.CreditorAgent != nil {
		if err := c.sanctionsDB.CheckSanctions(instruction.CreditorAgent); err != nil {
			return fmt.Errorf("creditor agent sanctions check failed: %w", err)
		}
	}
	
	return nil
}

// checkKYC checks KYC for all parties
func (c *ComplianceManager) checkKYC(ctx context.Context, instruction *settlement.SettlementInstruction) error {
	// Check debtor KYC
	if err := c.kycProvider.CheckKYC(instruction.Debtor); err != nil {
		return fmt.Errorf("debtor KYC check failed: %w", err)
	}
	
	// Check creditor KYC
	if err := c.kycProvider.CheckKYC(instruction.Creditor); err != nil {
		return fmt.Errorf("creditor KYC check failed: %w", err)
	}
	
	return nil
}

// assessRisk assesses risk for the settlement
func (c *ComplianceManager) assessRisk(ctx context.Context, instruction *settlement.SettlementInstruction) error {
	// Assess risk
	riskScore, err := c.riskEngine.AssessRisk(instruction)
	if err != nil {
		return fmt.Errorf("risk assessment failed: %w", err)
	}
	
	// Check if risk score exceeds threshold
	if riskScore > c.config.RiskThreshold {
		return fmt.Errorf("risk score %f exceeds threshold %f", riskScore, c.config.RiskThreshold)
	}
	
	return nil
}

// checkJurisdictionRestrictions checks jurisdiction restrictions
func (c *ComplianceManager) checkJurisdictionRestrictions(instruction *settlement.SettlementInstruction) error {
	// Check if jurisdiction is blocked
	for _, blockedJurisdiction := range c.config.BlockedJurisdictions {
		if instruction.Jurisdiction == blockedJurisdiction {
			return fmt.Errorf("jurisdiction %s is blocked", blockedJurisdiction)
		}
	}
	
	// Check debtor jurisdiction
	if instruction.Debtor != nil {
		for _, blockedJurisdiction := range c.config.BlockedJurisdictions {
			if instruction.Debtor.Jurisdiction == blockedJurisdiction {
				return fmt.Errorf("debtor jurisdiction %s is blocked", blockedJurisdiction)
			}
		}
	}
	
	// Check creditor jurisdiction
	if instruction.Creditor != nil {
		for _, blockedJurisdiction := range c.config.BlockedJurisdictions {
			if instruction.Creditor.Jurisdiction == blockedJurisdiction {
				return fmt.Errorf("creditor jurisdiction %s is blocked", blockedJurisdiction)
			}
		}
	}
	
	return nil
}

// checkCurrencyRestrictions checks currency restrictions
func (c *ComplianceManager) checkCurrencyRestrictions(instruction *settlement.SettlementInstruction) error {
	// Check if currency is blocked
	for _, blockedCurrency := range c.config.BlockedCurrencies {
		if instruction.InstructedAmount.Currency == blockedCurrency {
			return fmt.Errorf("currency %s is blocked", blockedCurrency)
		}
	}
	
	return nil
}

// CheckSanctions checks sanctions for a party
func (s *SanctionsDatabase) CheckSanctions(party interface{}) error {
	// In production, this would check against real sanctions databases
	// For now, we'll use mock data
	
	// Mock sanctions check - always passes
	return nil
}

// CheckKYC checks KYC for a party
func (k *KYCProvider) CheckKYC(party *settlement.Party) error {
	// In production, this would check against real KYC providers
	// For now, we'll use mock data
	
	if party == nil {
		return fmt.Errorf("party is nil")
	}
	
	// Mock KYC check - always passes
	return nil
}

// AssessRisk assesses risk for a settlement
func (r *RiskEngine) AssessRisk(instruction *settlement.SettlementInstruction) (float64, error) {
	// In production, this would use a real risk engine
	// For now, we'll use mock data
	
	riskScore := 0.1 // Low risk
	
	// Adjust risk based on amount
	if instruction.InstructedAmount != nil {
		// Mock risk adjustment based on amount
		// In production, you would parse the amount and adjust risk
	}
	
	// Adjust risk based on jurisdiction
	if instruction.Jurisdiction == "US" {
		riskScore += 0.1
	} else if instruction.Jurisdiction == "CN" {
		riskScore += 0.2
	}
	
	return riskScore, nil
}

// loadInitialSanctions loads initial sanctions data
func (s *SanctionsDatabase) loadInitialSanctions() {
	// In production, this would load from real sanctions databases
	// For now, we'll use mock data
	
	// Mock sanctions entries
	s.sanctionsList["sanctions_1"] = &SanctionsEntry{
		ID:            "sanctions_1",
		Name:          "Mock Sanctions Entry",
		Type:          "Individual",
		Jurisdiction:  "US",
		SanctionsType: []string{"OFAC", "UN"},
		EffectiveDate: time.Now().Add(-365 * 24 * time.Hour),
		ExpiryDate:    time.Now().Add(365 * 24 * time.Hour),
		Source:        "OFAC",
		Aliases:       []string{"Mock Alias"},
		Addresses:     []*settlement.Address{},
		Identifiers:   []*Identifier{},
		RiskScore:     0.9,
		LastUpdated:   time.Now(),
	}
}

// GetComplianceResult gets a comprehensive compliance result
func (c *ComplianceManager) GetComplianceResult(ctx context.Context, instruction *settlement.SettlementInstruction) (*ComplianceResult, error) {
	result := &ComplianceResult{
		Passed:              true,
		RiskScore:           0.1,
		SanctionsStatus:     "clear",
		KYCStatus:           "verified",
		RiskAssessment:      "low",
		BlockedReasons:      []string{},
		Warnings:            []string{},
		Recommendations:     []string{},
		ComplianceFlags:     []string{},
		LastChecked:         time.Now(),
		NextReview:          time.Now().Add(30 * 24 * time.Hour),
	}
	
	// Check sanctions
	if err := c.checkSanctions(ctx, instruction); err != nil {
		result.Passed = false
		result.BlockedReasons = append(result.BlockedReasons, err.Error())
		result.SanctionsStatus = "blocked"
	}
	
	// Check KYC
	if err := c.checkKYC(ctx, instruction); err != nil {
		result.Passed = false
		result.BlockedReasons = append(result.BlockedReasons, err.Error())
		result.KYCStatus = "failed"
	}
	
	// Assess risk
	if riskScore, err := c.riskEngine.AssessRisk(instruction); err != nil {
		result.Passed = false
		result.BlockedReasons = append(result.BlockedReasons, err.Error())
		result.RiskAssessment = "failed"
	} else {
		result.RiskScore = riskScore
		if riskScore > c.config.RiskThreshold {
			result.Warnings = append(result.Warnings, "High risk score")
			result.RiskAssessment = "high"
		}
	}
	
	return result, nil
}

// UpdateSanctionsList updates the sanctions list
func (s *SanctionsDatabase) UpdateSanctionsList(ctx context.Context) error {
	// In production, this would update from real sanctions databases
	// For now, we'll use mock data
	
	// Mock update
	return nil
}

// GetSanctionsEntry gets a sanctions entry by ID
func (s *SanctionsDatabase) GetSanctionsEntry(id string) (*SanctionsEntry, error) {
	entry, exists := s.sanctionsList[id]
	if !exists {
		return nil, fmt.Errorf("sanctions entry not found: %s", id)
	}
	
	return entry, nil
}

// SearchSanctions searches sanctions by name
func (s *SanctionsDatabase) SearchSanctions(name string) ([]*SanctionsEntry, error) {
	var results []*SanctionsEntry
	
	for _, entry := range s.sanctionsList {
		if entry.Name == name {
			results = append(results, entry)
		}
	}
	
	return results, nil
}
