// Basel III Compliance Adapter Testing Framework
// Enterprise-grade testing for regulatory compliance
package basel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBaselIIIComplianceEndToEnd tests the complete Basel III compliance workflow
func TestBaselIIIComplianceEndToEnd(t *testing.T) {
	adapter := setupTestAdapter(t)
	
	// Test with real-world risk calculation scenario
	req := &RWACalculationRequest{
		BankID:                  "TEST_BANK_001",
		CalculationDate:         time.Now(),
		CalculationCode:         loadTestRiskCalculation(),
		InputData:              loadTestMarketData(),
		CapitalRatio:           0.12,  // 12% - above minimum
		LeverageRatio:          0.05,  // 5% - above minimum  
		LiquidityCoverageRatio: 1.10,  // 110% - above minimum
		AssetPortfolio:         loadTestPortfolio(),
	}
	
	result, err := adapter.VerifyRiskWeightedAssets(context.Background(), req)
	require.NoError(t, err)
	
	// Verify compliance result
	assert.True(t, result.IsCompliant)
	assert.Empty(t, result.ViolationReasons)
	assert.True(t, result.CryptographicProof.MathematicalCertainty)
	
	// Verify performance (leverages your <15ms guarantee)
	assert.True(t, result.CryptographicProof.VerificationTime < 15*time.Millisecond)
	
	// Verify audit trail
	assert.NotEmpty(t, result.AuditTrail)
	assert.Equal(t, "Execution Started", result.AuditTrail[0].Event)
	assert.Equal(t, "Execution Completed", result.AuditTrail[1].Event)
}

// TestRegulatoryReportGeneration tests the generation of regulatory reports
func TestRegulatoryReportGeneration(t *testing.T) {
	adapter := setupTestAdapter(t)
	
	period := ReportingPeriod{
		StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2024, 3, 31, 23, 59, 59, 0, time.UTC),
		Quarter:   1,
		Year:      2024,
	}
	
	report, err := adapter.GenerateRegulatoryReport(context.Background(), period)
	require.NoError(t, err)
	
	// Verify report completeness
	assert.NotEmpty(t, report.ReportID)
	assert.NotEmpty(t, report.DigitalSignature)
	assert.Equal(t, "COMPLIANT", report.ComplianceStatus)
	assert.Equal(t, "TEST_BANK_001", report.BankID)
	
	// Verify executive summary
	assert.NotNil(t, report.ExecutiveSummary)
	assert.Equal(t, 0, report.ExecutiveSummary.TotalCalculations) // No test data
	assert.Equal(t, 0.0, report.ExecutiveSummary.CompliancePercentage)
}

// TestBaselComplianceViolations tests compliance violation detection
func TestBaselComplianceViolations(t *testing.T) {
	adapter := setupTestAdapter(t)
	
	// Test insufficient capital ratio
	req := &RWACalculationRequest{
		BankID:                  "TEST_BANK_002",
		CalculationDate:         time.Now(),
		CalculationCode:         loadTestRiskCalculation(),
		InputData:              loadTestMarketData(),
		CapitalRatio:           0.05,  // 5% - below minimum
		LeverageRatio:          0.05,  // 5% - above minimum
		LiquidityCoverageRatio: 1.10,  // 110% - above minimum
		AssetPortfolio:         loadTestPortfolio(),
	}
	
	result, err := adapter.VerifyRiskWeightedAssets(context.Background(), req)
	require.NoError(t, err)
	
	// Verify violation detection
	assert.False(t, result.IsCompliant)
	assert.Contains(t, result.ViolationReasons, "Insufficient capital adequacy ratio")
	
	// Verify cryptographic proof still works
	assert.True(t, result.CryptographicProof.MathematicalCertainty)
}

