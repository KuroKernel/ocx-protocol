package performance

import (
	"context"
	"os"
	"testing"
	"time"

	"ocx.local/pkg/deterministicvm"
)

// BenchmarkMemoryPool tests memory pool performance
func BenchmarkMemoryPool(b *testing.B) {
	pool := NewMemoryPool()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Get buffer
			buffer := pool.GetBuffer(1024)
			
			// Simulate work
			for i := range buffer {
				buffer[i] = byte(i % 256)
			}
			
			// Return buffer
			pool.PutBuffer(buffer)
		}
	})
}

// BenchmarkMemoryPoolVsAllocation compares pool vs direct allocation
func BenchmarkMemoryPoolVsAllocation(b *testing.B) {
	pool := NewMemoryPool()
	
	b.Run("Pool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buffer := pool.GetBuffer(1024)
			pool.PutBuffer(buffer)
		}
	})
	
	b.Run("Direct", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buffer := make([]byte, 1024)
			_ = buffer
		}
	})
}

// BenchmarkVMPool tests VM pool performance
func BenchmarkVMPool(b *testing.B) {
	// Create test artifact
	artifactPath := createTestArtifact()
	defer os.Remove(artifactPath)
	
	config := DefaultVMPoolConfig()
	pool := NewVMPool(config, 10)
	defer pool.Close()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			result, err := pool.Execute(ctx, artifactPath)
			cancel()
			
			if err != nil {
				b.Fatal(err)
			}
			
			_ = result
		}
	})
}

// BenchmarkVMPoolVsDirect compares pool vs direct VM creation
func BenchmarkVMPoolVsDirect(b *testing.B) {
	// Create test artifact
	artifactPath := createTestArtifact()
	defer os.Remove(artifactPath)
	
	config := DefaultVMPoolConfig()
	
	b.Run("Pool", func(b *testing.B) {
		pool := NewVMPool(config, 10)
		defer pool.Close()
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			result, err := pool.Execute(ctx, artifactPath)
			cancel()
			
			if err != nil {
				b.Fatal(err)
			}
			
			_ = result
		}
	})
	
	b.Run("Direct", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vm := &deterministicvm.OSProcessVM{}
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			result, err := vm.Run(ctx, deterministicvm.VMConfig{
				ArtifactPath: artifactPath,
				WorkingDir:   config.WorkingDir,
				Env:          config.Env,
				Timeout:      config.Timeout,
				StrictMode:   config.StrictMode,
			})
			cancel()
			
			if err != nil {
				b.Fatal(err)
			}
			
			_ = result
		}
	})
}

// BenchmarkReceiptGeneration tests receipt generation performance
func BenchmarkReceiptGeneration(b *testing.B) {
	// Create test artifact
	artifactPath := createTestArtifact()
	defer os.Remove(artifactPath)
	
	config := DefaultVMPoolConfig()
	pool := NewVMPool(config, 10)
	defer pool.Close()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			result, err := pool.Execute(ctx, artifactPath)
			cancel()
			
			if err != nil {
				b.Fatal(err)
			}
			
			// Simulate receipt generation
			receipt := generateTestReceipt(result)
			_ = receipt
		}
	})
}

// BenchmarkConcurrentExecution tests concurrent execution performance
func BenchmarkConcurrentExecution(b *testing.B) {
	// Create test artifact
	artifactPath := createTestArtifact()
	defer os.Remove(artifactPath)
	
	config := DefaultVMPoolConfig()
	pool := NewVMPool(config, 100)
	defer pool.Close()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			result, err := pool.Execute(ctx, artifactPath)
			cancel()
			
			if err != nil {
				b.Fatal(err)
			}
			
			_ = result
		}
	})
}

// BenchmarkMemoryUsage tests memory usage under load
func BenchmarkMemoryUsage(b *testing.B) {
	pool := NewMemoryPool()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Allocate various buffer sizes
		buffers := make([][]byte, 100)
		for j := range buffers {
			buffers[j] = pool.GetBuffer(1024 * (j%10 + 1))
		}
		
		// Return buffers
		for _, buffer := range buffers {
			pool.PutBuffer(buffer)
		}
	}
	
	// Force GC to measure memory usage
	ForceGC()
}

// Helper functions

// createTestArtifact creates a simple test artifact
func createTestArtifact() string {
	// Create a simple shell script
	script := `#!/bin/sh
echo "Hello, World!"
exit 0
`
	
	// Write to temporary file
	tmpfile, err := os.CreateTemp("", "ocx-test-*.sh")
	if err != nil {
		panic(err)
	}
	
	if _, err := tmpfile.WriteString(script); err != nil {
		panic(err)
	}
	
	if err := tmpfile.Close(); err != nil {
		panic(err)
	}
	
	// Make executable
	if err := os.Chmod(tmpfile.Name(), 0755); err != nil {
		panic(err)
	}
	
	return tmpfile.Name()
}

// generateTestReceipt simulates receipt generation
func generateTestReceipt(result *deterministicvm.ExecutionResult) map[string]interface{} {
	return map[string]interface{}{
		"exit_code": result.ExitCode,
		"stdout":    string(result.Stdout),
		"stderr":    string(result.Stderr),
		"duration":  result.Duration.String(),
		"gas_used":  result.GasUsed,
	}
}

// BenchmarkPoolStats tests pool statistics collection
func BenchmarkPoolStats(b *testing.B) {
	pool := NewMemoryPool()
	
	// Generate some activity
	for i := 0; i < 1000; i++ {
		buffer := pool.GetBuffer(1024)
		pool.PutBuffer(buffer)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats := pool.GetStats()
		_ = stats
	}
}

// BenchmarkVMPoolStats tests VM pool statistics collection
func BenchmarkVMPoolStats(b *testing.B) {
	config := DefaultVMPoolConfig()
	pool := NewVMPool(config, 10)
	defer pool.Close()
	
	// Generate some activity
	artifactPath := createTestArtifact()
	defer os.Remove(artifactPath)
	
	for i := 0; i < 100; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool.Execute(ctx, artifactPath)
		cancel()
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats := pool.GetStats()
		_ = stats
	}
}
