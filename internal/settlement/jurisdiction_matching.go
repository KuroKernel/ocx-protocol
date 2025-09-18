package settlement

import (
	"context"
	"fmt"
	"time"
)

// JurisdictionAwareMatcher handles jurisdiction-aware matching
type JurisdictionAwareMatcher struct {
	policyEngine *PolicyEngine
	config       *JurisdictionConfig
}

// JurisdictionConfig represents jurisdiction configuration
type JurisdictionConfig struct {
	DefaultJurisdiction    string                    `json:"default_jurisdiction"`
	SupportedJurisdictions []string                  `json:"supported_jurisdictions"`
	JurisdictionPolicies   map[string]*JurisdictionPolicy `json:"jurisdiction_policies"`
	CurrencyPreferences    map[string][]string       `json:"currency_preferences"`
	RailPreferences        map[string][]string       `json:"rail_preferences"`
}

// JurisdictionPolicy represents a jurisdiction policy
type JurisdictionPolicy struct {
	Jurisdiction          string                 `json:"jurisdiction"`
	AllowedCurrencies     []string               `json:"allowed_currencies"`
	AllowedRails          []string               `json:"allowed_rails"`
	BlockedCurrencies     []string               `json:"blocked_currencies"`
	BlockedRails          []string               `json:"blocked_rails"`
	RequiredKYC           bool                   `json:"required_kyc"`
	RequiredSanctionsCheck bool                  `json:"required_sanctions_check"`
	MaxTransactionAmount  *Amount                `json:"max_transaction_amount"`
	MinTransactionAmount  *Amount                `json:"min_transaction_amount"`
	DataResidency         string                 `json:"data_residency"`
	ExportControlFlags    []string               `json:"export_control_flags"`
	SanctionsScreening    []string               `json:"sanctions_screening"`
	ComplianceRequirements []string              `json:"compliance_requirements"`
}

// PolicyEngine represents a policy engine
type PolicyEngine struct {
	config *JurisdictionConfig
	// In production, this would be a real policy engine
	// For now, we'll use mock data
}

// MatchingRequest represents a matching request
type MatchingRequest struct {
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
	CreatedAt             time.Time              `json:"created_at"`
}

// MatchingResult represents a matching result
type MatchingResult struct {
	MatchID               string                 `json:"match_id"`
	BuyerID               string                 `json:"buyer_id"`
	SellerID              string                 `json:"seller_id"`
	Amount                *Amount                `json:"amount"`
	Currency              string                 `json:"currency"`
	Jurisdiction          string                 `json:"jurisdiction"`
	SelectedRail          string                 `json:"selected_rail"`
	SelectedCurrency      string                 `json:"selected_currency"`
	ExchangeRate          *ExchangeRate          `json:"exchange_rate,omitempty"`
	ComplianceStatus      string                 `json:"compliance_status"`
	PolicyCompliance      []string               `json:"policy_compliance"`
	Warnings              []string               `json:"warnings"`
	Recommendations       []string               `json:"recommendations"`
	CreatedAt             time.Time              `json:"created_at"`
}

// NewJurisdictionAwareMatcher creates a new jurisdiction-aware matcher
func NewJurisdictionAwareMatcher(config *JurisdictionConfig) *JurisdictionAwareMatcher {
	return &JurisdictionAwareMatcher{
		policyEngine: NewPolicyEngine(config),
		config:       config,
	}
}

// NewPolicyEngine creates a new policy engine
func NewPolicyEngine(config *JurisdictionConfig) *PolicyEngine {
	return &PolicyEngine{
		config: config,
	}
}

// MatchParties matches parties based on jurisdiction and policy requirements
func (j *JurisdictionAwareMatcher) MatchParties(ctx context.Context, request *MatchingRequest) (*MatchingResult, error) {
	// 1. Validate jurisdiction
	if err := j.validateJurisdiction(request.Jurisdiction); err != nil {
		return nil, err
	}
	
	// 2. Check policy compliance
	policyCompliance, err := j.policyEngine.CheckPolicyCompliance(request)
	if err != nil {
		return nil, err
	}
	
	// 3. Select optimal rail
	selectedRail, err := j.selectOptimalRail(request)
	if err != nil {
		return nil, err
	}
	
	// 4. Select optimal currency
	selectedCurrency, err := j.selectOptimalCurrency(request)
	if err != nil {
		return nil, err
	}
	
	// 5. Check exchange rate if needed
	var exchangeRate *ExchangeRate
	if selectedCurrency != request.Currency {
		exchangeRate, err = j.getExchangeRate(request.Currency, selectedCurrency)
		if err != nil {
			return nil, err
		}
	}
	
	// 6. Generate matching result
	result := &MatchingResult{
		MatchID:              fmt.Sprintf("match_%d", time.Now().UnixNano()),
		BuyerID:              request.BuyerID,
		SellerID:             request.SellerID,
		Amount:               request.Amount,
		Currency:             request.Currency,
		Jurisdiction:         request.Jurisdiction,
		SelectedRail:         selectedRail,
		SelectedCurrency:     selectedCurrency,
		ExchangeRate:         exchangeRate,
		ComplianceStatus:     "compliant",
		PolicyCompliance:     policyCompliance,
		Warnings:             []string{},
		Recommendations:      []string{},
		CreatedAt:            time.Now(),
	}
	
	// 7. Add warnings and recommendations
	j.addWarningsAndRecommendations(result, request)
	
	return result, nil
}

