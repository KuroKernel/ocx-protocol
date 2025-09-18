package consensus

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ResourceManager handles resource verification and management
type ResourceManager struct {
	db *sql.DB
}

// NewResourceManager creates a new resource manager
func NewResourceManager(db *sql.DB) *ResourceManager {
	return &ResourceManager{db: db}
}

// ResourceStatus represents the status of a compute resource
type ResourceStatus string

const (
	StatusAvailable   ResourceStatus = "available"
	StatusAllocated   ResourceStatus = "allocated"
	StatusMaintenance ResourceStatus = "maintenance"
	StatusOffline     ResourceStatus = "offline"
)

// ComputeUnit represents a compute unit
type ComputeUnit struct {
	UnitID              string         `json:"unit_id"`
	ProviderID          string         `json:"provider_id"`
	HardwareType        string         `json:"hardware_type"`
	GPUModel            string         `json:"gpu_model"`
	GPUMemoryGB         int            `json:"gpu_memory_gb"`
	CPUCores            int            `json:"cpu_cores"`
	RAMGB               int            `json:"ram_gb"`
	BasePricePerHour    float64        `json:"base_price_per_hour_usdc"`
	CurrentAvailability ResourceStatus `json:"current_availability"`
	GeographicRegion    string         `json:"geographic_region"`
	LastHeartbeat       time.Time      `json:"last_heartbeat"`
}

// VerifyResourceOwnership verifies that a provider owns a specific compute unit
func (rm *ResourceManager) VerifyResourceOwnership(ctx context.Context, providerID string, unitID string) (bool, error) {
	query := `
		SELECT provider_id 
		FROM compute_units 
		WHERE unit_id = $1 AND provider_id = $2
	`

	var ownerID string
	err := rm.db.QueryRowContext(ctx, query, unitID, providerID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // Resource not found or not owned by provider
		}
		return false, fmt.Errorf("failed to verify resource ownership: %w", err)
	}

	return ownerID == providerID, nil
}

// VerifyResourceAvailability verifies that a compute unit is available for allocation
func (rm *ResourceManager) VerifyResourceAvailability(ctx context.Context, unitID string) (bool, error) {
	query := `
		SELECT current_availability, last_heartbeat
		FROM compute_units 
		WHERE unit_id = $1
	`

	var availability string
	var lastHeartbeat time.Time
	err := rm.db.QueryRowContext(ctx, query, unitID).Scan(&availability, &lastHeartbeat)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // Resource not found
		}
		return false, fmt.Errorf("failed to verify resource availability: %w", err)
	}

	// Check if resource is available
	if availability != string(StatusAvailable) {
		return false, nil
	}

	// Check if provider is still active (heartbeat within last 5 minutes)
	if time.Since(lastHeartbeat) > 5*time.Minute {
		return false, nil
	}

	return true, nil
}

// ValidateMatchingCriteria validates that a provider's offer matches order requirements
func (rm *ResourceManager) ValidateMatchingCriteria(ctx context.Context, orderID string, providerID string, units []ComputeUnitOffer) error {
	// Get order requirements
	orderQuery := `
		SELECT hardware_type, gpu_model, gpu_memory_gb, max_price_per_hour_usdc, geographic_region
		FROM compute_orders 
		WHERE order_id = $1
	`

	var hardwareType, gpuModel, geographicRegion string
	var gpuMemoryGB int
	var maxPricePerHour float64

	err := rm.db.QueryRowContext(ctx, orderQuery, orderID).Scan(
		&hardwareType, &gpuModel, &gpuMemoryGB, &maxPricePerHour, &geographicRegion,
	)
	if err != nil {
		return fmt.Errorf("failed to get order requirements: %w", err)
	}

	// Validate each offered unit
	for _, unit := range units {
		// Check hardware type match
		if unit.HardwareType != hardwareType {
			return fmt.Errorf("hardware type mismatch: got %s, expected %s", unit.HardwareType, hardwareType)
		}

		// Check GPU model match (if specified)
		if gpuModel != "" && unit.GPUModel != gpuModel {
			return fmt.Errorf("GPU model mismatch: got %s, expected %s", unit.GPUModel, gpuModel)
		}

		// Check GPU memory match (if specified)
		if gpuMemoryGB > 0 && unit.GPUMemoryGB < gpuMemoryGB {
			return fmt.Errorf("insufficient GPU memory: got %dGB, expected %dGB", unit.GPUMemoryGB, gpuMemoryGB)
		}

		// Check price match
		if unit.BasePricePerHour > maxPricePerHour {
			return fmt.Errorf("price exceeds maximum: got $%.2f, expected max $%.2f", unit.BasePricePerHour, maxPricePerHour)
		}

		// Check geographic region match (if specified)
		if geographicRegion != "" && unit.GeographicRegion != geographicRegion {
			return fmt.Errorf("geographic region mismatch: got %s, expected %s", unit.GeographicRegion, geographicRegion)
		}
	}

	// Check provider reputation
	reputationQuery := `
		SELECT reputation_score 
		FROM providers 
		WHERE provider_id = $1
	`

	var reputationScore float64
	err = rm.db.QueryRowContext(ctx, reputationQuery, providerID).Scan(&reputationScore)
	if err != nil {
		return fmt.Errorf("failed to get provider reputation: %w", err)
	}

	// Check minimum reputation threshold
	minReputation := 0.5
	if reputationScore < minReputation {
		return fmt.Errorf("provider reputation %.2f below minimum %.2f", reputationScore, minReputation)
	}

	return nil
}

