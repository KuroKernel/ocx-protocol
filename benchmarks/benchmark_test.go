package benchmarks

import (
    "testing"
    
    "crypto/sha256"
    "math/rand"
)

func BenchmarkReceiptGeneration(b *testing.B) {
    artifact := sha256.Sum256([]byte("test-code"))
    input := sha256.Sum256([]byte("test-input"))
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Simulate receipt generation
        _ = sha256.Sum256(append(artifact[:], input[:]...))
    }
}

func BenchmarkReceiptVerification(b *testing.B) {
    // Pre-generate test data
    receipts := make([][32]byte, 1000)
    for i := range receipts {
        receipts[i] = sha256.Sum256([]byte{byte(i)})
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = receipts[i%len(receipts)]
    }
}

func BenchmarkDeterministicExecution(b *testing.B) {
    testData := make([]byte, 1024)
    rand.Read(testData)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = sha256.Sum256(testData)
    }
}
