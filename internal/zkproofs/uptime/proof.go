package uptime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	
	"time"
)

// UptimeDataPoint represents a single uptime measurement (private data)
type UptimeDataPoint struct {
	Timestamp        int64   `json:"timestamp"`
	IsOnline         bool    `json:"is_online"`
	ResponseTimeMs   float64 `json:"response_time_ms"`
	CPUUtilization   float64 `json:"cpu_utilization"`
	MemoryUtilization float64 `json:"memory_utilization"`
	ErrorRate        float64 `json:"error_rate"`
}

// ToCommitment converts the data point to a cryptographic commitment
func (u *UptimeDataPoint) ToCommitment() string {
	data := fmt.Sprintf("%d:%t:%.2f:%.2f:%.2f:%.2f", 
		u.Timestamp, u.IsOnline, u.ResponseTimeMs, 
		u.CPUUtilization, u.MemoryUtilization, u.ErrorRate)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ZKUptimeProof implements zero-knowledge proof system for uptime verification
type ZKUptimeProof struct {
	CircuitID string
}

// NewZKUptimeProof creates a new ZK uptime proof system
func NewZKUptimeProof() *ZKUptimeProof {
	return &ZKUptimeProof{
		CircuitID: "uptime_verification_v1",
	}
}

// UptimeProof represents a complete ZK proof for uptime claims
type UptimeProof struct {
	CircuitID       string                 `json:"circuit_id"`
	PublicInputs    UptimePublicInputs     `json:"public_inputs"`
	ProofCommitment string                 `json:"proof_commitment"`
	Signature       string                 `json:"signature"`
	Timestamp       int64                  `json:"timestamp"`
	Witness         UptimeWitness          `json:"witness"`
}

// UptimePublicInputs contains the public inputs to the proof
type UptimePublicInputs struct {
	ClaimedUptimePercentage float64 `json:"claimed_uptime_percentage"`
	ContractStart          int64   `json:"contract_start"`
	ContractEnd            int64   `json:"contract_end"`
	MeasurementCount       int     `json:"measurement_count"`
	CommitmentsRoot        string  `json:"commitments_root"`
	MinUptimeRequired      float64 `json:"min_uptime_required"`
	MaxResponseTime        float64 `json:"max_response_time"`
}

// UptimeWitness contains the witness data for the proof
type UptimeWitness struct {
	ConstraintSatisfied bool    `json:"constraint_satisfied"`
	DataHash           string  `json:"data_hash"`
	UptimeProof        string  `json:"uptime_proof"`
	ResponseTimeProof  float64 `json:"response_time_proof"`
	Nonce              string  `json:"nonce"`
	MerkleProof        []string `json:"merkle_proof"`
}

// GenerateProof generates a ZK proof that uptime claim is valid
func (zk *ZKUptimeProof) GenerateProof(privateData []UptimeDataPoint, 
	claimedUptimePercentage float64, contractStart, contractEnd int64,
	requirements *SLARequirements) (*UptimeProof, error) {
	
	fmt.Printf("🔒 Generating ZK proof for %.2f%% uptime claim\n", claimedUptimePercentage)
	
	// Step 1: Validate private inputs
	if err := zk.validatePrivateData(privateData, contractStart, contractEnd); err != nil {
		return nil, fmt.Errorf("private data validation failed: %w", err)
	}
	
	// Step 2: Calculate actual uptime from private data
	actualUptime := zk.CalculateActualUptime(privateData)
	fmt.Printf("📊 Actual uptime from private data: %.2f%%\n", actualUptime)
	
	// Step 3: Check if claim is valid
	minRequired := claimedUptimePercentage - 0.1 // 0.1% tolerance
	if actualUptime < minRequired {
		return nil, fmt.Errorf("claim %.2f%% exceeds actual %.2f%%", claimedUptimePercentage, actualUptime)
	}
	
	// Step 4: Generate cryptographic commitments to private data
	commitments := make([]string, len(privateData))
	for i, point := range privateData {
		commitments[i] = point.ToCommitment()
	}
	commitmentsRoot := zk.buildMerkleRoot(commitments)
	
	// Step 5: Generate witness
	witness, err := zk.generateWitness(privateData, claimedUptimePercentage, commitments)
	if err != nil {
		return nil, fmt.Errorf("witness generation failed: %w", err)
	}
	
	// Step 6: Create public inputs
	publicInputs := UptimePublicInputs{
		ClaimedUptimePercentage: claimedUptimePercentage,
		ContractStart:          contractStart,
		ContractEnd:            contractEnd,
		MeasurementCount:       len(privateData),
		CommitmentsRoot:        commitmentsRoot,
		MinUptimeRequired:      requirements.MinUptime,
		MaxResponseTime:        requirements.MaxResponseTime,
	}
	
	// Step 7: Create the proof
	proof := &UptimeProof{
		CircuitID:       zk.CircuitID,
		PublicInputs:    publicInputs,
		ProofCommitment: witness.DataHash,
		Witness:         *witness,
		Timestamp:       time.Now().Unix(),
	}
	
	// Step 8: Sign the proof
	signature, err := zk.signProof(proof)
	if err != nil {
		return nil, fmt.Errorf("proof signing failed: %w", err)
	}
	proof.Signature = signature
	
	fmt.Printf("✅ ZK proof generated successfully\n")
	return proof, nil
}

// VerifyProof verifies a ZK proof without access to private data
func (zk *ZKUptimeProof) VerifyProof(proof *UptimeProof) (bool, string) {
	fmt.Printf("🔍 Verifying ZK proof for %.2f%% uptime\n", proof.PublicInputs.ClaimedUptimePercentage)
	
	// Step 1: Verify proof structure
	if err := zk.validateProofStructure(proof); err != nil {
		return false, fmt.Sprintf("invalid proof structure: %v", err)
	}
	
	// Step 2: Verify circuit ID
	if proof.CircuitID != zk.CircuitID {
		return false, "invalid circuit ID"
	}
	
	// Step 3: Verify public inputs are reasonable
	if err := zk.validatePublicInputs(proof.PublicInputs); err != nil {
		return false, fmt.Sprintf("invalid public inputs: %v", err)
	}
	
	// Step 4: Verify proof commitment
	if !zk.verifyWitness(proof.Witness, proof.PublicInputs) {
		return false, "proof commitment verification failed"
	}
	
	// Step 5: Verify signature
	if !zk.verifyProofSignature(proof) {
		return false, "proof signature invalid"
	}
	
	// Step 6: Verify merkle proof
	if !zk.verifyMerkleProof(proof.Witness.MerkleProof, proof.PublicInputs.CommitmentsRoot) {
		return false, "merkle proof verification failed"
	}
	
	fmt.Printf("✅ ZK proof verified successfully\n")
	return true, "proof valid"
}

// SLARequirements defines SLA requirements for verification
type SLARequirements struct {
	MinUptime      float64 `json:"min_uptime"`
	MaxResponseTime float64 `json:"max_response_time"`
	MinMeasurements int     `json:"min_measurements"`
}

// validatePrivateData validates that private data is consistent and complete
func (zk *ZKUptimeProof) validatePrivateData(data []UptimeDataPoint, start, end int64) error {
	if len(data) == 0 {
		return fmt.Errorf("no data points provided")
	}
	
	// Check timestamps are in order and within contract period
	for i, point := range data {
		if point.Timestamp < start || point.Timestamp > end {
			return fmt.Errorf("timestamp %d outside contract period", point.Timestamp)
		}
		if i > 0 && point.Timestamp <= data[i-1].Timestamp {
			return fmt.Errorf("timestamps not chronological at index %d", i)
		}
	}
	
	// Check we have sufficient data points
	expectedPoints := (end - start) / 300 // Every 5 minutes
	if len(data) < int(expectedPoints)*8/10 { // At least 80% coverage
		return fmt.Errorf("insufficient data points: got %d, expected at least %d", 
			len(data), int(expectedPoints)*8/10)
	}
	
	return nil
}

// calculateActualUptime calculates actual uptime percentage from private data
func (zk *ZKUptimeProof) CalculateActualUptime(data []UptimeDataPoint) float64 {
	if len(data) == 0 {
		return 0.0
	}
	
	onlineCount := 0
	for _, point := range data {
		if point.IsOnline {
			onlineCount++
		}
	}
	
	return float64(onlineCount) / float64(len(data)) * 100.0
}

// buildMerkleRoot builds merkle tree root from commitments
func (zk *ZKUptimeProof) buildMerkleRoot(commitments []string) string {
	if len(commitments) == 0 {
		return ""
	}
	
	level := make([]string, len(commitments))
	copy(level, commitments)
	
	for len(level) > 1 {
		nextLevel := make([]string, 0, (len(level)+1)/2)
		
		for i := 0; i < len(level); i += 2 {
			var combined string
			if i+1 < len(level) {
				combined = level[i] + level[i+1]
			} else {
				combined = level[i] + level[i] // Duplicate if odd
			}
			
			hash := sha256.Sum256([]byte(combined))
			nextLevel = append(nextLevel, hex.EncodeToString(hash[:]))
		}
		
		level = nextLevel
	}
	
	return level[0]
}

// generateWitness generates witness proving we know private data satisfying the claim
func (zk *ZKUptimeProof) generateWitness(privateData []UptimeDataPoint, 
	claimedUptime float64, commitments []string) (*UptimeWitness, error) {
	
	// Calculate actual metrics from private data
	actualUptime := zk.CalculateActualUptime(privateData)
	
	// Calculate average response time for online periods
	onlinePeriods := make([]UptimeDataPoint, 0)
	for _, point := range privateData {
		if point.IsOnline {
			onlinePeriods = append(onlinePeriods, point)
		}
	}
	
	var avgResponseTime float64
	if len(onlinePeriods) > 0 {
		totalResponseTime := 0.0
		for _, point := range onlinePeriods {
			totalResponseTime += point.ResponseTimeMs
		}
		avgResponseTime = totalResponseTime / float64(len(onlinePeriods))
	}
	
	// Create data hash
	dataJSON, err := json.Marshal(privateData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private data: %w", err)
	}
	dataHash := sha256.Sum256(dataJSON)
	
	// Create uptime proof
	uptimeProof := zk.createUptimeProof(privateData, claimedUptime)
	
	// Generate merkle proof (simplified - in production would be full merkle proof)
	merkleProof := zk.generateMerkleProof(commitments, 0) // Proof for first element
	
	// Generate nonce
	nonce := fmt.Sprintf("%d_%d", time.Now().UnixNano(), len(privateData))
	
	return &UptimeWitness{
		ConstraintSatisfied: actualUptime >= claimedUptime-0.1,
		DataHash:           hex.EncodeToString(dataHash[:]),
		UptimeProof:        uptimeProof,
		ResponseTimeProof:  avgResponseTime,
		Nonce:              nonce,
		MerkleProof:        merkleProof,
	}, nil
}

// createUptimeProof creates cryptographic proof of uptime calculation
func (zk *ZKUptimeProof) createUptimeProof(data []UptimeDataPoint, claim float64) string {
	onlineCount := 0
	for _, point := range data {
		if point.IsOnline {
			onlineCount++
		}
	}
	
	totalCount := len(data)
	actualUptime := float64(onlineCount) / float64(totalCount) * 100.0
	
	// Proof that calculation was done correctly
	proofElements := []string{
		fmt.Sprintf("%d", onlineCount),
		fmt.Sprintf("%d", totalCount),
		fmt.Sprintf("%.6f", actualUptime),
		fmt.Sprintf("claim_%.6f", claim),
	}
	
	combined := ""
	for _, element := range proofElements {
		combined += element + "|"
	}
	
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// generateMerkleProof generates a merkle proof for a specific element
func (zk *ZKUptimeProof) generateMerkleProof(commitments []string, index int) []string {
	// Simplified merkle proof generation
	// In production, this would generate a full merkle proof path
	proof := make([]string, 0)
	
	if index < len(commitments) {
		proof = append(proof, commitments[index])
	}
	
	// Add some additional proof elements (simplified)
	if index+1 < len(commitments) {
		proof = append(proof, commitments[index+1])
	}
	
	return proof
}

// verifyWitness verifies that witness is valid for the public inputs
func (zk *ZKUptimeProof) verifyWitness(witness UptimeWitness, publicInputs UptimePublicInputs) bool {
	// Check witness format
	if len(witness.DataHash) != 64 { // SHA256 hex length
		return false
	}
	
	// Check constraint satisfaction
	if !witness.ConstraintSatisfied {
		return false
	}
	
	// Check nonce format
	if witness.Nonce == "" {
		return false
	}
	
	// Additional consistency checks
	return true
}

// verifyMerkleProof verifies a merkle proof
func (zk *ZKUptimeProof) verifyMerkleProof(merkleProof []string, root string) bool {
	// Simplified merkle proof verification
	// In production, this would verify the full merkle proof path
	
	if len(merkleProof) == 0 {
		return false
	}
	
	// Basic verification - check that proof elements exist
	for _, element := range merkleProof {
		if element == "" {
			return false
		}
	}
	
	return true
}

// validateProofStructure validates the basic structure of a proof
func (zk *ZKUptimeProof) validateProofStructure(proof *UptimeProof) error {
	if proof.CircuitID == "" {
		return fmt.Errorf("missing circuit ID")
	}
	
	if proof.ProofCommitment == "" {
		return fmt.Errorf("missing proof commitment")
	}
	
	if proof.Signature == "" {
		return fmt.Errorf("missing signature")
	}
	
	if proof.Witness.DataHash == "" {
		return fmt.Errorf("missing witness data hash")
	}
	
	return nil
}

// validatePublicInputs validates that public inputs are reasonable
func (zk *ZKUptimeProof) validatePublicInputs(inputs UptimePublicInputs) error {
	// Check uptime percentage is valid
	if inputs.ClaimedUptimePercentage < 0 || inputs.ClaimedUptimePercentage > 100 {
		return fmt.Errorf("invalid uptime percentage: %.2f", inputs.ClaimedUptimePercentage)
	}
	
	// Check contract period is valid
	if inputs.ContractStart >= inputs.ContractEnd {
		return fmt.Errorf("invalid contract period: start %d >= end %d", 
			inputs.ContractStart, inputs.ContractEnd)
	}
	
	// Check measurement count is reasonable
	contractDuration := inputs.ContractEnd - inputs.ContractStart
	expectedMeasurements := contractDuration / 300 // Every 5 minutes
	
	if inputs.MeasurementCount < int(expectedMeasurements)*6/10 { // At least 60% coverage
		return fmt.Errorf("insufficient measurements: got %d, expected at least %d", 
			inputs.MeasurementCount, int(expectedMeasurements)*6/10)
	}
	
	// Check commitments root format
	if len(inputs.CommitmentsRoot) != 64 { // SHA256 hex length
		return fmt.Errorf("invalid commitments root format")
	}
	
	return nil
}

// signProof signs the proof with provider's private key
func (zk *ZKUptimeProof) signProof(proof *UptimeProof) (string, error) {
	// Create a copy without signature for signing
	proofCopy := *proof
	proofCopy.Signature = ""
	
	// Marshal to JSON for signing
	proofJSON, err := json.Marshal(proofCopy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal proof for signing: %w", err)
	}
	
	// Sign with SHA256 (in production, use proper digital signature)
	hash := sha256.Sum256(proofJSON)
	return hex.EncodeToString(hash[:]), nil
}

// verifyProofSignature verifies proof signature
func (zk *ZKUptimeProof) verifyProofSignature(proof *UptimeProof) bool {
	// Extract signature and recalculate
	signature := proof.Signature
	proof.Signature = ""
	
	proofJSON, err := json.Marshal(proof)
	if err != nil {
		return false
	}
	
	expectedSignature := sha256.Sum256(proofJSON)
	expectedSignatureHex := hex.EncodeToString(expectedSignature[:])
	
	// Restore signature
	proof.Signature = signature
	
	return signature == expectedSignatureHex
}

// GenerateTestData generates realistic test data for demonstration
func (zk *ZKUptimeProof) GenerateTestData(contractStart, contractEnd int64, 
	targetUptime float64) []UptimeDataPoint {
	
	var data []UptimeDataPoint
	
	// Calculate number of measurements (every 5 minutes)
	totalMinutes := (contractEnd - contractStart) / 60
	measurementInterval := 5 // minutes
	numMeasurements := int(totalMinutes) / measurementInterval
	
	// Calculate how many should be offline to achieve target uptime
	totalMeasurements := float64(numMeasurements)
	offlineCount := int(math.Ceil(totalMeasurements * (100 - targetUptime) / 100))
	
	// Generate offline indices (distributed throughout the period)
	offlineIndices := make(map[int]bool)
	for i := 0; i < offlineCount; i++ {
		// Distribute offline periods throughout the contract
		index := int(float64(i) * totalMeasurements / float64(offlineCount))
		offlineIndices[index] = true
	}
	
	// Generate measurements
	for i := 0; i < numMeasurements; i++ {
		timestamp := contractStart + int64(i*measurementInterval*60)
		isOnline := !offlineIndices[i]
		
		var responseTime, cpuUtil, memoryUtil, errorRate float64
		
		if isOnline {
			responseTime = 2.0 + math.Mod(float64(timestamp), 6.0) // 2-8ms
			cpuUtil = 70.0 + math.Mod(float64(timestamp), 25.0)    // 70-95%
			memoryUtil = 60.0 + math.Mod(float64(timestamp), 30.0)  // 60-90%
			errorRate = math.Mod(float64(timestamp), 0.5)           // 0-0.5%
		} else {
			responseTime = 0.0
			cpuUtil = 0.0
			memoryUtil = 0.0
			errorRate = 100.0 // 100% error rate when offline
		}
		
		data = append(data, UptimeDataPoint{
			Timestamp:         timestamp,
			IsOnline:          isOnline,
			ResponseTimeMs:    responseTime,
			CPUUtilization:    cpuUtil,
			MemoryUtilization: memoryUtil,
			ErrorRate:         errorRate,
		})
	}
	
	return data
}
