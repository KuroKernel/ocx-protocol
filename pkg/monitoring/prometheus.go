package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMonitor provides Prometheus-based monitoring
type PrometheusMonitor struct {
	// Configuration
	config PrometheusConfig

	// Metrics
	systemMetrics   *SystemMetrics
	businessMetrics *BusinessMetrics
	customMetrics   *CustomMetrics

	// HTTP server
	server *http.Server

	// Control
	ctx    context.Context
	cancel context.CancelFunc
}

// PrometheusConfig defines configuration for Prometheus monitoring
type PrometheusConfig struct {
	// Server settings
	ListenAddress string `json:"listen_address"`
	MetricsPath   string `json:"metrics_path"`

	// Metric settings
	EnableSystemMetrics   bool `json:"enable_system_metrics"`
	EnableBusinessMetrics bool `json:"enable_business_metrics"`
	EnableCustomMetrics   bool `json:"enable_custom_metrics"`

	// Labels
	DefaultLabels map[string]string `json:"default_labels"`
}

// SystemMetrics represents system-level metrics
type SystemMetrics struct {
	// CPU metrics
	CPUUsagePercent prometheus.Gauge
	CPULoadAverage  prometheus.GaugeVec

	// Memory metrics
	MemoryUsageBytes   prometheus.Gauge
	MemoryUsagePercent prometheus.Gauge
	MemoryAvailable    prometheus.Gauge

	// Disk metrics
	DiskUsageBytes   prometheus.GaugeVec
	DiskUsagePercent prometheus.GaugeVec
	DiskIOReadBytes  prometheus.CounterVec
	DiskIOWriteBytes prometheus.CounterVec

	// Network metrics
	NetworkBytesReceived   prometheus.CounterVec
	NetworkBytesSent       prometheus.CounterVec
	NetworkPacketsReceived prometheus.CounterVec
	NetworkPacketsSent     prometheus.CounterVec

	// Process metrics
	ProcessCount       prometheus.Gauge
	ProcessMemoryUsage prometheus.Gauge
	ProcessCPUUsage    prometheus.Gauge
	ProcessOpenFiles   prometheus.Gauge
}

// BusinessMetrics represents business-level metrics
type BusinessMetrics struct {
	// Request metrics
	HTTPRequestsTotal   prometheus.CounterVec
	HTTPRequestDuration prometheus.HistogramVec
	HTTPRequestSize     prometheus.HistogramVec
	HTTPResponseSize    prometheus.HistogramVec

	// API metrics
	APIRequestsTotal prometheus.CounterVec
	APIDuration      prometheus.HistogramVec
	APIErrorsTotal   prometheus.CounterVec

	// Execution metrics
	ExecutionRequestsTotal prometheus.CounterVec
	ExecutionDuration      prometheus.HistogramVec
	ExecutionSuccessRate   prometheus.GaugeVec
	ExecutionErrorsTotal   prometheus.CounterVec

	// Verification metrics
	VerificationRequestsTotal prometheus.CounterVec
	VerificationDuration      prometheus.HistogramVec
	VerificationSuccessRate   prometheus.GaugeVec
	VerificationErrorsTotal   prometheus.CounterVec

	// Receipt metrics
	ReceiptsCreatedTotal   prometheus.CounterVec
	ReceiptsVerifiedTotal  prometheus.CounterVec
	ReceiptsStoredTotal    prometheus.CounterVec
	ReceiptsRetrievedTotal prometheus.CounterVec
}

// CustomMetrics represents custom application metrics
type CustomMetrics struct {
	// Security metrics
	SecurityAlertsTotal  prometheus.CounterVec
	SecurityScanDuration prometheus.HistogramVec
	SecurityScanResults  prometheus.GaugeVec

	// Performance metrics
	CacheHitRate         prometheus.GaugeVec
	CacheSize            prometheus.GaugeVec
	ConnectionPoolSize   prometheus.GaugeVec
	ConnectionPoolActive prometheus.GaugeVec

	// Database metrics
	DatabaseConnections   prometheus.GaugeVec
	DatabaseQueryDuration prometheus.HistogramVec
	DatabaseQueryErrors   prometheus.CounterVec

	// Key management metrics
	KeyRotationsTotal prometheus.CounterVec
	KeyExpirationTime prometheus.GaugeVec
	KeyUsageCount     prometheus.CounterVec
}

// NewPrometheusMonitor creates a new Prometheus monitor
func NewPrometheusMonitor(config PrometheusConfig) *PrometheusMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	pm := &PrometheusMonitor{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize metrics
	pm.initializeMetrics()

	// Start HTTP server
	pm.startHTTPServer()

	return pm
}

