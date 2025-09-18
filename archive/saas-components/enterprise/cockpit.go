package enterprise

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// OCX Protocol Standard - Enterprise Cockpit Reference Implementation
// This is the "Switzerland Play" - neutral protocol that everyone depends on

// ResourceType represents different types of compute resources
type ResourceType string

const (
	ResourceTypeA100     ResourceType = "A100"
	ResourceTypeH100     ResourceType = "H100"
	ResourceTypeV100     ResourceType = "V100"
	ResourceTypeTPUV4    ResourceType = "TPU_V4"
	ResourceTypeTPUV5    ResourceType = "TPU_V5"
	ResourceTypeMI300X   ResourceType = "MI300X"
	ResourceTypeCPUIntel ResourceType = "CPU_INTEL"
	ResourceTypeCPUAMD   ResourceType = "CPU_AMD"
)

// Region represents geographic regions for compute resources
type Region string

const (
	RegionUSEast      Region = "us-east"
	RegionUSWest      Region = "us-west"
	RegionEUWest      Region = "eu-west"
	RegionEUCentral   Region = "eu-central"
	RegionAsiaPacific Region = "asia-pacific"
	RegionAsiaSingapore Region = "asia-singapore"
	RegionAsiaTokyo   Region = "asia-tokyo"
	RegionMiddleEast  Region = "middle-east"
	RegionLATAM       Region = "latam"
)

// ReservationStatus represents the lifecycle status of a compute reservation
type ReservationStatus string

const (
	ReservationStatusDiscovering  ReservationStatus = "discovering"
	ReservationStatusAvailable    ReservationStatus = "available"
	ReservationStatusReserved     ReservationStatus = "reserved"
	ReservationStatusProvisioning ReservationStatus = "provisioning"
	ReservationStatusActive       ReservationStatus = "active"
	ReservationStatusCompleting   ReservationStatus = "completing"
	ReservationStatusCompleted    ReservationStatus = "completed"
	ReservationStatusFailed       ReservationStatus = "failed"
	ReservationStatusCancelled    ReservationStatus = "cancelled"
)

// BillingStatus represents the billing status of a reservation
type BillingStatus string

const (
	BillingStatusPending   BillingStatus = "pending"
	BillingStatusProcessing BillingStatus = "processing"
	BillingStatusPaid      BillingStatus = "paid"
	BillingStatusOverdue   BillingStatus = "overdue"
	BillingStatusDisputed  BillingStatus = "disputed"
	BillingStatusRefunded  BillingStatus = "refunded"
)

// SLAStatus represents SLA compliance status
type SLAStatus string

const (
	SLAStatusMeeting     SLAStatus = "meeting"
	SLAStatusAtRisk      SLAStatus = "at_risk"
	SLAStatusBreach      SLAStatus = "breach"
	SLAStatusRemediation SLAStatus = "remediation"
)

// ResourceSpec defines compute resource specifications
type ResourceSpec struct {
	ResourceType ResourceType `json:"resource_type"`
	Quantity     int          `json:"quantity"`
	MemoryGB     *int         `json:"memory_gb,omitempty"`
	StorageGB    *int         `json:"storage_gb,omitempty"`
	NetworkGbps  *int         `json:"network_gbps,omitempty"`
	Interconnect *string      `json:"interconnect,omitempty"`
}

// SLARequirement defines service level agreement requirements
type SLARequirement struct {
	UptimePercentage     float64 `json:"uptime_percentage"`
	MaxResponseTimeMs    float64 `json:"max_response_time_ms"`
	MaxSetupTimeMinutes  float64 `json:"max_setup_time_minutes"`
	AvailabilityGuarantee bool   `json:"availability_guarantee"`
	PerformanceBaseline  float64 `json:"performance_baseline"`
}

