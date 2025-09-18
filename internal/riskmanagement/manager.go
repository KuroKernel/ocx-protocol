package riskmanagement

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// ProviderRiskManager manages provider reliability and automatic failover
type ProviderRiskManager struct {
	providerHealth    map[string]*ProviderHealthMetrics
	riskProfiles      map[string]RiskLevel
	failureHistory    map[string][]FailureRecord
	activeFailovers   map[string]*FailoverRecord
	monitoringTasks   map[string]context.CancelFunc
	mu                sync.RWMutex
	stopChan          chan struct{}
	running           bool
}

// NewProviderRiskManager creates a new provider risk manager
func NewProviderRiskManager() *ProviderRiskManager {
	manager := &ProviderRiskManager{
		providerHealth:  make(map[string]*ProviderHealthMetrics),
		riskProfiles:    make(map[string]RiskLevel),
		failureHistory:  make(map[string][]FailureRecord),
		activeFailovers: make(map[string]*FailoverRecord),
		monitoringTasks: make(map[string]context.CancelFunc),
		stopChan:        make(chan struct{}),
	}
	
	manager.initializeProviderProfiles()
	return manager
}

// initializeProviderProfiles initializes baseline risk profiles for known providers
func (p *ProviderRiskManager) initializeProviderProfiles() {
	baselineProfiles := map[string]RiskLevel{
		"aws":       Minimal,
		"gcp":       Minimal,
		"azure":     Low,
		"runpod":    Medium,
		"lambdalabs": Medium,
		"vastai":    High,
		"coreweave": Low,
	}
	
	for provider, riskLevel := range baselineProfiles {
		p.riskProfiles[provider] = riskLevel
		
		// Generate initial health metrics
		baseReliability := map[RiskLevel]float64{
			Minimal:  0.999,
			Low:      0.995,
			Medium:   0.985,
			High:     0.975,
			Critical: 0.950,
		}[riskLevel]
		
		p.providerHealth[provider] = &ProviderHealthMetrics{
			ProviderID:           provider,
			Timestamp:            time.Now().Unix(),
			APIResponseTime:      rand.Float64()*(2.0-0.1) + 0.1,
			APISuccessRate:       baseReliability + rand.Float64()*0.02 - 0.01,
			InstanceFailureRate:  1 - baseReliability + rand.Float64()*0.01 - 0.005,
			NetworkLatency:       rand.Float64()*(50-5) + 5,
			CapacityAvailability: rand.Float64()*(0.95-0.7) + 0.7,
			BillingIssues:        rand.Intn(4),
			MaintenanceWindows:   []TimeWindow{},
			CustomerComplaints:   rand.Intn(6),
		}
	}
}

// StartMonitoring starts continuous provider health monitoring
func (p *ProviderRiskManager) StartMonitoring(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.running {
		return fmt.Errorf("monitoring already running")
	}
	
	p.running = true
	fmt.Println("🔄 Starting provider health monitoring...")
	
	// Start monitoring for each provider
	for providerID := range p.providerHealth {
		providerCtx, cancel := context.WithCancel(ctx)
		p.monitoringTasks[providerID] = cancel
		
		go p.monitorProviderHealth(providerCtx, providerID)
	}
	
	return nil
}

// StopMonitoring stops provider health monitoring
func (p *ProviderRiskManager) StopMonitoring() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.running {
		return
	}
	
	p.running = false
	
	// Cancel all monitoring tasks
	for _, cancel := range p.monitoringTasks {
		cancel()
	}
	
	close(p.stopChan)
	fmt.Println("⏹️  Provider health monitoring stopped")
}

// monitorProviderHealth monitors individual provider health continuously
func (p *ProviderRiskManager) monitorProviderHealth(ctx context.Context, providerID string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.collectProviderMetrics(providerID); err != nil {
				fmt.Printf("⚠️ Health monitoring error for %s: %v\n", providerID, err)
			}
		}
	}
}