// TestPricingCalculation tests the enterprise pricing calculations
func TestPricingCalculation(t *testing.T) {
	adapter := setupTestAdapter(t)
	
	// Test Tier 1 bank pricing
	tier1Pricing := adapter.CalculateMonthlyPricing(Tier1Bank, 15000)
	assert.Equal(t, "Tier 1 - Systemically Important", tier1Pricing.TierName)
	assert.Equal(t, int64(500000000), tier1Pricing.MonthlyBase) // $5M
	assert.Equal(t, int64(500000000), tier1Pricing.OverageCharges) // $5M overage
	assert.Equal(t, int64(1000000000), tier1Pricing.TotalMonthly) // $10M total
	
	// Test Tier 2 bank pricing
	tier2Pricing := adapter.CalculateMonthlyPricing(Tier2Bank, 6000)
	assert.Equal(t, "Tier 2 - Regional", tier2Pricing.TierName)
	assert.Equal(t, int64(50000000), tier2Pricing.MonthlyBase) // $500K
	assert.Equal(t, int64(10000000), tier2Pricing.OverageCharges) // $100K overage
	assert.Equal(t, int64(60000000), tier2Pricing.TotalMonthly) // $600K total
	
	// Test Tier 3 bank pricing
	tier3Pricing := adapter.CalculateMonthlyPricing(Tier3Bank, 500)
	assert.Equal(t, "Tier 3 - Community", tier3Pricing.TierName)
	assert.Equal(t, int64(5000000), tier3Pricing.MonthlyBase) // $50K
	assert.Equal(t, int64(0), tier3Pricing.OverageCharges) // No overage
	assert.Equal(t, int64(5000000), tier3Pricing.TotalMonthly) // $50K total
}

// TestEnterpriseSalesProposal tests the enterprise sales proposal generation
func TestEnterpriseSalesProposal(t *testing.T) {
	adapter := setupTestAdapter(t)
	salesManager := NewEnterpriseSalesManager(adapter)
	
	bankInfo := BankInfo{
		BankName:        "Test Regional Bank",
		BankID:          "TEST_BANK_003",
		Tier:            Tier2Bank,
		AssetsUnderManagement: 50000000000, // $50B
		RegulatoryJurisdiction: "US",
		CurrentComplianceCosts: 10000000, // $10M/year
		PainPoints:      []string{"Manual processes", "High error rates", "Regulatory uncertainty"},
	}
	
	requirements := ComplianceRequirements{
		MonthlyCalculations:    8000,
		ReportingFrequency:     Quarterly,
		AuditRetentionYears:    7,
		IntegrationRequirements: []string{"API integration", "Real-time reporting"},
		SLARequirements:        SLA_99_9,
		SupportRequirements:    SupportBusinessHours,
	}
	
	proposal := salesManager.GenerateProposal(bankInfo, requirements)
	
	// Verify proposal completeness
	assert.NotEmpty(t, proposal.ProposalID)
	assert.Equal(t, bankInfo.BankName, proposal.BankInfo.BankName)
	assert.Equal(t, requirements.MonthlyCalculations, proposal.Requirements.MonthlyCalculations)
	
	// Verify value proposition
	assert.NotNil(t, proposal.ValueProposition)
	assert.True(t, proposal.ValueProposition.CostSavings > 0)
	assert.Equal(t, 100.0, proposal.ValueProposition.ComplianceCertainty)
	assert.True(t, proposal.ValueProposition.ROI > 0)
	
	// Verify timeline
	assert.NotNil(t, proposal.Timeline)
	assert.True(t, proposal.Timeline.TotalDuration > 0)
	assert.Len(t, proposal.Timeline.Milestones, 4)
}

