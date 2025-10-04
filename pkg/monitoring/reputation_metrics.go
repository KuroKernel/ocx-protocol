package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ReputationMetrics represents reputation system-specific metrics
type ReputationMetrics struct {
	// Computation metrics
	ComputeRequestsTotal   prometheus.CounterVec
	ComputeDuration        prometheus.HistogramVec
	ComputeErrorsTotal     prometheus.CounterVec
	ComputeSuccessRate     prometheus.GaugeVec

	// Score metrics
	TrustScoreDistribution prometheus.HistogramVec
	ConfidenceDistribution prometheus.HistogramVec
	PlatformScores         prometheus.GaugeVec

	// Verification metrics
	VerificationRequestsTotal prometheus.CounterVec
	VerificationDuration      prometheus.HistogramVec
	VerificationErrorsTotal   prometheus.CounterVec
	VerificationSuccessRate   prometheus.GaugeVec

	// Badge metrics
	BadgeRequestsTotal prometheus.CounterVec
	BadgeDuration      prometheus.HistogramVec
	BadgeStyle         prometheus.CounterVec

	// Platform metrics
	PlatformRequestsTotal   prometheus.CounterVec
	PlatformScoresCollected prometheus.CounterVec
	PlatformErrorsTotal     prometheus.CounterVec

	// Receipt metrics
	ReputationReceiptsTotal prometheus.CounterVec
	ReceiptGasUsed          prometheus.HistogramVec
	ReceiptSignatureDuration prometheus.HistogramVec

	// OAuth metrics
	OAuthRequestsTotal   prometheus.CounterVec
	OAuthSuccessTotal    prometheus.CounterVec
	OAuthErrorsTotal     prometheus.CounterVec
	OAuthDuration        prometheus.HistogramVec
	OAuthTokenRefreshes  prometheus.CounterVec

	// Cache metrics
	ReputationCacheHits   prometheus.CounterVec
	ReputationCacheMisses prometheus.CounterVec
	ReputationCacheSize   prometheus.Gauge

	// Rate limiting metrics
	RateLimitExceeded prometheus.CounterVec
	RateLimitRemaining prometheus.GaugeVec
}

// InitializeReputationMetrics initializes reputation-specific Prometheus metrics
func InitializeReputationMetrics() *ReputationMetrics {
	rm := &ReputationMetrics{}

	// Computation metrics
	rm.ComputeRequestsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_compute_requests_total",
		Help: "Total number of reputation compute requests",
	}, []string{"status", "user_id"})

	rm.ComputeDuration = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_reputation_compute_duration_milliseconds",
		Help:    "Duration of reputation computation in milliseconds",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 25, 50, 100, 250, 500, 1000},
	}, []string{"platforms"})

	rm.ComputeErrorsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_compute_errors_total",
		Help: "Total number of reputation compute errors",
	}, []string{"error_type"})

	rm.ComputeSuccessRate = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_reputation_compute_success_rate",
		Help: "Success rate of reputation computations (0-1)",
	}, []string{"interval"})

	// Score metrics
	rm.TrustScoreDistribution = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_reputation_trust_score",
		Help:    "Distribution of trust scores (0-100)",
		Buckets: []float64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	}, []string{"platform_count"})

	rm.ConfidenceDistribution = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_reputation_confidence",
		Help:    "Distribution of confidence values (0-1)",
		Buckets: []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
	}, []string{"platform_count"})

	rm.PlatformScores = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_reputation_platform_score",
		Help: "Current platform scores (0-100)",
	}, []string{"platform", "user_id"})

	// Verification metrics
	rm.VerificationRequestsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_verification_requests_total",
		Help: "Total number of reputation verification requests",
	}, []string{"status"})

	rm.VerificationDuration = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_reputation_verification_duration_milliseconds",
		Help:    "Duration of reputation verification in milliseconds",
		Buckets: []float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	}, []string{"platforms"})

	rm.VerificationErrorsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_verification_errors_total",
		Help: "Total number of reputation verification errors",
	}, []string{"error_type"})

	rm.VerificationSuccessRate = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_reputation_verification_success_rate",
		Help: "Success rate of reputation verifications (0-1)",
	}, []string{"interval"})

	// Badge metrics
	rm.BadgeRequestsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_badge_requests_total",
		Help: "Total number of badge generation requests",
	}, []string{"style", "status"})

	rm.BadgeDuration = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_reputation_badge_duration_milliseconds",
		Help:    "Duration of badge generation in milliseconds",
		Buckets: []float64{1, 2, 5, 10, 25, 50, 100},
	}, []string{"style"})

	rm.BadgeStyle = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_badge_style_total",
		Help: "Count of badge requests by style",
	}, []string{"style"})

	// Platform metrics
	rm.PlatformRequestsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_platform_requests_total",
		Help: "Total number of platform API requests",
	}, []string{"platform", "status"})

	rm.PlatformScoresCollected = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_platform_scores_collected_total",
		Help: "Total number of platform scores collected",
	}, []string{"platform"})

	rm.PlatformErrorsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_platform_errors_total",
		Help: "Total number of platform API errors",
	}, []string{"platform", "error_type"})

	// Receipt metrics
	rm.ReputationReceiptsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_receipts_total",
		Help: "Total number of reputation receipts generated",
	}, []string{"status"})

	rm.ReceiptGasUsed = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_reputation_receipt_gas_used",
		Help:    "Gas used for reputation receipt generation",
		Buckets: []float64{50, 100, 150, 200, 238, 250, 300, 400, 500},
	}, []string{"platforms"})

	rm.ReceiptSignatureDuration = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_reputation_receipt_signature_duration_microseconds",
		Help:    "Duration of Ed25519 signature generation in microseconds",
		Buckets: []float64{100, 250, 500, 750, 1000, 2500, 5000},
	}, []string{})

	// OAuth metrics
	rm.OAuthRequestsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_oauth_requests_total",
		Help: "Total number of OAuth requests",
	}, []string{"platform", "flow"})

	rm.OAuthSuccessTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_oauth_success_total",
		Help: "Total number of successful OAuth requests",
	}, []string{"platform", "flow"})

	rm.OAuthErrorsTotal = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_oauth_errors_total",
		Help: "Total number of OAuth errors",
	}, []string{"platform", "error_type"})

	rm.OAuthDuration = *promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocx_reputation_oauth_duration_milliseconds",
		Help:    "Duration of OAuth operations in milliseconds",
		Buckets: []float64{100, 250, 500, 1000, 2000, 5000, 10000},
	}, []string{"platform", "flow"})

	rm.OAuthTokenRefreshes = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_oauth_token_refreshes_total",
		Help: "Total number of OAuth token refreshes",
	}, []string{"platform"})

	// Cache metrics
	rm.ReputationCacheHits = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_cache_hits_total",
		Help: "Total number of reputation cache hits",
	}, []string{"cache_type"})

	rm.ReputationCacheMisses = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_cache_misses_total",
		Help: "Total number of reputation cache misses",
	}, []string{"cache_type"})

	rm.ReputationCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ocx_reputation_cache_size_bytes",
		Help: "Current size of reputation cache in bytes",
	})

	// Rate limiting metrics
	rm.RateLimitExceeded = *promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ocx_reputation_rate_limit_exceeded_total",
		Help: "Total number of rate limit exceeded events",
	}, []string{"platform", "user_id"})

	rm.RateLimitRemaining = *promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ocx_reputation_rate_limit_remaining",
		Help: "Remaining rate limit quota",
	}, []string{"platform", "user_id"})

	return rm
}

