package matching

import (
	"database/sql"
	"fmt"
	"time"
)

// MatchingEngine handles order-to-compute-unit matching
type MatchingEngine struct {
	db *sql.DB
}

// NewMatchingEngine creates a new matching engine
func NewMatchingEngine(db *sql.DB) *MatchingEngine {
	return &MatchingEngine{db: db}
}

// Match represents a successful match between order and compute unit
type Match struct {
	MatchID     string    `json:"match_id"`
	OrderID     string    `json:"order_id"`
	UnitID      string    `json:"unit_id"`
	ProviderID  string    `json:"provider_id"`
	Price       float64   `json:"price_per_hour"`
	MatchedAt   time.Time `json:"matched_at"`
	Status      string    `json:"status"`
}

// MatchOrder attempts to match an order with available compute units
func (e *MatchingEngine) MatchOrder(orderID string) (*Match, error) {
	// Get order details
	order, err := e.getOrder(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	
	// Find available units
	units, err := e.findAvailableUnits(order)
	if err != nil {
		return nil, fmt.Errorf("failed to find available units: %w", err)
	}
	
	if len(units) == 0 {
		return nil, fmt.Errorf("no available units found for order %s", orderID)
	}
	
	// Select best unit (lowest price, highest reputation)
	bestUnit := e.selectBestUnit(units)
	
	// Create match
	match := &Match{
		MatchID:    fmt.Sprintf("match_%d", time.Now().UnixNano()),
		OrderID:    orderID,
		UnitID:     bestUnit.UnitID,
		ProviderID: bestUnit.ProviderID,
		Price:      bestUnit.Price,
		MatchedAt:  time.Now(),
		Status:     "matched",
	}
	
	// Save match to database
	if err := e.saveMatch(match); err != nil {
		return nil, fmt.Errorf("failed to save match: %w", err)
	}
	
	// Update order status
	if err := e.updateOrderStatus(orderID, "matched"); err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}
	
	// Update unit availability
	if err := e.updateUnitAvailability(bestUnit.UnitID, "reserved"); err != nil {
		return nil, fmt.Errorf("failed to update unit availability: %w", err)
	}
	
	return match, nil
}

