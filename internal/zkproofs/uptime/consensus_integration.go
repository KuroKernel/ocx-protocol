package uptime

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ZKConsensusVerifier integrates ZK proofs with Byzantine consensus
type ZKConsensusVerifier struct {
	zkProof     *ZKUptimeProof
	verifiers   map[string]*VerifierNode
	mu          sync.RWMutex
	threshold   float64 // Byzantine fault tolerance threshold (e.g., 0.67)
}

// VerifierNode represents a node in the consensus network
type VerifierNode struct {
	ID           string    `json:"id"`
	PublicKey    string    `json:"public_key"`
	Stake        float64   `json:"stake"`
	LastSeen     time.Time `json:"last_seen"`
	IsActive     bool      `json:"is_active"`
	VerificationCount int  `json:"verification_count"`
}

// VerificationResult represents the result of a verification round
type VerificationResult struct {
	ProofID        string                 `json:"proof_id"`
	ProviderID     string                 `json:"provider_id"`
	ClaimedUptime  float64                `json:"claimed_uptime"`
	Votes          map[string]bool        `json:"votes"` // verifier_id -> valid
	Consensus      bool                   `json:"consensus"`
	Confidence     float64                `json:"confidence"`
	Timestamp      int64                  `json:"timestamp"`
	ProcessingTime time.Duration          `json:"processing_time"`
}

// NewZKConsensusVerifier creates a new ZK consensus verifier
func NewZKConsensusVerifier(threshold float64) *ZKConsensusVerifier {
	return &ZKConsensusVerifier{
		zkProof:   NewZKUptimeProof(),
		verifiers: make(map[string]*VerifierNode),
		threshold: threshold,
	}
}

// AddVerifier adds a new verifier node to the consensus network
func (zkc *ZKConsensusVerifier) AddVerifier(verifierID, publicKey string, stake float64) {
	zkc.mu.Lock()
	defer zkc.mu.Unlock()
	
	zkc.verifiers[verifierID] = &VerifierNode{
		ID:        verifierID,
		PublicKey: publicKey,
		Stake:     stake,
		LastSeen:  time.Now(),
		IsActive:  true,
	}
	
	fmt.Printf("✅ Added verifier %s with stake $%.2f\n", verifierID, stake)
}

// RemoveVerifier removes a verifier from the network
func (zkc *ZKConsensusVerifier) RemoveVerifier(verifierID string) {
	zkc.mu.Lock()
	defer zkc.mu.Unlock()
	
	if verifier, exists := zkc.verifiers[verifierID]; exists {
		verifier.IsActive = false
		fmt.Printf("❌ Removed verifier %s\n", verifierID)
	}
}

// VerifyUptimeProof verifies an uptime proof using Byzantine consensus
func (zkc *ZKConsensusVerifier) VerifyUptimeProof(ctx context.Context, 
	proof *UptimeProof, providerID string) (*VerificationResult, error) {
	
	startTime := time.Now()
	proofID := fmt.Sprintf("proof_%d_%s", time.Now().Unix(), providerID)
	
	fmt.Printf("🏛️  Starting Byzantine consensus verification for proof %s\n", proofID)
	fmt.Printf("📊 Provider %s claims %.2f%% uptime\n", providerID, proof.PublicInputs.ClaimedUptimePercentage)
	
	// Get active verifiers
	activeVerifiers := zkc.getActiveVerifiers()
	if len(activeVerifiers) == 0 {
		return nil, fmt.Errorf("no active verifiers available")
	}
	
	fmt.Printf("👥 %d active verifiers participating in consensus\n", len(activeVerifiers))
	
	// Create verification result
	result := &VerificationResult{
		ProofID:       proofID,
		ProviderID:    providerID,
		ClaimedUptime: proof.PublicInputs.ClaimedUptimePercentage,
		Votes:         make(map[string]bool),
		Timestamp:     time.Now().Unix(),
	}
	
	// Each verifier independently verifies the proof
	var wg sync.WaitGroup
	voteChan := make(chan VerifierVote, len(activeVerifiers))
	
	for _, verifier := range activeVerifiers {
		wg.Add(1)
		go func(v *VerifierNode) {
			defer wg.Done()
			
			// Simulate verifier processing time
			processingDelay := time.Duration(100+len(proof.Witness.DataHash)%500) * time.Millisecond
			time.Sleep(processingDelay)
			
			// Verify the proof
			isValid, reason := zkc.zkProof.VerifyProof(proof)
			
			// Update verifier stats
			zkc.mu.Lock()
			v.VerificationCount++
			v.LastSeen = time.Now()
			zkc.mu.Unlock()
			
			vote := VerifierVote{
				VerifierID: v.ID,
				IsValid:    isValid,
				Reason:     reason,
				Stake:      v.Stake,
				Timestamp:  time.Now().Unix(),
			}
			
			voteChan <- vote
			
			fmt.Printf("   %s: %s (stake: $%.2f)\n", 
				v.ID, 
				map[bool]string{true: "✅ VALID", false: "❌ INVALID"}[isValid],
				v.Stake)
		}(verifier)
	}
	
	// Close channel when all verifiers are done
	go func() {
		wg.Wait()
		close(voteChan)
	}()
	
	// Collect votes
	var totalStake, validStake float64
	for vote := range voteChan {
		result.Votes[vote.VerifierID] = vote.IsValid
		
		if vote.IsValid {
			validStake += vote.Stake
		}
		totalStake += vote.Stake
	}
	
	// Calculate consensus
	result.Confidence = validStake / totalStake
	result.Consensus = result.Confidence >= zkc.threshold
	result.ProcessingTime = time.Since(startTime)
	
	fmt.Printf("🎯 Consensus result: %.1f%% of stake voted VALID\n", result.Confidence*100)
	
	if result.Consensus {
		fmt.Printf("✅ PROOF ACCEPTED by Byzantine consensus\n")
		fmt.Printf("💰 Provider's %.2f%% uptime claim is mathematically verified\n", result.ClaimedUptime)
	} else {
		fmt.Printf("❌ PROOF REJECTED by Byzantine consensus\n")
		fmt.Printf("⚠️  Insufficient consensus: %.1f%% < %.1f%% threshold\n", 
			result.Confidence*100, zkc.threshold*100)
	}
	
	return result, nil
}

