// multi_rail.go - Multi-Rail Settlement Manager for OCX Protocol
package settlement

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MultiRailSettlementManager manages settlement across multiple rails
type MultiRailSettlementManager struct {
	rails      map[string]SettlementRail
	mu         sync.RWMutex
	ledger     LedgerManager
	compliance ComplianceManager
}

// SettlementRail represents a settlement rail
type SettlementRail struct {
	RailID      string  `json:"rail_id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	IsActive    bool    `json:"is_active"`
	Cost        float64 `json:"cost"`
	Latency     int     `json:"latency_ms"`
	SuccessRate float64 `json:"success_rate"`
}

// SettlementInstruction represents a settlement instruction
type SettlementInstruction struct {
	ID              string                 `json:"id"`
	Amount          float64                `json:"amount"`
	Currency        string                 `json:"currency"`
	FromAccount     string                 `json:"from_account"`
	ToAccount       string                 `json:"to_account"`
	Priority        string                 `json:"priority"`
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedAt       time.Time              `json:"created_at"`
	ExpiresAt       time.Time              `json:"expires_at"`
}

// SettlementResult represents the result of a settlement
type SettlementResult struct {
	SettlementID string                 `json:"settlement_id"`
	RailUsed     string                 `json:"rail_used"`
	Status       string                 `json:"status"`
	Amount       float64                `json:"amount"`
	Currency     string                 `json:"currency"`
	Fee          float64                `json:"fee"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata"`
	Receipt      string                 `json:"receipt,omitempty"`
}

// SettlementStatus represents the status of a settlement
type SettlementStatus struct {
	SettlementID string    `json:"settlement_id"`
	Status       string    `json:"status"`
	LastUpdated  time.Time `json:"last_updated"`
	Message      string    `json:"message"`
}

// LedgerManager represents a ledger manager interface
type LedgerManager interface {
	RecordSettlement(ctx context.Context, instruction *SettlementInstruction, result *SettlementResult) error
	GetSettlement(ctx context.Context, settlementID string) (*SettlementResult, error)
}

// ComplianceManager represents a compliance manager interface
type ComplianceManager interface {
	ValidateSettlement(ctx context.Context, instruction *SettlementInstruction) error
	CheckSanctions(ctx context.Context, fromAccount, toAccount string) error
}

// NewMultiRailSettlementManager creates a new multi-rail settlement manager
func NewMultiRailSettlementManager(ledger LedgerManager, compliance ComplianceManager) *MultiRailSettlementManager {
	return &MultiRailSettlementManager{
		rails:      make(map[string]SettlementRail),
		ledger:     ledger,
		compliance: compliance,
	}
}

// AddRail adds a settlement rail
func (m *MultiRailSettlementManager) AddRail(rail SettlementRail) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rails[rail.RailID] = rail
}

// RemoveRail removes a settlement rail
func (m *MultiRailSettlementManager) RemoveRail(railID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rails, railID)
}

// GetRail returns a settlement rail by ID
func (m *MultiRailSettlementManager) GetRail(railID string) (SettlementRail, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rail, exists := m.rails[railID]
	return rail, exists
}

// ListRails returns all settlement rails
func (m *MultiRailSettlementManager) ListRails() []SettlementRail {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	rails := make([]SettlementRail, 0, len(m.rails))
	for _, rail := range m.rails {
		rails = append(rails, rail)
	}
	return rails
}

// ProcessSettlement processes a settlement instruction
func (m *MultiRailSettlementManager) ProcessSettlement(ctx context.Context, instruction *SettlementInstruction) (*SettlementResult, error) {
	// 1. Validate instruction
	if err := m.validateInstruction(instruction); err != nil {
		return nil, err
	}

	// 2. Check compliance
	if m.compliance != nil {
		if err := m.compliance.ValidateSettlement(ctx, instruction); err != nil {
			return nil, fmt.Errorf("compliance check failed: %w", err)
		}
	}

	// 3. Select optimal rail
	rail, err := m.selectOptimalRail(instruction)
	if err != nil {
		return nil, err
	}

	// 4. Process settlement
	result, err := m.processSettlementWithRail(ctx, rail, instruction)
	if err != nil {
		return nil, err
	}

	// 5. Record in ledger
	if m.ledger != nil {
		if err := m.ledger.RecordSettlement(ctx, instruction, result); err != nil {
			return nil, err
		}
	}

	// 6. Generate receipt
	receipt, err := m.generateReceipt(instruction, result)
	if err != nil {
		return nil, err
	}

	result.Receipt = receipt
	return result, nil
}

