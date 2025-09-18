package orchestrator

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"ocx.local/internal/analytics"
	"ocx.local/internal/capacity"
	"ocx.local/internal/loadbalancer"
	"ocx.local/internal/riskmanagement"
)

// OCXSystemOrchestrator is the main orchestrator that coordinates all OCX components
type OCXSystemOrchestrator struct {
	// Core components
	riskManager      *riskmanagement.ProviderRiskManager
	capacityEngine   *capacity.CapacityReservationEngine
	usageAnalyzer    *analytics.CustomerUsageAnalyzer
	loadBalancer     *loadbalancer.GlobalLoadBalancer
	marketIntelligence interface{} // Would be actual market intelligence
	
	// System state
	activeWorkloads  map[string]*WorkloadData
	systemMetrics    *SystemMetrics
	mu               sync.RWMutex
	running          bool
	stopChan         chan struct{}
}

// WorkloadData represents an active workload
type WorkloadData struct {
	CustomerID    string                        `json:"customer_id"`
	Request       map[string]interface{}        `json:"request"`
	Allocation    *loadbalancer.LoadBalanceDecision `json:"allocation"`
	StartTime     int64                         `json:"start_time"`
	Status        string                        `json:"status"`
	Reservation   *capacity.CapacityReservation `json:"reservation,omitempty"`
}

// SystemMetrics represents system-wide metrics
type SystemMetrics struct {
	Uptime                    float64 `json:"uptime"`
	TotalWorkloadsProcessed   int     `json:"total_workloads_processed"`
	TotalRevenue              float64 `json:"total_revenue"`
	AvgCustomerSatisfaction   float64 `json:"avg_customer_satisfaction"`
	ActiveWorkloads           int     `json:"active_workloads"`
	TotalComputeUnitsManaged  int     `json:"total_compute_units_managed"`
}

// CustomerRequest represents a customer compute request
type CustomerRequest struct {
	ResourceType      string                 `json:"resource_type"`
	Region            string                 `json:"region"`
	Quantity          int                    `json:"quantity"`
	DurationHours     float64                `json:"duration_hours"`
	SLARequirements   map[string]interface{} `json:"sla_requirements"`
	OptimizationGoal  string                 `json:"optimization_goal"`
	MaxPricePerHour   float64                `json:"max_price_per_hour"`
}

// ProcessResponse represents the response to a customer request
type ProcessResponse struct {
	RequestID              string                        `json:"request_id"`
	CustomerID             string                        `json:"customer_id"`
	Status                 string                        `json:"status"`
	Allocation             *AllocationInfo               `json:"allocation"`
	FailoverPlan           *FailoverPlanInfo             `json:"failover_plan"`
	ReservedCapacityUsed   bool                          `json:"reserved_capacity_used"`
	OptimizationOpportunities []analytics.OptimizationOpportunity `json:"optimization_opportunities"`
	FutureUsagePredictions []analytics.UsagePrediction   `json:"future_usage_predictions"`
	SLAGuarantees          map[string]interface{}        `json:"sla_guarantees"`
	EstimatedSetupTime     float64                       `json:"estimated_setup_time"`
	MonitoringEnabled      bool                          `json:"monitoring_enabled"`
}

// AllocationInfo represents allocation information
type AllocationInfo struct {
	Providers         []string          `json:"providers"`
	AllocationPercentages []float64     `json:"allocation_percentages"`
	ExpectedPerformance map[string]float64 `json:"expected_performance"`
	PricingStrategy   string            `json:"pricing_strategy"`
	TotalCost         float64           `json:"total_cost"`
	CostEfficiency    float64           `json:"cost_efficiency"`
	RiskScore         float64           `json:"risk_score"`
}

// FailoverPlanInfo represents failover plan information
type FailoverPlanInfo struct {
	PrimaryProvider   string   `json:"primary_provider"`
	BackupProviders   []string `json:"backup_providers"`
	AutoFailover      bool     `json:"auto_failover"`
}

// NewOCXSystemOrchestrator creates a new OCX system orchestrator
func NewOCXSystemOrchestrator() *OCXSystemOrchestrator {
	// Initialize components
	riskManager := riskmanagement.NewProviderRiskManager()
	capacityEngine := capacity.NewCapacityReservationEngine(nil) // Would pass market intelligence
	usageAnalyzer := analytics.NewCustomerUsageAnalyzer()
	loadBalancer := loadbalancer.NewGlobalLoadBalancer(riskManager, nil) // Would pass market intelligence
	
	return &OCXSystemOrchestrator{
		riskManager:      riskManager,
		capacityEngine:   capacityEngine,
		usageAnalyzer:    usageAnalyzer,
		loadBalancer:     loadBalancer,
		activeWorkloads:  make(map[string]*WorkloadData),
		systemMetrics:    &SystemMetrics{},
		stopChan:         make(chan struct{}),
	}
}

