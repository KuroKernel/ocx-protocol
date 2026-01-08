// Package metrics provides Prometheus metrics for OCX Protocol
package metrics

import (
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMetrics holds all Prometheus metrics for OCX Protocol
type PrometheusMetrics struct {
	// Receipt metrics
	ReceiptsCreated   prometheus.Counter
	ReceiptsVerified  prometheus.Counter
	ReceiptsFailed    prometheus.Counter
	ReceiptLatency    prometheus.Histogram
	ReceiptSize       prometheus.Histogram

	// Batch verification metrics
	BatchSize         prometheus.Histogram
	BatchLatency      prometheus.Histogram
	BatchThroughput   prometheus.Gauge
	BatchQueueDepth   prometheus.Gauge

	// Replay protection metrics
	NonceChecks       prometheus.Counter
	NonceRejections   prometheus.Counter
	NonceStoreSize    prometheus.Gauge

	// Merkle tree metrics
	TreesBuilt        prometheus.Counter
	ProofsGenerated   prometheus.Counter
	ProofsVerified    prometheus.Counter
	TreeBuildLatency  prometheus.Histogram

	// Compression metrics
	CompressionRatio  prometheus.Histogram
	CompressLatency   prometheus.Histogram
	DecompressLatency prometheus.Histogram

	// VM execution metrics
	ExecutionsTotal   prometheus.Counter
	ExecutionLatency  prometheus.Histogram
	GasUsed           prometheus.Histogram
	ExecutionErrors   *prometheus.CounterVec

	// API metrics
	RequestsTotal     *prometheus.CounterVec
	RequestLatency    *prometheus.HistogramVec
	ActiveConnections prometheus.Gauge

	// System metrics
	GoroutineCount    prometheus.Gauge
	MemoryUsage       prometheus.Gauge
	DBSize            prometheus.Gauge

	// Info metric
	BuildInfo         *prometheus.GaugeVec
}

var (
	promInstance *PrometheusMetrics
	promOnce     sync.Once
)

// GetPrometheus returns the singleton Prometheus metrics instance
func GetPrometheus() *PrometheusMetrics {
	promOnce.Do(func() {
		promInstance = newPrometheusMetrics()
		go promInstance.collectSystemMetrics()
	})
	return promInstance
}

func newPrometheusMetrics() *PrometheusMetrics {
	m := &PrometheusMetrics{
		// Receipt metrics
		ReceiptsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "ocx",
			Subsystem: "receipt",
			Name:      "created_total",
			Help:      "Total number of receipts created",
		}),
		ReceiptsVerified: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "ocx",
			Subsystem: "receipt",
			Name:      "verified_total",
			Help:      "Total number of receipts successfully verified",
		}),
		ReceiptsFailed: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "ocx",
			Subsystem: "receipt",
			Name:      "failed_total",
			Help:      "Total number of receipts that failed verification",
		}),
		ReceiptLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "ocx",
			Subsystem: "receipt",
			Name:      "creation_latency_seconds",
			Help:      "Latency of receipt creation in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		}),
		ReceiptSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "ocx",
			Subsystem: "receipt",
			Name:      "size_bytes",
			Help:      "Size of receipts in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 2, 10),
		}),

		// Batch verification metrics
		BatchSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "ocx",
			Subsystem: "batch",
			Name:      "size",
			Help:      "Number of receipts in verification batch",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		}),
		BatchLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "ocx",
			Subsystem: "batch",
			Name:      "latency_seconds",
			Help:      "Latency of batch verification in seconds",
			Buckets:   []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}),
		BatchThroughput: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "ocx",
			Subsystem: "batch",
			Name:      "throughput_per_second",
			Help:      "Current batch verification throughput",
		}),
		BatchQueueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "ocx",
			Subsystem: "batch",
			Name:      "queue_depth",
			Help:      "Current depth of batch verification queue",
		}),

		// Replay protection metrics
		NonceChecks: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "ocx",
			Subsystem: "nonce",
			Name:      "checks_total",
			Help:      "Total number of nonce checks",
		}),
		NonceRejections: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "ocx",
			Subsystem: "nonce",
			Name:      "rejections_total",
			Help:      "Total number of nonce rejections (replay attacks blocked)",
		}),
		NonceStoreSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "ocx",
			Subsystem: "nonce",
			Name:      "store_size",
			Help:      "Current number of nonces in replay store",
		}),

		// Merkle tree metrics
		TreesBuilt: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "ocx",
			Subsystem: "merkle",
			Name:      "trees_built_total",
			Help:      "Total number of Merkle trees built",
		}),
		ProofsGenerated: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "ocx",
			Subsystem: "merkle",
			Name:      "proofs_generated_total",
			Help:      "Total number of Merkle proofs generated",
		}),
		ProofsVerified: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "ocx",
			Subsystem: "merkle",
			Name:      "proofs_verified_total",
			Help:      "Total number of Merkle proofs verified",
		}),
		TreeBuildLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "ocx",
			Subsystem: "merkle",
			Name:      "tree_build_latency_seconds",
			Help:      "Latency of Merkle tree construction",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		}),

		// Compression metrics
		CompressionRatio: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "ocx",
			Subsystem: "compression",
			Name:      "ratio",
			Help:      "Compression ratio (compressed/original)",
			Buckets:   []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		}),
		CompressLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "ocx",
			Subsystem: "compression",
			Name:      "compress_latency_seconds",
			Help:      "Latency of compression operation",
			Buckets:   []float64{.0001, .0005, .001, .005, .01, .025, .05},
		}),
		DecompressLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "ocx",
			Subsystem: "compression",
			Name:      "decompress_latency_seconds",
			Help:      "Latency of decompression operation",
			Buckets:   []float64{.0001, .0005, .001, .005, .01, .025, .05},
		}),

		// VM execution metrics
		ExecutionsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "ocx",
			Subsystem: "vm",
			Name:      "executions_total",
			Help:      "Total number of VM executions",
		}),
		ExecutionLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "ocx",
			Subsystem: "vm",
			Name:      "execution_latency_seconds",
			Help:      "Latency of VM execution",
			Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
		}),
		GasUsed: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "ocx",
			Subsystem: "vm",
			Name:      "gas_used",
			Help:      "Gas used per execution",
			Buckets:   prometheus.ExponentialBuckets(100, 2, 15),
		}),
		ExecutionErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "ocx",
			Subsystem: "vm",
			Name:      "execution_errors_total",
			Help:      "Total number of execution errors by type",
		}, []string{"error_type"}),

		// API metrics
		RequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "ocx",
			Subsystem: "api",
			Name:      "requests_total",
			Help:      "Total API requests by endpoint and status",
		}, []string{"method", "endpoint", "status"}),
		RequestLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "ocx",
			Subsystem: "api",
			Name:      "request_latency_seconds",
			Help:      "API request latency by endpoint",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		}, []string{"method", "endpoint"}),
		ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "ocx",
			Subsystem: "api",
			Name:      "active_connections",
			Help:      "Number of active API connections",
		}),

		// System metrics
		GoroutineCount: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "ocx",
			Subsystem: "system",
			Name:      "goroutines",
			Help:      "Number of goroutines",
		}),
		MemoryUsage: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "ocx",
			Subsystem: "system",
			Name:      "memory_bytes",
			Help:      "Memory usage in bytes",
		}),
		DBSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "ocx",
			Subsystem: "system",
			Name:      "db_size_bytes",
			Help:      "Database size in bytes",
		}),

		// Build info
		BuildInfo: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "ocx",
			Name:      "build_info",
			Help:      "Build information",
		}, []string{"version", "commit", "go_version"}),
	}

	// Set build info
	m.BuildInfo.WithLabelValues("1.0.0", "unknown", runtime.Version()).Set(1)

	return m
}

