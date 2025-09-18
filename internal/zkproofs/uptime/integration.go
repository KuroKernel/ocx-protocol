package uptime

import (
	"fmt"
	"time"
)

// ZKProofsIntegration integrates uptime proofs with the main ZK proofs system
type ZKProofsIntegration struct {
	uptimeProof *ZKUptimeProof
	consensus   *ZKConsensusVerifier
}

// NewZKProofsIntegration creates a new integration instance
func NewZKProofsIntegration() *ZKProofsIntegration {
	return &ZKProofsIntegration{
		uptimeProof: NewZKUptimeProof(),
		consensus:   NewZKConsensusVerifier(0.67),
	}
}

// UptimeVerificationRequest represents a request to verify uptime
type UptimeVerificationRequest struct {
	ProviderID     string            `json:"provider_id"`
	WorkloadID     string            `json:"workload_id"`
	ClaimedUptime  float64           `json:"claimed_uptime"`
	ContractStart  int64             `json:"contract_start"`
	ContractEnd    int64             `json:"contract_end"`
	PrivateData    []UptimeDataPoint `json:"-"` // Private - not serialized
	SLARequirements *SLARequirements `json:"sla_requirements"`
}

// UptimeVerificationResult represents the result of uptime verification
type UptimeVerificationResult struct {
	RequestID      string                 `json:"request_id"`
	ProviderID     string                 `json:"provider_id"`
	WorkloadID     string                 `json:"workload_id"`
	ClaimedUptime  float64                `json:"claimed_uptime"`
	ActualUptime   float64                `json:"actual_uptime"`
	Proof          *UptimeProof           `json:"proof"`
	ConsensusResult *VerificationResult   `json:"consensus_result"`
	Verified       bool                   `json:"verified"`
	Timestamp      int64                  `json:"timestamp"`
	ProcessingTime time.Duration          `json:"processing_time"`
}

// VerifyUptime verifies uptime claims using ZK proofs and Byzantine consensus
func (zki *ZKProofsIntegration) VerifyUptime(request *UptimeVerificationRequest) (*UptimeVerificationResult, error) {
	startTime := time.Now()
	requestID := fmt.Sprintf("uptime_verify_%d_%s", time.Now().Unix(), request.ProviderID)
	
	fmt.Printf("🔒 Starting uptime verification for %s (workload: %s)\n", 
		request.ProviderID, request.WorkloadID)
	fmt.Printf("📊 Claimed uptime: %.2f%%\n", request.ClaimedUptime)
	
	// Calculate actual uptime from private data
	actualUptime := zki.uptimeProof.CalculateActualUptime(request.PrivateData)
	
	// Generate ZK proof
	proof, err := zki.uptimeProof.GenerateProof(
		request.PrivateData,
		request.ClaimedUptime,
		request.ContractStart,
		request.ContractEnd,
		request.SLARequirements,
	)
	if err != nil {
		return nil, fmt.Errorf("proof generation failed: %w", err)
	}
	
	// Verify proof using Byzantine consensus
	consensusResult, err := zki.consensus.VerifyUptimeProof(nil, proof, request.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("consensus verification failed: %w", err)
	}
	
	// Create result
	result := &UptimeVerificationResult{
		RequestID:      requestID,
		ProviderID:     request.ProviderID,
		WorkloadID:     request.WorkloadID,
		ClaimedUptime:  request.ClaimedUptime,
		ActualUptime:   actualUptime,
		Proof:          proof,
		ConsensusResult: consensusResult,
		Verified:       consensusResult.Consensus,
		Timestamp:      time.Now().Unix(),
		ProcessingTime: time.Since(startTime),
	}
	
	fmt.Printf("✅ Uptime verification completed: %s\n", 
		map[bool]string{true: "VERIFIED", false: "REJECTED"}[result.Verified])
	
	return result, nil
}

// BatchVerifyUptime verifies multiple uptime claims in batch
func (zki *ZKProofsIntegration) BatchVerifyUptime(requests []*UptimeVerificationRequest) ([]*UptimeVerificationResult, error) {
	fmt.Printf("🔄 Starting batch verification of %d uptime claims\n", len(requests))
	
	results := make([]*UptimeVerificationResult, 0, len(requests))
	
	for i, request := range requests {
		fmt.Printf("📋 Processing request %d/%d: %s\n", i+1, len(requests), request.ProviderID)
		
		result, err := zki.VerifyUptime(request)
		if err != nil {
			fmt.Printf("❌ Verification failed for %s: %v\n", request.ProviderID, err)
			continue
		}
		
		results = append(results, result)
	}
	
	verifiedCount := 0
	for _, result := range results {
		if result.Verified {
			verifiedCount++
		}
	}
	
	fmt.Printf("✅ Batch verification completed: %d/%d verified\n", verifiedCount, len(requests))
	
	return results, nil
}

