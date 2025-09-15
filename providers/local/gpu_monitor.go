// gpu_monitor.go - GPU Monitoring Service for RTX 5060
// Monitors GPU usage, temperature, and performance metrics

package local

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GPUMetrics represents real-time GPU metrics
type GPUMetrics struct {
	Timestamp     time.Time `json:"timestamp"`
	Utilization   int       `json:"utilization_percent"`
	Temperature   int       `json:"temperature_c"`
	MemoryUsed    int       `json:"memory_used_mb"`
	MemoryTotal   int       `json:"memory_total_mb"`
	PowerUsage    int       `json:"power_usage_w"`
	ClockGraphics int       `json:"clock_graphics_mhz"`
	ClockMemory   int       `json:"clock_memory_mhz"`
	FanSpeed      int       `json:"fan_speed_percent"`
	Processes     []Process `json:"processes"`
}

// Process represents a GPU process
type Process struct {
	PID         int    `json:"pid"`
	Name        string `json:"name"`
	MemoryUsed  int    `json:"memory_used_mb"`
	GPUUtil     int    `json:"gpu_utilization_percent"`
}

// GPUMonitor monitors the RTX 5060 GPU
type GPUMonitor struct {
	interval time.Duration
	stopCh   chan bool
	metrics  chan GPUMetrics
}

// NewGPUMonitor creates a new GPU monitor
func NewGPUMonitor(interval time.Duration) *GPUMonitor {
	return &GPUMonitor{
		interval: interval,
		stopCh:   make(chan bool),
		metrics:  make(chan GPUMetrics, 100),
	}
}

// Start starts monitoring the GPU
func (m *GPUMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	log.Printf("GPU monitoring started with %v interval", m.interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("GPU monitoring stopped by context")
			return
		case <-m.stopCh:
			log.Println("GPU monitoring stopped")
			return
		case <-ticker.C:
			metrics, err := m.collectMetrics()
			if err != nil {
				log.Printf("Failed to collect GPU metrics: %v", err)
				continue
			}

			select {
			case m.metrics <- metrics:
			default:
				// Channel full, skip this metric
			}
		}
	}
}

// Stop stops monitoring
func (m *GPUMonitor) Stop() {
	close(m.stopCh)
}

// GetMetrics returns the latest metrics
func (m *GPUMonitor) GetMetrics() <-chan GPUMetrics {
	return m.metrics
}

// collectMetrics collects current GPU metrics
func (m *GPUMonitor) collectMetrics() (GPUMetrics, error) {
	// Get basic GPU info
	cmd := exec.Command("nvidia-smi", 
		"--query-gpu=utilization.gpu,temperature.gpu,memory.used,memory.total,power.draw,clocks.gr,clocks.mem,fan.speed",
		"--format=csv,noheader,nounits")
	
	output, err := cmd.Output()
	if err != nil {
		return GPUMetrics{}, fmt.Errorf("failed to get GPU metrics: %w", err)
	}

	// Parse the output
	parts := strings.Split(strings.TrimSpace(string(output)), ", ")
	if len(parts) < 8 {
		return GPUMetrics{}, fmt.Errorf("unexpected GPU metrics format")
	}

	utilization, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	temperature, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	memoryUsed, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	memoryTotal, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
	powerUsage, _ := strconv.Atoi(strings.TrimSpace(parts[4]))
	clockGraphics, _ := strconv.Atoi(strings.TrimSpace(parts[5]))
	clockMemory, _ := strconv.Atoi(strings.TrimSpace(parts[6]))
	fanSpeed, _ := strconv.Atoi(strings.TrimSpace(parts[7]))

	// Get running processes
	processes, err := m.getGPUProcesses()
	if err != nil {
		log.Printf("Failed to get GPU processes: %v", err)
		processes = []Process{}
	}

	return GPUMetrics{
		Timestamp:     time.Now(),
		Utilization:   utilization,
		Temperature:   temperature,
		MemoryUsed:    memoryUsed,
		MemoryTotal:   memoryTotal,
		PowerUsage:    powerUsage,
		ClockGraphics: clockGraphics,
		ClockMemory:   clockMemory,
		FanSpeed:      fanSpeed,
		Processes:     processes,
	}, nil
}

// getGPUProcesses gets processes using the GPU
func (m *GPUMonitor) getGPUProcesses() ([]Process, error) {
	cmd := exec.Command("nvidia-smi", 
		"--query-compute-apps=pid,process_name,used_memory,utilization.gpu",
		"--format=csv,noheader,nounits")
	
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var processes []Process
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		parts := strings.Split(line, ", ")
		if len(parts) < 4 {
			continue
		}

		pid, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		name := strings.TrimSpace(parts[1])
		memoryUsed, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
		gpuUtil, _ := strconv.Atoi(strings.TrimSpace(parts[3]))

		processes = append(processes, Process{
			PID:        pid,
			Name:       name,
			MemoryUsed: memoryUsed,
			GPUUtil:    gpuUtil,
		})
	}

	return processes, nil
}

// GetHealthStatus returns the health status of the GPU
func (m *GPUMonitor) GetHealthStatus() (string, error) {
	metrics, err := m.collectMetrics()
	if err != nil {
		return "unknown", err
	}

	// Check various health indicators
	if metrics.Temperature > 85 {
		return "overheating", nil
	}
	
	if metrics.Utilization > 95 {
		return "overloaded", nil
	}
	
	if metrics.MemoryUsed > metrics.MemoryTotal*95/100 {
		return "memory_full", nil
	}
	
	if metrics.PowerUsage > 200 { // RTX 5060 max power
		return "power_limit", nil
	}

	return "healthy", nil
}

// GetPerformanceScore returns a performance score (0-100)
func (m *GPUMonitor) GetPerformanceScore() (int, error) {
	metrics, err := m.collectMetrics()
	if err != nil {
		return 0, err
	}

	// Calculate performance score based on utilization and temperature
	utilScore := metrics.Utilization
	tempScore := 100 - (metrics.Temperature - 30) // 30°C = 100%, 130°C = 0%
	if tempScore < 0 {
		tempScore = 0
	}
	if tempScore > 100 {
		tempScore = 100
	}

	// Weighted average
	score := (utilScore*60 + tempScore*40) / 100
	return score, nil
}

// LogMetrics logs current metrics to console
func (m *GPUMonitor) LogMetrics() {
	metrics, err := m.collectMetrics()
	if err != nil {
		log.Printf("Failed to get metrics: %v", err)
		return
	}

	log.Printf("GPU Metrics - Util: %d%%, Temp: %d°C, Memory: %d/%dMB, Power: %dW, Clock: %d/%dMHz, Fan: %d%%",
		metrics.Utilization, metrics.Temperature, metrics.MemoryUsed, metrics.MemoryTotal,
		metrics.PowerUsage, metrics.ClockGraphics, metrics.ClockMemory, metrics.FanSpeed)

	if len(metrics.Processes) > 0 {
		log.Printf("GPU Processes:")
		for _, proc := range metrics.Processes {
			log.Printf("  PID %d: %s (Memory: %dMB, Util: %d%%)",
				proc.PID, proc.Name, proc.MemoryUsed, proc.GPUUtil)
		}
	}
}

// ExportMetrics exports metrics to JSON
func (m *GPUMonitor) ExportMetrics() ([]byte, error) {
	metrics, err := m.collectMetrics()
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(metrics, "", "  ")
}