// DiscoveryFilter defines filters for resource discovery
type DiscoveryFilter struct {
	ResourceTypes        []ResourceType `json:"resource_types,omitempty"`
	Regions              []Region       `json:"regions,omitempty"`
	MinQuantity          *int           `json:"min_quantity,omitempty"`
	MaxQuantity          *int           `json:"max_quantity,omitempty"`
	MaxPricePerHour      *float64       `json:"max_price_per_hour,omitempty"`
	MinAvailability      *float64       `json:"min_availability,omitempty"`
	ProviderPreferences  []string       `json:"provider_preferences,omitempty"`
	ExcludeProviders     []string       `json:"exclude_providers,omitempty"`
	MinMemoryGB          *int           `json:"min_memory_gb,omitempty"`
	InterconnectRequired *string        `json:"interconnect_required,omitempty"`
}

// AvailableResource represents a discovered compute resource
type AvailableResource struct {
	ResourceID              string      `json:"resource_id"`
	ProviderID              string      `json:"provider_id"`
	ResourceSpec            ResourceSpec `json:"resource_spec"`
	Region                  Region      `json:"region"`
	PricePerHour            float64     `json:"price_per_hour"`
	AvailabilityPercentage  float64     `json:"availability_percentage"`
	EstimatedSetupTimeMinutes float64   `json:"estimated_setup_time_minutes"`
	SupportedFrameworks     []string    `json:"supported_frameworks"`
	LastUpdated             time.Time   `json:"last_updated"`
}

// TotalHourlyCost returns the total hourly cost for this resource
func (ar *AvailableResource) TotalHourlyCost() float64 {
	return ar.PricePerHour * float64(ar.ResourceSpec.Quantity)
}

// ComputeReservation represents a complete compute reservation
type ComputeReservation struct {
	ReservationID        string                `json:"reservation_id"`
	CustomerID           string                `json:"customer_id"`
	ResourceSpec         ResourceSpec          `json:"resource_spec"`
	Region               Region                `json:"region"`
	DurationHours        float64               `json:"duration_hours"`
	SLARequirements      SLARequirement        `json:"sla_requirements"`
	
	// Discovery results
	DiscoveredResources  []AvailableResource   `json:"discovered_resources,omitempty"`
	SelectedResource     *AvailableResource    `json:"selected_resource,omitempty"`
	
	// Reservation lifecycle
	Status               ReservationStatus     `json:"status"`
	CreatedAt            time.Time             `json:"created_at"`
	ReservedAt           *time.Time            `json:"reserved_at,omitempty"`
	StartedAt            *time.Time            `json:"started_at,omitempty"`
	CompletedAt          *time.Time            `json:"completed_at,omitempty"`
	
	// Cost and billing
	EstimatedCost        float64               `json:"estimated_cost"`
	ActualCost           float64               `json:"actual_cost"`
	EscrowAmount         float64               `json:"escrow_amount"`
	
	// Monitoring and SLA
	SLAStatus            SLAStatus             `json:"sla_status"`
	PerformanceMetrics   map[string]interface{} `json:"performance_metrics"`
	SLAViolations        []SLAViolation        `json:"sla_violations,omitempty"`
	
	// Connection details
	ConnectionInfo       map[string]interface{} `json:"connection_info,omitempty"`
}

// SLAViolation represents an SLA violation
type SLAViolation struct {
	Type      string    `json:"type"`
	Expected  float64   `json:"expected"`
	Actual    float64   `json:"actual"`
	Timestamp time.Time `json:"timestamp"`
}

// BillingRecord represents a billing record
type BillingRecord struct {
	BillingID         string       `json:"billing_id"`
	ReservationID     string       `json:"reservation_id"`
	CustomerID        string       `json:"customer_id"`
	ResourceHoursUsed float64      `json:"resource_hours_used"`
	BaseCost          float64      `json:"base_cost"`
	SLACredits         float64     `json:"sla_credits"`
	AdditionalFees     float64     `json:"additional_fees"`
	TotalCost          float64     `json:"total_cost"`
	Status             BillingStatus `json:"status"`
	InvoiceDate        *time.Time  `json:"invoice_date,omitempty"`
	DueDate            *time.Time  `json:"due_date,omitempty"`
	PaidDate           *time.Time  `json:"paid_date,omitempty"`
}

