// internal/consensus/telemetry/verification.go
package telemetry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// VerifierRole represents the role of a verifier node
type VerifierRole string

const (
	FullNode   VerifierRole = "full_node"   // Validates everything
	Watchdog   VerifierRole = "watchdog"    // Lightweight telemetry checker
	Validator  VerifierRole = "validator"   // Consensus participant
	Observer   VerifierRole = "observer"    // Read-only monitoring
)

// TelemetryEventType represents the type of telemetry event
type TelemetryEventType string

const (
	ComputeStart        TelemetryEventType = "compute_start"
	ComputeEnd          TelemetryEventType = "compute_end"
	Heartbeat           TelemetryEventType = "heartbeat"
	PerformanceSample   TelemetryEventType = "performance_sample"
	SLAViolation        TelemetryEventType = "sla_violation"
	ResourceAllocation  TelemetryEventType = "resource_allocation"
)

// ComputeWorkload represents a compute task that needs verification
type ComputeWorkload struct {
	WorkloadID      string                 `json:"workload_id"`
	ProviderID      string                 `json:"provider_id"`
	CustomerID      string                 `json:"customer_id"`
	ResourceSpec    map[string]interface{} `json:"resource_spec"`
	ExpectedDuration float64               `json:"expected_duration_seconds"`
	SLARequirements map[string]float64     `json:"sla_requirements"`
	StartTime       time.Time              `json:"start_time"`
	ChallengeSeed   string                 `json:"challenge_seed"`
}

// TelemetryEvent represents a single telemetry measurement with cryptographic proof
type TelemetryEvent struct {
	EventID          string                 `json:"event_id"`
	WorkloadID       string                 `json:"workload_id"`
	ProviderID       string                 `json:"provider_id"`
	EventType        TelemetryEventType     `json:"event_type"`
	Timestamp        time.Time              `json:"timestamp"`
	Metrics          map[string]float64     `json:"metrics"`
	ChallengeResponse string                `json:"challenge_response,omitempty"`
	Signature        string                 `json:"signature,omitempty"`
	Hash             string                 `json:"hash,omitempty"`
}