// VerifierVote represents a single verifier's vote
type VerifierVote struct {
	VerifierID string  `json:"verifier_id"`
	IsValid    bool    `json:"is_valid"`
	Reason     string  `json:"reason"`
	Stake      float64 `json:"stake"`
	Timestamp  int64   `json:"timestamp"`
}

// getActiveVerifiers returns all active verifiers
func (zkc *ZKConsensusVerifier) getActiveVerifiers() []*VerifierNode {
	zkc.mu.RLock()
	defer zkc.mu.RUnlock()
	
	var active []*VerifierNode
	for _, verifier := range zkc.verifiers {
		if verifier.IsActive {
			active = append(active, verifier)
		}
	}
	
	return active
}

// GetVerifierStats returns statistics about the verifier network
func (zkc *ZKConsensusVerifier) GetVerifierStats() map[string]interface{} {
	zkc.mu.RLock()
	defer zkc.mu.RUnlock()
	
	activeCount := 0
	totalStake := 0.0
	totalVerifications := 0
	
	for _, verifier := range zkc.verifiers {
		if verifier.IsActive {
			activeCount++
			totalStake += verifier.Stake
			totalVerifications += verifier.VerificationCount
		}
	}
	
	return map[string]interface{}{
		"active_verifiers":    activeCount,
		"total_verifiers":     len(zkc.verifiers),
		"total_stake":         totalStake,
		"total_verifications": totalVerifications,
		"consensus_threshold": zkc.threshold,
		"byzantine_tolerance": fmt.Sprintf("%.1f%%", (1-zkc.threshold)*100),
	}
}

// SimulateVerifierFailure simulates a verifier going offline
func (zkc *ZKConsensusVerifier) SimulateVerifierFailure(verifierID string) {
	zkc.mu.Lock()
	defer zkc.mu.Unlock()
	
	if verifier, exists := zkc.verifiers[verifierID]; exists {
		verifier.IsActive = false
		fmt.Printf("💥 Simulated failure of verifier %s\n", verifierID)
	}
}

// TestByzantineResistance tests the system's resistance to Byzantine failures
func (zkc *ZKConsensusVerifier) TestByzantineResistance(ctx context.Context, 
	proof *UptimeProof, providerID string) error {
	
	fmt.Println("🧪 Testing Byzantine fault tolerance")
	fmt.Println("====================================")
	
	// Test with different failure scenarios
	scenarios := []struct {
		name        string
		failCount   int
		description string
	}{
		{"No Failures", 0, "All verifiers operational"},
		{"Single Failure", 1, "One verifier offline"},
		{"Two Failures", 2, "Two verifiers offline"},
		{"Byzantine Limit", 3, "Maximum Byzantine failures (33%)"},
		{"Beyond Limit", 4, "Beyond Byzantine tolerance"},
	}
	
	for _, scenario := range scenarios {
		fmt.Printf("\n🔬 Scenario: %s\n", scenario.name)
		fmt.Printf("📝 %s\n", scenario.description)
		
		// Simulate failures
		verifierIDs := zkc.getVerifierIDs()
		for i := 0; i < scenario.failCount && i < len(verifierIDs); i++ {
			zkc.SimulateVerifierFailure(verifierIDs[i])
		}
		
		// Test verification
		result, err := zkc.VerifyUptimeProof(ctx, proof, providerID)
		if err != nil {
			fmt.Printf("❌ Verification failed: %v\n", err)
			continue
		}
		
		fmt.Printf("📊 Result: %s (%.1f%% confidence)\n", 
			map[bool]string{true: "ACCEPTED", false: "REJECTED"}[result.Consensus],
			result.Confidence*100)
		
		// Restore failed verifiers for next test
		for i := 0; i < scenario.failCount && i < len(verifierIDs); i++ {
			zkc.mu.Lock()
			if verifier, exists := zkc.verifiers[verifierIDs[i]]; exists {
				verifier.IsActive = true
			}
			zkc.mu.Unlock()
		}
	}
	
	return nil
}

// getVerifierIDs returns all verifier IDs
func (zkc *ZKConsensusVerifier) getVerifierIDs() []string {
	zkc.mu.RLock()
	defer zkc.mu.RUnlock()
	
	var ids []string
	for id := range zkc.verifiers {
		ids = append(ids, id)
	}
	return ids
}
