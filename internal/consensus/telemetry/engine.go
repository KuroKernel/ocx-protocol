// internal/consensus/telemetry/engine.go
package telemetry

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"time"
)

// TelemetryConsensusEngine is the main engine for Byzantine-grade telemetry verification
type TelemetryConsensusEngine struct {
	consensus      *ByzantineConsensus
	challengeGen   *VerificationChallenge
	workloads      map[string]*ComputeWorkload
	telemetryDB    map[string][]*TelemetryEvent
}

// NewTelemetryConsensusEngine creates a new telemetry consensus engine
func NewTelemetryConsensusEngine() *TelemetryConsensusEngine {
	// Initialize with 33% Byzantine tolerance (can handle up to 33% malicious nodes)
	consensus := NewByzantineConsensus(0.33)
	challengeGen := NewVerificationChallenge()
	
	engine := &TelemetryConsensusEngine{
		consensus:    consensus,
		challengeGen: challengeGen,
		workloads:    make(map[string]*ComputeWorkload),
		telemetryDB:  make(map[string][]*TelemetryEvent),
	}
	
	// Initialize verifier network
	engine.initializeVerifierNetwork()
	
	return engine
}

// initializeVerifierNetwork sets up the verifier network
func (tce *TelemetryConsensusEngine) initializeVerifierNetwork() {
	// Add different types of verifier nodes
	roles := []VerifierRole{FullNode, Validator, Watchdog, Observer}
	
	for i := 0; i < 7; i++ { // 7 nodes for good Byzantine tolerance
		role := roles[i%len(roles)]
		stake := 1.0 + float64(i)*0.2 // Varying stake weights
		
		node := NewVerifierNode(fmt.Sprintf("verifier_%d", i), role, stake)
		tce.consensus.AddVerifierNode(node)
	}
	
	// Simulate some Byzantine nodes (20% - below tolerance)
	tce.consensus.SimulateByzantineNodes(0.2)
	
	log.Printf("Initialized telemetry consensus network with %d verifier nodes", len(tce.consensus.VerifierNodes))
}

// StartWorkload starts a new compute workload and begins telemetry collection
func (tce *TelemetryConsensusEngine) StartWorkload(workload *ComputeWorkload) error {
	// Generate challenge seed
	seedBytes := make([]byte, 16)
	rand.Read(seedBytes)
	workload.ChallengeSeed = hex.EncodeToString(seedBytes)
	
	// Store workload
	tce.workloads[workload.WorkloadID] = workload
	tce.telemetryDB[workload.WorkloadID] = []*TelemetryEvent{}
	
	log.Printf("Started workload %s with challenge seed %s", workload.WorkloadID, workload.ChallengeSeed)
	return nil
}

// RecordTelemetryEvent records a telemetry event for a workload
func (tce *TelemetryConsensusEngine) RecordTelemetryEvent(workloadID string, event *TelemetryEvent) error {
	workload, exists := tce.workloads[workloadID]
	if !exists {
		return fmt.Errorf("workload %s not found", workloadID)
	}
	
	// Generate challenge response for proof of work
	challenge := tce.challengeGen.GenerateChallenge(workload)
	workProof := tce.challengeGen.GenerateWorkProof(event.WorkloadID, event.Metrics)
	
	// Simple proof-of-work (find nonce that makes hash start with 0000)
	nonce := 0
	for nonce < 100000 { // Safety limit
		response := fmt.Sprintf("%08d", nonce)
		if tce.challengeGen.VerifyChallengeResponse(challenge, response, workProof) {
			event.ChallengeResponse = response
			break
		}
		nonce++
	}
	
	if event.ChallengeResponse == "" {
		event.ChallengeResponse = "00000000" // Fallback
	}
	
	// Compute event hash
	event.Hash = event.ComputeHash()
	
	// Store event
	tce.telemetryDB[workloadID] = append(tce.telemetryDB[workloadID], event)
	
	log.Printf("Recorded telemetry event %s for workload %s", event.EventID, workloadID)
	return nil
}

// VerifyWorkloadCompletion verifies that a workload was completed according to SLA
func (tce *TelemetryConsensusEngine) VerifyWorkloadCompletion(workloadID string) (*ConsensusResult, error) {
	workload, exists := tce.workloads[workloadID]
	if !exists {
		return nil, fmt.Errorf("workload %s not found", workloadID)
	}
	
	events, exists := tce.telemetryDB[workloadID]
	if !exists || len(events) == 0 {
		return nil, fmt.Errorf("no telemetry events found for workload %s", workloadID)
	}
	
	log.Printf("Running Byzantine consensus verification for workload %s with %d telemetry events", workloadID, len(events))
	
	// Run Byzantine consensus
	result, err := tce.consensus.VerifyWorkloadConsensus(workload, events)
	if err != nil {
		return nil, fmt.Errorf("consensus verification failed: %w", err)
	}
	
	log.Printf("Consensus result for workload %s: %s", workloadID, result.ConsensusStatus)
	return result, nil
}

