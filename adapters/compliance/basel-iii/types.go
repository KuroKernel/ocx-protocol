// Basel III Compliance Data Structures
// Enterprise-grade types for regulatory compliance
package basel

import (
	"fmt"
	"time"
)

// RWACalculationRequest represents a Risk-Weighted Assets calculation request
type RWACalculationRequest struct {
	BankID                  string
	CalculationDate         time.Time
	CalculationCode         []byte // Deterministic risk calculation code
	InputData              []byte // Market data, position data
	CapitalRatio           float64
	LeverageRatio          float64
	LiquidityCoverageRatio float64
	AssetPortfolio         []Asset
}

// Asset represents a financial asset in the bank's portfolio
type Asset struct {
	AssetID     string
	AssetType   AssetType
	Value       float64
	RiskWeight  float64
	Currency    string
}

// AssetType represents different types of financial assets
type AssetType int

const (
	GovernmentBonds AssetType = iota
	CorporateBonds
	Mortgages
	CommercialLoans
	Derivatives
	Cash
)

// BaselComplianceResult represents the result of Basel III compliance verification
type BaselComplianceResult struct {
	ComplianceID       string
	BankID            string
	Timestamp         time.Time
	IsCompliant       bool
	ViolationReasons  []string
	CryptographicProof CryptographicProof
	AuditTrail        []AuditEntry
	RegulatoryGrade   string // A, B, C, D, F
}

// CryptographicProof provides mathematical certainty for compliance verification
type CryptographicProof struct {
	ExecutionHash        string
	SignatureProof       bool
	DeterministicProof   bool
	VerificationTime     time.Duration
	MathematicalCertainty bool // Always true with dual-library system
}

// AuditEntry represents an entry in the compliance audit trail
type AuditEntry struct {
	Timestamp   time.Time
	Event       string
	Details     map[string]interface{}
	SystemHash  string // Links to OCX receipt system
}

// RegulatoryReport represents a complete regulatory compliance report
type RegulatoryReport struct {
	ReportID          string
	BankID           string
	ReportingPeriod   ReportingPeriod
	GeneratedAt       time.Time
	ComplianceStatus  string
	Entries          []ReportEntry
	ExecutiveSummary  ExecutiveSummary
	DigitalSignature  []byte // Signed with OCX crypto system
}

// ReportingPeriod defines the time period for regulatory reporting
type ReportingPeriod struct {
	StartDate time.Time
	EndDate   time.Time
	Quarter   int
	Year      int
}

// ReportEntry represents a single entry in the regulatory report
type ReportEntry struct {
	CalculationID    string
	CalculationType  string
	Result          map[string]interface{}
	VerificationProof CryptographicProof
	ComplianceStatus bool
}

// ExecutiveSummary provides high-level compliance overview
type ExecutiveSummary struct {
	TotalCalculations      int
	CompliantCalculations  int
	CompliancePercentage   float64
	KeyRiskMetrics        map[string]float64
	RegulatoryRecommendations []string
}

// CalculationRecord represents a stored calculation for reporting
type CalculationRecord struct {
	ID               string
	Type             string
	Result           map[string]interface{}
	CryptographicProof CryptographicProof
	ComplianceResult *BaselComplianceResult
	Timestamp        time.Time
}

// BaselError represents Basel III specific errors
type BaselError struct {
	Code    ErrorCode
	Message string
}

// ErrorCode represents different types of Basel III errors
type ErrorCode int

const (
	ErrorCodeExecutionFailed ErrorCode = iota
	ErrorCodeVerificationFailed
	ErrorCodeReceiptGenerationFailed
	ErrorCodeComplianceViolation
	ErrorCodeReportGenerationFailed
)

func (e *BaselError) Error() string {
	return fmt.Sprintf("Basel III Error [%d]: %s", e.Code, e.Message)
}

// NewBaselError creates a new Basel III error
func NewBaselError(code ErrorCode, message string) *BaselError {
	return &BaselError{
		Code:    code,
		Message: message,
	}
}

// PricingQuote represents a pricing quote for Basel III compliance services
type PricingQuote struct {
	TierName          string
	MonthlyBase       int64 // USD cents
	OverageCharges    int64 // USD cents
	TotalMonthly      int64 // USD cents
	AnnualContract    int64 // USD cents
	CalculationsIncluded int
	ExcessCalculations int
}

// ServiceLevel represents different service level agreements
type ServiceLevel int

const (
	SLA_99_5 ServiceLevel = iota // 99.5% uptime
	SLA_99_9                     // 99.9% uptime
	SLA_99_99                    // 99.99% uptime
)

// SupportLevel represents different support levels
type SupportLevel int

const (
	SupportEmailOnly SupportLevel = iota
	SupportBusinessHours
	Support24x7Dedicated
)

// PricingTier represents a pricing tier for Basel III compliance
type PricingTier struct {
	TierName          string
	MonthlyBase       int64 // USD cents
	PerCalculation    int64 // USD cents per verification
	CalculationsIncluded int
	SLA              ServiceLevel
	Support          SupportLevel
}

// BaselPricingTiers defines the pricing structure for different bank tiers
var BaselPricingTiers = map[ComplianceLevel]PricingTier{
	Tier1Bank: { // JPMorgan Chase, Bank of America level
		TierName:          "Tier 1 - Systemically Important",
		MonthlyBase:       500000000, // $5M/month base
		PerCalculation:    100000,    // $1000/calculation
		CalculationsIncluded: 10000,
		SLA:              SLA_99_99,
		Support:          Support24x7Dedicated,
	},
	Tier2Bank: { // Regional banks
		TierName:          "Tier 2 - Regional",
		MonthlyBase:       50000000,  // $500K/month
		PerCalculation:    10000,     // $100/calculation  
		CalculationsIncluded: 5000,
		SLA:              SLA_99_9,
		Support:          SupportBusinessHours,
	},
	Tier3Bank: { // Community banks
		TierName:          "Tier 3 - Community",
		MonthlyBase:       5000000,   // $50K/month
		PerCalculation:    1000,      // $10/calculation
		CalculationsIncluded: 1000,
		SLA:              SLA_99_5,
		Support:          SupportEmailOnly,
	},
}

// CalculateMonthlyPricing calculates pricing for enterprise sales
func (b *BaselIIIAdapter) CalculateMonthlyPricing(bankTier ComplianceLevel, monthlyCalculations int) *PricingQuote {
	tier := BaselPricingTiers[bankTier]
	
	var overage int64 = 0
	if monthlyCalculations > tier.CalculationsIncluded {
		excess := monthlyCalculations - tier.CalculationsIncluded
		overage = int64(excess) * tier.PerCalculation
	}
	
	totalMonthly := tier.MonthlyBase + overage
	
	return &PricingQuote{
		TierName:          tier.TierName,
		MonthlyBase:       tier.MonthlyBase,
		OverageCharges:    overage,
		TotalMonthly:      totalMonthly,
		AnnualContract:    totalMonthly * 12,
		CalculationsIncluded: tier.CalculationsIncluded,
		ExcessCalculations: monthlyCalculations - tier.CalculationsIncluded,
	}
}