// ComputeHash computes a deterministic hash of the event
func (e *TelemetryEvent) ComputeHash() string {
	data := map[string]interface{}{
		"event_id":           e.EventID,
		"workload_id":        e.WorkloadID,
		"provider_id":        e.ProviderID,
		"event_type":         e.EventType,
		"timestamp":          e.Timestamp.UnixMilli(),
		"metrics":            e.Metrics,
		"challenge_response": e.ChallengeResponse,
	}
	
	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// VerificationChallenge generates computational challenges to prove work was done
type VerificationChallenge struct {
	DifficultyTarget string
}

// NewVerificationChallenge creates a new verification challenge system
func NewVerificationChallenge() *VerificationChallenge {
	return &VerificationChallenge{
		DifficultyTarget: "0000", // Require 4 leading zeros
	}
}

// GenerateChallenge generates a computational challenge for the workload
func (vc *VerificationChallenge) GenerateChallenge(workload *ComputeWorkload) string {
	challengeData := map[string]interface{}{
		"workload_id": workload.WorkloadID,
		"seed":        workload.ChallengeSeed,
		"timestamp":   time.Now().Unix(),
	}
	
	jsonData, _ := json.Marshal(challengeData)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// VerifyChallengeResponse verifies that the challenge response proves actual compute work
func (vc *VerificationChallenge) VerifyChallengeResponse(challenge, response, actualWorkProof string) bool {
	combined := fmt.Sprintf("%s%s%s", challenge, response, actualWorkProof)
	hash := sha256.Sum256([]byte(combined))
	hashStr := hex.EncodeToString(hash[:])
	return len(hashStr) >= len(vc.DifficultyTarget) && 
		   hashStr[:len(vc.DifficultyTarget)] == vc.DifficultyTarget
}

// GenerateWorkProof generates proof that actual compute work was performed
func (vc *VerificationChallenge) GenerateWorkProof(workloadID string, metrics map[string]float64) string {
	workSignature := map[string]interface{}{
		"workload_id": workloadID,
		"cpu_cycles":  metrics["cpu_utilization"] * 1000000,
		"memory_ops":  metrics["memory_usage_gb"] * 1024 * 1024,
		"gpu_flops":   metrics["gpu_utilization"] * 2000000000, // Approximate FLOPS
		"timestamp":   time.Now().UnixMilli(),
	}
	
	jsonData, _ := json.Marshal(workSignature)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// VerifierNode represents an independent verifier node in the consensus network
type VerifierNode struct {
	NodeID           string                 `json:"node_id"`
	Role             VerifierRole           `json:"role"`
	Stake            float64                `json:"stake"`
	IsByzantine      bool                   `json:"is_byzantine"`
	PeerNodes        map[string]bool        `json:"peer_nodes"`
	VerifiedWorkloads map[string]bool       `json:"verified_workloads"`
	ProposalsSeen    map[string]bool        `json:"proposals_seen"`
	VoteHistory      []Vote                 `json:"vote_history"`
	ChallengeGen     *VerificationChallenge `json:"-"`
}

// NewVerifierNode creates a new verifier node
func NewVerifierNode(nodeID string, role VerifierRole, stake float64) *VerifierNode {
	return &VerifierNode{
		NodeID:           nodeID,
		Role:             role,
		Stake:            stake,
		IsByzantine:      false,
		PeerNodes:        make(map[string]bool),
		VerifiedWorkloads: make(map[string]bool),
		ProposalsSeen:    make(map[string]bool),
		VoteHistory:      []Vote{},
		ChallengeGen:     NewVerificationChallenge(),
	}
}

// VerifyTelemetryEvent verifies a single telemetry event
func (vn *VerifierNode) VerifyTelemetryEvent(event *TelemetryEvent, workload *ComputeWorkload) (bool, string) {
	// Basic validation
	if event.WorkloadID != workload.WorkloadID {
		return false, "workload ID mismatch"
	}
	
	// Verify timestamp is reasonable
	now := time.Now()
	if math.Abs(event.Timestamp.Sub(now).Seconds()) > 3600 { // 1 hour tolerance
		return false, "timestamp too far from current time"
	}
	
	// Verify challenge response if present
	if event.ChallengeResponse != "" {
		challenge := vn.ChallengeGen.GenerateChallenge(workload)
		workProof := vn.ChallengeGen.GenerateWorkProof(event.WorkloadID, event.Metrics)
		
		if !vn.ChallengeGen.VerifyChallengeResponse(challenge, event.ChallengeResponse, workProof) {
			return false, "challenge response verification failed"
		}
	}
	
	// Verify metrics are within reasonable bounds
	if !vn.validateMetrics(event.Metrics) {
		return false, "metrics validation failed"
	}
	
	// Byzantine behavior simulation
	if vn.IsByzantine {
		// Randomly flip decision for Byzantine testing
		return time.Now().UnixNano()%2 == 0, "byzantine node random decision"
	}
	
	return true, "event verified successfully"
}

// validateMetrics validates that metrics are within reasonable bounds
func (vn *VerifierNode) validateMetrics(metrics map[string]float64) bool {
	bounds := map[string][2]float64{
		"cpu_utilization":    {0.0, 100.0},
		"memory_usage_gb":    {0.0, 1024.0},
		"gpu_utilization":    {0.0, 100.0},
		"network_io_mbps":    {0.0, 10000.0},
		"temperature_celsius": {10.0, 100.0},
		"power_consumption_watts": {50.0, 2000.0},
	}
	
	for metric, value := range metrics {
		if bounds, exists := bounds[metric]; exists {
			if value < bounds[0] || value > bounds[1] {
				return false
			}
		}
	}
	
	return true
}

// AssessSLACompliance assesses SLA compliance based on telemetry events
func (vn *VerifierNode) AssessSLACompliance(workload *ComputeWorkload, events []*TelemetryEvent) map[string]interface{} {
	if len(events) == 0 {
		return map[string]interface{}{
			"compliant": false,
			"reason":    "no telemetry data",
		}
	}
	
	// Calculate uptime
	totalEvents := len(events)
	validEvents := 0
	for _, event := range events {
		if event.Metrics["cpu_utilization"] > 0 {
			validEvents++
		}
	}
	uptimePercentage := float64(validEvents) / float64(totalEvents) * 100
	
	// Calculate average response time (simulated)
	responseTimes := make([]float64, len(events))
	for i, event := range events {
		responseTimes[i] = math.Max(1.0, event.Metrics["cpu_utilization"]/50.0*10)
	}
	
	avgResponseTime := 0.0
	for _, rt := range responseTimes {
		avgResponseTime += rt
	}
	avgResponseTime /= float64(len(responseTimes))
	
	// Check SLA requirements
	requiredUptime := workload.SLARequirements["uptime"]
	if requiredUptime == 0 {
		requiredUptime = 99.0 // Default
	}
	
	maxResponseTime := workload.SLARequirements["max_response_time"]
	if maxResponseTime == 0 {
		maxResponseTime = 10.0 // Default
	}
	
	uptimeOK := uptimePercentage >= requiredUptime
	responseOK := avgResponseTime <= maxResponseTime
	
	return map[string]interface{}{
		"compliant":         uptimeOK && responseOK,
		"uptime_percentage": uptimePercentage,
		"uptime_required":   requiredUptime,
		"uptime_met":        uptimeOK,
		"avg_response_time": avgResponseTime,
		"max_response_time": maxResponseTime,
		"response_time_met": responseOK,
		"total_events":      totalEvents,
		"valid_events":      validEvents,
	}
}

// Vote represents a vote on a verification proposal
type Vote struct {
	ProposalID  string    `json:"proposal_id"`
	VoterID     string    `json:"voter_id"`
	Vote        bool      `json:"vote"` // true = accept, false = reject
	Reasoning   string    `json:"reasoning"`
	Timestamp   time.Time `json:"timestamp"`
	Signature   string    `json:"signature"`
	StakeWeight float64   `json:"stake_weight"`
}

// VoteOnProposal votes on a verification proposal
func (vn *VerifierNode) VoteOnProposal(proposal *VerificationProposal, workload *ComputeWorkload) *Vote {
	// Independent verification of all events
	allValid := true
	rejectionReasons := []string{}
	
	for _, event := range proposal.TelemetryEvents {
		isValid, reason := vn.VerifyTelemetryEvent(event, workload)
		if !isValid {
			allValid = false
			rejectionReasons = append(rejectionReasons, fmt.Sprintf("event %s: %s", event.EventID, reason))
		}
	}
	
	// Verify SLA assessment
	ourAssessment := vn.AssessSLACompliance(workload, proposal.TelemetryEvents)
	assessmentMatches := math.Abs(ourAssessment["uptime_percentage"].(float64)-proposal.SLAAssessment["uptime_percentage"].(float64)) < 1.0
	
	if !assessmentMatches {
		allValid = false
		rejectionReasons = append(rejectionReasons, "SLA assessment mismatch")
	}
	
	// Byzantine behavior
	if vn.IsByzantine {
		allValid = !allValid // Flip the decision
		rejectionReasons = []string{"byzantine node opposing consensus"}
	}
	
	voteDecision := allValid
	reasoning := "all verifications passed"
	if !voteDecision {
		reasoning = fmt.Sprintf("%s", rejectionReasons)
	}
	
	// Create vote
	vote := &Vote{
		ProposalID:  proposal.ProposalID,
		VoterID:     vn.NodeID,
		Vote:        voteDecision,
		Reasoning:   reasoning,
		Timestamp:   time.Now(),
		Signature:   vn.signVote(proposal.ProposalID, voteDecision),
		StakeWeight: vn.Stake,
	}
	
	vn.VoteHistory = append(vn.VoteHistory, *vote)
	return vote
}

// signVote signs a vote (simplified for demo)
func (vn *VerifierNode) signVote(proposalID string, vote bool) string {
	data := fmt.Sprintf("%s:%s:%t:%d", proposalID, vn.NodeID, vote, time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
