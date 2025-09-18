package analytics

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// CustomerUsageAnalyzer analyzes customer usage patterns for demand prediction and optimization
type CustomerUsageAnalyzer struct {
	customerHistory map[string][]UsageRecord
	usagePatterns   map[string][]WorkloadPattern
	mu              sync.RWMutex
}

// NewCustomerUsageAnalyzer creates a new customer usage analyzer
func NewCustomerUsageAnalyzer() *CustomerUsageAnalyzer {
	return &CustomerUsageAnalyzer{
		customerHistory: make(map[string][]UsageRecord),
		usagePatterns:   make(map[string][]WorkloadPattern),
	}
}

// RecordUsage records customer usage for pattern analysis
func (c *CustomerUsageAnalyzer) RecordUsage(customerID string, usageData map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	usageRecord := UsageRecord{
		Timestamp:       now.Unix(),
		ResourceType:    usageData["resource_type"].(string),
		Region:          usageData["region"].(string),
		Quantity:        usageData["quantity"].(int),
		DurationHours:   usageData["duration_hours"].(float64),
		TotalCost:       usageData["total_cost"].(float64),
		SLARequirements: usageData["sla_requirements"].(map[string]interface{}),
		WorkloadType:    getStringOrDefault(usageData, "workload_type", "unknown"),
		StartTimeHour:   now.Hour(),
		DayOfWeek:       int(now.Weekday()),
		DayOfMonth:      now.Day(),
	}
	
	c.customerHistory[customerID] = append(c.customerHistory[customerID], usageRecord)
	
	// Limit history size
	if len(c.customerHistory[customerID]) > 1000 {
		c.customerHistory[customerID] = c.customerHistory[customerID][len(c.customerHistory[customerID])-1000:]
	}
	
	// Update patterns if sufficient data
	if len(c.customerHistory[customerID]) >= 10 {
		c.updateUsagePatterns(customerID)
	}
}

// updateUsagePatterns updates usage patterns for customer based on historical data
func (c *CustomerUsageAnalyzer) updateUsagePatterns(customerID string) {
	history := c.customerHistory[customerID]
	
	// Group by resource type
	resourceGroups := make(map[string][]UsageRecord)
	for _, usage := range history {
		resourceGroups[usage.ResourceType] = append(resourceGroups[usage.ResourceType], usage)
	}
	
	var patterns []WorkloadPattern
	
	for resourceType, usages := range resourceGroups {
		if len(usages) < 5 { // Need minimum data for pattern
			continue
		}
		
		pattern := c.extractPattern(customerID, resourceType, usages)
		if pattern != nil {
			patterns = append(patterns, *pattern)
		}
	}
	
	c.usagePatterns[customerID] = patterns
}

// extractPattern extracts usage pattern from historical data
func (c *CustomerUsageAnalyzer) extractPattern(customerID, resourceType string, usages []UsageRecord) *WorkloadPattern {
	// Calculate typical values
	quantities := make([]int, len(usages))
	durations := make([]float64, len(usages))
	startTimes := make([]int, len(usages))
	costs := make([]float64, len(usages))
	
	for i, usage := range usages {
		quantities[i] = usage.Quantity
		durations[i] = usage.DurationHours
		startTimes[i] = usage.StartTimeHour
		costs[i] = usage.TotalCost
	}
	
	// Find most common patterns
	typicalQuantity := medianInt(quantities)
	typicalDuration := medianFloat64(durations)
	
	// Find typical start time clusters
	startTimeClusters := c.clusterStartTimes(startTimes)
	
	// Calculate frequency (days between runs)
	frequencyDays := 7.0 // Default weekly
	if len(usages) >= 2 {
		timestamps := make([]int64, len(usages))
		for i, usage := range usages {
			timestamps[i] = usage.Timestamp
		}
		sort.Slice(timestamps, func(i, j int) bool {
			return timestamps[i] < timestamps[j]
		})
		
		intervals := make([]float64, len(timestamps)-1)
		for i := 1; i < len(timestamps); i++ {
			intervals[i-1] = float64(timestamps[i]-timestamps[i-1]) / 86400 // Convert to days
		}
		
		if len(intervals) > 0 {
			frequencyDays = medianFloat64(intervals)
		}
	}
	
	// Analyze cost sensitivity
	costSensitivity := c.calculateCostSensitivity(usages)
	
	// Extract seasonality patterns
	seasonality := c.extractSeasonality(usages)
	
	// Most common SLA requirements
	slaRequirements := c.extractCommonSLA(usages)
	
	patternID := fmt.Sprintf("%s_%s_%d", customerID, resourceType, time.Now().Unix())
	
	return &WorkloadPattern{
		CustomerID:        customerID,
		PatternID:         patternID,
		ResourceType:      resourceType,
		TypicalQuantity:   typicalQuantity,
		TypicalDuration:   typicalDuration,
		TypicalStartTimes: startTimeClusters,
		FrequencyDays:     frequencyDays,
		Seasonality:       seasonality,
		CostSensitivity:   costSensitivity,
		SLARequirements:   slaRequirements,
	}
}