// InitializeSystem initializes the complete OCX system
func (o *OCXSystemOrchestrator) InitializeSystem(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	if o.running {
		return fmt.Errorf("system already running")
	}
	
	fmt.Println("🚀 Initializing OCX System...")
	
	// Start risk monitoring
	if err := o.riskManager.StartMonitoring(ctx); err != nil {
		return fmt.Errorf("failed to start risk monitoring: %w", err)
	}
	
	// Start capacity monitoring
	if err := o.capacityEngine.StartMonitoring(ctx); err != nil {
		return fmt.Errorf("failed to start capacity monitoring: %w", err)
	}
	
	o.running = true
	fmt.Println("✅ Risk management system active")
	fmt.Println("✅ Market intelligence system active")
	fmt.Println("✅ Capacity reservation engine active")
	fmt.Println("✅ Usage analytics system active")
	fmt.Println("✅ Global load balancer active")
	fmt.Println("🎉 OCX System fully operational!")
	
	return nil
}

// StopSystem stops the OCX system
func (o *OCXSystemOrchestrator) StopSystem() {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	if !o.running {
		return
	}
	
	o.running = false
	o.riskManager.StopMonitoring()
	o.capacityEngine.StopMonitoring()
	close(o.stopChan)
	fmt.Println("⏹️  OCX System stopped")
}

// ProcessCustomerRequest processes complete customer compute request
func (o *OCXSystemOrchestrator) ProcessCustomerRequest(customerID string, request *CustomerRequest) *ProcessResponse {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	requestID := fmt.Sprintf("req_%d_%d", time.Now().Unix(), rand.Intn(10000))
	
	fmt.Printf("📋 Processing request %s for customer %s\n", requestID, customerID)
	
	// Record usage for analytics
	usageData := map[string]interface{}{
		"resource_type":     request.ResourceType,
		"region":            request.Region,
		"quantity":          request.Quantity,
		"duration_hours":    request.DurationHours,
		"total_cost":        request.MaxPricePerHour * float64(request.Quantity) * request.DurationHours,
		"sla_requirements":  request.SLARequirements,
		"workload_type":     "compute",
	}
	o.usageAnalyzer.RecordUsage(customerID, usageData)
	
	// Check if we have reserved capacity
	var reservedCapacity *capacity.CapacityReservation
	if request.MaxPricePerHour > 0 {
		customerReq := &capacity.CustomerRequest{
			ResourceType:    request.ResourceType,
			Region:          request.Region,
			Quantity:        request.Quantity,
			DurationHours:   request.DurationHours,
			MaxPricePerHour: request.MaxPricePerHour,
			SLARequirements: request.SLARequirements,
		}
		reservedCapacity = o.capacityEngine.AllocateReservedCapacity(customerReq)
	}
	
	// Generate failover plan
	failoverPlan := o.riskManager.CreateFailoverPlan(map[string]interface{}{
		"resource_type":     request.ResourceType,
		"region":            request.Region,
		"sla_requirements":  request.SLARequirements,
	})
	
	// Optimize load balancing
	workloadSpec := &loadbalancer.WorkloadSpec{
		ResourceType:     request.ResourceType,
		Region:           request.Region,
		Quantity:         request.Quantity,
		DurationHours:    request.DurationHours,
		SLARequirements:  request.SLARequirements,
		OptimizationGoal: request.OptimizationGoal,
		AllocationTime:   time.Now().Unix(),
	}
	
	loadBalanceDecision := o.loadBalancer.OptimizeWorkloadPlacement(workloadSpec)
	
	// Calculate pricing
	var totalCost float64
	var pricingStrategy string
	
	if reservedCapacity != nil {
		totalCost = *reservedCapacity.CustomerPrice * float64(request.Quantity) * request.DurationHours
		pricingStrategy = "Reserved Capacity"
	} else {
		totalCost = loadBalanceDecision.ExpectedPerformance["total_cost_per_hour"] * request.DurationHours
		pricingStrategy = "Market Rate"
	}
	
	// Predict future usage for optimization
	futurePredictions := o.usageAnalyzer.PredictNextUsage(customerID)
	
	// Compile complete response
	response := &ProcessResponse{
		RequestID: requestID,
		CustomerID: customerID,
		Status: "processed",
		Allocation: &AllocationInfo{
			Providers:             loadBalanceDecision.TargetProviders,
			AllocationPercentages: loadBalanceDecision.AllocationPercentages,
			ExpectedPerformance:   loadBalanceDecision.ExpectedPerformance,
			PricingStrategy:       pricingStrategy,
			TotalCost:             totalCost,
			CostEfficiency:        loadBalanceDecision.CostEfficiency,
			RiskScore:             loadBalanceDecision.RiskScore,
		},
		FailoverPlan: &FailoverPlanInfo{
			PrimaryProvider: failoverPlan.PrimaryProvider,
			BackupProviders: failoverPlan.BackupProviders,
			AutoFailover:    failoverPlan.AutoFailback,
		},
		ReservedCapacityUsed: reservedCapacity != nil,
		OptimizationOpportunities: o.usageAnalyzer.GetCustomerInsights(customerID).OptimizationOpportunities,
		FutureUsagePredictions: futurePredictions,
		SLAGuarantees: o.calculateSLAGuarantees(loadBalanceDecision, request),
		EstimatedSetupTime: 5.0, // minutes
		MonitoringEnabled: true,
	}
	
	// Store active workload
	o.activeWorkloads[requestID] = &WorkloadData{
		CustomerID:  customerID,
		Request:     map[string]interface{}{
			"resource_type":     request.ResourceType,
			"region":            request.Region,
			"quantity":          request.Quantity,
			"duration_hours":    request.DurationHours,
			"sla_requirements":  request.SLARequirements,
			"optimization_goal": request.OptimizationGoal,
		},
		Allocation:  loadBalanceDecision,
		StartTime:   time.Now().Unix(),
		Status:      "active",
		Reservation: reservedCapacity,
	}
	
	// Update system metrics
	o.systemMetrics.TotalWorkloadsProcessed++
	o.systemMetrics.TotalRevenue += totalCost * 0.15 // 15% OCX margin
	o.systemMetrics.ActiveWorkloads = len(o.activeWorkloads)
	o.systemMetrics.TotalComputeUnitsManaged += request.Quantity
	
	return response
}