// GetVerificationStats returns statistics about verification performance
func (zki *ZKProofsIntegration) GetVerificationStats() map[string]interface{} {
	consensusStats := zki.consensus.GetVerifierStats()
	
	return map[string]interface{}{
		"consensus_network": consensusStats,
		"uptime_proof_system": map[string]interface{}{
			"circuit_id": zki.uptimeProof.CircuitID,
			"supported_sla_types": []string{"uptime", "response_time", "availability"},
			"privacy_level": "maximum",
			"verification_method": "zero_knowledge",
		},
		"integration_status": "active",
	}
}

// AddVerifier adds a verifier to the consensus network
func (zki *ZKProofsIntegration) AddVerifier(verifierID, publicKey string, stake float64) {
	zki.consensus.AddVerifier(verifierID, publicKey, stake)
}

// RemoveVerifier removes a verifier from the consensus network
func (zki *ZKProofsIntegration) RemoveVerifier(verifierID string) {
	zki.consensus.RemoveVerifier(verifierID)
}

// TestUptimeVerification tests the complete uptime verification system
func (zki *ZKProofsIntegration) TestUptimeVerification() error {
	fmt.Println("🧪 Testing Complete Uptime Verification System")
	fmt.Println("==============================================")
	
	// Set up test verifiers
	testVerifiers := []struct {
		id        string
		publicKey string
		stake     float64
	}{
		{"test_verifier_1", "pk_test_1", 10000.0},
		{"test_verifier_2", "pk_test_2", 15000.0},
		{"test_verifier_3", "pk_test_3", 12000.0},
		{"test_verifier_4", "pk_test_4", 18000.0},
		{"test_verifier_5", "pk_test_5", 20000.0},
	}
	
	for _, v := range testVerifiers {
		zki.AddVerifier(v.id, v.publicKey, v.stake)
	}
	
	// Test 1: Valid uptime claim
	fmt.Println("\n🔬 Test 1: Valid Uptime Claim")
	contractStart := time.Now().Add(-12 * time.Hour).Unix()
	contractEnd := time.Now().Unix()
	claimedUptime := 99.5
	
	privateData := zki.uptimeProof.GenerateTestData(contractStart, contractEnd, claimedUptime)
	
	request := &UptimeVerificationRequest{
		ProviderID:     "test_provider_1",
		WorkloadID:     "workload_123",
		ClaimedUptime:  claimedUptime,
		ContractStart:  contractStart,
		ContractEnd:    contractEnd,
		PrivateData:    privateData,
		SLARequirements: &SLARequirements{
			MinUptime:      99.0,
			MaxResponseTime: 10.0,
			MinMeasurements: 100,
		},
	}
	
	result, err := zki.VerifyUptime(request)
	if err != nil {
		return fmt.Errorf("test 1 failed: %w", err)
	}
	
	fmt.Printf("✅ Test 1 result: %s (%.1f%% confidence)\n", 
		map[bool]string{true: "VERIFIED", false: "REJECTED"}[result.Verified],
		result.ConsensusResult.Confidence*100)
	
	// Test 2: Invalid uptime claim
	fmt.Println("\n🔬 Test 2: Invalid Uptime Claim")
	invalidRequest := *request
	invalidRequest.ClaimedUptime = 99.9 // Claiming higher than actual
	
	_, err = zki.VerifyUptime(&invalidRequest)
	if err != nil {
		fmt.Printf("✅ Test 2 correctly rejected invalid claim: %v\n", err)
	} else {
		fmt.Printf("❌ Test 2 should have failed but didn't\n")
	}
	
	// Test 3: Batch verification
	fmt.Println("\n🔬 Test 3: Batch Verification")
	
	var batchRequests []*UptimeVerificationRequest
	for i := 0; i < 3; i++ {
		providerID := fmt.Sprintf("batch_provider_%d", i+1)
		claimedUptime := 99.0 + float64(i)*0.2
		
		privateData := zki.uptimeProof.GenerateTestData(contractStart, contractEnd, claimedUptime)
		
		batchRequest := &UptimeVerificationRequest{
			ProviderID:     providerID,
			WorkloadID:     fmt.Sprintf("workload_%d", i+1),
			ClaimedUptime:  claimedUptime,
			ContractStart:  contractStart,
			ContractEnd:    contractEnd,
			PrivateData:    privateData,
			SLARequirements: request.SLARequirements,
		}
		
		batchRequests = append(batchRequests, batchRequest)
	}
	
	batchResults, err := zki.BatchVerifyUptime(batchRequests)
	if err != nil {
		return fmt.Errorf("batch test failed: %w", err)
	}
	
	verifiedCount := 0
	for _, result := range batchResults {
		if result.Verified {
			verifiedCount++
		}
	}
	
	fmt.Printf("✅ Batch test result: %d/%d verified\n", verifiedCount, len(batchResults))
	
	// Show final stats
	stats := zki.GetVerificationStats()
	fmt.Printf("\n📊 Final Statistics:\n")
	fmt.Printf("   Active verifiers: %v\n", stats["consensus_network"].(map[string]interface{})["active_verifiers"])
	fmt.Printf("   Total stake: $%.0f\n", stats["consensus_network"].(map[string]interface{})["total_stake"])
	fmt.Printf("   Integration status: %s\n", stats["integration_status"])
	
	fmt.Println("\n🎉 All tests completed successfully!")
	return nil
}
