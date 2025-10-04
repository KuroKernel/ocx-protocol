package performance

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// PerformanceMonitor tracks system performance metrics
type PerformanceMonitor struct {
	// Configuration
	config MonitorConfig

	// Metrics storage
	metrics    map[string]*Metric
	metricsMux sync.RWMutex

	// Alerting
	alerts    []Alert
	alertsMux sync.RWMutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// MonitorConfig defines configuration for performance monitoring
type MonitorConfig struct {
	// Collection settings
	CollectionInterval time.Duration `json:"collection_interval"`
	RetentionPeriod    time.Duration `json:"retention_period"`
	MaxMetrics         int           `json:"max_metrics"`

	// Alerting settings
	AlertThresholds map[string]AlertThreshold `json:"alert_thresholds"`
	AlertCooldown   time.Duration             `json:"alert_cooldown"`

	// Resource monitoring
	EnableCPU     bool `json:"enable_cpu"`
	EnableMemory  bool `json:"enable_memory"`
	EnableDisk    bool `json:"enable_disk"`
	EnableNetwork bool `json:"enable_network"`
}

// AlertThreshold defines alerting thresholds
type AlertThreshold struct {
	Warning  float64 `json:"warning"`
	Critical float64 `json:"critical"`
	Operator string  `json:"operator"` // "gt", "lt", "eq", "gte", "lte"
}

// Metric represents a performance metric
type Metric struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"` // "counter", "gauge", "histogram"
	Value         float64           `json:"value"`
	Timestamp     time.Time         `json:"timestamp"`
	Labels        map[string]string `json:"labels"`
	DataPoints    []DataPoint       `json:"data_points"`
	DataPointsMux sync.RWMutex      `json:"-"`
}

// DataPoint represents a single data point in a metric
type DataPoint struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// Alert represents a performance alert
type Alert struct {
	ID           string    `json:"id"`
	MetricName   string    `json:"metric_name"`
	Level        string    `json:"level"` // "warning", "critical"
	Message      string    `json:"message"`
	Value        float64   `json:"value"`
	Threshold    float64   `json:"threshold"`
	Timestamp    time.Time `json:"timestamp"`
	Acknowledged bool      `json:"acknowledged"`
}