// initializeMetrics initializes all Prometheus metrics
func (pm *PrometheusMonitor) initializeMetrics() {
	if pm.config.EnableSystemMetrics {
		pm.initializeSystemMetrics()
	}

	if pm.config.EnableBusinessMetrics {
		pm.initializeBusinessMetrics()
	}

	if pm.config.EnableCustomMetrics {
		pm.initializeCustomMetrics()
	}
}

// initializeSystemMetrics initializes system-level metrics
func (pm *PrometheusMonitor) initializeSystemMetrics() {
	systemMetrics := &SystemMetrics{}

	// CPU metrics
	systemMetrics.CPUUsagePercent = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ocx_system_cpu_usage_percent",
		Help: "CPU usage percentage",
	})

	systemMetrics.CPULoadAverage = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_system_cpu_load_average",
		Help: "CPU load average",
	}, []string{"period"})

	// Memory metrics
	systemMetrics.MemoryUsageBytes = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ocx_system_memory_usage_bytes",
		Help: "Memory usage in bytes",
	})

	systemMetrics.MemoryUsagePercent = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ocx_system_memory_usage_percent",
		Help: "Memory usage percentage",
	})

	systemMetrics.MemoryAvailable = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ocx_system_memory_available_bytes",
		Help: "Available memory in bytes",
	})

	// Disk metrics
	systemMetrics.DiskUsageBytes = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_system_disk_usage_bytes",
		Help: "Disk usage in bytes",
	}, []string{"device", "mountpoint"})

	systemMetrics.DiskUsagePercent = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_system_disk_usage_percent",
		Help: "Disk usage percentage",
	}, []string{"device", "mountpoint"})

	// Network metrics
	systemMetrics.NetworkBytesReceived = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_system_network_bytes_received_total",
		Help: "Total network bytes received",
	}, []string{"interface"})

	systemMetrics.NetworkBytesSent = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_system_network_bytes_sent_total",
		Help: "Total network bytes sent",
	}, []string{"interface"})

	// Process metrics
	systemMetrics.ProcessCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ocx_system_process_count",
		Help: "Number of processes",
	})

	systemMetrics.ProcessMemoryUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ocx_system_process_memory_usage_bytes",
		Help: "Process memory usage in bytes",
	})

	systemMetrics.ProcessCPUUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ocx_system_process_cpu_usage_percent",
		Help: "Process CPU usage percentage",
	})

	pm.systemMetrics = systemMetrics
}

// initializeBusinessMetrics initializes business-level metrics
func (pm *PrometheusMonitor) initializeBusinessMetrics() {
	businessMetrics := &BusinessMetrics{}

	// HTTP metrics
	businessMetrics.HTTPRequestsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "path", "status_code"})

	businessMetrics.HTTPRequestDuration = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	businessMetrics.HTTPRequestSize = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_http_request_size_bytes",
		Help:    "HTTP request size in bytes",
		Buckets: prometheus.ExponentialBuckets(100, 10, 8),
	}, []string{"method", "path"})

	businessMetrics.HTTPResponseSize = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_http_response_size_bytes",
		Help:    "HTTP response size in bytes",
		Buckets: prometheus.ExponentialBuckets(100, 10, 8),
	}, []string{"method", "path"})

	// API metrics
	businessMetrics.APIRequestsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_api_requests_total",
		Help: "Total API requests",
	}, []string{"endpoint", "method", "status_code"})

	businessMetrics.APIDuration = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_api_duration_seconds",
		Help:    "API request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"endpoint", "method"})

	businessMetrics.APIErrorsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_api_errors_total",
		Help: "Total API errors",
	}, []string{"endpoint", "method", "error_type"})

	// Execution metrics
	businessMetrics.ExecutionRequestsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_execution_requests_total",
		Help: "Total execution requests",
	}, []string{"vm_type", "status"})

	businessMetrics.ExecutionDuration = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_execution_duration_seconds",
		Help:    "Execution duration in seconds",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~32s
	}, []string{"vm_type"})

	businessMetrics.ExecutionSuccessRate = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_execution_success_rate",
		Help: "Execution success rate",
	}, []string{"vm_type"})

	businessMetrics.ExecutionErrorsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_execution_errors_total",
		Help: "Total execution errors",
	}, []string{"vm_type", "error_type"})

	// Verification metrics
	businessMetrics.VerificationRequestsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_verification_requests_total",
		Help: "Total verification requests",
	}, []string{"status"})

	businessMetrics.VerificationDuration = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_verification_duration_seconds",
		Help:    "Verification duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{})

	businessMetrics.VerificationSuccessRate = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_verification_success_rate",
		Help: "Verification success rate",
	}, []string{})

	businessMetrics.VerificationErrorsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_verification_errors_total",
		Help: "Total verification errors",
	}, []string{"error_type"})

	// Receipt metrics
	businessMetrics.ReceiptsCreatedTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_receipts_created_total",
		Help: "Total receipts created",
	}, []string{"vm_type"})

	businessMetrics.ReceiptsVerifiedTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_receipts_verified_total",
		Help: "Total receipts verified",
	}, []string{"status"})

	businessMetrics.ReceiptsStoredTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_receipts_stored_total",
		Help: "Total receipts stored",
	}, []string{"store_type"})

	businessMetrics.ReceiptsRetrievedTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_receipts_retrieved_total",
		Help: "Total receipts retrieved",
	}, []string{"store_type"})

	pm.businessMetrics = businessMetrics
}