// TestROIAnalysis tests the ROI analysis generation
func TestROIAnalysis(t *testing.T) {
	adapter := setupTestAdapter(t)
	salesManager := NewEnterpriseSalesManager(adapter)
	
	bankInfo := BankInfo{
		BankName:        "Test Community Bank",
		BankID:          "TEST_BANK_004",
		Tier:            Tier3Bank,
		AssetsUnderManagement: 1000000000, // $1B
		RegulatoryJurisdiction: "US",
		CurrentComplianceCosts: 2000000, // $2M/year
		PainPoints:      []string{"High compliance costs", "Manual reporting"},
	}
	
	requirements := ComplianceRequirements{
		MonthlyCalculations:    2000,
		ReportingFrequency:     Monthly,
		AuditRetentionYears:    5,
		IntegrationRequirements: []string{"Basic API integration"},
		SLARequirements:        SLA_99_5,
		SupportRequirements:    SupportEmailOnly,
	}
	
	roiAnalysis := salesManager.GenerateROIAnalysis(bankInfo, requirements)
	
	// Debug output
	t.Logf("Current Annual Costs: $%.2f", roiAnalysis.CurrentAnnualCosts)
	t.Logf("OCX Annual Costs: $%.2f", roiAnalysis.OCXAnnualCosts)
	t.Logf("Annual Savings: $%.2f", roiAnalysis.AnnualSavings)
	t.Logf("ROI Percentage: %.2f%%", roiAnalysis.ROIPercentage)
	t.Logf("Payback Period: %.2f months", roiAnalysis.PaybackPeriod)
	
	// Verify ROI analysis
	assert.True(t, roiAnalysis.CurrentAnnualCosts > roiAnalysis.OCXAnnualCosts)
	assert.True(t, roiAnalysis.AnnualSavings > 0)
	assert.True(t, roiAnalysis.ROIPercentage > 0)
	assert.True(t, roiAnalysis.PaybackPeriod > 0)
	assert.True(t, roiAnalysis.ThreeYearSavings > roiAnalysis.AnnualSavings)
	assert.True(t, roiAnalysis.FiveYearSavings > roiAnalysis.ThreeYearSavings)
}

// TestPerformanceUnderLoad tests performance under enterprise load
func TestPerformanceUnderLoad(t *testing.T) {
	adapter := setupTestAdapter(t)
	
	// Simulate enterprise load (1000 calculations)
	startTime := time.Now()
	
	for i := 0; i < 1000; i++ {
		req := &RWACalculationRequest{
			BankID:                  "LOAD_TEST_BANK",
			CalculationDate:         time.Now(),
			CalculationCode:         loadTestRiskCalculation(),
			InputData:              loadTestMarketData(),
			CapitalRatio:           0.12,
			LeverageRatio:          0.05,
			LiquidityCoverageRatio: 1.10,
			AssetPortfolio:         loadTestPortfolio(),
		}
		
		result, err := adapter.VerifyRiskWeightedAssets(context.Background(), req)
		require.NoError(t, err)
		assert.True(t, result.IsCompliant)
	}
	
	totalDuration := time.Since(startTime)
	avgDuration := totalDuration / 1000
	
	// Verify enterprise performance (should handle 1000 calculations in reasonable time)
	assert.True(t, totalDuration < 30*time.Second, "1000 calculations took too long: %v", totalDuration)
	assert.True(t, avgDuration < 30*time.Millisecond, "Average calculation took too long: %v", avgDuration)
	
	t.Logf("Performance test: 1000 calculations in %v (avg: %v per calculation)", totalDuration, avgDuration)
}

// Helper functions for testing
func setupTestAdapter(t *testing.T) *BaselIIIAdapter {
	config := &BaselConfig{
		BankID:              "TEST_BANK_001",
		RegulatoryAuthority: "US_FED",
		ComplianceLevel:     Tier2Bank,
		AuditRetention:      7 * 365 * 24 * time.Hour, // 7 years
	}
	
	adapter, err := NewBaselIIIAdapter(config)
	require.NoError(t, err)
	
	return adapter
}

func loadTestRiskCalculation() []byte {
	// This load actual risk calculation code
	// For testing, return a simple deterministic calculation
	return []byte("test_risk_calculation_code")
}

func loadTestMarketData() []byte {
	// This load actual market data
	// For testing, return sample market data
	return []byte("test_market_data")
}

func loadTestPortfolio() []Asset {
	return []Asset{
		{
			AssetID:    "GOVT_BOND_001",
			AssetType:  GovernmentBonds,
			Value:      1000000,
			RiskWeight: 0.0, // Government bonds have 0% risk weight
			Currency:   "USD",
		},
		{
			AssetID:    "CORP_BOND_001",
			AssetType:  CorporateBonds,
			Value:      500000,
			RiskWeight: 0.2, // Corporate bonds have 20% risk weight
			Currency:   "USD",
		},
		{
			AssetID:    "MORTGAGE_001",
			AssetType:  Mortgages,
			Value:      2000000,
			RiskWeight: 0.35, // Mortgages have 35% risk weight
			Currency:   "USD",
		},
	}
}