// Handler returns the Prometheus HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// collectSystemMetrics periodically collects system metrics
func (m *PrometheusMetrics) collectSystemMetrics() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		m.GoroutineCount.Set(float64(runtime.NumGoroutine()))
		m.MemoryUsage.Set(float64(memStats.Alloc))
	}
}

// RecordReceiptCreation records a receipt creation event
func (m *PrometheusMetrics) RecordReceiptCreation(duration time.Duration, size int) {
	m.ReceiptsCreated.Inc()
	m.ReceiptLatency.Observe(duration.Seconds())
	m.ReceiptSize.Observe(float64(size))
}

// RecordReceiptVerification records a receipt verification event
func (m *PrometheusMetrics) RecordReceiptVerification(success bool) {
	if success {
		m.ReceiptsVerified.Inc()
	} else {
		m.ReceiptsFailed.Inc()
	}
}

// RecordBatchVerification records a batch verification event
func (m *PrometheusMetrics) RecordBatchVerification(size int, duration time.Duration) {
	m.BatchSize.Observe(float64(size))
	m.BatchLatency.Observe(duration.Seconds())
	if duration > 0 {
		throughput := float64(size) / duration.Seconds()
		m.BatchThroughput.Set(throughput)
	}
}

// RecordNonceCheck records a nonce check event
func (m *PrometheusMetrics) RecordNonceCheck(rejected bool) {
	m.NonceChecks.Inc()
	if rejected {
		m.NonceRejections.Inc()
	}
}

// RecordMerkleTreeBuild records a Merkle tree build event
func (m *PrometheusMetrics) RecordMerkleTreeBuild(duration time.Duration) {
	m.TreesBuilt.Inc()
	m.TreeBuildLatency.Observe(duration.Seconds())
}

// RecordCompression records a compression event
func (m *PrometheusMetrics) RecordCompression(originalSize, compressedSize int, duration time.Duration) {
	if originalSize > 0 {
		ratio := float64(compressedSize) / float64(originalSize)
		m.CompressionRatio.Observe(ratio)
	}
	m.CompressLatency.Observe(duration.Seconds())
}

// RecordExecution records a VM execution event
func (m *PrometheusMetrics) RecordExecution(duration time.Duration, gasUsed uint64, err error) {
	m.ExecutionsTotal.Inc()
	m.ExecutionLatency.Observe(duration.Seconds())
	m.GasUsed.Observe(float64(gasUsed))
	if err != nil {
		m.ExecutionErrors.WithLabelValues(classifyError(err)).Inc()
	}
}

// RecordAPIRequest records an API request
func (m *PrometheusMetrics) RecordAPIRequest(method, endpoint, status string, duration time.Duration) {
	m.RequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	m.RequestLatency.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

func classifyError(err error) string {
	if err == nil {
		return "none"
	}
	errStr := err.Error()
	switch {
	case containsSubstr(errStr, "timeout"):
		return "timeout"
	case containsSubstr(errStr, "memory"):
		return "memory"
	case containsSubstr(errStr, "gas"):
		return "gas_limit"
	case containsSubstr(errStr, "permission"):
		return "permission"
	default:
		return "other"
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