// initializeCustomMetrics initializes custom application metrics
func (pm *PrometheusMonitor) initializeCustomMetrics() {
	customMetrics := &CustomMetrics{}

	// Security metrics
	customMetrics.SecurityAlertsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_security_alerts_total",
		Help: "Total security alerts",
	}, []string{"severity", "source"})

	customMetrics.SecurityScanDuration = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_security_scan_duration_seconds",
		Help:    "Security scan duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"scan_type"})

	customMetrics.SecurityScanResults = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_security_scan_results",
		Help: "Security scan results",
	}, []string{"scan_type", "severity"})

	// Performance metrics
	customMetrics.CacheHitRate = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_cache_hit_rate",
		Help: "Cache hit rate",
	}, []string{"cache_type"})

	customMetrics.CacheSize = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_cache_size",
		Help: "Cache size",
	}, []string{"cache_type"})

	customMetrics.ConnectionPoolSize = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_connection_pool_size",
		Help: "Connection pool size",
	}, []string{"pool_type"})

	customMetrics.ConnectionPoolActive = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_connection_pool_active",
		Help: "Active connections in pool",
	}, []string{"pool_type"})

	// Database metrics
	customMetrics.DatabaseConnections = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_database_connections",
		Help: "Database connections",
	}, []string{"database", "status"})

	customMetrics.DatabaseQueryDuration = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_database_query_duration_seconds",
		Help:    "Database query duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"database", "query_type"})

	customMetrics.DatabaseQueryErrors = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_database_query_errors_total",
		Help: "Total database query errors",
	}, []string{"database", "error_type"})

	// Key management metrics
	customMetrics.KeyRotationsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_key_rotations_total",
		Help: "Total key rotations",
	}, []string{"key_type"})

	customMetrics.KeyExpirationTime = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_key_expiration_timestamp",
		Help: "Key expiration timestamp",
	}, []string{"key_id"})

	customMetrics.KeyUsageCount = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_key_usage_total",
		Help: "Total key usage count",
	}, []string{"key_id", "operation"})

	pm.customMetrics = customMetrics
}

// startHTTPServer starts the HTTP server for metrics
func (pm *PrometheusMonitor) startHTTPServer() {
	mux := http.NewServeMux()
	mux.Handle(pm.config.MetricsPath, promhttp.Handler())

	pm.server = &http.Server{
		Addr:    pm.config.ListenAddress,
		Handler: mux,
	}

	go func() {
		if err := pm.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Prometheus metrics server error: %v\n", err)
		}
	}()
}

// GetSystemMetrics returns system metrics
func (pm *PrometheusMonitor) GetSystemMetrics() *SystemMetrics {
	return pm.systemMetrics
}

// GetBusinessMetrics returns business metrics
func (pm *PrometheusMonitor) GetBusinessMetrics() *BusinessMetrics {
	return pm.businessMetrics
}

// GetCustomMetrics returns custom metrics
func (pm *PrometheusMonitor) GetCustomMetrics() *CustomMetrics {
	return pm.customMetrics
}

// RecordHTTPRequest records an HTTP request
func (pm *PrometheusMonitor) RecordHTTPRequest(method, path, statusCode string, duration time.Duration, requestSize, responseSize int64) {
	businessMetrics := pm.GetBusinessMetrics()
	if businessMetrics == nil {
		return
	}

	businessMetrics.HTTPRequestsTotal.WithLabelValues(method, path, statusCode).Inc()
	businessMetrics.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
	businessMetrics.HTTPRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
	businessMetrics.HTTPResponseSize.WithLabelValues(method, path).Observe(float64(responseSize))
}