// MonitoringMetrics represents real-time monitoring data
type MonitoringMetrics struct {
	Timestamp           time.Time `json:"timestamp"`
	ReservationID       string    `json:"reservation_id"`
	CPUUtilization      float64   `json:"cpu_utilization"`
	MemoryUtilization   float64   `json:"memory_utilization"`
	GPUUtilization      float64   `json:"gpu_utilization"`
	NetworkIOMbps       float64   `json:"network_io_mbps"`
	StorageIOMbps       float64   `json:"storage_io_mbps"`
	ResponseTimeMs      float64   `json:"response_time_ms"`
	ThroughputOpsPerSec float64   `json:"throughput_ops_per_sec"`
	ErrorRate           float64   `json:"error_rate"`
	UptimePercentage    float64   `json:"uptime_percentage"`
	SLAComplianceScore  float64   `json:"sla_compliance_score"`
}

// ProviderConnector interface for different cloud providers
type ProviderConnector interface {
	DiscoverResources(spec ResourceSpec, region Region, duration float64, filters *DiscoveryFilter) ([]AvailableResource, error)
	ReserveResource(resourceID string, duration float64) error
	ReleaseResource(resourceID string) error
	GetResourceStatus(resourceID string) (string, error)
}

// ReservationEngine handles the complete reservation lifecycle
type ReservationEngine struct {
	Reservations map[string]*ComputeReservation `json:"reservations"`
	Discovery    *ResourceDiscovery             `json:"-"`
	mu           sync.RWMutex                   `json:"-"`
}

// BillingEngine handles billing and invoicing
type BillingEngine struct {
	BillingRecords map[string]*BillingRecord `json:"billing_records"`
	mu             sync.RWMutex               `json:"-"`
}

// EnterpriseCockpit is the main OCX Protocol reference implementation
type EnterpriseCockpit struct {
	ReservationEngine *ReservationEngine `json:"-"`
	BillingEngine     *BillingEngine     `json:"-"`
	Customers         map[string]map[string]interface{} `json:"customers"`
	mu                sync.RWMutex       `json:"-"`
}

// NewEnterpriseCockpit creates a new Enterprise Cockpit instance
func NewEnterpriseCockpit() *EnterpriseCockpit {
	return &EnterpriseCockpit{
		ReservationEngine: &ReservationEngine{
			Reservations: make(map[string]*ComputeReservation),
			Discovery:    NewResourceDiscovery(),
		},
		BillingEngine: &BillingEngine{
			BillingRecords: make(map[string]*BillingRecord),
		},
		Customers: make(map[string]map[string]interface{}),
	}
}

// Reserve is the primary enterprise API for reserving compute resources
func (ec *EnterpriseCockpit) Reserve(ctx context.Context, customerID string, quantity int, 
	resourceType string, duration string, region string, options map[string]interface{}) (*ComputeReservation, error) {
	
	// Parse and validate inputs
	rt, err := parseResourceType(resourceType)
	if err != nil {
		return nil, fmt.Errorf("invalid resource type: %w", err)
	}
	
	reg, err := parseRegion(region)
	if err != nil {
		return nil, fmt.Errorf("invalid region: %w", err)
	}
	
	durationHours, err := parseDuration(duration)
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %w", err)
	}
	
	// Create resource specification
	resourceSpec := ResourceSpec{
		ResourceType: rt,
		Quantity:     quantity,
	}
	
	// Set defaults based on resource type
	setResourceDefaults(&resourceSpec)
	
	// Create SLA requirements
	slaRequirements := SLARequirement{
		UptimePercentage:     99.9,
		MaxResponseTimeMs:    10.0,
		MaxSetupTimeMinutes:  15.0,
		AvailabilityGuarantee: true,
		PerformanceBaseline:  0.95,
	}
	
	// Apply custom SLA if provided
	if sla, ok := options["sla"].(map[string]interface{}); ok {
		if uptime, ok := sla["uptime"].(float64); ok {
			slaRequirements.UptimePercentage = uptime
		}
		if responseTime, ok := sla["max_response_time"].(float64); ok {
			slaRequirements.MaxResponseTimeMs = responseTime
		}
		if setupTime, ok := sla["max_setup_time"].(float64); ok {
			slaRequirements.MaxSetupTimeMinutes = setupTime
		}
	}
	
	// Create reservation
	reservationID := fmt.Sprintf("rsv_%s", uuid.New().String()[:12])
	reservation := &ComputeReservation{
		ReservationID:   reservationID,
		CustomerID:      customerID,
		ResourceSpec:    resourceSpec,
		Region:          reg,
		DurationHours:   durationHours,
		SLARequirements: slaRequirements,
		Status:          ReservationStatusDiscovering,
		CreatedAt:       time.Now(),
		PerformanceMetrics: make(map[string]interface{}),
	}
	
	// Store reservation
	ec.ReservationEngine.mu.Lock()
	ec.ReservationEngine.Reservations[reservationID] = reservation
	ec.ReservationEngine.mu.Unlock()
	
	// Start reservation process asynchronously
	go ec.processReservation(ctx, reservation)
	
	return reservation, nil
}

