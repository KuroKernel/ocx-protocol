package normalizers

import (
	"math"

	"ocx.local/integrations/oauth"
)

// LinkedInNormalizer converts LinkedIn statistics to a 0-100 reputation score
type LinkedInNormalizer struct {
	// Weights for different metrics (must sum to 1.0)
	weightConnections float64
	weightEndorsements float64
	weightPosts       float64
	weightFollowers   float64
	weightProfileViews float64
	weightExperience  float64
}

// NewLinkedInNormalizer creates a new LinkedIn normalizer with default weights
func NewLinkedInNormalizer() *LinkedInNormalizer {
	return &LinkedInNormalizer{
		weightConnections:  0.25, // 25% - network size
		weightEndorsements: 0.20, // 20% - skill validation
		weightPosts:        0.15, // 15% - content creation
		weightFollowers:    0.15, // 15% - influence
		weightProfileViews: 0.15, // 15% - visibility
		weightExperience:   0.10, // 10% - years of experience
	}
}

// Normalize converts LinkedIn statistics to a 0-100 score
func (n *LinkedInNormalizer) Normalize(stats *oauth.LinkedInUserStats) float64 {
	// Connections: log scale, max ~5000 connections (500 is LinkedIn's display limit)
	connectionScore := n.normalizeLogScale(float64(stats.Connections), math.Log10(5000))

	// Endorsements: log scale, max ~1000 endorsements
	endorsementScore := n.normalizeLogScale(float64(stats.Endorsements), math.Log10(1000))

	// Posts: log scale, max ~500 posts
	postScore := n.normalizeLogScale(float64(stats.Posts), math.Log10(500))

	// Followers: log scale, max ~10k followers
	followerScore := n.normalizeLogScale(float64(stats.Followers), math.Log10(10000))

	// Profile views: log scale, max ~1000 views/month
	viewScore := n.normalizeLogScale(float64(stats.ProfileViews), math.Log10(1000))

	// Experience: linear scale, max 20 years
	experienceScore := n.normalizeLinear(float64(stats.YearsExp), 20.0)

	// Weighted average
	rawScore := (connectionScore * n.weightConnections) +
		(endorsementScore * n.weightEndorsements) +
		(postScore * n.weightPosts) +
		(followerScore * n.weightFollowers) +
		(viewScore * n.weightProfileViews) +
		(experienceScore * n.weightExperience)

	return clamp(rawScore, 0.0, 100.0)
}

// normalizeLogScale applies logarithmic normalization
func (n *LinkedInNormalizer) normalizeLogScale(value float64, maxLog float64) float64 {
	if value <= 0 {
		return 0.0
	}

	logValue := math.Log10(value + 1.0)
	normalized := (logValue / maxLog) * 100.0

	return clamp(normalized, 0.0, 100.0)
}

// normalizeLinear applies linear normalization
func (n *LinkedInNormalizer) normalizeLinear(value float64, maxValue float64) float64 {
	if maxValue == 0 {
		return 0.0
	}

	normalized := (value / maxValue) * 100.0
	return clamp(normalized, 0.0, 100.0)
}
