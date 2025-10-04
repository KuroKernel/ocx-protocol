package normalizers

import (
	"math"

	"ocx.local/integrations/oauth"
)

// UberNormalizer converts Uber statistics to a 0-100 reputation score
type UberNormalizer struct {
	// Weights for different metrics (must sum to 1.0)
	weightRating      float64
	weightTrips       float64
	weightLongevity   float64
	weightCompliments float64
	weightReliability float64
}

// NewUberNormalizer creates a new Uber normalizer with default weights
func NewUberNormalizer() *UberNormalizer {
	return &UberNormalizer{
		weightRating:      0.40, // 40% - star rating (most important)
		weightTrips:       0.25, // 25% - trip volume
		weightLongevity:   0.15, // 15% - years of service
		weightCompliments: 0.10, // 10% - compliments received
		weightReliability: 0.10, // 10% - cancellation rate (inverse)
	}
}

// Normalize converts Uber statistics to a 0-100 score
func (n *UberNormalizer) Normalize(stats *oauth.UberUserStats) float64 {
	// Rating: linear scale from 1.0-5.0 to 0-100
	// Uber ratings are typically 4.5-5.0, so we shift the scale
	ratingScore := n.normalizeRating(stats.Rating)

	// Trips: log scale, max ~10k trips for regular users
	tripScore := n.normalizeLogScale(float64(stats.TotalTrips), math.Log10(10000))

	// Longevity: linear scale, max 10 years
	longevityScore := n.normalizeLinear(float64(stats.TotalYears), 10.0)

	// Compliments: log scale, max ~100 compliments
	complimentScore := n.normalizeLogScale(float64(stats.Compliments), math.Log10(100))

	// Reliability: inverse of cancellation rate
	// Assume max 20% cancellation rate (high cancellations are bad)
	var reliabilityScore float64
	if stats.TotalTrips > 0 {
		cancellationRate := float64(stats.Cancellations) / float64(stats.TotalTrips)
		// Invert: 0% cancellation = 100 score, 20% cancellation = 0 score
		reliabilityScore = clamp((1.0-(cancellationRate/0.20))*100.0, 0.0, 100.0)
	} else {
		reliabilityScore = 50.0 // Neutral score for new users
	}

	// Weighted average
	rawScore := (ratingScore * n.weightRating) +
		(tripScore * n.weightTrips) +
		(longevityScore * n.weightLongevity) +
		(complimentScore * n.weightCompliments) +
		(reliabilityScore * n.weightReliability)

	return clamp(rawScore, 0.0, 100.0)
}

// normalizeRating converts Uber's 1-5 star rating to 0-100 scale
// Shifted to account for typical 4.5-5.0 range
func (n *UberNormalizer) normalizeRating(rating float64) float64 {
	if rating < 1.0 {
		return 0.0
	}

	// Linear transformation: 4.0 → 50, 5.0 → 100
	// This gives meaningful differentiation in the typical 4.5-5.0 range
	score := ((rating - 4.0) / 1.0) * 50.0 + 50.0

	return clamp(score, 0.0, 100.0)
}

// normalizeLogScale applies logarithmic normalization
func (n *UberNormalizer) normalizeLogScale(value float64, maxLog float64) float64 {
	if value <= 0 {
		return 0.0
	}

	logValue := math.Log10(value + 1.0)
	normalized := (logValue / maxLog) * 100.0

	return clamp(normalized, 0.0, 100.0)
}

// normalizeLinear applies linear normalization
func (n *UberNormalizer) normalizeLinear(value float64, maxValue float64) float64 {
	if maxValue == 0 {
		return 0.0
	}

	normalized := (value / maxValue) * 100.0
	return clamp(normalized, 0.0, 100.0)
}