// processReservation handles the complete reservation lifecycle
func (ec *EnterpriseCockpit) processReservation(ctx context.Context, reservation *ComputeReservation) {
	defer func() {
		if r := recover(); r != nil {
			reservation.Status = ReservationStatusFailed
			fmt.Printf("Reservation %s failed with panic: %v\n", reservation.ReservationID, r)
		}
	}()
	
	// Phase 1: Discovery
	reservation.Status = ReservationStatusDiscovering
	fmt.Printf("🔍 Phase 1: Discovering resources for %s\n", reservation.ReservationID)
	
	discovered, err := ec.ReservationEngine.Discovery.Discover(ctx, reservation.ResourceSpec, 
		reservation.Region, reservation.DurationHours, nil)
	if err != nil {
		reservation.Status = ReservationStatusFailed
		fmt.Printf("❌ Discovery failed for %s: %v\n", reservation.ReservationID, err)
		return
	}
	
	reservation.DiscoveredResources = discovered
	
	if len(discovered) == 0 {
		reservation.Status = ReservationStatusFailed
		fmt.Printf("❌ No resources found for %s\n", reservation.ReservationID)
		return
	}
	
	// Phase 2: Selection and reservation
	reservation.Status = ReservationStatusAvailable
	fmt.Printf("🎯 Phase 2: Selecting optimal resource for %s\n", reservation.ReservationID)
	
	// Select best resource (lowest cost with sufficient SLA)
	var bestResource *AvailableResource
	for i := range discovered {
		resource := &discovered[i]
		if resource.AvailabilityPercentage >= reservation.SLARequirements.UptimePercentage {
			if resource.EstimatedSetupTimeMinutes <= reservation.SLARequirements.MaxSetupTimeMinutes {
				bestResource = resource
				break
			}
		}
	}
	
	if bestResource == nil {
		reservation.Status = ReservationStatusFailed
		fmt.Printf("❌ No suitable resource found for %s\n", reservation.ReservationID)
		return
	}
	
	reservation.SelectedResource = bestResource
	reservation.EstimatedCost = bestResource.TotalHourlyCost() * reservation.DurationHours
	reservation.EscrowAmount = reservation.EstimatedCost * 1.2 // 20% buffer
	
	reservation.Status = ReservationStatusReserved
	now := time.Now()
	reservation.ReservedAt = &now
	
	fmt.Printf("✅ Reserved %dx %s for %s\n", 
		bestResource.ResourceSpec.Quantity, 
		bestResource.ResourceSpec.ResourceType, 
		reservation.ReservationID)
	fmt.Printf("💰 Estimated cost: $%.2f\n", reservation.EstimatedCost)
	fmt.Printf("🏦 Escrow: $%.2f\n", reservation.EscrowAmount)
	
	// Phase 3: Provisioning
	ec.provisionResource(ctx, reservation)
	
	// Phase 4: Monitoring (runs in background)
	if reservation.Status == ReservationStatusActive {
		go ec.monitorReservation(ctx, reservation)
	}
}

