// local_gpu_provider.go - Local GPU Provider for NVIDIA GPU Testing
package local

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LocalGPUProvider manages the local NVIDIA GPU
type LocalGPUProvider struct {
	providerID     string
	gpuAvailable   bool
	currentLease   string
	leaseStartTime time.Time
	mu             sync.RWMutex
	hostname       string
	username       string
	sshPort        int
}

// GPUInfo represents GPU information
type GPUInfo struct {
	Name           string  `json:"name"`
	MemoryTotal    int     `json:"memory_total_mb"`
	MemoryUsed     int     `json:"memory_used_mb"`
	DriverVersion  string  `json:"driver_version"`
	Temperature    int     `json:"temperature_c"`
	Utilization    int     `json:"utilization_percent"`
	PowerDraw      float64 `json:"power_draw_w"`
}

// LeaseRequest represents a lease request
type LeaseRequest struct {
	Hours    int    `json:"hours"`
	Memory   int    `json:"memory_gb,omitempty"`
	Priority string `json:"priority,omitempty"`
}

// NewLocalGPUProvider creates a new local GPU provider
func NewLocalGPUProvider() (*LocalGPUProvider, error) {
	// Get system information
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	username := os.Getenv("USER")
	if username == "" {
		username = "kurokernel"
	}

	provider := &LocalGPUProvider{
		providerID:   "local-nvidia-provider",
		gpuAvailable: true,
		hostname:     hostname,
		username:     username,
		sshPort:      22,
	}

	// Verify GPU is available
	if err := provider.verifyGPU(); err != nil {
		return nil, fmt.Errorf("GPU verification failed: %w", err)
	}

	log.Printf("Local GPU Provider initialized: %s", provider.providerID)
	return provider, nil
}

// verifyGPU checks if the NVIDIA GPU is available and working
func (p *LocalGPUProvider) verifyGPU() error {
	// Check if nvidia-smi is available
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total,driver_version", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("nvidia-smi not available: %w", err)
	}

	// Parse GPU information
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("no GPUs detected")
	}

	// Check if NVIDIA GPU is present
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "nvidia") {
			return nil
		}
	}

	return fmt.Errorf("no NVIDIA GPU found")
}

// GetGPUInfo returns current GPU information
func (p *LocalGPUProvider) GetGPUInfo() (*GPUInfo, error) {
	cmd := exec.Command("nvidia-smi", 
		"--query-gpu=name,memory.total,memory.used,driver_version,temperature.gpu,utilization.gpu,power.draw",
		"--format=csv,noheader,nounits")
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get GPU info: %w", err)
	}

	// Parse the output
	parts := strings.Split(strings.TrimSpace(string(output)), ", ")
	if len(parts) < 7 {
		return nil, fmt.Errorf("unexpected GPU info format")
	}

	name := strings.TrimSpace(parts[0])
	memoryTotal, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	memoryUsed, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	driverVersion := strings.TrimSpace(parts[3])
	temperature, _ := strconv.Atoi(strings.TrimSpace(parts[4]))
	utilization, _ := strconv.Atoi(strings.TrimSpace(parts[5]))
	powerDraw, _ := strconv.ParseFloat(strings.TrimSpace(parts[6]), 64)

	return &GPUInfo{
		Name:          name,
		MemoryTotal:   memoryTotal,
		MemoryUsed:    memoryUsed,
		DriverVersion: driverVersion,
		Temperature:   temperature,
		Utilization:   utilization,
		PowerDraw:     powerDraw,
	}, nil
}

// RequestLease requests a lease for GPU compute
func (p *LocalGPUProvider) RequestLease(ctx context.Context, req *LeaseRequest) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.gpuAvailable {
		return "", fmt.Errorf("GPU not available")
	}

	if p.currentLease != "" {
		return "", fmt.Errorf("GPU already leased")
	}

	// Generate lease ID
	leaseID := fmt.Sprintf("lease-%d", time.Now().Unix())
	p.currentLease = leaseID
	p.leaseStartTime = time.Now()

	log.Printf("GPU lease granted: %s for %d hours", leaseID, req.Hours)
	return leaseID, nil
}

// ReleaseLease releases a GPU lease
func (p *LocalGPUProvider) ReleaseLease(ctx context.Context, leaseID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentLease != leaseID {
		return fmt.Errorf("lease not found: %s", leaseID)
	}

	p.currentLease = ""
	p.leaseStartTime = time.Time{}

	log.Printf("GPU lease released: %s", leaseID)
	return nil
}

// GetLeaseStatus returns the status of a lease
func (p *LocalGPUProvider) GetLeaseStatus(ctx context.Context, leaseID string) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.currentLease == leaseID, nil
}

// GetProviderInfo returns provider information
func (p *LocalGPUProvider) GetProviderInfo() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"provider_id":    p.providerID,
		"gpu_available":  p.gpuAvailable,
		"current_lease":  p.currentLease,
		"hostname":       p.hostname,
		"username":       p.username,
		"ssh_port":       p.sshPort,
		"lease_duration": time.Since(p.leaseStartTime).String(),
	}
}

// HealthCheck performs a health check
func (p *LocalGPUProvider) HealthCheck() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.gpuAvailable {
		return fmt.Errorf("GPU not available")
	}

	// Check if nvidia-smi is still working
	cmd := exec.Command("nvidia-smi", "--query-gpu=name", "--format=csv,noheader")
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("nvidia-smi health check failed: %w", err)
	}

	return nil
}