// RecordAPIRequest records an API request
func (pm *PrometheusMonitor) RecordAPIRequest(endpoint, method, statusCode string, duration time.Duration) {
	businessMetrics := pm.GetBusinessMetrics()
	if businessMetrics == nil {
		return
	}

	businessMetrics.APIRequestsTotal.WithLabelValues(endpoint, method, statusCode).Inc()
	businessMetrics.APIDuration.WithLabelValues(endpoint, method).Observe(duration.Seconds())
}

// RecordAPIError records an API error
func (pm *PrometheusMonitor) RecordAPIError(endpoint, method, errorType string) {
	businessMetrics := pm.GetBusinessMetrics()
	if businessMetrics == nil {
		return
	}

	businessMetrics.APIErrorsTotal.WithLabelValues(endpoint, method, errorType).Inc()
}

// RecordExecution records an execution
func (pm *PrometheusMonitor) RecordExecution(vmType, status string, duration time.Duration) {
	businessMetrics := pm.GetBusinessMetrics()
	if businessMetrics == nil {
		return
	}

	businessMetrics.ExecutionRequestsTotal.WithLabelValues(vmType, status).Inc()
	businessMetrics.ExecutionDuration.WithLabelValues(vmType).Observe(duration.Seconds())
}

// RecordExecutionError records an execution error
func (pm *PrometheusMonitor) RecordExecutionError(vmType, errorType string) {
	businessMetrics := pm.GetBusinessMetrics()
	if businessMetrics == nil {
		return
	}

	businessMetrics.ExecutionErrorsTotal.WithLabelValues(vmType, errorType).Inc()
}

// RecordVerification records a verification
func (pm *PrometheusMonitor) RecordVerification(status string, duration time.Duration) {
	businessMetrics := pm.GetBusinessMetrics()
	if businessMetrics == nil {
		return
	}

	businessMetrics.VerificationRequestsTotal.WithLabelValues(status).Inc()
	businessMetrics.VerificationDuration.WithLabelValues().Observe(duration.Seconds())
}

// RecordVerificationError records a verification error
func (pm *PrometheusMonitor) RecordVerificationError(errorType string) {
	businessMetrics := pm.GetBusinessMetrics()
	if businessMetrics == nil {
		return
	}

	businessMetrics.VerificationErrorsTotal.WithLabelValues(errorType).Inc()
}

// RecordSecurityAlert records a security alert
func (pm *PrometheusMonitor) RecordSecurityAlert(severity, source string) {
	customMetrics := pm.GetCustomMetrics()
	if customMetrics == nil {
		return
	}

	customMetrics.SecurityAlertsTotal.WithLabelValues(severity, source).Inc()
}

// RecordSecurityScan records a security scan
func (pm *PrometheusMonitor) RecordSecurityScan(scanType string, duration time.Duration, results map[string]int) {
	customMetrics := pm.GetCustomMetrics()
	if customMetrics == nil {
		return
	}

	customMetrics.SecurityScanDuration.WithLabelValues(scanType).Observe(duration.Seconds())

	for severity, count := range results {
		customMetrics.SecurityScanResults.WithLabelValues(scanType, severity).Set(float64(count))
	}
}

// RecordCacheMetrics records cache metrics
func (pm *PrometheusMonitor) RecordCacheMetrics(cacheType string, hitRate float64, size int64) {
	customMetrics := pm.GetCustomMetrics()
	if customMetrics == nil {
		return
	}

	customMetrics.CacheHitRate.WithLabelValues(cacheType).Set(hitRate)
	customMetrics.CacheSize.WithLabelValues(cacheType).Set(float64(size))
}

// RecordKeyRotation records a key rotation
func (pm *PrometheusMonitor) RecordKeyRotation(keyType string) {
	customMetrics := pm.GetCustomMetrics()
	if customMetrics == nil {
		return
	}

	customMetrics.KeyRotationsTotal.WithLabelValues(keyType).Inc()
}

// RecordKeyUsage records key usage
func (pm *PrometheusMonitor) RecordKeyUsage(keyID, operation string) {
	customMetrics := pm.GetCustomMetrics()
	if customMetrics == nil {
		return
	}

	customMetrics.KeyUsageCount.WithLabelValues(keyID, operation).Inc()
}

// Stop stops the Prometheus monitor
func (pm *PrometheusMonitor) Stop() {
	pm.cancel()

	if pm.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		pm.server.Shutdown(ctx)
	}
}