// provisionResource provisions the reserved resource
func (ec *EnterpriseCockpit) provisionResource(ctx context.Context, reservation *ComputeReservation) {
	reservation.Status = ReservationStatusProvisioning
	fmt.Printf("⚙️  Phase 3: Provisioning %s\n", reservation.ReservationID)
	
	// Simulate provisioning time
	setupTime := reservation.SelectedResource.EstimatedSetupTimeMinutes
	time.Sleep(time.Duration(setupTime/10) * time.Minute) // Accelerated for demo
	
	// Generate connection details
	reservation.ConnectionInfo = map[string]interface{}{
		"ssh_host":     fmt.Sprintf("%s-%s.compute.ocx.ai", reservation.SelectedResource.ProviderID, reservation.Region),
		"ssh_port":     22,
		"ssh_key":      fmt.Sprintf("-----BEGIN RSA PRIVATE KEY-----\n%s\n-----END RSA PRIVATE KEY-----", generateRandomString(64)),
		"jupyter_url":  fmt.Sprintf("https://%s.jupyter.ocx.ai", reservation.ReservationID),
		"jupyter_token": generateRandomString(32),
		"api_endpoint": fmt.Sprintf("https://%s.api.ocx.ai", reservation.ReservationID),
		"api_key":      fmt.Sprintf("ocx_key_%s", generateRandomString(24)),
		"resource_ids": generateResourceIDs(reservation.ResourceSpec.Quantity),
	}
	
	reservation.Status = ReservationStatusActive
	now := time.Now()
	reservation.StartedAt = &now
	
	fmt.Printf("🟢 Resource active: %s\n", reservation.ConnectionInfo["ssh_host"])
	fmt.Printf("🔗 Jupyter: %s\n", reservation.ConnectionInfo["jupyter_url"])
	fmt.Printf("🔑 API: %s\n", reservation.ConnectionInfo["api_endpoint"])
}

// monitorReservation monitors active reservation for SLA compliance
func (ec *EnterpriseCockpit) monitorReservation(ctx context.Context, reservation *ComputeReservation) {
	fmt.Printf("📊 Starting monitoring for %s\n", reservation.ReservationID)
	
	monitoringInterval := 30 * time.Second
	
	for reservation.Status == ReservationStatusActive {
		select {
		case <-ctx.Done():
			return
		case <-time.After(monitoringInterval):
			// Generate monitoring metrics
			metrics := ec.generateMetrics(reservation)
			
			// Check SLA compliance
			ec.checkSLACompliance(reservation, metrics)
			
			// Store metrics
			if reservation.PerformanceMetrics["monitoring_history"] == nil {
				reservation.PerformanceMetrics["monitoring_history"] = []MonitoringMetrics{}
			}
			
			history := reservation.PerformanceMetrics["monitoring_history"].([]MonitoringMetrics)
			history = append(history, metrics)
			
			// Keep only last 1000 metrics
			if len(history) > 1000 {
				history = history[len(history)-1000:]
			}
			reservation.PerformanceMetrics["monitoring_history"] = history
			
			// Check if reservation should end
			elapsedHours := time.Since(*reservation.StartedAt).Hours()
			if elapsedHours >= reservation.DurationHours {
				ec.completeReservation(reservation)
				return
			}
		}
	}
}

// generateMetrics generates realistic monitoring metrics
func (ec *EnterpriseCockpit) generateMetrics(reservation *ComputeReservation) MonitoringMetrics {
	// Simulate some performance degradation over time
	elapsedHours := time.Since(*reservation.StartedAt).Hours()
	degradation := math.Max(0.9, 1.0-(elapsedHours*0.01))
	
	return MonitoringMetrics{
		Timestamp:           time.Now(),
		ReservationID:       reservation.ReservationID,
		CPUUtilization:      85.0*degradation + randomFloat(-10, 10),
		MemoryUtilization:   78.0*degradation + randomFloat(-15, 15),
		GPUUtilization:      92.0*degradation + randomFloat(-5, 5),
		NetworkIOMbps:       1200.0 + randomFloat(-200, 200),
		StorageIOMbps:       800.0 + randomFloat(-100, 100),
		ResponseTimeMs:      reservation.SLARequirements.MaxResponseTimeMs*0.7 + randomFloat(-2, 4),
		ThroughputOpsPerSec: 1000.0*degradation + randomFloat(-100, 100),
		ErrorRate:           randomFloat(0, 0.5),
		UptimePercentage:    99.9 + randomFloat(-0.5, 0.1),
		SLAComplianceScore:  math.Min(100.0, 95.0+randomFloat(0, 5)),
	}
}

