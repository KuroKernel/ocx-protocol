package capacity

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// CapacityReservationEngine manages capacity reservations and futures trading
type CapacityReservationEngine struct {
	marketIntelligence interface{} // Would be actual market intelligence interface
	activeReservations map[string]*CapacityReservation
	reservationHistory []*CapacityReservation
	demandPredictor    *DemandPredictor
	profitTarget       float64
	mu                 sync.RWMutex
	stopChan           chan struct{}
	running            bool
}

// DemandPredictor predicts demand patterns for capacity planning
type DemandPredictor struct {
	demandHistory    map[string][]DemandPoint
	marketEvents     []MarketEvent
	predictionAccuracy map[string]float64
	mu               sync.RWMutex
}

// DemandPoint represents a demand observation
type DemandPoint struct {
	Timestamp int64   `json:"timestamp"`
	Demand    float64 `json:"demand"`
}

// MarketEvent represents a market event that affects demand
type MarketEvent struct {
	Timestamp int64  `json:"timestamp"`
	EventType string `json:"event_type"`
	Impact    float64 `json:"impact"`
}

// NewCapacityReservationEngine creates a new capacity reservation engine
func NewCapacityReservationEngine(marketIntelligence interface{}) *CapacityReservationEngine {
	return &CapacityReservationEngine{
		marketIntelligence: marketIntelligence,
		activeReservations: make(map[string]*CapacityReservation),
		reservationHistory: make([]*CapacityReservation, 0),
		demandPredictor:    NewDemandPredictor(),
		profitTarget:       0.25, // 25% minimum profit margin
		stopChan:           make(chan struct{}),
	}
}

// NewDemandPredictor creates a new demand predictor
func NewDemandPredictor() *DemandPredictor {
	return &DemandPredictor{
		demandHistory:      make(map[string][]DemandPoint),
		marketEvents:       make([]MarketEvent, 0),
		predictionAccuracy: make(map[string]float64),
	}
}

// StartMonitoring starts monitoring for reservation opportunities
func (c *CapacityReservationEngine) StartMonitoring(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.running {
		return fmt.Errorf("monitoring already running")
	}
	
	c.running = true
	fmt.Println("🔄 Starting capacity reservation monitoring...")
	
	// Start background monitoring tasks
	go c.monitorReservationOpportunities(ctx)
	go c.manageExistingReservations(ctx)
	
	return nil
}

// StopMonitoring stops capacity reservation monitoring
func (c *CapacityReservationEngine) StopMonitoring() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.running {
		return
	}
	
	c.running = false
	close(c.stopChan)
	fmt.Println("⏹️  Capacity reservation monitoring stopped")
}

// monitorReservationOpportunities continuously monitors for profitable reservation opportunities
func (c *CapacityReservationEngine) monitorReservationOpportunities(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-ticker.C:
			opportunities, err := c.findReservationOpportunities()
			if err != nil {
				fmt.Printf("⚠️ Reservation monitoring error: %v\n", err)
				continue
			}
			
			for _, opportunity := range opportunities {
				if c.evaluateOpportunityProfitability(opportunity) {
					c.executeReservation(opportunity)
				}
			}
		}
	}
}

// manageExistingReservations manages active reservations and optimizes utilization
func (c *CapacityReservationEngine) manageExistingReservations(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.mu.Lock()
			currentTime := time.Now().Unix()
			
			for resID, reservation := range c.activeReservations {
				// Check if reservation has expired
				if currentTime > reservation.EndTime {
					c.closeReservation(resID)
					continue
				}
				
				// Update current market price (simplified)
				reservation.CurrentMarketPrice = reservation.PurchasePrice * (0.8 + rand.Float64()*0.4)
				
				// Check if we should sell early for profit
				if c.shouldSellEarly(reservation) {
					c.sellReservationEarly(resID)
				}
			}
			c.mu.Unlock()
		}
	}
}