// collectProviderMetrics collects real-time health metrics for provider
func (p *ProviderRiskManager) collectProviderMetrics(providerID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	baseMetrics := p.providerHealth[providerID]
	if baseMetrics == nil {
		return fmt.Errorf("provider %s not found", providerID)
	}
	
	// Simulate metric collection with realistic drift
	apiResponseTime := maxFloat(0.05, baseMetrics.APIResponseTime+rand.NormFloat64()*0.1)
	
	// Simulate occasional API failures
	apiSuccessRate := baseMetrics.APISuccessRate
	if rand.Float64() < 0.01 { // 1% chance of degradation
		apiSuccessRate *= rand.Float64()*(0.95-0.8) + 0.8
	}
	
	// Simulate network latency variation
	networkLatency := maxFloat(1, baseMetrics.NetworkLatency+rand.NormFloat64()*5)
	
	// Capacity availability changes
	capacityTrend := rand.Float64()*(1.05-0.95) + 0.95
	capacityAvailability := minFloat(1.0, maxFloat(0.1, baseMetrics.CapacityAvailability*capacityTrend))
	
	// Update metrics
	p.providerHealth[providerID] = &ProviderHealthMetrics{
		ProviderID:           providerID,
		Timestamp:            time.Now().Unix(),
		APIResponseTime:      apiResponseTime,
		APISuccessRate:       apiSuccessRate,
		InstanceFailureRate:  maxFloat(0, 1-apiSuccessRate+rand.NormFloat64()*0.01),
		NetworkLatency:       networkLatency,
		CapacityAvailability: capacityAvailability,
		BillingIssues:        baseMetrics.BillingIssues + func() int { if rand.Float64() < 0.001 { return 1 } else { return 0 } }(),
		BillingIssues:        baseMetrics.BillingIssues + func() int { if rand.Float64() < 0.001 { return 1 } else { return 0 } }(),
		BillingIssues:        baseMetrics.BillingIssues + func() int { if rand.Float64() < 0.001 { return 1 } else { return 0 } }(),
	// Update risk profile based on recent performance
		CustomerComplaints:   baseMetrics.CustomerComplaints + func() int { if rand.Float64() < 0.002 { return 1 } else { return 0 } }(),
	
	// Check for failover triggers
	p.checkFailoverConditions(providerID, p.providerHealth[providerID]),
	
}
	return nil
}

// updateRiskProfile updates provider risk profile based on recent performance
func (p *ProviderRiskManager) updateRiskProfile(providerID string, metrics *ProviderHealthMetrics) {
	healthScore := metrics.CalculateHealthScore()
	
	var newRisk RiskLevel
	if healthScore >= 95 {
		newRisk = Minimal
	} else if healthScore >= 90 {
		newRisk = Low
	} else if healthScore >= 80 {
		newRisk = Medium
	} else if healthScore >= 70 {
		newRisk = High
	} else {
		newRisk = Critical
	}
	
	oldRisk := p.riskProfiles[providerID]
	if newRisk != oldRisk {
		p.riskProfiles[providerID] = newRisk
		fmt.Printf("🔄 Risk profile update: %s %s → %s\n", providerID, oldRisk, newRisk)
	}
}

// checkFailoverConditions checks if failover should be triggered
func (p *ProviderRiskManager) checkFailoverConditions(providerID string, metrics *ProviderHealthMetrics) {
	// Critical conditions that trigger immediate failover
	criticalConditions := []bool{
		metrics.APISuccessRate < 0.95,        // <95% API success
		metrics.InstanceFailureRate > 0.1,    // >10% instance failures
		metrics.CapacityAvailability < 0.2,   // <20% capacity available
		metrics.APIResponseTime > 10.0,       // >10s API response time
	}
	
	if any(criticalConditions) {
		p.triggerFailover(providerID, metrics)
	}
}

// any checks if any condition is true
func any(conditions []bool) bool {
	for _, condition := range conditions {
		if condition {
			return true
		}
	}
	return false
}