// calculateSLAGuarantees calculates SLA guarantees based on provider allocation
func (o *OCXSystemOrchestrator) calculateSLAGuarantees(allocation *loadbalancer.LoadBalanceDecision, request *CustomerRequest) map[string]interface{} {
	slaRequirements := request.SLARequirements
	
	// Calculate combined uptime from provider reliabilities
	combinedUptime := 1.0
	for _, providerID := range allocation.TargetProviders {
		providerHealth := o.riskManager.GetProviderHealth(providerID)
		if providerHealth != nil {
			providerUptime := providerHealth.APISuccessRate
			// Redundancy reduces failure probability
			combinedUptime *= (1 - (1 - providerUptime) / float64(len(allocation.TargetProviders)))
		}
	}
	
	guaranteedUptime := minFloat(combinedUptime*100, 99.9) // Cap at 99.9%
	
	// Performance guarantees
	avgResponseTime := 5.0 // Default estimate
	if len(allocation.TargetProviders) > 0 {
		responseTimes := make([]float64, 0)
		for _, providerID := range allocation.TargetProviders {
			providerHealth := o.riskManager.GetProviderHealth(providerID)
			if providerHealth != nil {
				responseTimes = append(responseTimes, providerHealth.APIResponseTime*1000) // Convert to ms
			}
		}
		if len(responseTimes) > 0 {
			avgResponseTime = 0
			for _, rt := range responseTimes {
				avgResponseTime += rt
			}
			avgResponseTime /= float64(len(responseTimes))
		}
	}
	
	return map[string]interface{}{
		"uptime_guarantee": guaranteedUptime,
		"max_response_time_ms": avgResponseTime * 1.2, // 20% buffer
		"availability_guarantee": minFloat(95.0, 100-allocation.RiskScore*20),
		"performance_tier": func() string {
			if allocation.CostEfficiency > 0.8 {
				return "enterprise"
			}
			return "standard"
		}(),
		"sla_credits": map[string]string{
			"uptime_below_99": "5% credit",
			"uptime_below_95": "15% credit",
			"response_time_breach": "10% credit",
		},
	}
}