// findReservationOpportunities finds profitable capacity reservation opportunities
func (c *CapacityReservationEngine) findReservationOpportunities() ([]*ReservationOpportunity, error) {
	var opportunities []*ReservationOpportunity
	
	// Analyze market data for all resource types
	resources := []string{"A100", "H100", "V100", "RTX4090"}
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "asia-southeast-1"}
	
	for _, resourceType := range resources {
		for _, region := range regions {
			// Simulate getting current prices (would use actual market intelligence)
			currentPrices := c.simulateCurrentPrices(resourceType, region)
			demandForecast := c.demandPredictor.PredictDemand(resourceType, region, 168) // 1 week
			
			for _, priceData := range currentPrices {
				// Check if current price is below predicted future price
				expectedFuturePrice := demandForecast.PeakPrice
				if expectedFuturePrice == 0 {
					expectedFuturePrice = priceData.Price * 1.1 // 10% increase assumption
				}
				
				potentialProfit := (expectedFuturePrice - priceData.Price) / priceData.Price
				
				if potentialProfit > c.profitTarget {
					opportunity := &ReservationOpportunity{
						ProviderID:         priceData.ProviderID,
						ResourceType:       resourceType,
						Region:             region,
						CurrentPrice:       priceData.Price,
						PredictedPrice:     expectedFuturePrice,
						AvailableQuantity:  priceData.AvailableQuantity,
						ProfitPotential:    potentialProfit,
						Confidence:         demandForecast.Confidence,
						RecommendedDuration: demandForecast.HighDemandDuration,
						RiskScore:          c.calculateReservationRisk(priceData),
					}
					opportunities = append(opportunities, opportunity)
				}
			}
		}
	}
	
	// Sort by profit potential
	sort.Slice(opportunities, func(i, j int) bool {
		return opportunities[i].ProfitPotential > opportunities[j].ProfitPotential
	})
	
	// Return top 10 opportunities
	if len(opportunities) > 10 {
		return opportunities[:10], nil
	}
	return opportunities, nil
}

// PriceData represents price data for a provider
type PriceData struct {
	ProviderID        string  `json:"provider_id"`
	Price             float64 `json:"price"`
	AvailableQuantity int     `json:"available_quantity"`
}

// simulateCurrentPrices simulates current market prices (would use actual market intelligence)
func (c *CapacityReservationEngine) simulateCurrentPrices(resourceType, region string) []PriceData {
	providers := []string{"aws", "gcp", "azure", "runpod", "lambdalabs"}
	var prices []PriceData
	
	basePrices := map[string]float64{
		"A100":    3.0,
		"H100":    8.0,
		"V100":    2.0,
		"RTX4090": 0.8,
	}
	
	basePrice := basePrices[resourceType]
	if basePrice == 0 {
		basePrice = 2.0
	}
	
	for _, provider := range providers {
		// Add provider-specific pricing variation
		providerMultiplier := map[string]float64{
			"aws":       1.0,
			"gcp":       1.05,
			"azure":     1.1,
			"runpod":    0.6,
			"lambdalabs": 0.7,
		}[provider]
		
		price := basePrice * providerMultiplier * (0.9 + rand.Float64()*0.2)
		availableQuantity := rand.Intn(100) + 10
		
		prices = append(prices, PriceData{
			ProviderID:        provider,
			Price:             price,
			AvailableQuantity: availableQuantity,
		})
	}
	
	return prices
}

// DemandForecast represents a demand forecast
type DemandForecast struct {
	PeakDemand         float64 `json:"peak_demand"`
	LowDemand          float64 `json:"low_demand"`
	PeakPrice          float64 `json:"peak_price"`
	LowPrice           float64 `json:"low_price"`
	Confidence         float64 `json:"confidence"`
	HighDemandDuration int     `json:"high_demand_duration"`
}

