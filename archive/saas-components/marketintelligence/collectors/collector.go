package collectors

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"ocx.local/internal/marketintelligence"
	"ocx.local/internal/marketintelligence/connectors"
)

// MarketDataCollector collects real-time market data from all providers
type MarketDataCollector struct {
	connectors      map[string]connectors.ProviderConnector
	marketData      map[string]*marketintelligence.MarketDataBuffer
	priceHistory    map[string][]float64
	mu              sync.RWMutex
	lastCollectionTime time.Time
	collectionInterval time.Duration
	running         bool
	stopChan        chan struct{}
}

// NewMarketDataCollector creates a new market data collector
func NewMarketDataCollector() *MarketDataCollector {
	return &MarketDataCollector{
		connectors:      make(map[string]connectors.ProviderConnector),
		marketData:      make(map[string]*marketintelligence.MarketDataBuffer),
		priceHistory:    make(map[string][]float64),
		collectionInterval: 30 * time.Second,
		stopChan:        make(chan struct{}),
	}
}

// InitializeConnectors initializes all provider API connectors
func (m *MarketDataCollector) InitializeConnectors() {
	// In production, these would use real API credentials
	m.connectors["aws"] = connectors.NewAWSConnector(map[string]string{
		"access_key": "mock_aws_key",
		"secret_key": "mock_aws_secret",
	})
	
	m.connectors["gcp"] = connectors.NewGCPConnector(map[string]string{
		"project_id":       "mock_gcp_project",
		"credentials_file": "mock_credentials.json",
	})
	
	m.connectors["runpod"] = connectors.NewRunPodConnector(map[string]string{
		"api_key": "mock_runpod_key",
	})
	
	// Initialize market data buffers for each provider/resource/region combination
	resources := []string{"A100", "H100", "V100", "RTX4090", "T4"}
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "asia-southeast-1"}
	
	for _, provider := range m.connectors {
		for _, resource := range resources {
			for _, region := range regions {
				key := fmt.Sprintf("%s:%s:%s", provider.GetProviderID(), resource, region)
				m.marketData[key] = marketintelligence.NewMarketDataBuffer(1000)
			}
		}
	}
}

// StartCollection starts continuous market data collection
func (m *MarketDataCollector) StartCollection(ctx context.Context) error {
	if m.running {
		return fmt.Errorf("collection already running")
	}
	
	m.running = true
	fmt.Println("🔄 Starting market data collection...")
	
	go m.collectionLoop(ctx)
	return nil
}

// StopCollection stops market data collection
func (m *MarketDataCollector) StopCollection() {
	if !m.running {
		return
	}
	
	m.running = false
	close(m.stopChan)
	fmt.Println("⏹️  Market data collection stopped")
}

// collectionLoop runs the main collection loop
func (m *MarketDataCollector) collectionLoop(ctx context.Context) {
	ticker := time.NewTicker(m.collectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			if err := m.collectMarketSnapshot(ctx); err != nil {
				fmt.Printf("❌ Collection error: %v\n", err)
			}
		}
	}
}

// collectMarketSnapshot collects complete market snapshot from all providers
func (m *MarketDataCollector) collectMarketSnapshot(ctx context.Context) error {
	snapshotTime := time.Now()
	
	// Resources and regions to monitor
	resources := []string{"A100", "H100", "V100", "RTX4090", "T4"}
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "asia-southeast-1"}
	
	var wg sync.WaitGroup
	dataChan := make(chan *marketintelligence.MarketDataPoint, 100)
	
	// Create goroutines for all provider/resource/region combinations
	for _, provider := range m.connectors {
		for _, resource := range resources {
			for _, region := range regions {
				wg.Add(1)
				go func(p connectors.ProviderConnector, r, reg string) {
					defer wg.Done()
					
					dataPoint, err := m.collectProviderData(ctx, p, r, reg)
					if err != nil {
						fmt.Printf("⚠️  Failed to collect from %s: %v\n", p.GetProviderID(), err)
						return
					}
					
					if dataPoint != nil {
						dataChan <- dataPoint
					}
				}(provider, resource, region)
			}
		}
	}
	
	// Close channel when all goroutines are done
	go func() {
		wg.Wait()
		close(dataChan)
	}()
	
	// Process collected data
	validDataPoints := 0
	for dataPoint := range dataChan {
		m.storeMarketData(dataPoint)
		validDataPoints++
	}
	
	m.lastCollectionTime = snapshotTime
	fmt.Printf("📊 Collected %d market data points\n", validDataPoints)
	
	return nil
}

