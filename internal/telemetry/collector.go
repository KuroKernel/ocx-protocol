// internal/telemetry/collector.go
package telemetry

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// TelemetryCollector gathers real-time GPU and system metrics for OCX sessions
type TelemetryCollector struct {
	db        *sql.DB
	sessionID string
	interval  time.Duration
	running   bool
}

// MetricsSnapshot contains comprehensive system telemetry for OCX sessions
type MetricsSnapshot struct {
	SessionID       string    `json:"session_id"`
	Timestamp       time.Time `json:"timestamp"`
	
	// GPU Metrics
	GPUUtilization  int     `json:"gpu_utilization_percent"`
	GPUMemoryUsed   int     `json:"gpu_memory_used_mb"`
	GPUMemoryTotal  int     `json:"gpu_memory_total_mb"`
	GPUTemperature  int     `json:"gpu_temperature_celsius"`
	GPUPowerDraw    int     `json:"gpu_power_draw_watts"`
	GPUClockCore    int     `json:"gpu_clock_core_mhz"`
	GPUClockMemory  int     `json:"gpu_clock_memory_mhz"`
	
	// System Metrics
	CPUUtilization  int     `json:"cpu_utilization_percent"`
	RAMUsed         float64 `json:"ram_used_gb"`
	RAMTotal        float64 `json:"ram_total_gb"`
	DiskIORead      float64 `json:"disk_io_read_mbps"`
	DiskIOWrite     float64 `json:"disk_io_write_mbps"`
	NetworkRX       float64 `json:"network_rx_mbps"`
	NetworkTX       float64 `json:"network_tx_mbps"`
	
	// Performance Metrics
	TrainingStepsSec    float64 `json:"training_steps_per_second"`
	InferenceTokensSec  float64 `json:"inference_tokens_per_second"`
	BatchSize          int     `json:"batch_size_processed"`
	MemoryPeak         int     `json:"memory_peak_mb"`
	
	// Integrity
	MetricsHash     string `json:"metrics_hash"`
	ProviderSig     string `json:"provider_signature"`
}

// SLACompliance tracks adherence to service level agreements
type SLACompliance struct {
	SessionID           string    `json:"session_id"`
	MinGPUUtilization   int       `json:"min_gpu_utilization"`
	MaxTemperature      int       `json:"max_temperature"`
	MaxDowntime         time.Duration `json:"max_downtime_minutes"`
	GuaranteedUptime    float64   `json:"guaranteed_uptime_percent"`
	
	// Actual Performance
	ActualAvgUtilization float64   `json:"actual_avg_utilization"`
	ActualMaxTemp        int       `json:"actual_max_temperature"`
	ActualUptime         float64   `json:"actual_uptime_percent"`
	TotalDowntime        time.Duration `json:"total_downtime"`
	
	// Compliance Status
	IsCompliant         bool      `json:"is_compliant"`
	Violations          []string  `json:"violations"`
	ComplianceScore     float64   `json:"compliance_score"`
}

// NewTelemetryCollector creates a metrics collection system for OCX sessions
func NewTelemetryCollector(db *sql.DB, sessionID string, interval time.Duration) *TelemetryCollector {
	return &TelemetryCollector{
		db:        db,
		sessionID: sessionID,
		interval:  interval,
		running:   false,
	}
}

// StartCollection begins continuous telemetry gathering for OCX session
func (tc *TelemetryCollector) StartCollection(ctx context.Context) error {
	if tc.running {
		return fmt.Errorf("collection already running for session %s", tc.sessionID)
	}
	
	tc.running = true
	
	// Create metrics table if not exists
	if err := tc.createMetricsTable(); err != nil {
		return fmt.Errorf("failed to create metrics table: %w", err)
	}
	
	go tc.collectMetrics(ctx)
	
	log.Printf("Started OCX telemetry collection for session %s", tc.sessionID)
	return nil
}

// StopCollection ends telemetry gathering
func (tc *TelemetryCollector) StopCollection() error {
	tc.running = false
	log.Printf("Stopped OCX telemetry collection for session %s", tc.sessionID)
	return nil
}

// collectMetrics runs the main collection loop
func (tc *TelemetryCollector) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(tc.interval)
	defer ticker.Stop()
	
	for tc.running {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics, err := tc.gatherMetrics()
			if err != nil {
				log.Printf("Error gathering OCX metrics for session %s: %v", tc.sessionID, err)
				continue
			}
			
			// Add cryptographic integrity
			metrics.MetricsHash = tc.calculateMetricsHash(metrics)
			metrics.ProviderSig = tc.signMetrics(metrics)
			
			if err := tc.storeMetrics(metrics); err != nil {
				log.Printf("Error storing OCX metrics for session %s: %v", tc.sessionID, err)
			}
		}
	}
}