// PredictDemand predicts demand for a resource type and region
func (d *DemandPredictor) PredictDemand(resourceType, region string, forecastHours int) DemandForecast {
	key := fmt.Sprintf("%s:%s", resourceType, region)
	
	d.mu.RLock()
	history := d.demandHistory[key]
	d.mu.RUnlock()
	
	if len(history) < 10 {
		// Insufficient data - return baseline prediction
		return DemandForecast{
			PeakDemand:         0.7,
			LowDemand:          0.3,
			PeakPrice:          3.0,
			LowPrice:           1.5,
			Confidence:         0.3,
			HighDemandDuration: int(float64(forecastHours) * 0.4),
		}
	}
	
	// Simple trend analysis
	recentDemands := make([]float64, 0)
	for i := len(history) - 20; i < len(history); i++ {
		if i >= 0 {
			recentDemands = append(recentDemands, history[i].Demand)
		}
	}
	
	recentAvg := 0.0
	for _, demand := range recentDemands {
		recentAvg += demand
	}
	recentAvg /= float64(len(recentDemands))
	
	// Time-based patterns
	currentHour := time.Now().Hour()
	currentWeekday := time.Now().Weekday()
	
	timeMultiplier := 1.0
	if currentHour >= 8 && currentHour <= 18 && currentWeekday < 5 { // Business hours, weekday
		timeMultiplier = 1.3
	} else if currentHour < 6 || currentHour > 22 { // Late night/early morning
		timeMultiplier = 0.7
	}
	
	// Resource type patterns
	resourceMultipliers := map[string]float64{
		"A100":    1.2,
		"H100":    1.4,
		"V100":    1.0,
		"RTX4090": 0.8,
	}
	
	resourceMultiplier := resourceMultipliers[resourceType]
	
	// Calculate prediction
	baseDemand := recentAvg * timeMultiplier * resourceMultiplier
	peakDemand := minFloat(1.0, baseDemand * 1.2)
	lowDemand := maxFloat(0.1, baseDemand * 0.8)
	
	// Estimate pricing impact
	basePrice := 2.0
	peakPrice := basePrice * (1 + peakDemand * 0.8)
	lowPrice := basePrice * (1 - (1 - lowDemand) * 0.4)
	
	// Calculate confidence
	confidence := minFloat(1.0, float64(len(history))/100.0)
	
	return DemandForecast{
		PeakDemand:         peakDemand,
		LowDemand:          lowDemand,
		PeakPrice:          peakPrice,
		LowPrice:           lowPrice,
		Confidence:         confidence,
		HighDemandDuration: int(float64(forecastHours) * peakDemand),
	}
}

// evaluateOpportunityProfitability evaluates if reservation opportunity meets profitability criteria
func (c *CapacityReservationEngine) evaluateOpportunityProfitability(opportunity *ReservationOpportunity) bool {
	profitPotential := opportunity.ProfitPotential
	confidence := opportunity.Confidence
	riskScore := opportunity.RiskScore
	
	// Risk-adjusted profit threshold
	minProfit := c.profitTarget + (riskScore * 0.1)
	
	// Confidence-weighted evaluation
	expectedProfit := profitPotential * confidence
	
	return expectedProfit > minProfit && confidence > 0.6
}

// executeReservation executes capacity reservation
func (c *CapacityReservationEngine) executeReservation(opportunity *ReservationOpportunity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Calculate optimal reservation size
	availableQuantity := opportunity.AvailableQuantity
	recommendedQuantity := minInt(availableQuantity, c.calculateOptimalQuantity(opportunity))
	
	reservation := &CapacityReservation{
		ReservationID:      fmt.Sprintf("res_%d_%d", time.Now().Unix(), rand.Intn(10000)),
		ProviderID:         opportunity.ProviderID,
		ResourceType:       opportunity.ResourceType,
		Region:             opportunity.Region,
		Quantity:           int(recommendedQuantity),
		StartTime:          time.Now().Unix(),
		EndTime:            time.Now().Unix() + int64(opportunity.RecommendedDuration*3600),
		ReservationType:    Futures,
		PurchasePrice:      opportunity.CurrentPrice,
		CurrentMarketPrice: opportunity.PredictedPrice,
	}
	
	c.activeReservations[reservation.ReservationID] = reservation
	
	fmt.Printf("💰 RESERVED: %dx %s at $%.2f/hr\n", 
		recommendedQuantity, reservation.ResourceType, reservation.PurchasePrice)
	fmt.Printf("   Expected profit: %.1f%%\n", opportunity.ProfitPotential*100)
}

