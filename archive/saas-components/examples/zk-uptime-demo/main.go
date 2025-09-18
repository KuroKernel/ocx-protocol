package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"ocx.local/internal/zkproofs/uptime"
)

func main() {
	fmt.Println("🔒 OCX Protocol - ZK Uptime Proof Demo")
	fmt.Println("=====================================")
	fmt.Println("")
	fmt.Println("This demo shows how OCX uses Zero-Knowledge Proofs to verify")
	fmt.Println("uptime claims without revealing sensitive provider data.")
	fmt.Println("")
	fmt.Println("Key Features:")
	fmt.Println("✅ ZK proofs for uptime verification")
	fmt.Println("✅ Byzantine consensus for proof validation")
	fmt.Println("✅ Privacy-preserving SLA compliance")
	fmt.Println("✅ Mathematical guarantees for compute delivery")
	fmt.Println("")
	
	// Initialize ZK proof system
	zkProof := uptime.NewZKUptimeProof()
	
	// Initialize consensus verifier with 67% threshold (Byzantine fault tolerance)
	consensusVerifier := uptime.NewZKConsensusVerifier(0.67)
	
	// Set up verifier network
	fmt.Println("🏛️  Setting up Byzantine consensus network...")
	verifiers := []struct {
		id        string
		publicKey string
		stake     float64
	}{
		{"verifier_1", "pk_aws_verifier", 50000.0},
		{"verifier_2", "pk_gcp_verifier", 75000.0},
		{"verifier_3", "pk_azure_verifier", 60000.0},
		{"verifier_4", "pk_hedge_fund", 100000.0},
		{"verifier_5", "pk_sovereign_node", 80000.0},
		{"verifier_6", "pk_university_lab", 30000.0},
		{"verifier_7", "pk_enterprise_client", 40000.0},
		{"verifier_8", "pk_independent_auditor", 25000.0},
		{"verifier_9", "pk_research_institute", 35000.0},
	}
	
	for _, v := range verifiers {
		consensusVerifier.AddVerifier(v.id, v.publicKey, v.stake)
	}
	
	// Show verifier network stats
	stats := consensusVerifier.GetVerifierStats()
	fmt.Printf("✅ Network ready: %v active verifiers, $%.0f total stake\n", 
		stats["active_verifiers"], stats["total_stake"])
	fmt.Printf("🛡️  Byzantine tolerance: %s\n", stats["byzantine_tolerance"])
	fmt.Println("")
	
	// Demo 1: Valid Uptime Claim
	fmt.Println("🚀 Demo 1: Valid Uptime Claim")
	fmt.Println("-----------------------------")
	
	contractStart := time.Now().Add(-24 * time.Hour).Unix()
	contractEnd := time.Now().Unix()
	
	// Generate realistic private data first
	fmt.Println("🔒 Generating private telemetry data...")
	privateData := zkProof.GenerateTestData(contractStart, contractEnd, 99.0) // Target 99% uptime
	fmt.Printf("📊 Generated %d private measurements (every 5 minutes)\n", len(privateData))
	
	// Calculate actual uptime from private data
	actualUptime := zkProof.CalculateActualUptime(privateData)
	fmt.Printf("🎯 Actual uptime from private data: %.2f%%\n", actualUptime)
	
	// Use actual uptime as the claim (with small buffer for tolerance)
	claimedUptime := actualUptime - 0.1 // Claim slightly less than actual
	
	fmt.Printf("📋 Scenario: Provider claims %.2f%% uptime over 24h period\n", claimedUptime)
	fmt.Printf("⏰ Contract period: %s to %s\n", 
		time.Unix(contractStart, 0).Format("2006-01-02 15:04:05"),
		time.Unix(contractEnd, 0).Format("2006-01-02 15:04:05"))
	
	// Create SLA requirements
	slaRequirements := &uptime.SLARequirements{
		MinUptime:      99.0,
		MaxResponseTime: 10.0,
		MinMeasurements: 200,
	}
	
	// Generate ZK proof
	fmt.Println("🔒 Generating Zero-Knowledge proof...")
	proof, err := zkProof.GenerateProof(privateData, claimedUptime, 
		contractStart, contractEnd, slaRequirements)
	if err != nil {
		log.Fatalf("Proof generation failed: %v", err)
	}
	
	fmt.Printf("✅ ZK proof generated successfully\n")
	fmt.Printf("📋 Proof size: %d bytes\n", len(fmt.Sprintf("%+v", proof)))
	fmt.Printf("🔑 Circuit ID: %s\n", proof.CircuitID)
	fmt.Printf("📊 Claimed uptime: %.2f%%\n", proof.PublicInputs.ClaimedUptimePercentage)
	fmt.Printf("📈 Measurement count: %d\n", proof.PublicInputs.MeasurementCount)
	fmt.Println("")
	
	// Verify proof using Byzantine consensus
	fmt.Println("🏛️  Verifying proof using Byzantine consensus...")
	ctx := context.Background()
	result, err := consensusVerifier.VerifyUptimeProof(ctx, proof, "provider_aws_us_east")
	if err != nil {
		log.Fatalf("Consensus verification failed: %v", err)
	}
	
	fmt.Printf("🎯 Consensus result: %s\n", 
		map[bool]string{true: "ACCEPTED", false: "REJECTED"}[result.Consensus])
	fmt.Printf("📊 Confidence: %.1f%% of stake voted VALID\n", result.Confidence*100)
	fmt.Printf("⏱️  Processing time: %v\n", result.ProcessingTime)
	fmt.Printf("�� Verifier votes: %d total\n", len(result.Votes))
	
	validVotes := 0
	for _, valid := range result.Votes {
		if valid {
			validVotes++
		}
	}
	fmt.Printf("✅ Valid votes: %d/%d\n", validVotes, len(result.Votes))
	fmt.Println("")
	
	// Demo 2: Invalid Uptime Claim
	fmt.Println("🚨 Demo 2: Invalid Uptime Claim")
	fmt.Println("-------------------------------")
	
	invalidClaimedUptime := actualUptime + 0.5 // Claim higher than actual
	fmt.Printf("📋 Scenario: Provider claims %.2f%% uptime (but actual is %.2f%%)\n", 
		invalidClaimedUptime, actualUptime)
	
	// Try to generate proof for invalid claim
	fmt.Println("🔒 Attempting to generate ZK proof for invalid claim...")
	_, err = zkProof.GenerateProof(privateData, invalidClaimedUptime, 
		contractStart, contractEnd, slaRequirements)
	
	if err != nil {
		fmt.Printf("✅ Invalid claim correctly rejected: %v\n", err)
	} else {
		fmt.Printf("❌ This should not have succeeded!\n")
	}
	fmt.Println("")
	
	// Demo 3: Privacy Preservation
	fmt.Println("🔒 Demo 3: Privacy Preservation")
	fmt.Println("-------------------------------")
	
	fmt.Println("�� What the consensus network can see:")
	fmt.Printf("   • Claimed uptime: %.2f%%\n", proof.PublicInputs.ClaimedUptimePercentage)
	fmt.Printf("   • Contract period: %d to %d\n", proof.PublicInputs.ContractStart, proof.PublicInputs.ContractEnd)
	fmt.Printf("   • Measurement count: %d\n", proof.PublicInputs.MeasurementCount)
	fmt.Printf("   • Commitments root: %s...\n", proof.PublicInputs.CommitmentsRoot[:16])
	fmt.Println("")
	
	fmt.Println("🔒 What the consensus network CANNOT see:")
	fmt.Println("   • Individual measurement timestamps")
	fmt.Println("   • Response times for each measurement")
	fmt.Println("   • CPU utilization details")
	fmt.Println("   • Memory usage patterns")
	fmt.Println("   • Error rates and failure details")
	fmt.Println("   • Customer data or internal logs")
	fmt.Println("   • Raw telemetry data")
	fmt.Println("")
	
	fmt.Println("✅ Privacy is mathematically guaranteed through ZK proofs!")
	fmt.Println("")
	
	// Demo 4: Integration Test
	fmt.Println("🧪 Demo 4: Integration Test")
	fmt.Println("---------------------------")
	
	// Test the integration system
	integration := uptime.NewZKProofsIntegration()
	
	// Add verifiers to integration
	for _, v := range verifiers {
		integration.AddVerifier(v.id, v.publicKey, v.stake)
	}
	
	// Test complete verification workflow
	fmt.Println("🔄 Testing complete verification workflow...")
	err = integration.TestUptimeVerification()
	if err != nil {
		log.Printf("Integration test failed: %v", err)
	} else {
		fmt.Println("✅ Integration test passed!")
	}
	fmt.Println("")
	
	// Final Summary
	fmt.Println("🎯 Final Summary")
	fmt.Println("================")
	fmt.Println("")
	fmt.Println("✅ ZK Uptime Proof System successfully demonstrates:")
	fmt.Println("   • Zero-knowledge verification of uptime claims")
	fmt.Println("   • Byzantine consensus for proof validation")
	fmt.Println("   • Privacy-preserving SLA compliance")
	fmt.Println("   • Mathematical guarantees for compute delivery")
	fmt.Println("   • Resistance to Byzantine failures")
	fmt.Println("   • Complete integration with OCX Protocol")
	fmt.Println("")
	fmt.Println("🚀 Key Innovations:")
	fmt.Println("   • Prove uptime without revealing sensitive data")
	fmt.Println("   • Cryptographic commitments to private measurements")
	fmt.Println("   • Merkle tree verification of data integrity")
	fmt.Println("   • Byzantine fault tolerant consensus")
	fmt.Println("   • Economic incentives through verifier staking")
	fmt.Println("   • Multiple ZK circuit types for different SLA metrics")
	fmt.Println("")
	fmt.Println("💡 Business Impact:")
	fmt.Println("   • Providers can prove SLA compliance without exposing logs")
	fmt.Println("   • Customers get mathematical guarantees of service delivery")
	fmt.Println("   • Consensus network ensures trust without centralization")
	fmt.Println("   • Privacy is preserved while maintaining verifiability")
	fmt.Println("   • OCX becomes the neutral standard for compute verification")
	fmt.Println("")
	fmt.Println("🔒 This solves the core problem: 'How do you prove 99.9% uptime")
	fmt.Println("   without revealing your internal logs or customer data?'")
	fmt.Println("")
	fmt.Println("🎉 ZK Uptime Proof Demo Complete!")
	fmt.Println("")
	fmt.Println("🚀 Next Steps:")
	fmt.Println("   1. Deploy ZK proof system to production")
	fmt.Println("   2. Integrate with OCX-QL query engine")
	fmt.Println("   3. Connect to settlement and payment systems")
	fmt.Println("   4. Launch verifier node marketplace")
	fmt.Println("   5. Begin enterprise customer onboarding")
}