// triggerFailover executes failover for degraded provider
func (p *ProviderRiskManager) triggerFailover(providerID string, metrics *ProviderHealthMetrics) {
	if _, exists := p.activeFailovers[providerID]; exists {
		return // Already in failover
	}
	
	fmt.Printf("🚨 TRIGGERING FAILOVER for %s\n", providerID)
	
	// Find suitable backup providers
	backupProviders := p.findBackupProviders(providerID)
	
	failoverRecord := &FailoverRecord{
		PrimaryProvider:   providerID,
		BackupProviders:   backupProviders,
		TriggerTime:       time.Now().Unix(),
		TriggerReason:     fmt.Sprintf("Health score: %.1f", metrics.CalculateHealthScore()),
		AffectedWorkloads: []string{}, // Would be populated with actual workloads
		Status:            "active",
	}
	
	p.activeFailovers[providerID] = failoverRecord
	
	// Record failure in history
	failureRecord := FailureRecord{
		Timestamp:   time.Now().Unix(),
		FailureType: p.classifyFailureType(metrics),
		HealthScore: metrics.CalculateHealthScore(),
		Metrics:     *metrics,
	}
	
	p.failureHistory[providerID] = append(p.failureHistory[providerID], failureRecord)
	
	// Keep only recent failure history
	if len(p.failureHistory[providerID]) > 100 {
		p.failureHistory[providerID] = p.failureHistory[providerID][len(p.failureHistory[providerID])-100:]
	}
}

// findBackupProviders finds suitable backup providers for failover
func (p *ProviderRiskManager) findBackupProviders(failedProvider string) []string {
	// Get all healthy providers except the failed one
	var healthyProviders []string
	for providerID, risk := range p.riskProfiles {
		if providerID != failedProvider && (risk == Minimal || risk == Low) {
			healthyProviders = append(healthyProviders, providerID)
		}
	}
	
	// Sort by health score
	sort.Slice(healthyProviders, func(i, j int) bool {
		scoreI := p.providerHealth[healthyProviders[i]].CalculateHealthScore()
		scoreJ := p.providerHealth[healthyProviders[j]].CalculateHealthScore()
		return scoreI > scoreJ
	})
	
	// Return top 3 backup providers
	if len(healthyProviders) > 3 {
		return healthyProviders[:3]
	}
	return healthyProviders
}

// classifyFailureType classifies the type of failure based on metrics
func (p *ProviderRiskManager) classifyFailureType(metrics *ProviderHealthMetrics) FailureType {
	if metrics.APISuccessRate < 0.8 {
		return API
	} else if metrics.InstanceFailureRate > 0.2 {
		return Hardware
	} else if metrics.NetworkLatency > 100 {
		return Network
	} else if metrics.CapacityAvailability < 0.1 {
		return Capacity
	}
	return Unknown
}

// CreateFailoverPlan creates failover plan for specific workload
func (p *ProviderRiskManager) CreateFailoverPlan(workloadSpec map[string]interface{}) *FailoverPlan {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	resourceType := workloadSpec["resource_type"].(string)
	region := workloadSpec["region"].(string)
	slaRequirements := workloadSpec["sla_requirements"].(map[string]interface{})
	
	// Select primary provider based on best combination of price and reliability
	var suitableProviders []string
	for providerID, risk := range p.riskProfiles {
		if risk == Minimal || risk == Low || risk == Medium {
			suitableProviders = append(suitableProviders, providerID)
		}
	}
	
	if len(suitableProviders) == 0 {
		// Fallback to all providers
		for providerID := range p.riskProfiles {
			suitableProviders = append(suitableProviders, providerID)
		}
	}
	
	// Score providers (health score + cost efficiency)
	type providerScore struct {
		score      float64
		providerID string
	}
	
	var providerScores []providerScore
	for _, providerID := range suitableProviders {
		healthScore := p.providerHealth[providerID].CalculateHealthScore()
		// Simulate cost factor (would come from pricing engine)
		costFactor := rand.Float64()*(1.3-0.7) + 0.7
		combinedScore := healthScore * (2.0 - costFactor) // Higher is better
		providerScores = append(providerScores, providerScore{combinedScore, providerID})
	}
	
	// Sort by combined score
	sort.Slice(providerScores, func(i, j int) bool {
		return providerScores[i].score > providerScores[j].score
	})
	
	primaryProvider := providerScores[0].providerID
	var backupProviders []string
	for i := 1; i < minFloat(4, len(providerScores)); i++ {
		backupProviders = append(backupProviders, providerScores[i].providerID)
	}
	
	// Define trigger conditions based on SLA requirements
	minUptime := 99.0
	if uptime, ok := slaRequirements["uptime"].(float64); ok {
		minUptime = uptime
	}
	
	triggerConditions := map[string]interface{}{
		"min_api_success_rate":     minUptime / 100,
		"max_instance_failure_rate": 1 - minUptime/100,
		"max_api_response_time":    5.0,
		"min_capacity_availability": 0.2,
	}
	
	return &FailoverPlan{
		PrimaryProvider:       primaryProvider,
		BackupProviders:       backupProviders,
		TriggerConditions:     triggerConditions,
		FailoverLatencySeconds: 30.0,
		CostMultiplier:        1.2, // 20% cost increase during failover
		AutoFailback:          true,
	}
}

