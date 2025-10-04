package normalizers

import (
	"math"

	"ocx.local/integrations/oauth"
)

// GitHubNormalizer converts GitHub statistics to a 0-100 reputation score
type GitHubNormalizer struct {
	// Weights for different metrics (must sum to 1.0)
	weightCommits      float64
	weightStars        float64
	weightFollowers    float64
	weightOrgs         float64
	weightAccountAge   float64
	weightRepos        float64
}

// NewGitHubNormalizer creates a new GitHub normalizer with default weights
func NewGitHubNormalizer() *GitHubNormalizer {
	return &GitHubNormalizer{
		weightCommits:    0.35, // 35% - code contributions are most important
		weightStars:      0.25, // 25% - stars indicate quality
		weightFollowers:  0.15, // 15% - network effect
		weightOrgs:       0.10, // 10% - organizational involvement
		weightAccountAge: 0.10, // 10% - account longevity
		weightRepos:      0.05, // 5%  - repository count
	}
}

// Normalize converts GitHub statistics to a 0-100 score using deterministic math
// All operations use IEEE 754 floating-point arithmetic for cross-platform consistency
func (n *GitHubNormalizer) Normalize(stats *oauth.GitHubUserStats) float64 {
	// Each metric is normalized to 0-100 scale using log-scale transformation
	// This handles the wide range of values (1 commit vs 100k commits)

	// Commits: log10(commits + 1) normalized to max ~100k commits
	commitScore := n.normalizeLogScale(float64(stats.TotalCommits), 5.0) // log10(100000) ≈ 5

	// Stars: log10(stars + 1) normalized to max ~10k stars
	starScore := n.normalizeLogScale(float64(stats.TotalStars), 4.0) // log10(10000) ≈ 4

	// Followers: log10(followers + 1) normalized to max ~1k followers
	followerScore := n.normalizeLogScale(float64(stats.Followers), 3.0) // log10(1000) ≈ 3

	// Organizations: linear scale, max 20 orgs
	orgScore := n.normalizeLinear(float64(stats.Organizations), 20.0)

	// Account age: linear scale, max 10 years
	accountAgeYears := float64(stats.AccountAge) / 365.0
	ageScore := n.normalizeLinear(accountAgeYears, 10.0)

	// Repositories: log10(repos + 1) normalized to max ~100 repos
	repoScore := n.normalizeLogScale(float64(stats.PublicRepos), 2.0) // log10(100) ≈ 2

	// Weighted average
	rawScore := (commitScore * n.weightCommits) +
		(starScore * n.weightStars) +
		(followerScore * n.weightFollowers) +
		(orgScore * n.weightOrgs) +
		(ageScore * n.weightAccountAge) +
		(repoScore * n.weightRepos)

	// Clamp to 0-100 range
	return clamp(rawScore, 0.0, 100.0)
}

// normalizeLogScale applies logarithmic normalization
// value: raw value to normalize
// maxLog: log10 of the maximum expected value
func (n *GitHubNormalizer) normalizeLogScale(value float64, maxLog float64) float64 {
	if value <= 0 {
		return 0.0
	}

	// log10(value + 1) to handle 0 values
	logValue := math.Log10(value + 1.0)

	// Normalize to 0-100 scale
	normalized := (logValue / maxLog) * 100.0

	return clamp(normalized, 0.0, 100.0)
}

// normalizeLinear applies linear normalization
// value: raw value to normalize
// maxValue: maximum expected value
func (n *GitHubNormalizer) normalizeLinear(value float64, maxValue float64) float64 {
	if maxValue == 0 {
		return 0.0
	}

	normalized := (value / maxValue) * 100.0
	return clamp(normalized, 0.0, 100.0)
}

// clamp restricts a value to a given range
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
