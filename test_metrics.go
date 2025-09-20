package main

import (
	"fmt"
	"time"
	"ocx.local/pkg/metrics"
)

func main() {
	fmt.Println("Testing metrics package...")
	
	// Test execution metrics
	metrics.RecordExecution(100, time.Millisecond, true)
	metrics.RecordExecution(200, 2*time.Millisecond, true)
	
	// Test verification metrics
	metrics.RecordVerification(time.Millisecond, true)
	metrics.RecordVerification(2*time.Millisecond, true)
	
	// Print results
	fmt.Printf("Execute counter: %d\n", metrics.ExecuteCounter.Value())
	fmt.Printf("Verify counter: %d\n", metrics.VerifyCounter.Value())
	fmt.Printf("Execute latency P99: %.3f\n", metrics.ExecuteLatency.P99())
	fmt.Printf("Verify latency P99: %.3f\n", metrics.VerifyLatency.P99())
}