// GetProviderReliabilityReport generates comprehensive provider reliability report
func (p *ProviderRiskManager) GetProviderReliabilityReport() *ProviderReliabilityReport {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	report := &ProviderReliabilityReport{
		Timestamp:       time.Now().Unix(),
		ProviderCount:   len(p.providerHealth),
		ActiveFailovers: len(p.activeFailovers),
		Providers:       make(map[string]ProviderStats),
	}
	
	for providerID, metrics := range p.providerHealth {
		healthScore := metrics.CalculateHealthScore()
		riskLevel := p.riskProfiles[providerID]
		
		// Calculate failure statistics
		recentFailures := 0
		cutoffTime := time.Now().Unix() - 86400 // Last 24 hours
		for _, failure := range p.failureHistory[providerID] {
			if failure.Timestamp > cutoffTime {
				recentFailures++
			}
		}
		
		mtbf := p.calculateMTBF(providerID)
		
		report.Providers[providerID] = ProviderStats{
			HealthScore:         healthScore,
			RiskLevel:           string(riskLevel),
			APIResponseTime:     metrics.APIResponseTime,
			UptimePercentage:    metrics.APISuccessRate * 100,
			CapacityUtilization: (1 - metrics.CapacityAvailability) * 100,
			RecentFailures:      recentFailures,
			MTBFHours:           mtbf,
			IsInFailover:        p.activeFailovers[providerID] != nil,
			LastUpdated:         metrics.Timestamp,
		}
	}
	
	return report
}

// calculateMTBF calculates Mean Time Between Failures for provider
func (p *ProviderRiskManager) calculateMTBF(providerID string) float64 {
	failures := p.failureHistory[providerID]
	if len(failures) < 2 {
		return 0 // No failures or only one failure
	}
	
	// Calculate time intervals between failures
	var intervals []float64
	for i := 1; i < len(failures); i++ {
		interval := float64(failures[i].Timestamp - failures[i-1].Timestamp)
		intervals = append(intervals, interval)
	}
	
	if len(intervals) == 0 {
		return 0
	}
	
	// Calculate average interval
	totalInterval := 0.0
	for _, interval := range intervals {
		totalInterval += interval
	}
	
	return totalInterval / float64(len(intervals)) / 3600 // Convert to hours
}

// GetProviderHealth returns current health metrics for a provider
func (p *ProviderRiskManager) GetProviderHealth(providerID string) *ProviderHealthMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return p.providerHealth[providerID]
}

// GetRiskLevel returns current risk level for a provider
func (p *ProviderRiskManager) GetRiskLevel(providerID string) RiskLevel {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return p.riskProfiles[providerID]
}

// IsProviderHealthy checks if provider is currently healthy
func (p *ProviderRiskManager) IsProviderHealthy(providerID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	health := p.providerHealth[providerID]
	if health == nil {
		return false
	}
	
	return health.CalculateHealthScore() > 70
}