// checkSLACompliance checks and updates SLA compliance status
func (ec *EnterpriseCockpit) checkSLACompliance(reservation *ComputeReservation, metrics MonitoringMetrics) {
	var violations []SLAViolation
	
	// Check response time
	if metrics.ResponseTimeMs > reservation.SLARequirements.MaxResponseTimeMs {
		violations = append(violations, SLAViolation{
			Type:      "response_time",
			Expected:  reservation.SLARequirements.MaxResponseTimeMs,
			Actual:    metrics.ResponseTimeMs,
			Timestamp: metrics.Timestamp,
		})
	}
	
	// Check uptime
	if metrics.UptimePercentage < reservation.SLARequirements.UptimePercentage {
		violations = append(violations, SLAViolation{
			Type:      "uptime",
			Expected:  reservation.SLARequirements.UptimePercentage,
			Actual:    metrics.UptimePercentage,
			Timestamp: metrics.Timestamp,
		})
	}
	
	// Check performance baseline
	expectedGPU := 90.0
	performanceRatio := metrics.GPUUtilization / expectedGPU
	if performanceRatio < reservation.SLARequirements.PerformanceBaseline {
		violations = append(violations, SLAViolation{
			Type:      "performance",
			Expected:  reservation.SLARequirements.PerformanceBaseline,
			Actual:    performanceRatio,
			Timestamp: metrics.Timestamp,
		})
	}
	
	// Update SLA status
	if len(violations) > 0 {
		reservation.SLAViolations = append(reservation.SLAViolations, violations...)
		
		// Count recent violations
		recentViolations := 0
		for _, v := range reservation.SLAViolations {
			if metrics.Timestamp.Sub(v.Timestamp) < 15*time.Minute {
				recentViolations++
			}
		}
		
		if recentViolations >= 3 {
			reservation.SLAStatus = SLAStatusBreach
			fmt.Printf("🚨 SLA BREACH detected for %s\n", reservation.ReservationID)
		} else if recentViolations >= 1 {
			reservation.SLAStatus = SLAStatusAtRisk
			fmt.Printf("⚠️  SLA AT RISK for %s\n", reservation.ReservationID)
		}
	} else {
		reservation.SLAStatus = SLAStatusMeeting
	}
}

// completeReservation completes reservation and handles billing
func (ec *EnterpriseCockpit) completeReservation(reservation *ComputeReservation) {
	reservation.Status = ReservationStatusCompleting
	now := time.Now()
	reservation.CompletedAt = &now
	
	// Calculate actual usage and costs
	actualHours := reservation.CompletedAt.Sub(*reservation.StartedAt).Hours()
	reservation.ActualCost = reservation.SelectedResource.TotalHourlyCost() * actualHours
	
	fmt.Printf("✅ Reservation %s completed\n", reservation.ReservationID)
	fmt.Printf("⏱️  Duration: %.2f hours\n", actualHours)
	fmt.Printf("💰 Final cost: $%.2f\n", reservation.ActualCost)
	
	reservation.Status = ReservationStatusCompleted
}

// GetReservation retrieves a reservation by ID
func (ec *EnterpriseCockpit) GetReservation(reservationID string) (*ComputeReservation, bool) {
	ec.ReservationEngine.mu.RLock()
	defer ec.ReservationEngine.mu.RUnlock()
	
	reservation, exists := ec.ReservationEngine.Reservations[reservationID]
	return reservation, exists
}