// clusterStartTimes finds clusters in start times (hours of day)
func (c *CustomerUsageAnalyzer) clusterStartTimes(startTimes []int) []int {
	if len(startTimes) == 0 {
		return []int{9} // Default 9 AM
	}
	
	// Simple clustering - find most common hours
	timeCounts := make(map[int]int)
	for _, hour := range startTimes {
		timeCounts[hour]++
	}
	
	// Convert to slice and sort by count
	type timeCount struct {
		hour  int
		count int
	}
	
	var timeCountSlice []timeCount
	for hour, count := range timeCounts {
		timeCountSlice = append(timeCountSlice, timeCount{hour, count})
	}
	
	sort.Slice(timeCountSlice, func(i, j int) bool {
		return timeCountSlice[i].count > timeCountSlice[j].count
	})
	
	// Return top 3 most common start times
	result := make([]int, 0, 3)
	for i, tc := range timeCountSlice {
		if i >= 3 {
			break
		}
		result = append(result, tc.hour)
	}
	
	return result
}

// calculateCostSensitivity calculates how cost-sensitive the customer is (0-1)
func (c *CustomerUsageAnalyzer) calculateCostSensitivity(usages []UsageRecord) float64 {
	// Look at price variance acceptance
	var costsPerHour []float64
	for _, usage := range usages {
		if usage.DurationHours > 0 && usage.Quantity > 0 {
			costPerUnitHour := usage.TotalCost / (float64(usage.Quantity) * usage.DurationHours)
			costsPerHour = append(costsPerHour, costPerUnitHour)
		}
	}
	
	if len(costsPerHour) < 3 {
		return 0.5 // Default medium sensitivity
	}
	
	// Higher variance = lower sensitivity to price
	avgCost := 0.0
	for _, cost := range costsPerHour {
		avgCost += cost
	}
	avgCost /= float64(len(costsPerHour))
	
	variance := 0.0
	for _, cost := range costsPerHour {
		diff := cost - avgCost
		variance += diff * diff
	}
	variance /= float64(len(costsPerHour))
	
	priceVariance := variance / avgCost
	
	// Convert to sensitivity score (lower variance = higher sensitivity)
	sensitivity := maxFloat(0, minFloat(1, 1-(priceVariance*2)))
	return sensitivity
}