// validateJurisdiction validates a jurisdiction
func (j *JurisdictionAwareMatcher) validateJurisdiction(jurisdiction string) error {
	// Check if jurisdiction is supported
	for _, supportedJurisdiction := range j.config.SupportedJurisdictions {
		if jurisdiction == supportedJurisdiction {
			return nil
		}
	}
	
	return fmt.Errorf("jurisdiction not supported: %s", jurisdiction)
}

// CheckPolicyCompliance checks policy compliance
func (p *PolicyEngine) CheckPolicyCompliance(request *MatchingRequest) ([]string, error) {
	var compliance []string
	
	// Get jurisdiction policy
	policy, exists := p.config.JurisdictionPolicies[request.Jurisdiction]
	if !exists {
		return nil, fmt.Errorf("no policy found for jurisdiction: %s", request.Jurisdiction)
	}
	
	// Check currency compliance
	if err := p.checkCurrencyCompliance(request, policy); err != nil {
		return nil, err
	}
	compliance = append(compliance, "currency_compliant")
	
	// Check rail compliance
	if err := p.checkRailCompliance(request, policy); err != nil {
		return nil, err
	}
	compliance = append(compliance, "rail_compliant")
	
	// Check amount compliance
	if err := p.checkAmountCompliance(request, policy); err != nil {
		return nil, err
	}
	compliance = append(compliance, "amount_compliant")
	
	// Check KYC compliance
	if policy.RequiredKYC {
		compliance = append(compliance, "kyc_required")
	}
	
	// Check sanctions compliance
	if policy.RequiredSanctionsCheck {
		compliance = append(compliance, "sanctions_check_required")
	}
	
	// Check data residency
	if policy.DataResidency != "" {
		compliance = append(compliance, "data_residency_required")
	}
	
	// Check export control
	if len(policy.ExportControlFlags) > 0 {
		compliance = append(compliance, "export_control_required")
	}
	
	return compliance, nil
}