// ListReservations lists reservations for a customer
func (ec *EnterpriseCockpit) ListReservations(customerID string, status *ReservationStatus) []*ComputeReservation {
	ec.ReservationEngine.mu.RLock()
	defer ec.ReservationEngine.mu.RUnlock()
	
	var reservations []*ComputeReservation
	for _, r := range ec.ReservationEngine.Reservations {
		if r.CustomerID == customerID {
			if status == nil || r.Status == *status {
				reservations = append(reservations, r)
			}
		}
	}
	
	// Sort by creation time (newest first)
	sort.Slice(reservations, func(i, j int) bool {
		return reservations[i].CreatedAt.After(reservations[j].CreatedAt)
	})
	
	return reservations
}

// GetConnectionInfo returns connection information for an active reservation
func (ec *EnterpriseCockpit) GetConnectionInfo(reservationID string) (map[string]interface{}, bool) {
	reservation, exists := ec.GetReservation(reservationID)
	if !exists || reservation.Status != ReservationStatusActive {
		return nil, false
	}
	
	return reservation.ConnectionInfo, true
}

// GetMonitoring returns monitoring data for a reservation
func (ec *EnterpriseCockpit) GetMonitoring(reservationID string, lastNPoints int) (map[string]interface{}, bool) {
	reservation, exists := ec.GetReservation(reservationID)
	if !exists {
		return nil, false
	}
	
	history, ok := reservation.PerformanceMetrics["monitoring_history"].([]MonitoringMetrics)
	if !ok {
		return map[string]interface{}{
			"reservation_id": reservationID,
			"status":        reservation.Status,
			"sla_status":    reservation.SLAStatus,
			"recent_metrics": []MonitoringMetrics{},
		}, true
	}
	
	// Get last N points
	start := 0
	if len(history) > lastNPoints {
		start = len(history) - lastNPoints
	}
	recentMetrics := history[start:]
	
	return map[string]interface{}{
		"reservation_id":  reservationID,
		"status":         reservation.Status,
		"sla_status":     reservation.SLAStatus,
		"sla_violations": len(reservation.SLAViolations),
		"recent_metrics": recentMetrics,
		"summary_stats":  ec.calculateMonitoringSummary(recentMetrics),
	}, true
}

// calculateMonitoringSummary calculates summary statistics from monitoring history
func (ec *EnterpriseCockpit) calculateMonitoringSummary(metrics []MonitoringMetrics) map[string]float64 {
	if len(metrics) == 0 {
		return map[string]float64{}
	}
	
	var cpuUtils, gpuUtils, responseTimes []float64
	for _, m := range metrics {
		cpuUtils = append(cpuUtils, m.CPUUtilization)
		gpuUtils = append(gpuUtils, m.GPUUtilization)
		responseTimes = append(responseTimes, m.ResponseTimeMs)
	}
	
	avgCPU := average(cpuUtils)
	avgGPU := average(gpuUtils)
	avgResponse := average(responseTimes)
	maxResponse := maxFloat(responseTimes)
	minResponse := minFloat(responseTimes)
	
	return map[string]float64{
		"avg_cpu_utilization":  avgCPU,
		"avg_gpu_utilization":  avgGPU,
		"avg_response_time_ms": avgResponse,
		"max_response_time_ms": maxResponse,
		"min_response_time_ms": minResponse,
		"data_points":          float64(len(metrics)),
	}
}

// HealthCheck returns system health status
func (ec *EnterpriseCockpit) HealthCheck() map[string]interface{} {
	ec.ReservationEngine.mu.RLock()
	defer ec.ReservationEngine.mu.RUnlock()
	
	totalReservations := len(ec.ReservationEngine.Reservations)
	activeReservations := 0
	for _, r := range ec.ReservationEngine.Reservations {
		if r.Status == ReservationStatusActive {
			activeReservations++
		}
	}
	
	return map[string]interface{}{
		"status":              "healthy",
		"timestamp":           time.Now(),
		"total_reservations":  totalReservations,
		"active_reservations": activeReservations,
		"supported_resource_types": []string{
			string(ResourceTypeA100), string(ResourceTypeH100), string(ResourceTypeV100),
			string(ResourceTypeTPUV4), string(ResourceTypeTPUV5), string(ResourceTypeMI300X),
			string(ResourceTypeCPUIntel), string(ResourceTypeCPUAMD),
		},
		"supported_regions": []string{
			string(RegionUSEast), string(RegionUSWest), string(RegionEUWest),
			string(RegionEUCentral), string(RegionAsiaPacific), string(RegionAsiaSingapore),
			string(RegionAsiaTokyo), string(RegionMiddleEast), string(RegionLATAM),
		},
		"api_version": "1.0.0",
	}
}