// extractSeasonality extracts seasonal usage patterns
func (c *CustomerUsageAnalyzer) extractSeasonality(usages []UsageRecord) map[string]float64 {
	monthlyCounts := make(map[int]int)
	weeklyCounts := make(map[int]int)
	
	for _, usage := range usages {
		t := time.Unix(usage.Timestamp, 0)
		monthlyCounts[t.Month()]++
		weeklyCounts[int(t.Weekday())]++
	}
	
	// Convert to multipliers (relative to average)
	totalUsages := len(usages)
	avgMonthly := float64(totalUsages) / 12
	avgWeekly := float64(totalUsages) / 7
	
	seasonality := make(map[string]float64)
	
	for month := 1; month <= 12; month++ {
		count := monthlyCounts[month]
		multiplier := float64(count) / avgMonthly
		if avgMonthly > 0 {
			seasonality[fmt.Sprintf("month_%d", month)] = multiplier
		} else {
			seasonality[fmt.Sprintf("month_%d", month)] = 1.0
		}
	}
	
	for day := 0; day < 7; day++ {
		count := weeklyCounts[day]
		multiplier := float64(count) / avgWeekly
		if avgWeekly > 0 {
			seasonality[fmt.Sprintf("weekday_%d", day)] = multiplier
		} else {
			seasonality[fmt.Sprintf("weekday_%d", day)] = 1.0
		}
	}
	
	return seasonality
}

// extractCommonSLA extracts most common SLA requirements
func (c *CustomerUsageAnalyzer) extractCommonSLA(usages []UsageRecord) map[string]interface{} {
	slaPatterns := make(map[string][]interface{})
	
	for _, usage := range usages {
		for key, value := range usage.SLARequirements {
			slaPatterns[key] = append(slaPatterns[key], value)
		}
	}
	
	commonSLA := make(map[string]interface{})
	for key, values := range slaPatterns {
		if len(values) == 0 {
			continue
		}
		
		// For numeric values, use median
		if len(values) > 0 {
			if _, ok := values[0].(float64); ok {
				// Convert to float64 slice and calculate median
				floatValues := make([]float64, len(values))
				for i, v := range values {
					if f, ok := v.(float64); ok {
						floatValues[i] = f
					}
				}
				commonSLA[key] = medianFloat64(floatValues)
			} else {
				// For non-numeric values, take most common
				valueCounts := make(map[interface{}]int)
				for _, val := range values {
					valueCounts[val]++
				}
				
				maxCount := 0
				var mostCommon interface{}
				for val, count := range valueCounts {
					if count > maxCount {
						maxCount = count
						mostCommon = val
					}
				}
				commonSLA[key] = mostCommon
			}
		}
	}
	
	return commonSLA
}

// PredictNextUsage predicts customer's next likely usage
func (c *CustomerUsageAnalyzer) PredictNextUsage(customerID string) []UsagePrediction {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	patterns := c.usagePatterns[customerID]
	if len(patterns) == 0 {
		return []UsagePrediction{}
	}
	
	var predictions []UsagePrediction
	currentTime := time.Now().Unix()
	
	for _, pattern := range patterns {
		// Calculate time since last usage of this pattern
		relevantHistory := make([]UsageRecord, 0)
		for _, usage := range c.customerHistory[customerID] {
			if usage.ResourceType == pattern.ResourceType {
				relevantHistory = append(relevantHistory, usage)
			}
		}
		
		if len(relevantHistory) == 0 {
			continue
		}
		
		lastUsageTime := relevantHistory[0].Timestamp
		for _, usage := range relevantHistory {
			if usage.Timestamp > lastUsageTime {
				lastUsageTime = usage.Timestamp
			}
		}
		
		timeSinceLast := float64(currentTime-lastUsageTime) / 86400 // days
		
		// Predict next usage based on frequency
		if timeSinceLast >= pattern.FrequencyDays*0.8 { // 80% of typical interval
			// Calculate prediction confidence
			confidence := minFloat(1.0, timeSinceLast/pattern.FrequencyDays)
			
			// Predict most likely start time
			currentHour := time.Now().Hour()
			nextStartTime := c.predictNextStartTime(pattern.TypicalStartTimes, currentHour)
			
			// Apply seasonality adjustments
			currentMonth := time.Now().Month()
			currentWeekday := time.Now().Weekday()
			
		seasonalMultiplier := (pattern.Seasonality[fmt.Sprintf("month_%d", int(currentMonth))] + pattern.Seasonality[fmt.Sprintf("weekday_%d", int(currentWeekday))]) / 2
		seasonalMultiplier := (pattern.Seasonality[fmt.Sprintf("month_%d", int(currentMonth))] + pattern.Seasonality[fmt.Sprintf("weekday_%d", int(currentWeekday))]) / 2
			prediction := UsagePrediction{
				CustomerID:            customerID,
				PredictedResourceType: pattern.ResourceType,
				PredictedQuantity:     adjustedQuantity,
				PredictedDuration:     pattern.TypicalDuration,
				PredictedStartTime:    currentTime + int64(nextStartTime*3600),
				Confidence:            confidence,
				PatternID:             pattern.PatternID,
				CostSensitivity:       pattern.CostSensitivity,
				SLARequirements:       pattern.SLARequirements,
			}
			
			predictions = append(predictions, prediction)
		}
	}
	
	// Sort by confidence
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].Confidence > predictions[j].Confidence
	})
	
	if len(predictions) > 5 {
		return predictions[:5] // Top 5 predictions
	}
	return predictions
}