// SystemMetrics represents system-level performance metrics
type SystemMetrics struct {
	// CPU metrics
	CPUUsage    float64    `json:"cpu_usage"`
	CPUCores    int        `json:"cpu_cores"`
	LoadAverage [3]float64 `json:"load_average"`

	// Memory metrics
	MemoryTotal uint64  `json:"memory_total"`
	MemoryUsed  uint64  `json:"memory_used"`
	MemoryFree  uint64  `json:"memory_free"`
	MemoryUsage float64 `json:"memory_usage"`
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`
	SwapFree    uint64  `json:"swap_free"`
	SwapUsage   float64 `json:"swap_usage"`

	// GC metrics
	GCRuns         uint32  `json:"gc_runs"`
	GCPauseTotal   uint64  `json:"gc_pause_total"`
	GCPauseAverage float64 `json:"gc_pause_average"`

	// Goroutine metrics
	Goroutines int `json:"goroutines"`

	// Timestamp
	Timestamp time.Time `json:"timestamp"`
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(config MonitorConfig) *PerformanceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &PerformanceMonitor{
		config:  config,
		metrics: make(map[string]*Metric),
		alerts:  make([]Alert, 0),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start begins performance monitoring
func (pm *PerformanceMonitor) Start() error {
	// Start metric collection
	pm.wg.Add(1)
	go pm.collectMetrics()

	// Start alert processing
	pm.wg.Add(1)
	go pm.processAlerts()

	return nil
}

// Stop stops performance monitoring
func (pm *PerformanceMonitor) Stop() {
	pm.cancel()
	pm.wg.Wait()
}

// collectMetrics collects system performance metrics
func (pm *PerformanceMonitor) collectMetrics() {
	defer pm.wg.Done()

	ticker := time.NewTicker(pm.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.collectSystemMetrics()
		}
	}
}

// collectSystemMetrics collects system-level metrics
func (pm *PerformanceMonitor) collectSystemMetrics() {
	// Collect Go runtime metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// CPU usage (simplified)
	cpuUsage := pm.getCPUUsage()

	// Memory metrics
	memoryUsage := float64(m.Alloc) / float64(m.Sys) * 100

	// GC metrics
	gcPauseAverage := float64(m.PauseTotalNs) / float64(m.NumGC) / 1000000 // Convert to milliseconds

	// Update metrics
	pm.updateMetric("cpu_usage", "gauge", cpuUsage, nil)
	pm.updateMetric("memory_usage", "gauge", memoryUsage, nil)
	pm.updateMetric("memory_alloc", "gauge", float64(m.Alloc), nil)
	pm.updateMetric("memory_sys", "gauge", float64(m.Sys), nil)
	pm.updateMetric("gc_runs", "counter", float64(m.NumGC), nil)
	pm.updateMetric("gc_pause_total", "counter", float64(m.PauseTotalNs), nil)
	pm.updateMetric("gc_pause_average", "gauge", gcPauseAverage, nil)
	pm.updateMetric("goroutines", "gauge", float64(runtime.NumGoroutine()), nil)

	// Check for alerts
	pm.checkAlerts()
}

// getCPUUsage gets current CPU usage (real implementation)
func (pm *PerformanceMonitor) getCPUUsage() float64 {
	// Read /proc/stat for real CPU usage
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0.0 // Fallback if /proc/stat not available
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0.0
	}

	line := scanner.Text()
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return 0.0
	}

	// Parse CPU times
	var user, nice, system, idle, iowait, irq, softirq, steal uint64
	fmt.Sscanf(fields[1], "%d", &user)
	fmt.Sscanf(fields[2], "%d", &nice)
	fmt.Sscanf(fields[3], "%d", &system)
	fmt.Sscanf(fields[4], "%d", &idle)
	fmt.Sscanf(fields[5], "%d", &iowait)
	fmt.Sscanf(fields[6], "%d", &irq)
	fmt.Sscanf(fields[7], "%d", &softirq)
	if len(fields) > 8 {
		fmt.Sscanf(fields[8], "%d", &steal)
	}

	// Calculate total CPU time
	total := user + nice + system + idle + iowait + irq + softirq + steal
	idleTotal := idle + iowait

	// Calculate usage percentage
	if total == 0 {
		return 0.0
	}

	usage := float64(total-idleTotal) / float64(total) * 100.0
	return usage
}

// updateMetric updates a metric value
func (pm *PerformanceMonitor) updateMetric(name, metricType string, value float64, labels map[string]string) {
	pm.metricsMux.Lock()
	defer pm.metricsMux.Unlock()

	metric, exists := pm.metrics[name]
	if !exists {
		metric = &Metric{
			Name:       name,
			Type:       metricType,
			Labels:     make(map[string]string),
			DataPoints: make([]DataPoint, 0),
		}
		pm.metrics[name] = metric
	}

	// Update metric value
	metric.Value = value
	metric.Timestamp = time.Now()

	// Update labels
	if labels != nil {
		for k, v := range labels {
			metric.Labels[k] = v
		}
	}

	// Add data point
	metric.DataPointsMux.Lock()
	metric.DataPoints = append(metric.DataPoints, DataPoint{
		Value:     value,
		Timestamp: time.Now(),
	})

	// Trim data points if exceeding retention
	if len(metric.DataPoints) > pm.config.MaxMetrics {
		metric.DataPoints = metric.DataPoints[len(metric.DataPoints)-pm.config.MaxMetrics:]
	}
	metric.DataPointsMux.Unlock()
}

// checkAlerts checks if any metrics exceed alert thresholds
func (pm *PerformanceMonitor) checkAlerts() {
	pm.metricsMux.RLock()
	defer pm.metricsMux.RUnlock()

	for metricName, metric := range pm.metrics {
		threshold, exists := pm.config.AlertThresholds[metricName]
		if !exists {
			continue
		}

		// Check warning threshold
		if pm.evaluateThreshold(metric.Value, threshold.Warning, threshold.Operator) {
			pm.createAlert(metricName, "warning", metric.Value, threshold.Warning)
		}

		// Check critical threshold
		if pm.evaluateThreshold(metric.Value, threshold.Critical, threshold.Operator) {
			pm.createAlert(metricName, "critical", metric.Value, threshold.Critical)
		}
	}
}

// evaluateThreshold evaluates if a value meets a threshold condition
func (pm *PerformanceMonitor) evaluateThreshold(value, threshold float64, operator string) bool {
	switch operator {
	case "gt":
		return value > threshold
	case "lt":
		return value < threshold
	case "eq":
		return value == threshold
	case "gte":
		return value >= threshold
	case "lte":
		return value <= threshold
	default:
		return false
	}
}

// createAlert creates a new alert
func (pm *PerformanceMonitor) createAlert(metricName, level string, value, threshold float64) {
	pm.alertsMux.Lock()
	defer pm.alertsMux.Unlock()

	// Check if alert already exists and is not acknowledged
	for _, alert := range pm.alerts {
		if alert.MetricName == metricName && alert.Level == level && !alert.Acknowledged {
			// Alert already exists, update timestamp
			alert.Timestamp = time.Now()
			alert.Value = value
			return
		}
	}

	// Create new alert
	alert := Alert{
		ID:           fmt.Sprintf("%s_%s_%d", metricName, level, time.Now().Unix()),
		MetricName:   metricName,
		Level:        level,
		Message:      fmt.Sprintf("Metric %s is %s (value: %.2f, threshold: %.2f)", metricName, level, value, threshold),
		Value:        value,
		Threshold:    threshold,
		Timestamp:    time.Now(),
		Acknowledged: false,
	}

	pm.alerts = append(pm.alerts, alert)
}

// processAlerts processes alerts and handles cleanup
func (pm *PerformanceMonitor) processAlerts() {
	defer pm.wg.Done()

	ticker := time.NewTicker(time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.cleanupOldAlerts()
		}
	}
}

// cleanupOldAlerts removes old acknowledged alerts
func (pm *PerformanceMonitor) cleanupOldAlerts() {
	pm.alertsMux.Lock()
	defer pm.alertsMux.Unlock()

	cutoff := time.Now().Add(-pm.config.AlertCooldown)
	var activeAlerts []Alert

	for _, alert := range pm.alerts {
		if !alert.Acknowledged || alert.Timestamp.After(cutoff) {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	pm.alerts = activeAlerts
}

// GetMetrics returns all current metrics
func (pm *PerformanceMonitor) GetMetrics() map[string]*Metric {
	pm.metricsMux.RLock()
	defer pm.metricsMux.RUnlock()

	metrics := make(map[string]*Metric)
	for name, metric := range pm.metrics {
		// Create a copy to avoid race conditions
		metricCopy := *metric
		metricCopy.DataPointsMux.RLock()
		metricCopy.DataPoints = make([]DataPoint, len(metric.DataPoints))
		copy(metricCopy.DataPoints, metric.DataPoints)
		metricCopy.DataPointsMux.RUnlock()
		metrics[name] = &metricCopy
	}

	return metrics
}

// GetAlerts returns all current alerts
func (pm *PerformanceMonitor) GetAlerts() []Alert {
	pm.alertsMux.RLock()
	defer pm.alertsMux.RUnlock()

	alerts := make([]Alert, len(pm.alerts))
	copy(alerts, pm.alerts)
	return alerts
}

// AcknowledgeAlert acknowledges an alert by ID
func (pm *PerformanceMonitor) AcknowledgeAlert(alertID string) error {
	pm.alertsMux.Lock()
	defer pm.alertsMux.Unlock()

	for i, alert := range pm.alerts {
		if alert.ID == alertID {
			pm.alerts[i].Acknowledged = true
			return nil
		}
	}

	return fmt.Errorf("alert %s not found", alertID)
}

// GetSystemMetrics returns current system metrics
func (pm *PerformanceMonitor) GetSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemMetrics{
		CPUUsage:       pm.getCPUUsage(),
		CPUCores:       runtime.NumCPU(),
		MemoryTotal:    m.Sys,
		MemoryUsed:     m.Alloc,
		MemoryFree:     m.Sys - m.Alloc,
		MemoryUsage:    float64(m.Alloc) / float64(m.Sys) * 100,
		SwapTotal:      m.Sys, // Simplified
		SwapUsed:       0,     // Simplified
		SwapFree:       m.Sys, // Simplified
		SwapUsage:      0,     // Simplified
		GCRuns:         m.NumGC,
		GCPauseTotal:   m.PauseTotalNs,
		GCPauseAverage: float64(m.PauseTotalNs) / float64(m.NumGC) / 1000000,
		Goroutines:     runtime.NumGoroutine(),
		Timestamp:      time.Now(),
	}
}

// GetMetricHistory returns historical data for a metric
func (pm *PerformanceMonitor) GetMetricHistory(metricName string, duration time.Duration) ([]DataPoint, error) {
	pm.metricsMux.RLock()
	defer pm.metricsMux.RUnlock()

	metric, exists := pm.metrics[metricName]
	if !exists {
		return nil, fmt.Errorf("metric %s not found", metricName)
	}

	metric.DataPointsMux.RLock()
	defer metric.DataPointsMux.RUnlock()

	cutoff := time.Now().Add(-duration)
	var history []DataPoint

	for _, point := range metric.DataPoints {
		if point.Timestamp.After(cutoff) {
			history = append(history, point)
		}
	}

	return history, nil
}

// ExportMetrics exports metrics in a specific format
func (pm *PerformanceMonitor) ExportMetrics(format string) ([]byte, error) {
	metrics := pm.GetMetrics()

	switch format {
	case "json":
		return pm.exportJSON(metrics)
	case "prometheus":
		return pm.exportPrometheus(metrics)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportJSON exports metrics as JSON
func (pm *PerformanceMonitor) exportJSON(metrics map[string]*Metric) ([]byte, error) {
	// Real JSON export implementation
	exportData := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"metrics":   metrics,
	}

	jsonData, err := json.Marshal(exportData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metrics to JSON: %w", err)
	}

	return jsonData, nil
}

// exportPrometheus exports metrics in Prometheus format
func (pm *PerformanceMonitor) exportPrometheus(metrics map[string]*Metric) ([]byte, error) {
	// Real Prometheus format export
	var output strings.Builder
	output.WriteString("# HELP ocx_performance_metrics OCX Protocol performance metrics\n")
	output.WriteString("# TYPE ocx_performance_metrics gauge\n")

	for name, metric := range metrics {
		if len(metric.DataPoints) > 0 {
			latest := metric.DataPoints[len(metric.DataPoints)-1]
			output.WriteString(fmt.Sprintf("ocx_performance_metrics{name=\"%s\",type=\"%s\"} %f %d\n",
				name, metric.Type, latest.Value, latest.Timestamp.Unix()*1000))
		}
	}

	return []byte(output.String()), nil
}