// gatherMetrics collects current system state for OCX session
func (tc *TelemetryCollector) gatherMetrics() (*MetricsSnapshot, error) {
	metrics := &MetricsSnapshot{
		SessionID: tc.sessionID,
		Timestamp: time.Now(),
	}
	
	// GPU Metrics via nvidia-smi
	gpuMetrics, err := tc.getGPUMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get GPU metrics: %w", err)
	}
	
	metrics.GPUUtilization = gpuMetrics.Utilization
	metrics.GPUMemoryUsed = gpuMetrics.MemoryUsed
	metrics.GPUMemoryTotal = gpuMetrics.MemoryTotal
	metrics.GPUTemperature = gpuMetrics.Temperature
	metrics.GPUPowerDraw = gpuMetrics.PowerDraw
	metrics.GPUClockCore = gpuMetrics.ClockCore
	metrics.GPUClockMemory = gpuMetrics.ClockMemory
	
	// System Metrics
	systemMetrics, err := tc.getSystemMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to get system metrics: %w", err)
	}
	
	metrics.CPUUtilization = systemMetrics.CPUUtil
	metrics.RAMUsed = systemMetrics.RAMUsed
	metrics.RAMTotal = systemMetrics.RAMTotal
	metrics.DiskIORead = systemMetrics.DiskRead
	metrics.DiskIOWrite = systemMetrics.DiskWrite
	metrics.NetworkRX = systemMetrics.NetworkRX
	metrics.NetworkTX = systemMetrics.NetworkTX
	
	// Performance Metrics (if available)
	perfMetrics := tc.getPerformanceMetrics()
	metrics.TrainingStepsSec = perfMetrics.TrainingSteps
	metrics.InferenceTokensSec = perfMetrics.InferenceTokens
	metrics.BatchSize = perfMetrics.BatchSize
	metrics.MemoryPeak = perfMetrics.MemoryPeak
	
	return metrics, nil
}

// GPU metrics collection using nvidia-smi
type GPUMetrics struct {
	Utilization  int
	MemoryUsed   int
	MemoryTotal  int
	Temperature  int
	PowerDraw    int
	ClockCore    int
	ClockMemory  int
}

func (tc *TelemetryCollector) getGPUMetrics() (*GPUMetrics, error) {
	cmd := exec.Command("nvidia-smi", "--query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu,power.draw,clocks.current.graphics,clocks.current.memory", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi failed: %w", err)
	}
	
	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) < 7 {
		return nil, fmt.Errorf("unexpected nvidia-smi output format")
	}
	
	metrics := &GPUMetrics{}
	
	if val, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
		metrics.Utilization = val
	}
	if val, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
		metrics.MemoryUsed = val
	}
	if val, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
		metrics.MemoryTotal = val
	}
	if val, err := strconv.Atoi(strings.TrimSpace(parts[3])); err == nil {
		metrics.Temperature = val
	}
	if val, err := strconv.ParseFloat(strings.TrimSpace(parts[4]), 64); err == nil {
		metrics.PowerDraw = int(val)
	}
	if val, err := strconv.Atoi(strings.TrimSpace(parts[5])); err == nil {
		metrics.ClockCore = val
	}
	if val, err := strconv.Atoi(strings.TrimSpace(parts[6])); err == nil {
		metrics.ClockMemory = val
	}
	
	return metrics, nil
}

// System metrics collection
type SystemMetrics struct {
	CPUUtil    int
	RAMUsed    float64
	RAMTotal   float64
	DiskRead   float64
	DiskWrite  float64
	NetworkRX  float64
	NetworkTX  float64
}

func (tc *TelemetryCollector) getSystemMetrics() (*SystemMetrics, error) {
	metrics := &SystemMetrics{}
	
	// CPU utilization
	if cpu, err := tc.getCPUUtilization(); err == nil {
		metrics.CPUUtil = cpu
	}
	
	// Memory usage
	if ram, total, err := tc.getMemoryUsage(); err == nil {
		metrics.RAMUsed = ram
		metrics.RAMTotal = total
	}
	
	// Disk I/O
	if read, write, err := tc.getDiskIO(); err == nil {
		metrics.DiskRead = read
		metrics.DiskWrite = write
	}
	
	// Network I/O
	if rx, tx, err := tc.getNetworkIO(); err == nil {
		metrics.NetworkRX = rx
		metrics.NetworkTX = tx
	}
	
	return metrics, nil
}

