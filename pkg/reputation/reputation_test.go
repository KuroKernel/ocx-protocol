package reputation

import (
	"testing"
)

// TestComputeReputation tests the reputation computation algorithm
func TestComputeReputation(t *testing.T) {
	tests := []struct {
		name       string
		platforms  map[string]float64
		wantScore  float64
		wantConf   float64
		tolerance  float64
	}{
		{
			name: "GitHub only",
			platforms: map[string]float64{
				"github": 85.5,
			},
			wantScore: 85.5,
			wantConf:  0.4,
			tolerance: 0.01,
		},
		{
			name: "All platforms",
			platforms: map[string]float64{
				"github":   85.5,
				"linkedin": 72.3,
				"uber":     90.1,
			},
			wantScore: 82.03,
			wantConf:  1.0,
			tolerance: 0.01,
		},
		{
			name: "GitHub + Uber",
			platforms: map[string]float64{
				"github": 95.0,
				"uber":   88.0,
			},
			wantScore: 92.31,
			wantConf:  0.65,
			tolerance: 0.01,
		},
		{
			name: "Perfect score all platforms",
			platforms: map[string]float64{
				"github":   100.0,
				"linkedin": 100.0,
				"uber":     100.0,
			},
			wantScore: 100.0,
			wantConf:  1.0,
			tolerance: 0.01,
		},
		{
			name: "Zero score all platforms",
			platforms: map[string]float64{
				"github":   0.0,
				"linkedin": 0.0,
				"uber":     0.0,
			},
			wantScore: 0.0,
			wantConf:  1.0,
			tolerance: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, conf := ComputeWeightedScore(tt.platforms)

			// Check score
			if diff := abs(score - tt.wantScore); diff > tt.tolerance {
				t.Errorf("ComputeWeightedScore() score = %.2f, want %.2f (diff: %.4f)", score, tt.wantScore, diff)
			}

			// Check confidence
			if diff := abs(conf - tt.wantConf); diff > tt.tolerance {
				t.Errorf("ComputeWeightedScore() confidence = %.2f, want %.2f (diff: %.4f)", conf, tt.wantConf, diff)
			}
		})
	}
}

// TestDeterminism tests that the computation is deterministic
func TestDeterminism(t *testing.T) {
	platforms := map[string]float64{
		"github":   85.5,
		"linkedin": 72.3,
		"uber":     90.1,
	}

	runs := 100
	results := make([]float64, runs)

	for i := 0; i < runs; i++ {
		score, _ := ComputeWeightedScore(platforms)
		results[i] = score
	}

	// All results should be identical
	firstScore := results[0]
	for i := 1; i < runs; i++ {
		if results[i] != firstScore {
			t.Errorf("Non-deterministic behavior detected: run 0 = %.10f, run %d = %.10f", firstScore, i, results[i])
		}
	}
}

// TestBadgeColor tests badge color selection
func TestBadgeColor(t *testing.T) {
	tests := []struct {
		score float64
		want  BadgeColor
	}{
		{95.0, BadgeColorBrightGreen},
		{90.0, BadgeColorBrightGreen},
		{80.0, BadgeColorGreen},
		{75.0, BadgeColorGreen},
		{65.0, BadgeColorYellow},
		{60.0, BadgeColorYellow},
		{50.0, BadgeColorOrange},
		{40.0, BadgeColorOrange},
		{30.0, BadgeColorRed},
		{10.0, BadgeColorRed},
		{0.0, BadgeColorGray},
	}

	for _, tt := range tests {
		t.Run(formatFloat(tt.score), func(t *testing.T) {
			got := GetBadgeColor(tt.score)
			if got != tt.want {
				t.Errorf("GetBadgeColor(%.1f) = %s, want %s", tt.score, got, tt.want)
			}
		})
	}
}

// TestGenerateBadgeSVG tests SVG badge generation
func TestGenerateBadgeSVG(t *testing.T) {
	tests := []struct {
		style BadgeStyle
		score float64
	}{
		{BadgeStyleFlat, 85.5},
		{BadgeStyleFlatSquare, 92.3},
		{BadgeStyleForTheBadge, 78.1},
	}

	for _, tt := range tests {
		t.Run(string(tt.style), func(t *testing.T) {
			svg := GenerateBadgeSVG("TrustScore", tt.score, tt.style)

			// Check that it's valid SVG
			if len(svg) == 0 {
				t.Error("GenerateBadgeSVG() returned empty string")
			}

			// Check for SVG tag
			if !contains(svg, "<svg") {
				t.Error("GenerateBadgeSVG() did not return valid SVG")
			}

			// Check for score
			scoreStr := formatFloat(tt.score)
			if !contains(svg, scoreStr) {
				t.Errorf("GenerateBadgeSVG() does not contain score %.1f", tt.score)
			}
		})
	}
}

// TestInvalidInputs tests handling of invalid inputs
func TestInvalidInputs(t *testing.T) {
	tests := []struct {
		name      string
		platforms map[string]float64
	}{
		{
			name: "Negative score",
			platforms: map[string]float64{
				"github": -10.0,
			},
		},
		{
			name: "Score > 100",
			platforms: map[string]float64{
				"github": 150.0,
			},
		},
		{
			name: "Unknown platform",
			platforms: map[string]float64{
				"twitter": 85.0,
			},
		},
		{
			name:      "Empty platforms",
			platforms: map[string]float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, conf := ComputeWeightedScore(tt.platforms)

			// Should return 0 for invalid inputs
			if score != 0.0 || conf != 0.0 {
				t.Errorf("Expected (0.0, 0.0) for invalid input, got (%.2f, %.2f)", score, conf)
			}
		})
	}
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func formatFloat(f float64) string {
	// Simple float formatting for tests
	if f == 0.0 {
		return "0"
	}
	return string(rune(int(f)) + '0')
}

// ComputeWeightedScore computes weighted reputation score
func ComputeWeightedScore(platforms map[string]float64) (float64, float64) {
	var totalScore float64
	var totalWeight float64

	weights := map[string]float64{
		"github":   0.4,
		"linkedin": 0.35,
		"uber":     0.25,
	}

	for platform, score := range platforms {
		// Validate score range
		if score < 0 || score > 100 {
			continue
		}

		weight, ok := weights[platform]
		if !ok {
			continue
		}

		totalScore += score * weight
		totalWeight += weight
	}

	finalScore := 0.0
	if totalWeight > 0 {
		finalScore = totalScore / totalWeight
	}

	return finalScore, totalWeight
}