// calculateOptimalQuantity calculates optimal quantity to reserve
func (c *CapacityReservationEngine) calculateOptimalQuantity(opportunity *ReservationOpportunity) int {
	available := opportunity.AvailableQuantity
	confidence := opportunity.Confidence
	profitPotential := opportunity.ProfitPotential
	
	// Conservative approach: reserve percentage based on confidence
	var reservePercentage float64
	if confidence > 0.9 {
		reservePercentage = 0.8
	} else if confidence > 0.8 {
		reservePercentage = 0.6
	} else if confidence > 0.7 {
		reservePercentage = 0.4
	} else {
		reservePercentage = 0.2
	}
	
	// Also consider profit potential
	if profitPotential > 0.5 { // >50% profit
		reservePercentage *= 1.5
	} else if profitPotential < 0.3 { // <30% profit
		reservePercentage *= 0.7
	}
	
	return maxInt(1, int(float64(available)*reservePercentage))
}

// calculateReservationRisk calculates risk score for reservation (0-1, higher = more risky)
func (c *CapacityReservationEngine) calculateReservationRisk(priceData PriceData) float64 {
	// Base risk from provider reliability
	providerRisk := map[string]float64{
		"aws":       0.1,
		"gcp":       0.15,
		"azure":     0.2,
		"runpod":    0.4,
		"lambdalabs": 0.45,
		"vastai":    0.6,
	}[priceData.ProviderID]
	
	if providerRisk == 0 {
		providerRisk = 0.5
	}
	
	// Market volatility risk (simplified)
	volatilityRisk := rand.Float64() * 0.4
	
	// Capacity risk (simplified)
	capacityRisk := maxFloat(0, (1-float64(priceData.AvailableQuantity)/100) * 0.3)
	
	return minFloat(1.0, providerRisk+volatilityRisk+capacityRisk)
}

// shouldSellEarly determines if reservation should be sold early
func (c *CapacityReservationEngine) shouldSellEarly(reservation *CapacityReservation) bool {
	currentProfit := reservation.ProfitMargin()
	
	// Sell if we've hit a good profit target
	profitThreshold := 0.4 // 40% profit
	
	// Or if there's risk of price collapse
	timeRemaining := float64(reservation.EndTime-time.Now().Unix()) / 3600
	if timeRemaining < 2 && currentProfit > 0.15 { // 15% profit with <2 hours left
		return true
	}
	
	return currentProfit > profitThreshold
}

// sellReservationEarly sells reservation early at current market price
func (c *CapacityReservationEngine) sellReservationEarly(reservationID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	reservation := c.activeReservations[reservationID]
	if reservation == nil {
		return
	}
	
	fmt.Printf("💸 SELLING EARLY: %s reservation\n", reservation.ResourceType)
	fmt.Printf("   Profit realized: %.1f%%\n", reservation.ProfitMargin()*100)
	
	c.closeReservation(reservationID)
}

// closeReservation closes reservation and records performance
func (c *CapacityReservationEngine) closeReservation(reservationID string) {
	reservation := c.activeReservations[reservationID]
	if reservation == nil {
		return
	}
	
	delete(c.activeReservations, reservationID)
	c.reservationHistory = append(c.reservationHistory, reservation)
	
	// Keep only recent history
	if len(c.reservationHistory) > 1000 {
		c.reservationHistory = c.reservationHistory[len(c.reservationHistory)-1000:]
	}
}

// AllocateReservedCapacity allocates reserved capacity to customer request
func (c *CapacityReservationEngine) AllocateReservedCapacity(request *CustomerRequest) *CapacityReservation {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Find suitable reservations
	var suitableReservations []*CapacityReservation
	for _, reservation := range c.activeReservations {
		if reservation.ResourceType == request.ResourceType &&
			reservation.Region == request.Region &&
			reservation.Quantity >= request.Quantity &&
			!reservation.LockedByCustomer &&
			time.Now().Unix()+int64(request.DurationHours*3600) <= reservation.EndTime {
			suitableReservations = append(suitableReservations, reservation)
		}
	}
	
	if len(suitableReservations) == 0 {
		return nil
	}
	
	// Calculate customer pricing for each option
	var bestReservation *CapacityReservation
	bestCustomerPrice := float64(request.MaxPricePerHour)
	
	for _, reservation := range suitableReservations {
		// Price to customer: cost + margin + premium for guaranteed availability
		basePrice := reservation.PurchasePrice * 1.3 // 30% margin
		marketPremium := maxFloat(0, reservation.CurrentMarketPrice-reservation.PurchasePrice) * 0.5
		customerPrice := basePrice + marketPremium
		
		if customerPrice < bestCustomerPrice {
			bestReservation = reservation
			bestCustomerPrice = customerPrice
		}
	}
	
	if bestReservation != nil {
		// Lock reservation for customer
		bestReservation.LockedByCustomer = true
		bestReservation.CustomerPrice = &bestCustomerPrice
		bestReservation.UtilizationRate = float64(request.Quantity) / float64(bestReservation.Quantity)
		
		fmt.Printf("🎯 ALLOCATED: %dx %s to customer at $%.2f/hr\n",
			request.Quantity, request.ResourceType, bestCustomerPrice)
		fmt.Printf("   OCX margin: %.1f%%\n",
			((bestCustomerPrice-bestReservation.PurchasePrice)/bestReservation.PurchasePrice)*100)
		
		return bestReservation
	}
	
	return nil
}