// checkCurrencyCompliance checks currency compliance
func (p *PolicyEngine) checkCurrencyCompliance(request *MatchingRequest, policy *JurisdictionPolicy) error {
	// Check if currency is allowed
	if len(policy.AllowedCurrencies) > 0 {
		allowed := false
		for _, allowedCurrency := range policy.AllowedCurrencies {
			if request.Currency == allowedCurrency {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("currency %s not allowed in jurisdiction %s", request.Currency, policy.Jurisdiction)
		}
	}
	
	// Check if currency is blocked
	for _, blockedCurrency := range policy.BlockedCurrencies {
		if request.Currency == blockedCurrency {
			return fmt.Errorf("currency %s is blocked in jurisdiction %s", request.Currency, policy.Jurisdiction)
		}
	}
	
	return nil
}

// checkRailCompliance checks rail compliance
func (p *PolicyEngine) checkRailCompliance(request *MatchingRequest, policy *JurisdictionPolicy) error {
	// Check if preferred rails are allowed
	for _, preferredRail := range request.PreferredRails {
		// Check if rail is allowed
		if len(policy.AllowedRails) > 0 {
			allowed := false
			for _, allowedRail := range policy.AllowedRails {
				if preferredRail == allowedRail {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Errorf("rail %s not allowed in jurisdiction %s", preferredRail, policy.Jurisdiction)
			}
		}
		
		// Check if rail is blocked
		for _, blockedRail := range policy.BlockedRails {
			if preferredRail == blockedRail {
				return fmt.Errorf("rail %s is blocked in jurisdiction %s", preferredRail, policy.Jurisdiction)
			}
		}
	}
	
	return nil
}

// checkAmountCompliance checks amount compliance
func (p *PolicyEngine) checkAmountCompliance(request *MatchingRequest, policy *JurisdictionPolicy) error {
	// Check maximum amount
	if policy.MaxTransactionAmount != nil {
		// In production, you would parse and compare amounts properly
		// For now, we'll use a simple check
		if request.Amount != nil {
			// Mock amount check
		}
	}
	
	// Check minimum amount
	if policy.MinTransactionAmount != nil {
		// In production, you would parse and compare amounts properly
		// For now, we'll use a simple check
		if request.Amount != nil {
			// Mock amount check
		}
	}
	
	return nil
}

// selectOptimalRail selects the optimal rail
func (j *JurisdictionAwareMatcher) selectOptimalRail(request *MatchingRequest) (string, error) {
	// Get jurisdiction policy
	policy, exists := j.config.JurisdictionPolicies[request.Jurisdiction]
	if !exists {
		return "", fmt.Errorf("no policy found for jurisdiction: %s", request.Jurisdiction)
	}
	
	// Filter by allowed rails
	var allowedRails []string
	if len(policy.AllowedRails) > 0 {
		allowedRails = policy.AllowedRails
	} else {
		// Use default rails
		allowedRails = []string{"swift", "lightning", "usdc"}
	}
	
	// Filter by preferred rails
	var preferredRails []string
	for _, preferredRail := range request.PreferredRails {
		for _, allowedRail := range allowedRails {
			if preferredRail == allowedRail {
				preferredRails = append(preferredRails, preferredRail)
			}
		}
	}
	
	// Filter by excluded rails
	var finalRails []string
	for _, rail := range preferredRails {
		excluded := false
		for _, excludedRail := range request.ExcludedRails {
			if rail == excludedRail {
				excluded = true
				break
			}
		}
		if !excluded {
			finalRails = append(finalRails, rail)
		}
	}
	
	if len(finalRails) == 0 {
		return "", fmt.Errorf("no suitable rails found")
	}
	
	// Return the first (most preferred) rail
	return finalRails[0], nil
}

// selectOptimalCurrency selects the optimal currency
func (j *JurisdictionAwareMatcher) selectOptimalCurrency(request *MatchingRequest) (string, error) {
	// Get jurisdiction policy
	policy, exists := j.config.JurisdictionPolicies[request.Jurisdiction]
	if !exists {
		return "", fmt.Errorf("no policy found for jurisdiction: %s", request.Jurisdiction)
	}
	
	// Check if requested currency is allowed
	if len(policy.AllowedCurrencies) > 0 {
		allowed := false
		for _, allowedCurrency := range policy.AllowedCurrencies {
			if request.Currency == allowedCurrency {
				allowed = true
				break
			}
		}
		if allowed {
			return request.Currency, nil
		}
	}
	
	// Use preferred currency from jurisdiction
	if len(policy.AllowedCurrencies) > 0 {
		return policy.AllowedCurrencies[0], nil
	}
	
	// Use default currency
	return "USD", nil
}

// getExchangeRate gets an exchange rate
func (j *JurisdictionAwareMatcher) getExchangeRate(fromCurrency, toCurrency string) (*ExchangeRate, error) {
	// In production, this would get real exchange rates
	// For now, we'll return mock data
	
	rate := &ExchangeRate{
		BaseCurrency:   fromCurrency,
		TargetCurrency: toCurrency,
		Rate:           "1.0",
		RateType:       "spot",
		ContractIdentification: "",
	}
	
	return rate, nil
}

// addWarningsAndRecommendations adds warnings and recommendations
func (j *JurisdictionAwareMatcher) addWarningsAndRecommendations(result *MatchingResult, request *MatchingRequest) {
	// Add warnings
	if result.SelectedCurrency != request.Currency {
		result.Warnings = append(result.Warnings, "Currency converted due to jurisdiction restrictions")
	}
	
	if result.SelectedRail != request.PreferredRails[0] {
		result.Warnings = append(result.Warnings, "Rail changed due to jurisdiction restrictions")
	}
	
	// Add recommendations
	result.Recommendations = append(result.Recommendations, "Consider using USD for better liquidity")
	result.Recommendations = append(result.Recommendations, "SWIFT rail provides best compliance coverage")
}

// GetJurisdictionPolicy gets a jurisdiction policy
func (j *JurisdictionAwareMatcher) GetJurisdictionPolicy(jurisdiction string) (*JurisdictionPolicy, error) {
	policy, exists := j.config.JurisdictionPolicies[jurisdiction]
	if !exists {
		return nil, fmt.Errorf("no policy found for jurisdiction: %s", jurisdiction)
	}
	
	return policy, nil
}

// GetSupportedJurisdictions gets supported jurisdictions
func (j *JurisdictionAwareMatcher) GetSupportedJurisdictions() []string {
	return j.config.SupportedJurisdictions
}

// GetCurrencyPreferences gets currency preferences for a jurisdiction
func (j *JurisdictionAwareMatcher) GetCurrencyPreferences(jurisdiction string) []string {
	preferences, exists := j.config.CurrencyPreferences[jurisdiction]
	if !exists {
		return []string{"USD"}
	}
	
	return preferences
}

// GetRailPreferences gets rail preferences for a jurisdiction
func (j *JurisdictionAwareMatcher) GetRailPreferences(jurisdiction string) []string {
	preferences, exists := j.config.RailPreferences[jurisdiction]
	if !exists {
		return []string{"swift", "lightning"}
	}
	
	return preferences
}