// GetSettlementStatus gets the status of a settlement
func (m *MultiRailSettlementManager) GetSettlementStatus(ctx context.Context, settlementID string) (*SettlementStatus, error) {
	// Get settlement from ledger
	var settlement *SettlementResult
	var err error
	if m.ledger != nil {
		settlement, err = m.ledger.GetSettlement(ctx, settlementID)
	} else {
		// Mock settlement for disabled ledger
		settlement = &SettlementResult{SettlementID: settlementID, RailUsed: "unknown"}
	}

	if err != nil {
		return nil, err
	}

	// Get rail status
	rail, exists := m.rails[settlement.RailUsed]
	if !exists {
		return &SettlementStatus{
			SettlementID: settlementID,
			Status:       "unknown",
			LastUpdated:  time.Now(),
			Message:      "Rail not found",
		}, nil
	}

	return m.getRailStatus(ctx, rail, settlementID)
}

// validateInstruction validates a settlement instruction
func (m *MultiRailSettlementManager) validateInstruction(instruction *SettlementInstruction) error {
	if instruction.ID == "" {
		return fmt.Errorf("instruction ID is required")
	}
	if instruction.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if instruction.Currency == "" {
		return fmt.Errorf("currency is required")
	}
	if instruction.FromAccount == "" {
		return fmt.Errorf("from account is required")
	}
	if instruction.ToAccount == "" {
		return fmt.Errorf("to account is required")
	}
	return nil
}

// selectOptimalRail selects the optimal rail for settlement
func (m *MultiRailSettlementManager) selectOptimalRail(instruction *SettlementInstruction) (SettlementRail, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var bestRail SettlementRail
	var bestScore float64

	for _, rail := range m.rails {
		if !rail.IsActive {
			continue
		}

		// Calculate score based on cost, latency, and success rate
		score := rail.SuccessRate * 0.5 + (1.0/rail.Cost) * 0.3 + (1.0/float64(rail.Latency)) * 0.2

		if score > bestScore {
			bestScore = score
			bestRail = rail
		}
	}

	if bestRail.RailID == "" {
		return SettlementRail{}, fmt.Errorf("no active rails available")
	}

	return bestRail, nil
}

// generateReceipt generates a receipt for the settlement
func (m *MultiRailSettlementManager) generateReceipt(instruction *SettlementInstruction, result *SettlementResult) (string, error) {
	receipt := fmt.Sprintf("Settlement Receipt\n"+
		"ID: %s\n"+
		"Amount: %.2f %s\n"+
		"Fee: %.2f %s\n"+
		"Rail: %s\n"+
		"Status: %s\n"+
		"Timestamp: %s\n",
		result.SettlementID,
		result.Amount,
		result.Currency,
		result.Fee,
		result.Currency,
		result.RailUsed,
		result.Status,
		result.Timestamp.Format(time.RFC3339))

	return receipt, nil
}

// GetSettlementStats returns settlement statistics
func (m *MultiRailSettlementManager) GetSettlementStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalRails := len(m.rails)
	activeRails := 0
	var totalCost, avgLatency, avgSuccessRate float64

	for _, rail := range m.rails {
		if rail.IsActive {
			activeRails++
		}
		totalCost += rail.Cost
		avgLatency += float64(rail.Latency)
		avgSuccessRate += rail.SuccessRate
	}

	if totalRails > 0 {
		avgLatency /= float64(totalRails)
		avgSuccessRate /= float64(totalRails)
	}

	return map[string]interface{}{
		"total_rails":      totalRails,
		"active_rails":     activeRails,
		"avg_cost":         totalCost / float64(totalRails),
		"avg_latency":      avgLatency,
		"avg_success_rate": avgSuccessRate,
	}
}

// processSettlementWithRail processes settlement with a specific rail
func (m *MultiRailSettlementManager) processSettlementWithRail(ctx context.Context, rail SettlementRail, instruction *SettlementInstruction) (*SettlementResult, error) {
	// Simulate settlement processing
	result := &SettlementResult{
		SettlementID: fmt.Sprintf("settlement_%d", time.Now().Unix()),
		RailUsed:     rail.RailID,
		Status:       "completed",
		Amount:       instruction.Amount,
		Currency:     instruction.Currency,
		Fee:          rail.Cost,
		Timestamp:    time.Now(),
		Metadata:     instruction.Metadata,
	}

	return result, nil
}

// getRailStatus gets the status from a specific rail
func (m *MultiRailSettlementManager) getRailStatus(ctx context.Context, rail SettlementRail, settlementID string) (*SettlementStatus, error) {
	// Simulate rail status check
	status := &SettlementStatus{
		SettlementID: settlementID,
		Status:       "completed",
		LastUpdated:  time.Now(),
		Message:      fmt.Sprintf("Settlement processed via %s rail", rail.Name),
	}

	return status, nil
}
