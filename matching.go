// matching.go — OCX Matching Engine with Fee Calculation
// Implements min-cost assignment and auction mechanics

package main

import (
	"fmt"
	"sort"
	"time"
)

type MatchingEngine struct {
	offers map[ID]*Offer
	orders map[ID]*Order
	leases map[ID]*Lease
}

func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{
		offers: make(map[ID]*Offer),
		orders: make(map[ID]*Order),
		leases: make(map[ID]*Lease),
	}
}

// MatchingResult represents the result of a matching operation
type MatchingResult struct {
	Success   bool   `json:"success"`
	LeaseID   ID     `json:"lease_id,omitempty"`
	OfferID   ID     `json:"offer_id"`
	OrderID   ID     `json:"order_id"`
	Price     Money  `json:"price"`
	Fee       *Money `json:"fee,omitempty"`
	PayTo     string `json:"pay_to,omitempty"`
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
}

// OfferScore represents an offer with its computed score for matching
type OfferScore struct {
	Offer *Offer
	Score float64
	Price Money
}

// AddOffer adds an offer to the matching engine
func (me *MatchingEngine) AddOffer(offer *Offer) error {
	// Validate offer
	if err := me.validateOffer(offer); err != nil {
		return fmt.Errorf("invalid offer: %w", err)
	}
	
	me.offers[offer.OfferID] = offer
	return nil
}

// ProcessOrder attempts to match an order with available offers
func (me *MatchingEngine) ProcessOrder(order *Order) (*MatchingResult, error) {
	// Validate order
	if err := me.validateOrder(order); err != nil {
		return &MatchingResult{
			Success: false,
			OrderID: order.OrderID,
			Error:   fmt.Sprintf("invalid order: %v", err),
		}, nil
	}

	// Find compatible offers
	compatibleOffers := me.findCompatibleOffers(order)
	if len(compatibleOffers) == 0 {
		return &MatchingResult{
			Success: false,
			OrderID: order.OrderID,
			Message: "No compatible offers found",
		}, nil
	}

	// Score and rank offers (min-cost assignment)
	scoredOffers := me.scoreOffers(compatibleOffers, order)
	
	// Select best offer
	bestOffer := scoredOffers[0]
	
	// Create lease
	lease, err := me.createLease(order, bestOffer.Offer)
	if err != nil {
		return &MatchingResult{
			Success: false,
			OrderID: order.OrderID,
			OfferID: bestOffer.Offer.OfferID,
			Error:   fmt.Sprintf("failed to create lease: %v", err),
		}, nil
	}

	// Store lease
	me.leases[lease.LeaseID] = lease

	// Update order status
	order.State = OrderAccepted
	order.UpdatedAt = time.Now()

	return &MatchingResult{
		Success: true,
		LeaseID: lease.LeaseID,
		OfferID: bestOffer.Offer.OfferID,
		OrderID: order.OrderID,
		Price:   bestOffer.Price,
		Message: "Order matched successfully",
	}, nil
}

// validateOffer validates an offer
func (me *MatchingEngine) validateOffer(offer *Offer) error {
	if offer.OfferID == "" {
		return fmt.Errorf("offer ID is required")
	}
	
	if offer.MinGPUs <= 0 || offer.MaxGPUs <= 0 || offer.MinGPUs > offer.MaxGPUs {
		return fmt.Errorf("invalid GPU constraints: min=%d, max=%d", offer.MinGPUs, offer.MaxGPUs)
	}
	
	if offer.MinHours <= 0 || offer.MaxHours <= 0 || offer.MinHours > offer.MaxHours {
		return fmt.Errorf("invalid hour constraints: min=%d, max=%d", offer.MinHours, offer.MaxHours)
	}
	
	if offer.ValidTo.Before(time.Now()) {
		return fmt.Errorf("offer has expired")
	}
	
	if offer.UnitPrice.Amount == "" || offer.UnitPrice.Currency == "" {
		return fmt.Errorf("unit price is required")
	}
	
	return nil
}

// validateOrder validates an order
func (me *MatchingEngine) validateOrder(order *Order) error {
	if order.OrderID == "" {
		return fmt.Errorf("order ID is required")
	}
	
	if order.RequestedGPUs <= 0 {
		return fmt.Errorf("requested GPUs must be positive")
	}
	
	if order.Hours <= 0 {
		return fmt.Errorf("hours must be positive")
	}
	
	return nil
}

// findCompatibleOffers finds offers that can satisfy the order requirements
func (me *MatchingEngine) findCompatibleOffers(order *Order) []*Offer {
	var compatible []*Offer
	
	for _, offer := range me.offers {
		if me.isCompatible(order, offer) {
			compatible = append(compatible, offer)
		}
	}
	
	return compatible
}

// isCompatible checks if an offer can satisfy an order
func (me *MatchingEngine) isCompatible(order *Order, offer *Offer) bool {
	// Check GPU count
	if order.RequestedGPUs < offer.MinGPUs || order.RequestedGPUs > offer.MaxGPUs {
		return false
	}
	
	// Check duration
	if order.Hours < offer.MinHours || order.Hours > offer.MaxHours {
		return false
	}
	
	// Check validity
	if offer.ValidTo.Before(time.Now()) {
		return false
	}
	
	// Check budget if specified
	if order.BudgetCap != nil {
		totalCost := me.calculateTotalCost(order, offer)
		if !me.isWithinBudget(totalCost, *order.BudgetCap) {
			return false
		}
	}
	
	// Check currency compatibility
	if order.BudgetCap != nil && order.BudgetCap.Currency != offer.UnitPrice.Currency {
		return false
	}
	
	return true
}