// predictNextStartTime predicts next start time based on patterns
func (c *CustomerUsageAnalyzer) predictNextStartTime(typicalStartTimes []int, currentHour int) int {
	if len(typicalStartTimes) == 0 {
		return 9 // Default 9 AM
	}
	
	// Find next typical start time after current hour
	for _, startTime := range typicalStartTimes {
		if startTime > currentHour {
			return startTime // Next start time today
		}
	}
	
	// Return first start time tomorrow
	return typicalStartTimes[0]
}

// GetCustomerInsights returns comprehensive insights about customer usage
func (c *CustomerUsageAnalyzer) GetCustomerInsights(customerID string) *CustomerInsights {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	history := c.customerHistory[customerID]
	patterns := c.usagePatterns[customerID]
	
	if len(history) == 0 {
		return &CustomerInsights{
			CustomerID: customerID,
		}
	}
	
	// Calculate basic statistics
	totalUsageHours := 0.0
	totalSpend := 0.0
	for _, usage := range history {
		totalUsageHours += usage.DurationHours
		totalSpend += usage.TotalCost
	}
	
	avgCostPerHour := totalSpend / totalUsageHours
	if totalUsageHours == 0 {
		avgCostPerHour = 0
	}
	
	// Resource usage breakdown
	resourceUsage := make(map[string]ResourceUsage)
	for _, usage := range history {
		resType := usage.ResourceType
		if _, exists := resourceUsage[resType]; !exists {
			resourceUsage[resType] = ResourceUsage{}
		}
		
		ru := resourceUsage[resType]
		ru.Count++
		ru.TotalHours += usage.DurationHours
		ru.TotalCost += usage.TotalCost
		resourceUsage[resType] = ru
	}
	
	// Usage frequency analysis
	avgFrequencyDays := 0.0
	if len(history) >= 2 {
		timestamps := make([]int64, len(history))
		for i, usage := range history {
			timestamps[i] = usage.Timestamp
		}
		sort.Slice(timestamps, func(i, j int) bool {
			return timestamps[i] < timestamps[j]
		})
		
		intervals := make([]float64, len(timestamps)-1)
		for i := 1; i < len(timestamps); i++ {
			intervals[i-1] = float64(timestamps[i]-timestamps[i-1]) / 86400
		}
		
		if len(intervals) > 0 {
			avgFrequencyDays = medianFloat64(intervals)
		}
	}
	
	// Peak usage times
	hours := make([]int, len(history))
	for i, usage := range history {
		hours[i] = usage.StartTimeHour
	}
	peakHours := c.findPeakHours(hours)
	
	// Predict next usage
	nextUsagePredictions := c.PredictNextUsage(customerID)
	
	// Get overall cost sensitivity
	overallCostSensitivity := c.getOverallCostSensitivity(patterns)
	
	// Identify optimization opportunities
	optimizationOpportunities := c.identifyOptimizationOpportunities(customerID)
	
	return &CustomerInsights{
		CustomerID: customerID,
		UsageSummary: UsageSummary{
			TotalSessions:     len(history),
			TotalComputeHours: totalUsageHours,
			TotalSpend:        totalSpend,
			AvgCostPerHour:    avgCostPerHour,
			AvgFrequencyDays:  avgFrequencyDays,
		},
		ResourceBreakdown:         resourceUsage,
		UsagePatterns:             len(patterns),
		PeakUsageHours:            peakHours,
		CostSensitivity:           overallCostSensitivity,
		NextUsagePredictions:      nextUsagePredictions,
		OptimizationOpportunities: optimizationOpportunities,
	}
}