// RecordComputeRequest records a reputation compute request
func (rm *ReputationMetrics) RecordComputeRequest(status string, userID string) {
	rm.ComputeRequestsTotal.WithLabelValues(status, userID).Inc()
}

// RecordComputeDuration records the duration of a reputation computation
func (rm *ReputationMetrics) RecordComputeDuration(platforms string, durationMS float64) {
	rm.ComputeDuration.WithLabelValues(platforms).Observe(durationMS)
}

// RecordTrustScore records a trust score
func (rm *ReputationMetrics) RecordTrustScore(platformCount string, score float64) {
	rm.TrustScoreDistribution.WithLabelValues(platformCount).Observe(score)
}

// RecordConfidence records a confidence value
func (rm *ReputationMetrics) RecordConfidence(platformCount string, confidence float64) {
	rm.ConfidenceDistribution.WithLabelValues(platformCount).Observe(confidence)
}

// RecordBadgeRequest records a badge generation request
func (rm *ReputationMetrics) RecordBadgeRequest(style, status string) {
	rm.BadgeRequestsTotal.WithLabelValues(style, status).Inc()
	rm.BadgeStyle.WithLabelValues(style).Inc()
}

// RecordBadgeDuration records the duration of badge generation
func (rm *ReputationMetrics) RecordBadgeDuration(style string, durationMS float64) {
	rm.BadgeDuration.WithLabelValues(style).Observe(durationMS)
}

// RecordPlatformScore records a platform score
func (rm *ReputationMetrics) RecordPlatformScore(platform, userID string, score float64) {
	rm.PlatformScores.WithLabelValues(platform, userID).Set(score)
	rm.PlatformScoresCollected.WithLabelValues(platform).Inc()
}

// RecordReputationReceipt records a reputation receipt generation
func (rm *ReputationMetrics) RecordReputationReceipt(status string, gasUsed float64, platforms string) {
	rm.ReputationReceiptsTotal.WithLabelValues(status).Inc()
	rm.ReceiptGasUsed.WithLabelValues(platforms).Observe(gasUsed)
}

// RecordCacheOperation records a cache operation
func (rm *ReputationMetrics) RecordCacheOperation(cacheType string, hit bool) {
	if hit {
		rm.ReputationCacheHits.WithLabelValues(cacheType).Inc()
	} else {
		rm.ReputationCacheMisses.WithLabelValues(cacheType).Inc()
	}
}

// RecordRateLimitExceeded records a rate limit exceeded event
func (rm *ReputationMetrics) RecordRateLimitExceeded(platform, userID string) {
	rm.RateLimitExceeded.WithLabelValues(platform, userID).Inc()
}

// UpdateCacheSize updates the reputation cache size
func (rm *ReputationMetrics) UpdateCacheSize(sizeBytes float64) {
	rm.ReputationCacheSize.Set(sizeBytes)
}