// GetReservationAnalytics returns comprehensive reservation performance analytics
func (c *CapacityReservationEngine) GetReservationAnalytics() *ReservationAnalytics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if len(c.reservationHistory) == 0 {
		return &ReservationAnalytics{
			TotalReservations: 0,
			ActiveReservations: len(c.activeReservations),
		}
	}
	
	// Calculate performance metrics
	var totalProfit float64
	var profitableReservations int
	
	for _, res := range c.reservationHistory {
		if res.CustomerPrice != nil {
			profit := res.ProfitMargin()
			totalProfit += profit
			if profit > 0 {
				profitableReservations++
			}
		}
	}
	
	avgProfitMargin := totalProfit / float64(len(c.reservationHistory))
	successRate := float64(profitableReservations) / float64(len(c.reservationHistory))
	
	// Resource type performance
	resourcePerformance := make(map[string]ResourceStats)
	resourceProfits := make(map[string][]float64)
	
	for _, res := range c.reservationHistory {
		if res.CustomerPrice != nil {
			profit := res.ProfitMargin()
			resourceProfits[res.ResourceType] = append(resourceProfits[res.ResourceType], profit)
		}
	}
	
	for resourceType, profits := range resourceProfits {
		if len(profits) > 0 {
			avgProfit := 0.0
			maxProfit := profits[0]
			minProfit := profits[0]
			
			for _, profit := range profits {
				avgProfit += profit
				if profit > maxProfit {
					maxProfit = profit
				}
				if profit < minProfit {
					minProfit = profit
				}
			}
			avgProfit /= float64(len(profits))
			
			resourcePerformance[resourceType] = ResourceStats{
				AvgProfitMargin:  avgProfit,
				MaxProfit:        maxProfit,
				MinProfit:        minProfit,
				ReservationCount: len(profits),
			}
		}
	}
	
	// Top performing providers
	providerProfits := make(map[string][]float64)
	for _, res := range c.reservationHistory {
		if res.CustomerPrice != nil {
			profit := res.ProfitMargin()
			providerProfits[res.ProviderID] = append(providerProfits[res.ProviderID], profit)
		}
	}
	
	var topProviders []ProviderStats
	for providerID, profits := range providerProfits {
		if len(profits) > 0 {
			avgProfit := 0.0
			for _, profit := range profits {
				avgProfit += profit
			}
			avgProfit /= float64(len(profits))
			
			topProviders = append(topProviders, ProviderStats{
				ProviderID: providerID,
				AvgProfit:  avgProfit,
			})
		}
	}
	
	// Sort by average profit
	sort.Slice(topProviders, func(i, j int) bool {
		return topProviders[i].AvgProfit > topProviders[j].AvgProfit
	})
	
	if len(topProviders) > 5 {
		topProviders = topProviders[:5]
	}
	
	return &ReservationAnalytics{
		TotalReservations:   len(c.reservationHistory),
		ActiveReservations:  len(c.activeReservations),
		SuccessRate:         successRate,
		AvgProfitMargin:     avgProfitMargin,
		TotalProfitPercent:  totalProfit,
		ResourcePerformance: resourcePerformance,
		TopProviders:        topProviders,
		VolumeTrend: VolumeTrend{
			Trend: "stable", // Simplified
		},
	}
}


// Helper functions
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
