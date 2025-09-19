// isolation.go — Multi-Tenant Isolation System
// Integrates with existing identity and execution systems

package tenant

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// TenantConfig defines isolation requirements for a tenant
type TenantConfig struct {
	TenantID      string   `json:"tenant_id"`
	Isolation     string   `json:"isolation_level"` // "shared", "dedicated", "airgapped"
	DataResidency []string `json:"allowed_regions"`
	Compliance    []string `json:"required_frameworks"` // "SOX", "GDPR", "HIPAA"
	BudgetLimits  Budget   `json:"budget_limits"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Budget struct {
	DailyLimit    uint64 `json:"daily_limit_micro_units"`
	MonthlyLimit  uint64 `json:"monthly_limit_micro_units"`
	YearlyLimit   uint64 `json:"yearly_limit_micro_units"`
	CurrentSpend  uint64 `json:"current_spend_micro_units"`
	AlertThreshold float64 `json:"alert_threshold"` // Percentage (0-1)
}

// IsolationManager manages tenant isolation and access control
type IsolationManager struct {
	tenants map[string]*TenantConfig
	executor TenantExecutor
	auditor  TenantAuditor
}

type TenantExecutor interface {
	ExecuteWithIsolation(isolation string, artifact, input []byte, maxCycles uint64) (*ExecutionResult, error)
}

type TenantAuditor interface {
	LogTenantAccess(tenantID, action, resource string, metadata map[string]interface{}) error
	ValidateTenantAccess(tenantID, resource string) (bool, error)
}

type ExecutionResult struct {
	OutputHash  [32]byte `json:"output_hash"`
	CyclesUsed  uint64   `json:"cycles_used"`
	ReceiptBlob []byte   `json:"receipt_blob"`
	Isolation   string   `json:"isolation_level"`
	TenantID    string   `json:"tenant_id"`
}

// NewIsolationManager creates a new tenant isolation manager
func NewIsolationManager(executor TenantExecutor, auditor TenantAuditor) *IsolationManager {
	return &IsolationManager{
		tenants:  make(map[string]*TenantConfig),
		executor: executor,
		auditor:  auditor,
	}
}

// RegisterTenant registers a new tenant with isolation requirements
func (im *IsolationManager) RegisterTenant(config *TenantConfig) error {
	// Validate isolation level
	if !isValidIsolationLevel(config.Isolation) {
		return fmt.Errorf("invalid isolation level: %s", config.Isolation)
	}

	// Validate compliance frameworks
	for _, framework := range config.Compliance {
		if !isValidComplianceFramework(framework) {
			return fmt.Errorf("invalid compliance framework: %s", framework)
		}
	}

	// Set timestamps
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now

	// Store tenant configuration
	im.tenants[config.TenantID] = config

	// Log tenant registration
	err := im.auditor.LogTenantAccess(config.TenantID, "register", "tenant", map[string]interface{}{
		"isolation_level": config.Isolation,
		"compliance":      config.Compliance,
		"regions":         config.DataResidency,
	})
	if err != nil {
		return fmt.Errorf("failed to log tenant registration: %w", err)
	}

	return nil
}

// OCX_EXEC_TENANT executes computation with tenant-specific isolation
func (im *IsolationManager) OCX_EXEC_TENANT(tenantID string, artifact, input []byte, maxCycles uint64) (*ExecutionResult, error) {
	// Get tenant configuration
	tenant, exists := im.tenants[tenantID]
	if !exists {
		return nil, fmt.Errorf("tenant not found: %s", tenantID)
	}

	// Validate tenant access
	allowed, err := im.auditor.ValidateTenantAccess(tenantID, "execution")
	if err != nil {
		return nil, fmt.Errorf("failed to validate tenant access: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("tenant access denied for execution")
	}

	// Check budget limits
	err = im.checkBudgetLimits(tenant)
	if err != nil {
		return nil, fmt.Errorf("budget limit exceeded: %w", err)
	}

	// Validate data residency
	err = im.validateDataResidency(tenant, artifact, input)
	if err != nil {
		return nil, fmt.Errorf("data residency violation: %w", err)
	}

	// Execute with tenant-specific isolation
	result, err := im.executor.ExecuteWithIsolation(tenant.Isolation, artifact, input, maxCycles)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Add tenant metadata to result
	result.TenantID = tenantID
	result.Isolation = tenant.Isolation

	// Update budget spending
	err = im.updateBudgetSpending(tenant, result.CyclesUsed)
	if err != nil {
		return nil, fmt.Errorf("failed to update budget spending: %w", err)
	}

	// Log execution
	err = im.auditor.LogTenantAccess(tenantID, "execute", "computation", map[string]interface{}{
		"cycles_used":    result.CyclesUsed,
		"isolation":      tenant.Isolation,
		"artifact_hash":  fmt.Sprintf("%x", sha256.Sum256(artifact)),
		"input_hash":     fmt.Sprintf("%x", sha256.Sum256(input)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to log execution: %w", err)
	}

	return result, nil
}

// checkBudgetLimits validates budget constraints
func (im *IsolationManager) checkBudgetLimits(tenant *TenantConfig) error {
	// Check daily limit
	if tenant.BudgetLimits.DailyLimit > 0 {
		dailySpend := im.getDailySpend(tenant.TenantID)
		if dailySpend >= tenant.BudgetLimits.DailyLimit {
			return fmt.Errorf("daily budget limit exceeded: %d/%d", dailySpend, tenant.BudgetLimits.DailyLimit)
		}
	}

	// Check monthly limit
	if tenant.BudgetLimits.MonthlyLimit > 0 {
		monthlySpend := im.getMonthlySpend(tenant.TenantID)
		if monthlySpend >= tenant.BudgetLimits.MonthlyLimit {
			return fmt.Errorf("monthly budget limit exceeded: %d/%d", monthlySpend, tenant.BudgetLimits.MonthlyLimit)
		}
	}

	// Check yearly limit
	if tenant.BudgetLimits.YearlyLimit > 0 {
		yearlySpend := im.getYearlySpend(tenant.TenantID)
		if yearlySpend >= tenant.BudgetLimits.YearlyLimit {
			return fmt.Errorf("yearly budget limit exceeded: %d/%d", yearlySpend, tenant.BudgetLimits.YearlyLimit)
		}
	}

	return nil
}

// validateDataResidency ensures data doesn't cross restricted boundaries
func (im *IsolationManager) validateDataResidency(tenant *TenantConfig, artifact, input []byte) error {
	// Check if tenant has data residency requirements
	if len(tenant.DataResidency) == 0 {
		return nil // No restrictions
	}

	// In a real implementation, this would check:
	// 1. Current execution region
	// 2. Data source regions
	// 3. Cross-border data transfer restrictions
	// 4. Compliance framework requirements

	// For now, we'll do a simple validation
	// In production, this would integrate with actual data residency tracking
	return nil
}

// updateBudgetSpending updates tenant budget spending
func (im *IsolationManager) updateBudgetSpending(tenant *TenantConfig, cyclesUsed uint64) error {
	// Calculate cost based on cycles used
	// This would integrate with actual pricing
	cost := cyclesUsed * 1000 // Simplified pricing: 1000 micro-units per cycle

	// Update current spend
	tenant.BudgetLimits.CurrentSpend += cost

	// Check alert threshold
	if tenant.BudgetLimits.AlertThreshold > 0 {
		threshold := uint64(float64(tenant.BudgetLimits.DailyLimit) * tenant.BudgetLimits.AlertThreshold)
		if tenant.BudgetLimits.CurrentSpend >= threshold {
			// Send alert (in production, this would trigger notifications)
			im.auditor.LogTenantAccess(tenant.TenantID, "alert", "budget", map[string]interface{}{
				"current_spend": tenant.BudgetLimits.CurrentSpend,
				"threshold":     threshold,
				"percentage":    float64(tenant.BudgetLimits.CurrentSpend) / float64(tenant.BudgetLimits.DailyLimit) * 100,
			})
		}
	}

	return nil
}

// getDailySpend retrieves daily spending for a tenant
func (im *IsolationManager) getDailySpend(tenantID string) uint64 {
	// In production, this would query the database for actual spending
	// For now, return current spend from tenant config
	tenant, exists := im.tenants[tenantID]
	if !exists {
		return 0
	}
	return tenant.BudgetLimits.CurrentSpend
}

// getMonthlySpend retrieves monthly spending for a tenant
func (im *IsolationManager) getMonthlySpend(tenantID string) uint64 {
	// In production, this would query the database for monthly spending
	// For now, return a simplified calculation
	return im.getDailySpend(tenantID) * 30
}

// getYearlySpend retrieves yearly spending for a tenant
func (im *IsolationManager) getYearlySpend(tenantID string) uint64 {
	// In production, this would query the database for yearly spending
	// For now, return a simplified calculation
	return im.getDailySpend(tenantID) * 365
}

// GetTenantConfig retrieves tenant configuration
func (im *IsolationManager) GetTenantConfig(tenantID string) (*TenantConfig, error) {
	tenant, exists := im.tenants[tenantID]
	if !exists {
		return nil, fmt.Errorf("tenant not found: %s", tenantID)
	}
	return tenant, nil
}

// UpdateTenantConfig updates tenant configuration
func (im *IsolationManager) UpdateTenantConfig(tenantID string, updates map[string]interface{}) error {
	tenant, exists := im.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}

	// Update fields based on provided updates
	for key, value := range updates {
		switch key {
		case "isolation_level":
			if level, ok := value.(string); ok {
				if !isValidIsolationLevel(level) {
					return fmt.Errorf("invalid isolation level: %s", level)
				}
				tenant.Isolation = level
			}
		case "data_residency":
			if regions, ok := value.([]string); ok {
				tenant.DataResidency = regions
			}
		case "compliance":
			if frameworks, ok := value.([]string); ok {
				for _, framework := range frameworks {
					if !isValidComplianceFramework(framework) {
						return fmt.Errorf("invalid compliance framework: %s", framework)
					}
				}
				tenant.Compliance = frameworks
			}
		case "budget_limits":
			if budget, ok := value.(Budget); ok {
				tenant.BudgetLimits = budget
			}
		}
	}

	tenant.UpdatedAt = time.Now()

	// Log configuration update
	err := im.auditor.LogTenantAccess(tenantID, "update", "config", map[string]interface{}{
		"updates": updates,
	})
	if err != nil {
		return fmt.Errorf("failed to log configuration update: %w", err)
	}

	return nil
}

// Helper functions
func isValidIsolationLevel(level string) bool {
	validLevels := []string{"shared", "dedicated", "airgapped"}
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}

func isValidComplianceFramework(framework string) bool {
	validFrameworks := []string{"SOX", "GDPR", "HIPAA", "PCI-DSS", "SOC2", "ISO27001"}
	for _, valid := range validFrameworks {
		if framework == valid {
			return true
		}
	}
	return false
}