// GenerateSampleTelemetry generates sample telemetry data for testing
func (tce *TelemetryConsensusEngine) GenerateSampleTelemetry(workloadID string, durationHours float64) error {
	workload, exists := tce.workloads[workloadID]
	if !exists {
		return fmt.Errorf("workload %s not found", workloadID)
	}
	
	// Generate events every 5 minutes
	intervalMinutes := 5.0
	totalEvents := int(durationHours * 60 / intervalMinutes)
	
	startTime := workload.StartTime
	for i := 0; i < totalEvents; i++ {
		eventTime := startTime.Add(time.Duration(i) * time.Duration(intervalMinutes) * time.Minute)
		
		// Simulate slight degradation over time
		degradationFactor := 1.0 - (float64(i) * 0.01)
		
		// Generate realistic metrics
		metrics := map[string]float64{
			"cpu_utilization":     math.Max(0, 85.0*degradationFactor + tce.randomFloat(-5, 5)),
			"memory_usage_gb":     math.Max(0, 280.0 + tce.randomFloat(-10, 10)),
			"gpu_utilization":     math.Max(0, 95.0*degradationFactor + tce.randomFloat(-3, 3)),
			"network_io_mbps":     math.Max(0, 1000.0 + tce.randomFloat(-100, 100)),
			"temperature_celsius": math.Max(10, 72.0 + tce.randomFloat(0, 8)),
			"power_consumption_watts": math.Max(50, 400.0 + tce.randomFloat(-50, 50)),
		}
		
		event := &TelemetryEvent{
			EventID:    fmt.Sprintf("event_%s_%03d", workloadID, i),
			WorkloadID: workloadID,
			ProviderID: workload.ProviderID,
			EventType:  PerformanceSample,
			Timestamp:  eventTime,
			Metrics:    metrics,
		}
		
		if err := tce.RecordTelemetryEvent(workloadID, event); err != nil {
			return fmt.Errorf("failed to record event %d: %w", i, err)
		}
	}
	
	log.Printf("Generated %d telemetry events for workload %s", totalEvents, workloadID)
	return nil
}

// randomFloat generates a random float between min and max
func (tce *TelemetryConsensusEngine) randomFloat(min, max float64) float64 {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	
	// Convert to float64 between 0 and 1
	val := float64(bytes[0]) / 255.0
	
	// Scale to desired range
	return min + val*(max-min)
}

// GetNetworkStatus returns the current network status
func (tce *TelemetryConsensusEngine) GetNetworkStatus() *NetworkStatus {
	return tce.consensus.GetNetworkStatus()
}

// GetWorkloadStatus returns the status of a specific workload
func (tce *TelemetryConsensusEngine) GetWorkloadStatus(workloadID string) (*WorkloadStatus, error) {
	workload, exists := tce.workloads[workloadID]
	if !exists {
		return nil, fmt.Errorf("workload %s not found", workloadID)
	}
	
	events, exists := tce.telemetryDB[workloadID]
	if !exists {
		events = []*TelemetryEvent{}
	}
	
	// Calculate basic metrics
	totalEvents := len(events)
	validEvents := 0
	for _, event := range events {
		if event.Metrics["cpu_utilization"] > 0 {
			validEvents++
		}
	}
	
	uptimePercentage := 0.0
	if totalEvents > 0 {
		uptimePercentage = float64(validEvents) / float64(totalEvents) * 100
	}
	
	return &WorkloadStatus{
		WorkloadID:       workloadID,
		ProviderID:       workload.ProviderID,
		CustomerID:       workload.CustomerID,
		StartTime:        workload.StartTime,
		TotalEvents:      totalEvents,
		ValidEvents:      validEvents,
		UptimePercentage: uptimePercentage,
		SLARequirements:  workload.SLARequirements,
		LastEventTime:    tce.getLastEventTime(events),
		Status:           tce.determineWorkloadStatus(workload, events),
	}, nil
}

// getLastEventTime gets the timestamp of the last telemetry event
func (tce *TelemetryConsensusEngine) getLastEventTime(events []*TelemetryEvent) *time.Time {
	if len(events) == 0 {
		return nil
	}
	
	lastEvent := events[len(events)-1]
	return &lastEvent.Timestamp
}

// determineWorkloadStatus determines the current status of a workload
func (tce *TelemetryConsensusEngine) determineWorkloadStatus(workload *ComputeWorkload, events []*TelemetryEvent) string {
	if len(events) == 0 {
		return "not_started"
	}
	
	// Check if workload should be completed
	expectedEndTime := workload.StartTime.Add(time.Duration(workload.ExpectedDuration) * time.Second)
	if time.Now().After(expectedEndTime) {
		return "completed"
	}
	
	// Check if recent events exist
	lastEvent := events[len(events)-1]
	if time.Since(lastEvent.Timestamp) > 10*time.Minute {
		return "stalled"
	}
	
	return "running"
}

// WorkloadStatus represents the status of a workload
type WorkloadStatus struct {
	WorkloadID       string                 `json:"workload_id"`
	ProviderID       string                 `json:"provider_id"`
	CustomerID       string                 `json:"customer_id"`
	StartTime        time.Time              `json:"start_time"`
	TotalEvents      int                    `json:"total_events"`
	ValidEvents      int                    `json:"valid_events"`
	UptimePercentage float64                `json:"uptime_percentage"`
	SLARequirements  map[string]float64     `json:"sla_requirements"`
	LastEventTime    *time.Time             `json:"last_event_time"`
	Status           string                 `json:"status"`
}