// scoreOffers scores and sorts offers by cost (min-cost assignment)
func (me *MatchingEngine) scoreOffers(offers []*Offer, order *Order) []OfferScore {
	var scored []OfferScore
	
	for _, offer := range offers {
		totalCost := me.calculateTotalCost(order, offer)
		
		// Simple scoring: lower cost = better score
		// In production, you'd add factors like:
		// - Provider reputation
		// - Geographic proximity
		// - SLA guarantees
		// - Past performance
		score := -me.costToFloat(totalCost) // Negative because we want min-cost
		
		scored = append(scored, OfferScore{
			Offer: offer,
			Score: score,
			Price: totalCost,
		})
	}
	
	// Sort by score (descending, so best scores first)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	
	return scored
}

// calculateTotalCost calculates the total cost for an order given an offer
func (me *MatchingEngine) calculateTotalCost(order *Order, offer *Offer) Money {
	// Simple calculation: unit_price * gpus * hours
	// In production, you'd handle different pricing models:
	// - Spot vs reserved pricing
	// - Volume discounts
	// - Dynamic pricing
	// - Egress fees
	
	unitCost := me.costToFloat(offer.UnitPrice)
	totalCost := unitCost * float64(order.RequestedGPUs) * float64(order.Hours)
	
	return Money{
		Currency: offer.UnitPrice.Currency,
		Amount:   fmt.Sprintf("%.2f", totalCost),
		Scale:    offer.UnitPrice.Scale,
	}
}

// costToFloat converts Money to float64 for calculations
func (me *MatchingEngine) costToFloat(money Money) float64 {
	// Simplified conversion - in production use decimal library
	var amount float64
	fmt.Sscanf(money.Amount, "%f", &amount)
	
	// Apply scale
	for i := 0; i < money.Scale; i++ {
		amount /= 10.0
	}
	
	return amount
}

// isWithinBudget checks if cost is within budget
func (me *MatchingEngine) isWithinBudget(cost, budget Money) bool {
	if cost.Currency != budget.Currency {
		return false
	}
	
	costValue := me.costToFloat(cost)
	budgetValue := me.costToFloat(budget)
	
	return costValue <= budgetValue
}

// createLease creates a lease from a matched order and offer
func (me *MatchingEngine) createLease(order *Order, offer *Offer) (*Lease, error) {
	leaseID := generateULID()
	now := time.Now()
	endTime := now.Add(time.Duration(order.Hours) * time.Hour)
	
	// Create access specification
	access := AccessSpec{
		Method:    "ssh",
		Endpoints: []string{fmt.Sprintf("gpu-node-%s.ocx.example.com:22", offer.FleetID[:8])},
		CACertPEM: "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
		CredsRef:  generateULID(),
	}
	
	// Create policy reference
	policy := PolicyRef{
		PolicyID: "ocx-standard-policy-v1",
		Revision: "1.0.0",
		Hash: Hash{
			Alg:   "sha256",
			Value: "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		},
	}
	
	// Create SLA specification
	sla := &SLASpec{
		AvailabilityPct: 99.5,
		MinEgressMbps:   1000,
		MaxJitterMs:     10,
		Remedy:          "credit_50_percent",
	}
	
	lease := &Lease{
		LeaseID:      leaseID,
		Version:      V010,
		OrderID:      order.OrderID,
		FleetID:      offer.FleetID,
		AssignedGPUs: order.RequestedGPUs,
		StartAt:      now,
		EndAt:        &endTime,
		State:        LeaseProvisioning,
		Access:       access,
		Policy:       policy,
		SLA:          sla,
	}
	
	return lease, nil
}

// GetLease retrieves a lease by ID
func (me *MatchingEngine) GetLease(leaseID ID) (*Lease, bool) {
	lease, exists := me.leases[leaseID]
	return lease, exists
}

// GetActiveLeases returns all active leases
func (me *MatchingEngine) GetActiveLeases() []*Lease {
	var active []*Lease
	for _, lease := range me.leases {
		if lease.State == LeaseRunning || lease.State == LeaseProvisioning {
			active = append(active, lease)
		}
	}
	return active
}

// UpdateLeaseState updates the state of a lease
func (me *MatchingEngine) UpdateLeaseState(leaseID ID, newState LeaseState) error {
	lease, exists := me.leases[leaseID]
	if !exists {
		return fmt.Errorf("lease not found: %s", leaseID)
	}
	
	lease.State = newState
	return nil
}

// GetMarketStats returns basic market statistics
func (me *MatchingEngine) GetMarketStats() map[string]interface{} {
	totalOffers := len(me.offers)
	totalOrders := len(me.orders)
	totalLeases := len(me.leases)
	
	activeLeases := 0
	for _, lease := range me.leases {
		if lease.State == LeaseRunning || lease.State == LeaseProvisioning {
			activeLeases++
		}
	}
	
	return map[string]interface{}{
		"total_offers":  totalOffers,
		"total_orders":  totalOrders,
		"total_leases":  totalLeases,
		"active_leases": activeLeases,
		"match_rate":    float64(totalLeases) / float64(totalOrders) * 100,
	}
}