// UpdateResourceStatus updates the status of a compute unit
func (rm *ResourceManager) UpdateResourceStatus(ctx context.Context, unitID string, status ResourceStatus) error {
	// Use a transaction to ensure atomicity
	tx, err := rm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update resource status
	updateQuery := `
		UPDATE compute_units 
		SET current_availability = $1, last_heartbeat = $2
		WHERE unit_id = $3
	`

	result, err := tx.ExecContext(ctx, updateQuery, string(status), time.Now(), unitID)
	if err != nil {
		return fmt.Errorf("failed to update resource status: %w", err)
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("resource not found: %s", unitID)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetResourceDetails gets detailed information about a compute unit
func (rm *ResourceManager) GetResourceDetails(ctx context.Context, unitID string) (*ComputeUnit, error) {
	query := `
		SELECT unit_id, provider_id, hardware_type, gpu_model, gpu_memory_gb, 
		       cpu_cores, ram_gb, base_price_per_hour_usdc, current_availability, 
		       geographic_region, last_heartbeat
		FROM compute_units 
		WHERE unit_id = $1
	`

	var unit ComputeUnit
	err := rm.db.QueryRowContext(ctx, query, unitID).Scan(
		&unit.UnitID,
		&unit.ProviderID,
		&unit.HardwareType,
		&unit.GPUModel,
		&unit.GPUMemoryGB,
		&unit.CPUCores,
		&unit.RAMGB,
		&unit.BasePricePerHour,
		&unit.CurrentAvailability,
		&unit.GeographicRegion,
		&unit.LastHeartbeat,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("resource not found: %s", unitID)
		}
		return nil, fmt.Errorf("failed to get resource details: %w", err)
	}

	return &unit, nil
}

// ListAvailableResources lists all available compute resources matching criteria
func (rm *ResourceManager) ListAvailableResources(ctx context.Context, criteria ResourceCriteria) ([]*ComputeUnit, error) {
	query := `
		SELECT cu.unit_id, cu.provider_id, cu.hardware_type, cu.gpu_model, 
		       cu.gpu_memory_gb, cu.cpu_cores, cu.ram_gb, cu.base_price_per_hour_usdc, 
		       cu.current_availability, cu.geographic_region, cu.last_heartbeat
		FROM compute_units cu
		JOIN providers p ON cu.provider_id = p.provider_id
		WHERE cu.current_availability = 'available'
		AND p.status = 'active'
		AND cu.last_heartbeat > NOW() - INTERVAL '5 minutes'
	`

	// Add filters based on criteria
	args := []interface{}{}
	argIndex := 1

	if criteria.HardwareType != "" {
		query += fmt.Sprintf(" AND cu.hardware_type = $%d", argIndex)
		args = append(args, criteria.HardwareType)
		argIndex++
	}

	if criteria.GPUModel != "" {
		query += fmt.Sprintf(" AND cu.gpu_model = $%d", argIndex)
		args = append(args, criteria.GPUModel)
		argIndex++
	}

	if criteria.MinGPUMemory > 0 {
		query += fmt.Sprintf(" AND cu.gpu_memory_gb >= $%d", argIndex)
		args = append(args, criteria.MinGPUMemory)
		argIndex++
	}

	if criteria.MaxPrice > 0 {
		query += fmt.Sprintf(" AND cu.base_price_per_hour_usdc <= $%d", argIndex)
		args = append(args, criteria.MaxPrice)
		argIndex++
	}

	if criteria.GeographicRegion != "" {
		query += fmt.Sprintf(" AND cu.geographic_region = $%d", argIndex)
		args = append(args, criteria.GeographicRegion)
		argIndex++
	}

	if criteria.MinReputation > 0 {
		query += fmt.Sprintf(" AND p.reputation_score >= $%d", argIndex)
		args = append(args, criteria.MinReputation)
		argIndex++
	}

	query += " ORDER BY cu.base_price_per_hour_usdc ASC"

	rows, err := rm.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query available resources: %w", err)
	}
	defer rows.Close()

	var resources []*ComputeUnit
	for rows.Next() {
		var unit ComputeUnit
		err := rows.Scan(
			&unit.UnitID,
			&unit.ProviderID,
			&unit.HardwareType,
			&unit.GPUModel,
			&unit.GPUMemoryGB,
			&unit.CPUCores,
			&unit.RAMGB,
			&unit.BasePricePerHour,
			&unit.CurrentAvailability,
			&unit.GeographicRegion,
			&unit.LastHeartbeat,
		)
		if err != nil {
			continue
		}
		resources = append(resources, &unit)
	}

	return resources, nil
}

// ResourceCriteria represents criteria for filtering resources
type ResourceCriteria struct {
	HardwareType      string  `json:"hardware_type"`
	GPUModel          string  `json:"gpu_model"`
	MinGPUMemory      int     `json:"min_gpu_memory_gb"`
	MaxPrice          float64 `json:"max_price_per_hour_usdc"`
	GeographicRegion  string  `json:"geographic_region"`
	MinReputation     float64 `json:"min_reputation_score"`
}

// ComputeUnitOffer represents an offer for a compute unit
type ComputeUnitOffer struct {
	UnitID           string  `json:"unit_id"`
	HardwareType     string  `json:"hardware_type"`
	GPUModel         string  `json:"gpu_model"`
	GPUMemoryGB      int     `json:"gpu_memory_gb"`
	CPUCores         int     `json:"cpu_cores"`
	RAMGB            int     `json:"ram_gb"`
	BasePricePerHour float64 `json:"base_price_per_hour_usdc"`
	GeographicRegion string  `json:"geographic_region"`
}
