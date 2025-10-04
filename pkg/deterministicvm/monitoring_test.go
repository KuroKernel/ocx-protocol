//go:build linux
// +build linux

package deterministicvm

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestPerfCyclesFallbacks(t *testing.T) {
	// Use a CPU-intensive process to ensure it consumes cycles
	cmd := exec.Command("sh", "-c", "while true; do :; done")
	if err := cmd.Start(); err != nil { 
		t.Fatal(err) 
	}
	defer cmd.Process.Kill()
	
	pid := cmd.Process.Pid
	t.Logf("Testing with PID: %d", pid)
	
	// Wait a bit for the process to start
	time.Sleep(10 * time.Millisecond)
	
	duration := 50 * time.Millisecond
	t.Logf("Duration: %v", duration)
	t.Logf("Duration nanoseconds: %d", duration.Nanoseconds())
	
	c := (&OSProcessVM{}).calculateCycles(pid, duration)
	t.Logf("Measured cycles: %d", c)
	
	if c == 0 {
		t.Fatal("cycles should be > 0")
	}
}

func TestMemoryUsage(t *testing.T) {
	mu, err := getMemoryUsage(os.Getpid())
	if err != nil {
		t.Skip("kernel may not expose smaps_rollup; status/statm fallback may fail in CI containers")
	}
	if mu.RSSBytes == 0 {
		t.Fatal("RSSBytes should be > 0")
	}
	t.Logf("Memory usage: RSS=%d bytes, PSS=%d bytes", mu.RSSBytes, mu.PSSBytes)
}

func TestCgroupManager(t *testing.T) {
	// Test cgroup manager creation (may fail due to permissions)
	cgManager, err := NewCgroupManager("test-vm")
	if err != nil {
		t.Logf("Cgroup manager creation failed (expected in some environments): %v", err)
		return
	}
	defer cgManager.Cleanup()
	
	// Test applying limits to current process
	limits := CgroupLimits{
		CPUQuotaMicros: 50000, // 50% of 1 CPU
		MemoryMaxBytes: 100 * 1024 * 1024, // 100MB
		PidsMax:        100,
	}
	
	err = cgManager.Apply(os.Getpid(), limits)
	if err != nil {
		t.Logf("Cgroup limits application failed (expected in some environments): %v", err)
		return
	}
	
	// Test memory usage reading
	memUsage, err := cgManager.GetMemoryUsage()
	if err != nil {
		t.Logf("Memory usage reading failed: %v", err)
	} else {
		t.Logf("Cgroup memory usage: %d bytes", memUsage)
	}
}

func TestRealMonitoringIntegration(t *testing.T) {
	// Test the integrated monitoring approach
	vm := &OSProcessVM{}
	
	// Create a simple test command
	cmd := exec.Command("echo", "test")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	
	pid := cmd.Process.Pid
	
	// Wait a bit for the process to complete
	time.Sleep(10 * time.Millisecond)
	
	// Test cycle calculation
	cycles := vm.calculateCycles(pid, 10*time.Millisecond)
	t.Logf("Process cycles: %d", cycles)
	
	// Test memory usage
	memory := vm.getActualMemoryUsage(pid, nil)
	t.Logf("Process memory: %d bytes", memory)
	
	// Wait for process to complete
	cmd.Wait()
}