// GetMatches returns matches for an order
func (e *MatchingEngine) GetMatches(orderID string) ([]Match, error) {
	query := `
		SELECT match_id, order_id, unit_id, provider_id, price_per_hour, matched_at, status
		FROM order_matches 
		WHERE order_id = $1
		ORDER BY matched_at DESC
	`
	
	rows, err := e.db.Query(query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var matches []Match
	for rows.Next() {
		var match Match
		err := rows.Scan(&match.MatchID, &match.OrderID, &match.UnitID, 
			&match.ProviderID, &match.Price, &match.MatchedAt, &match.Status)
		if err != nil {
			continue
		}
		matches = append(matches, match)
	}
	
	return matches, nil
}

// ProcessPendingOrders processes all pending orders
func (e *MatchingEngine) ProcessPendingOrders() (int, error) {
	// Get pending orders
	orders, err := e.getPendingOrders()
	if err != nil {
		return 0, fmt.Errorf("failed to get pending orders: %w", err)
	}
	
	matchedCount := 0
	for _, order := range orders {
		_, err := e.MatchOrder(order.OrderID)
		if err != nil {
			// Log error but continue with other orders
			fmt.Printf("Failed to match order %s: %v\n", order.OrderID, err)
			continue
		}
		matchedCount++
	}
	
	return matchedCount, nil
}

// Order represents an order from the database
type Order struct {
	OrderID           string  `json:"order_id"`
	RequesterID       string  `json:"requester_id"`
	HardwareType      string  `json:"required_hardware_type"`
	MaxPrice          float64 `json:"max_price_per_hour_usdc"`
	MinReputation     float64 `json:"min_provider_reputation"`
	EstimatedDuration float64 `json:"estimated_duration_hours"`
	Status            string  `json:"order_status"`
}

// ComputeUnit represents a compute unit from the database
type ComputeUnit struct {
	UnitID      string  `json:"unit_id"`
	ProviderID  string  `json:"provider_id"`
	HardwareType string `json:"hardware_type"`
	GPUModel    string  `json:"gpu_model"`
	GPUMemory   int     `json:"gpu_memory_gb"`
	Price       float64 `json:"base_price_per_hour_usdc"`
	Availability string `json:"current_availability"`
	Reputation  float64 `json:"reputation_score"`
}

// Helper methods
func (e *MatchingEngine) getOrder(orderID string) (*Order, error) {
	query := `
		SELECT order_id, requester_id, required_hardware_type, max_price_per_hour_usdc,
		       min_provider_reputation, estimated_duration_hours, order_status
		FROM compute_orders 
		WHERE order_id = $1
	`
	
	var order Order
	err := e.db.QueryRow(query, orderID).Scan(
		&order.OrderID, &order.RequesterID, &order.HardwareType, &order.MaxPrice,
		&order.MinReputation, &order.EstimatedDuration, &order.Status,
	)
	
	return &order, err
}

func (e *MatchingEngine) findAvailableUnits(order *Order) ([]ComputeUnit, error) {
	query := `
		SELECT cu.unit_id, cu.provider_id, cu.hardware_type, cu.gpu_model,
		       cu.gpu_memory_gb, cu.base_price_per_hour_usdc, cu.current_availability,
		       p.reputation_score
		FROM compute_units cu
		JOIN providers p ON cu.provider_id = p.provider_id
		WHERE cu.hardware_type = $1 
		  AND cu.current_availability = 'available'
		  AND cu.base_price_per_hour_usdc <= $2
		  AND p.reputation_score >= $3
		ORDER BY cu.base_price_per_hour_usdc ASC, p.reputation_score DESC
	`
	
	rows, err := e.db.Query(query, order.HardwareType, order.MaxPrice, order.MinReputation)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var units []ComputeUnit
	for rows.Next() {
		var unit ComputeUnit
		err := rows.Scan(&unit.UnitID, &unit.ProviderID, &unit.HardwareType,
			&unit.GPUModel, &unit.GPUMemory, &unit.Price, &unit.Availability, &unit.Reputation)
		if err != nil {
			continue
		}
		units = append(units, unit)
	}
	
	return units, nil
}

func (e *MatchingEngine) selectBestUnit(units []ComputeUnit) ComputeUnit {
	// Simple selection: first unit (already sorted by price, then reputation)
	return units[0]
}

func (e *MatchingEngine) saveMatch(match *Match) error {
	query := `
		INSERT INTO order_matches (match_id, order_id, unit_id, provider_id, price_per_hour, matched_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	_, err := e.db.Exec(query, match.MatchID, match.OrderID, match.UnitID,
		match.ProviderID, match.Price, match.MatchedAt, match.Status)
	
	return err
}

func (e *MatchingEngine) updateOrderStatus(orderID, status string) error {
	query := `UPDATE compute_orders SET order_status = $1 WHERE order_id = $2`
	_, err := e.db.Exec(query, status, orderID)
	return err
}

func (e *MatchingEngine) updateUnitAvailability(unitID, availability string) error {
	query := `UPDATE compute_units SET current_availability = $1 WHERE unit_id = $2`
	_, err := e.db.Exec(query, availability, unitID)
	return err
}

func (e *MatchingEngine) getPendingOrders() ([]Order, error) {
	query := `
		SELECT order_id, requester_id, required_hardware_type, max_price_per_hour_usdc,
		       min_provider_reputation, estimated_duration_hours, order_status
		FROM compute_orders 
		WHERE order_status = 'pending_matching'
		ORDER BY placed_at ASC
	`
	
	rows, err := e.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var orders []Order
	for rows.Next() {
		var order Order
		err := rows.Scan(&order.OrderID, &order.RequesterID, &order.HardwareType,
			&order.MaxPrice, &order.MinReputation, &order.EstimatedDuration, &order.Status)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}
	
	return orders, nil
}