// collectProviderData collects data from single provider for specific resource/region
func (m *MarketDataCollector) collectProviderData(ctx context.Context, provider connectors.ProviderConnector, 
	resource, region string) (*marketintelligence.MarketDataPoint, error) {
	
	// Get pricing and availability concurrently
	pricingCtx, pricingCancel := context.WithTimeout(ctx, 10*time.Second)
	availabilityCtx, availabilityCancel := context.WithTimeout(ctx, 10*time.Second)
	defer pricingCancel()
	defer availabilityCancel()
	
	pricingChan := make(chan map[string]interface{}, 1)
	availabilityChan := make(chan map[string]interface{}, 1)
	errorChan := make(chan error, 2)
	
	// Collect pricing data
	go func() {
		pricing, err := provider.GetPricing(pricingCtx, resource, region)
		if err != nil {
			errorChan <- err
			return
		}
		pricingChan <- pricing
	}()
	
	// Collect availability data
	go func() {
		availability, err := provider.GetAvailability(availabilityCtx, resource, region)
		if err != nil {
			errorChan <- err
			return
		}
		availabilityChan <- availability
	}()
	
	// Wait for both results
	var pricingData, availabilityData map[string]interface{}
	completed := 0
	
	for completed < 2 {
		select {
		case pricing := <-pricingChan:
			pricingData = pricing
			completed++
		case availability := <-availabilityChan:
			availabilityData = availability
			completed++
		case err := <-errorChan:
			return nil, err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	// Calculate demand indicator based on availability vs capacity
	totalCapacity := 1
	if cap, ok := availabilityData["total_capacity"].(int); ok {
		totalCapacity = cap
	}
	
	available := 0
	if avail, ok := availabilityData["available_quantity"].(int); ok {
		available = avail
	}
	
	demandIndicator := 1.0 - (float64(available) / float64(maxFloat(totalCapacity, 1)))
	
	// Calculate quality score (simplified for demo)
	qualityScore := 0.8 + rand.Float64()*0.2 // Random between 0.8-1.0
	
	// Use spot/preemptible price if available, otherwise on-demand
	var price float64
	if spotPrice, ok := pricingData["spot_price"].(float64); ok && spotPrice > 0 {
		price = spotPrice
	} else if preemptiblePrice, ok := pricingData["preemptible_price"].(float64); ok && preemptiblePrice > 0 {
		price = preemptiblePrice
	} else if onDemandPrice, ok := pricingData["on_demand_price"].(float64); ok {
		price = onDemandPrice
	} else {
		return nil, fmt.Errorf("no valid price found")
	}
	
	dataPoint := &marketintelligence.MarketDataPoint{
		Timestamp:        time.Now().Unix(),
		ProviderID:       provider.GetProviderID(),
		ResourceType:     resource,
		Region:           region,
		PricePerHour:     price,
		AvailableQuantity: available,
		DemandIndicator:  demandIndicator,
		QualityScore:     qualityScore,
	}
	
	return dataPoint, nil
}

// storeMarketData stores market data point with indexing
func (m *MarketDataCollector) storeMarketData(dataPoint *marketintelligence.MarketDataPoint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Create composite key for indexing
	key := fmt.Sprintf("%s:%s:%s", dataPoint.ProviderID, dataPoint.ResourceType, dataPoint.Region)
	
	// Store in time-series buffer
	if buffer, exists := m.marketData[key]; exists {
		buffer.Add(*dataPoint)
	}
	
	// Store price history for trend analysis
	m.priceHistory[key] = append(m.priceHistory[key], dataPoint.PricePerHour)
	
	// Keep only recent price history (last 100 points)
	if len(m.priceHistory[key]) > 100 {
		m.priceHistory[key] = m.priceHistory[key][len(m.priceHistory[key])-100:]
	}
}

// GetCurrentPrices gets current prices for resource type across all providers
func (m *MarketDataCollector) GetCurrentPrices(resourceType, region string) []marketintelligence.MarketDataPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var currentPrices []marketintelligence.MarketDataPoint
	
	for key, buffer := range m.marketData {
		// Parse key
		parts := splitKey(key)
		if len(parts) != 3 {
			continue
		}
		
		_, resType, reg := parts[0], parts[1], parts[2]
		
		if resType == resourceType && reg == region {
			// Get most recent data point
			if latest := buffer.GetLatest(); latest != nil {
				currentPrices = append(currentPrices, *latest)
			}
		}
	}
	
	return currentPrices
}

// GetPriceTrend analyzes price trend for specific provider/resource/region
func (m *MarketDataCollector) GetPriceTrend(providerID, resourceType, region string, 
	lookbackPoints int) *marketintelligence.PriceTrend {
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	key := fmt.Sprintf("%s:%s:%s", providerID, resourceType, region)
	
	prices, exists := m.priceHistory[key]
	if !exists || len(prices) < 2 {
		return &marketintelligence.PriceTrend{
			CurrentPrice:       0,
			TrendDirection:     "no_data",
			PriceChangePercent: 0,
			Volatility:         0,
			MinPrice:           0,
			MaxPrice:           0,
			AvgPrice:           0,
		}
	}
	
	// Get recent prices
	if len(prices) > lookbackPoints {
		prices = prices[len(prices)-lookbackPoints:]
	}
	
	// Calculate trend metrics
	currentPrice := prices[len(prices)-1]
	firstPrice := prices[0]
	
	priceChangePercent := ((currentPrice / firstPrice) - 1) * 100
	
	trendDirection := "stable"
	if priceChangePercent > 5 {
		trendDirection = "rising"
	} else if priceChangePercent < -5 {
		trendDirection = "falling"
	}
	
	// Calculate volatility (standard deviation / mean)
	avgPrice := calculateAverage(prices)
	volatility := calculateVolatility(prices, avgPrice) / avgPrice
	
	minPrice := calculateMin(prices)
	maxPrice := calculateMax(prices)
	
	return &marketintelligence.PriceTrend{
		CurrentPrice:       currentPrice,
		TrendDirection:     trendDirection,
		PriceChangePercent: priceChangePercent,
		Volatility:         volatility,
		MinPrice:           minPrice,
		MaxPrice:           maxPrice,
		AvgPrice:           avgPrice,
	}
}

// GetMarketStats returns overall market statistics
func (m *MarketDataCollector) GetMarketStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	totalDataPoints := 0
	for _, buffer := range m.marketData {
		totalDataPoints += len(buffer.GetAll())
	}
	
	activeProviders := len(m.connectors)
	marketKeys := len(m.marketData)
	
	return map[string]interface{}{
		"total_data_points": totalDataPoints,
		"active_providers":  activeProviders,
		"market_keys":       marketKeys,
		"last_collection":   m.lastCollectionTime,
		"collection_interval": m.collectionInterval.Seconds(),
	}
}

// Helper functions
func maxFloat(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func splitKey(key string) []string {
	// Simple key splitting - in production would be more robust
	parts := make([]string, 0, 3)
	start := 0
	for i, char := range key {
		if char == ':' {
			parts = append(parts, key[start:i])
			start = i + 1
		}
	}
	parts = append(parts, key[start:])
	return parts
}

func calculateAverage(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, price := range prices {
		sum += price
	}
	return sum / float64(len(prices))
}

func calculateVolatility(prices []float64, avg float64) float64 {
	if len(prices) <= 1 {
		return 0
	}
	
	sumSquaredDiffs := 0.0
	for _, price := range prices {
		diff := price - avg
		sumSquaredDiffs += diff * diff
	}
	
	variance := sumSquaredDiffs / float64(len(prices)-1)
	return variance // Return variance, not standard deviation for volatility calculation
}

func calculateMin(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	
	min := prices[0]
	for _, price := range prices {
		if price < min {
			min = price
		}
	}
	return min
}

func calculateMax(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	
	max := prices[0]
	for _, price := range prices {
		if price > max {
			max = price
		}
	}
	return max
}