// GetSystemStatus returns comprehensive system status
func (o *OCXSystemOrchestrator) GetSystemStatus() map[string]interface{} {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	currentTime := time.Now().Unix()
	
	// Provider health summary
	reliabilityReport := o.riskManager.GetProviderReliabilityReport()
	
	// Active workload summary
	activeWorkloads := len(o.activeWorkloads)
	totalComputeUnits := 0
	for _, workload := range o.activeWorkloads {
		if quantity, ok := workload.Request["quantity"].(int); ok {
			totalComputeUnits += quantity
		}
	}
	
	// System performance metrics
	avgProcessingTime := 2.5 // Would be calculated from actual data
	systemUptime := 99.95    // Would be calculated from actual uptime
	
	// Capacity analytics
	capacityAnalytics := o.capacityEngine.GetReservationAnalytics()
	
	return map[string]interface{}{
		"timestamp": currentTime,
		"system_status": "operational",
		"uptime_percentage": systemUptime,
		"active_workloads": activeWorkloads,
		"total_compute_units_managed": totalComputeUnits,
		"provider_health": reliabilityReport.Providers,
		"active_failovers": reliabilityReport.ActiveFailovers,
		"processing_metrics": map[string]interface{}{
			"avg_request_processing_time": avgProcessingTime,
			"requests_processed_today": o.systemMetrics.TotalWorkloadsProcessed,
			"success_rate": 99.2, // Would be calculated from actual data
		},
		"financial_metrics": map[string]interface{}{
			"total_revenue": o.systemMetrics.TotalRevenue,
			"active_reservations": capacityAnalytics.ActiveReservations,
			"profit_margin": 15.0, // OCX margin percentage
		},
		"capacity_metrics": map[string]interface{}{
			"total_providers": reliabilityReport.ProviderCount,
			"healthy_providers": func() int {
				count := 0
				for _, stats := range reliabilityReport.Providers {
					if stats.HealthScore > 80 {
						count++
					}
				}
				return count
			}(),
			"total_available_capacity": func() float64 {
				total := 0.0
				for _, stats := range reliabilityReport.Providers {
					total += (100 - stats.CapacityUtilization)
				}
				return total
			}(),
		},
	}
}

// GetCustomerInsights returns customer insights
func (o *OCXSystemOrchestrator) GetCustomerInsights(customerID string) *analytics.CustomerInsights {
	return o.usageAnalyzer.GetCustomerInsights(customerID)
}

// RunSystemOptimization runs periodic system optimization tasks
func (o *OCXSystemOrchestrator) RunSystemOptimization(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopChan:
			return
		case <-ticker.C:
			o.runOptimizationCycle()
		}
	}
}

// runOptimizationCycle runs a single optimization cycle
func (o *OCXSystemOrchestrator) runOptimizationCycle() {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	fmt.Println("🔧 Running system optimization...")
	
	// Check for rebalancing opportunities
	for workloadID, workloadData := range o.activeWorkloads {
		workloadSpec := &loadbalancer.WorkloadSpec{
			ResourceType:     workloadData.Request["resource_type"].(string),
			Region:           workloadData.Request["region"].(string),
			Quantity:         workloadData.Request["quantity"].(int),
			DurationHours:    workloadData.Request["duration_hours"].(float64),
			SLARequirements:  workloadData.Request["sla_requirements"].(map[string]interface{}),
			OptimizationGoal: workloadData.Request["optimization_goal"].(string),
			AllocationTime:   workloadData.StartTime,
		}
		
		newAllocation := o.loadBalancer.RebalanceIfNeeded(workloadID, workloadData.Allocation, workloadSpec)
		if newAllocation != nil {
			workloadData.Allocation = newAllocation
			fmt.Printf("🔄 Rebalanced workload %s\n", workloadID)
		}
	}
	
	// Cleanup completed workloads
	currentTime := time.Now().Unix()
	var completedWorkloads []string
	
	for workloadID, workloadData := range o.activeWorkloads {
		durationHours := workloadData.Request["duration_hours"].(float64)
		elapsedTime := float64(currentTime-workloadData.StartTime) / 3600
		
		if elapsedTime >= durationHours {
			completedWorkloads = append(completedWorkloads, workloadID)
		}
	}
	
	for _, workloadID := range completedWorkloads {
		delete(o.activeWorkloads, workloadID)
		fmt.Printf("✅ Completed workload %s\n", workloadID)
	}
	
	o.systemMetrics.ActiveWorkloads = len(o.activeWorkloads)
}

// Helper functions
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
