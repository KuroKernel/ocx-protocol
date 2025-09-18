// consensus_integration_test.go - Integration tests for consensus module
package consensus

import (
	"context"
	"testing"
	"time"
)

func TestConsensusIntegration(t *testing.T) {
	ctx := context.Background()
	
	t.Run("blockchain consensus flow", func(t *testing.T) {
		// Create test blockchain
		blockchain := NewBlockchain()
		
		// Add some test blocks
		for i := 0; i < 5; i++ {
			block := &Block{
				Index:     i,
				Timestamp: time.Now(),
				Data:      []byte("test data"),
				Hash:      "",
				PrevHash:  "",
			}
			
			err := blockchain.AddBlock(block)
			if err != nil {
				t.Fatalf("Failed to add block: %v", err)
			}
		}
		
		// Verify blockchain integrity
		if !blockchain.IsValid() {
			t.Error("Blockchain is not valid")
		}
		
		// Test consensus mechanism
		consensus := NewConsensus(blockchain)
		
		// Simulate multiple nodes
		nodes := make([]*Node, 3)
		for i := range nodes {
			nodes[i] = NewNode(i, consensus)
		}
		
		// Start consensus process
		for _, node := range nodes {
			go node.Start(ctx)
		}
		
		// Wait for consensus
		time.Sleep(100 * time.Millisecond)
		
		// Verify all nodes have same blockchain
		for i := 1; i < len(nodes); i++ {
			if !nodes[0].GetBlockchain().IsEqual(nodes[i].GetBlockchain()) {
				t.Error("Nodes have different blockchains")
			}
		}
	})
	
	t.Run("reputation system integration", func(t *testing.T) {
		reputation := NewReputationSystem()
		
		// Add some providers
		providers := []string{"provider1", "provider2", "provider3"}
		for _, provider := range providers {
			reputation.AddProvider(provider)
		}
		
		// Simulate some transactions
		for i := 0; i < 10; i++ {
			provider := providers[i%len(providers)]
			success := i%3 != 0 // 2/3 success rate
			
			reputation.RecordTransaction(provider, success)
		}
		
		// Check reputation scores
		for _, provider := range providers {
			score := reputation.GetScore(provider)
			if score < 0 || score > 100 {
				t.Errorf("Invalid reputation score for %s: %f", provider, score)
			}
		}
	})
}

func TestCryptoIntegration(t *testing.T) {
	t.Run("key generation and signing", func(t *testing.T) {
		// Generate key pair
		keyPair, err := GenerateKeyPair()
		if err != nil {
			t.Fatalf("Failed to generate key pair: %v", err)
		}
		
		// Test message
		message := []byte("test message")
		
		// Sign message
		signature, err := SignMessage(message, keyPair.PrivateKey)
		if err != nil {
			t.Fatalf("Failed to sign message: %v", err)
		}
		
		// Verify signature
		valid, err := VerifySignature(message, signature, keyPair.PublicKey)
		if err != nil {
			t.Fatalf("Failed to verify signature: %v", err)
		}
		
		if !valid {
			t.Error("Signature verification failed")
		}
	})
	
	t.Run("hash functions", func(t *testing.T) {
		data := []byte("test data")
		
		// Test SHA256
		hash1 := SHA256Hash(data)
		hash2 := SHA256Hash(data)
		
		if string(hash1) != string(hash2) {
			t.Error("SHA256 hash is not deterministic")
		}
		
		// Test different data produces different hash
		hash3 := SHA256Hash([]byte("different data"))
		if string(hash1) == string(hash3) {
			t.Error("Different data produced same hash")
		}
	})
}