// Helper functions

func parseResourceType(s string) (ResourceType, error) {
	switch s {
	case "A100", "a100":
		return ResourceTypeA100, nil
	case "H100", "h100":
		return ResourceTypeH100, nil
	case "V100", "v100":
		return ResourceTypeV100, nil
	case "TPU_V4", "tpu_v4", "tpu-v4":
		return ResourceTypeTPUV4, nil
	case "TPU_V5", "tpu_v5", "tpu-v5":
		return ResourceTypeTPUV5, nil
	case "MI300X", "mi300x", "mi-300x":
		return ResourceTypeMI300X, nil
	case "CPU_INTEL", "cpu_intel", "cpu-intel":
		return ResourceTypeCPUIntel, nil
	case "CPU_AMD", "cpu_amd", "cpu-amd":
		return ResourceTypeCPUAMD, nil
	default:
		return "", fmt.Errorf("unsupported resource type: %s", s)
	}
}

func parseRegion(s string) (Region, error) {
	switch s {
	case "us", "us-east":
		return RegionUSEast, nil
	case "us-west":
		return RegionUSWest, nil
	case "eu", "europe", "eu-west":
		return RegionEUWest, nil
	case "eu-central":
		return RegionEUCentral, nil
	case "asia", "asia-pacific":
		return RegionAsiaPacific, nil
	case "singapore":
		return RegionAsiaSingapore, nil
	case "tokyo", "japan":
		return RegionAsiaTokyo, nil
	case "middle-east":
		return RegionMiddleEast, nil
	case "latam":
		return RegionLATAM, nil
	default:
		return "", fmt.Errorf("unsupported region: %s", s)
	}
}

func parseDuration(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("duration cannot be empty")
	}
	
	// Handle numeric durations (assume hours)
	if duration, err := time.ParseDuration(s + "h"); err == nil {
		return duration.Hours(), nil
	}
	
	// Handle common duration formats
	switch {
	case s[len(s)-1] == 'h':
		return parseFloat(s[:len(s)-1])
	case s[len(s)-1] == 'd':
		days, err := parseFloat(s[:len(s)-1])
		return days * 24, err
	case s[len(s)-1] == 'w':
		weeks, err := parseFloat(s[:len(s)-1])
		return weeks * 24 * 7, err
	case s[len(s)-1] == 'm':
		minutes, err := parseFloat(s[:len(s)-1])
		return minutes / 60, err
	default:
		return parseFloat(s) // Assume hours
	}
}

func setResourceDefaults(spec *ResourceSpec) {
	if spec.MemoryGB == nil {
		memoryDefaults := map[ResourceType]int{
			ResourceTypeA100:     40,
			ResourceTypeH100:     80,
			ResourceTypeV100:     16,
			ResourceTypeTPUV4:    128,
			ResourceTypeTPUV5:    256,
			ResourceTypeMI300X:   192,
			ResourceTypeCPUIntel: 16,
			ResourceTypeCPUAMD:   16,
		}
		if mem, ok := memoryDefaults[spec.ResourceType]; ok {
			spec.MemoryGB = &mem
		}
	}
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

func generateResourceIDs(quantity int) []string {
	ids := make([]string, quantity)
	for i := 0; i < quantity; i++ {
		ids[i] = fmt.Sprintf("gpu_%d", i)
	}
	return ids
}

func randomFloat(min, max float64) float64 {
	return min + (max-min)*math.Mod(float64(time.Now().UnixNano()), 1.0)
}

func parseFloat(s string) (float64, error) {
	// Simple float parsing - in production, use strconv.ParseFloat
	return 0, fmt.Errorf("not implemented")
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func maxFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func minFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}