// Performance metrics for ML workloads
type PerformanceMetrics struct {
	TrainingSteps   float64
	InferenceTokens float64
	BatchSize       int
	MemoryPeak      int
}

func (tc *TelemetryCollector) getPerformanceMetrics() *PerformanceMetrics {
	// This would integrate with ML frameworks to get actual performance data
	// For now, return placeholder data
	return &PerformanceMetrics{
		TrainingSteps:   0.0,
		InferenceTokens: 0.0,
		BatchSize:       0,
		MemoryPeak:      0,
	}
}

// Cryptographic integrity functions
func (tc *TelemetryCollector) calculateMetricsHash(metrics *MetricsSnapshot) string {
	// Create deterministic hash of metrics data
	data := fmt.Sprintf("%s_%d_%d_%d_%d_%d",
		metrics.SessionID,
		metrics.Timestamp.Unix(),
		metrics.GPUUtilization,
		metrics.GPUMemoryUsed,
		metrics.GPUTemperature,
		metrics.GPUPowerDraw,
	)
	
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (tc *TelemetryCollector) signMetrics(metrics *MetricsSnapshot) string {
	// This would use the provider's private key to sign the metrics
	// For now, return a placeholder signature
	return "ocx_provider_signature_placeholder"
}

// SLA Compliance checking for OCX sessions
func (tc *TelemetryCollector) CheckSLACompliance(sessionID string, requirements *SLACompliance) (*SLACompliance, error) {
	// Query historical metrics for the session
	metrics, err := tc.getSessionMetrics(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session metrics: %w", err)
	}
	
	compliance := &SLACompliance{
		SessionID:              sessionID,
		MinGPUUtilization:     requirements.MinGPUUtilization,
		MaxTemperature:        requirements.MaxTemperature,
		MaxDowntime:           requirements.MaxDowntime,
		GuaranteedUptime:      requirements.GuaranteedUptime,
		Violations:            []string{},
	}
	
	// Calculate actual performance
	totalMetrics := len(metrics)
	if totalMetrics == 0 {
		compliance.IsCompliant = false
		compliance.Violations = append(compliance.Violations, "No metrics data available")
		return compliance, nil
	}
	
	// Average utilization
	var totalUtil int
	var maxTemp int
	var downtimeCount int
	
	for _, metric := range metrics {
		totalUtil += metric.GPUUtilization
		if metric.GPUTemperature > maxTemp {
			maxTemp = metric.GPUTemperature
		}
		if metric.GPUUtilization == 0 {
			downtimeCount++
		}
	}
	
	compliance.ActualAvgUtilization = float64(totalUtil) / float64(totalMetrics)
	compliance.ActualMaxTemp = maxTemp
	compliance.ActualUptime = float64(totalMetrics-downtimeCount) / float64(totalMetrics) * 100
	compliance.TotalDowntime = time.Duration(downtimeCount) * tc.interval
	
	// Check violations
	if compliance.ActualAvgUtilization < float64(requirements.MinGPUUtilization) {
		compliance.Violations = append(compliance.Violations, 
			fmt.Sprintf("GPU utilization %.1f%% below minimum %d%%", 
				compliance.ActualAvgUtilization, requirements.MinGPUUtilization))
	}
	
	if maxTemp > requirements.MaxTemperature {
		compliance.Violations = append(compliance.Violations, 
			fmt.Sprintf("GPU temperature %d°C exceeded maximum %d°C", 
				maxTemp, requirements.MaxTemperature))
	}
	
	if compliance.ActualUptime < requirements.GuaranteedUptime {
		compliance.Violations = append(compliance.Violations, 
			fmt.Sprintf("Uptime %.1f%% below guaranteed %.1f%%", 
				compliance.ActualUptime, requirements.GuaranteedUptime))
	}
	
	if compliance.TotalDowntime > requirements.MaxDowntime {
		compliance.Violations = append(compliance.Violations, 
			fmt.Sprintf("Downtime %v exceeded maximum %v", 
				compliance.TotalDowntime, requirements.MaxDowntime))
	}
	
	compliance.IsCompliant = len(compliance.Violations) == 0
	
	// Calculate compliance score (0.0 to 1.0)
	score := 1.0
	if !compliance.IsCompliant {
		score = 0.5 // Base score for violations
		if compliance.ActualAvgUtilization >= float64(requirements.MinGPUUtilization)*0.8 {
			score += 0.2 // Partial credit for near-compliance
		}
		if compliance.ActualUptime >= requirements.GuaranteedUptime*0.9 {
			score += 0.2
		}
	}
	compliance.ComplianceScore = score
	
	return compliance, nil
}

// Database operations
func (tc *TelemetryCollector) createMetricsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS ocx_session_metrics (
		session_id TEXT NOT NULL,
		timestamp TIMESTAMPTZ NOT NULL,
		gpu_utilization_percent INTEGER,
		gpu_memory_used_mb INTEGER,
		gpu_memory_total_mb INTEGER,
		gpu_temperature_celsius INTEGER,
		gpu_power_draw_watts INTEGER,
		gpu_clock_core_mhz INTEGER,
		gpu_clock_memory_mhz INTEGER,
		cpu_utilization_percent INTEGER,
		ram_used_gb DECIMAL(8,3),
		ram_total_gb DECIMAL(8,3),
		disk_io_read_mbps DECIMAL(8,2),
		disk_io_write_mbps DECIMAL(8,2),
		network_rx_mbps DECIMAL(8,2),
		network_tx_mbps DECIMAL(8,2),
		training_steps_per_second DECIMAL(10,2),
		inference_tokens_per_second DECIMAL(10,2),
		batch_size_processed INTEGER,
		memory_peak_mb INTEGER,
		metrics_hash TEXT,
		provider_signature TEXT,
		PRIMARY KEY (session_id, timestamp)
	);
	
	CREATE INDEX IF NOT EXISTS idx_ocx_session_metrics_session 
	ON ocx_session_metrics (session_id, timestamp DESC);
	`
	
	_, err := tc.db.Exec(query)
	return err
}

func (tc *TelemetryCollector) storeMetrics(metrics *MetricsSnapshot) error {
	query := `
	INSERT INTO ocx_session_metrics (
		session_id, timestamp, gpu_utilization_percent, gpu_memory_used_mb,
		gpu_memory_total_mb, gpu_temperature_celsius, gpu_power_draw_watts,
		gpu_clock_core_mhz, gpu_clock_memory_mhz, cpu_utilization_percent,
		ram_used_gb, ram_total_gb, disk_io_read_mbps, disk_io_write_mbps,
		network_rx_mbps, network_tx_mbps, training_steps_per_second,
		inference_tokens_per_second, batch_size_processed, memory_peak_mb,
		metrics_hash, provider_signature
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`
	
	_, err := tc.db.Exec(query,
		metrics.SessionID, metrics.Timestamp, metrics.GPUUtilization,
		metrics.GPUMemoryUsed, metrics.GPUMemoryTotal, metrics.GPUTemperature,
		metrics.GPUPowerDraw, metrics.GPUClockCore, metrics.GPUClockMemory,
		metrics.CPUUtilization, metrics.RAMUsed, metrics.RAMTotal,
		metrics.DiskIORead, metrics.DiskIOWrite, metrics.NetworkRX,
		metrics.NetworkTX, metrics.TrainingStepsSec, metrics.InferenceTokensSec,
		metrics.BatchSize, metrics.MemoryPeak, metrics.MetricsHash,
		metrics.ProviderSig,
	)
	
	return err
}

func (tc *TelemetryCollector) getSessionMetrics(sessionID string) ([]*MetricsSnapshot, error) {
	query := `
	SELECT timestamp, gpu_utilization_percent, gpu_temperature_celsius
	FROM ocx_session_metrics 
	WHERE session_id = $1 
	ORDER BY timestamp ASC
	`
	
	rows, err := tc.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var metrics []*MetricsSnapshot
	for rows.Next() {
		metric := &MetricsSnapshot{SessionID: sessionID}
		err := rows.Scan(&metric.Timestamp, &metric.GPUUtilization, &metric.GPUTemperature)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	
	return metrics, nil
}

// System helper functions (simplified implementations)
func (tc *TelemetryCollector) getCPUUtilization() (int, error) {
	// Implementation would use /proc/stat or similar
	return tc.getRealCPUUsage()
}

func (tc *TelemetryCollector) getMemoryUsage() (float64, float64, error) {
	// Implementation would use /proc/meminfo
	return tc.getRealMemoryUsage()
}

func (tc *TelemetryCollector) getDiskIO() (float64, float64, error) {
	// Implementation would use /proc/diskstats
	return tc.getRealDiskIO()
}

func (tc *TelemetryCollector) getNetworkIO() (float64, float64, error) {
	// Implementation would use /proc/net/dev
	return tc.getRealNetworkIO()
}