// findPeakHours finds peak usage hours
func (c *CustomerUsageAnalyzer) findPeakHours(hours []int) []int {
	if len(hours) == 0 {
		return []int{}
	}
	
	hourCounts := make(map[int]int)
	for _, hour := range hours {
		hourCounts[hour]++
	}
	
	// Find hours with maximum count
	maxCount := 0
	for _, count := range hourCounts {
		if count > maxCount {
			maxCount = count
		}
	}
	
	var peakHours []int
	for hour, count := range hourCounts {
		if count == maxCount {
			peakHours = append(peakHours, hour)
		}
	}
	
	sort.Ints(peakHours)
	return peakHours
}

// getOverallCostSensitivity gets overall cost sensitivity across all patterns
func (c *CustomerUsageAnalyzer) getOverallCostSensitivity(patterns []WorkloadPattern) float64 {
	if len(patterns) == 0 {
		return 0.5
	}
	
	totalSensitivity := 0.0
	for _, pattern := range patterns {
		totalSensitivity += pattern.CostSensitivity
	}
	
	return totalSensitivity / float64(len(patterns))
}

// identifyOptimizationOpportunities identifies cost optimization opportunities for customer
func (c *CustomerUsageAnalyzer) identifyOptimizationOpportunities(customerID string) []OptimizationOpportunity {
	c.mu.RLock()
	patterns := c.usagePatterns[customerID]
	c.mu.RUnlock()
	
	var opportunities []OptimizationOpportunity
	
	for _, pattern := range patterns {
		// Check if they could benefit from reservations
		if pattern.FrequencyDays <= 7 && pattern.TypicalDuration >= 4 {
			opportunities = append(opportunities, OptimizationOpportunity{
				Type:             "capacity_reservation",
				ResourceType:     pattern.ResourceType,
				PotentialSavings: "15-25%",
				Description:      fmt.Sprintf("Regular %s usage could benefit from capacity reservations", pattern.ResourceType),
			})
		}
		
		// Check if they could use cheaper providers
		if pattern.CostSensitivity < 0.3 { // Low cost sensitivity
			opportunities = append(opportunities, OptimizationOpportunity{
				Type:             "provider_optimization",
				ResourceType:     pattern.ResourceType,
				PotentialSavings: "20-40%",
				Description:      fmt.Sprintf("Could use lower-cost providers for %s workloads", pattern.ResourceType),
			})
		}
		
		// Check if they could optimize timing
		peakHours := pattern.TypicalStartTimes
		hasBusinessHours := false
		for _, hour := range peakHours {
			if hour >= 8 && hour <= 18 { // Business hours
				hasBusinessHours = true
				break
			}
		}
		
		if hasBusinessHours {
			opportunities = append(opportunities, OptimizationOpportunity{
				Type:             "timing_optimization",
				ResourceType:     pattern.ResourceType,
				PotentialSavings: "10-20%",
				Description:      fmt.Sprintf("Running %s workloads during off-peak hours could reduce costs", pattern.ResourceType),
			})
		}
	}
	
	return opportunities
}

// Helper functions
func getStringOrDefault(data map[string]interface{}, key, defaultValue string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return defaultValue
}

func medianInt(values []int) int {
	if len(values) == 0 {
		return 0
	}
	
	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)
	
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func medianFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

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
