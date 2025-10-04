use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Input structure for reputation verification
#[derive(Debug, Clone, Deserialize)]
pub struct ReputationInput {
    pub user_id: String,
    pub platforms: Vec<PlatformScore>,
    #[serde(default)]
    pub weights: ScoreWeights,
    #[serde(default)]
    pub timestamp: u64,
}

/// Score from a single platform
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PlatformScore {
    pub platform_type: String,
    pub score: f64,
    pub weight: f64,
    #[serde(default)]
    pub metadata: HashMap<String, serde_json::Value>,
}

/// Weighting factors for score calculation
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct ScoreWeights {
    /// Weight for recent activity (default: 0.3)
    #[serde(default = "default_recency")]
    pub recency: f64,

    /// Weight for volume of contributions (default: 0.3)
    #[serde(default = "default_volume")]
    pub volume: f64,

    /// Weight for platform diversity (default: 0.4)
    #[serde(default = "default_diversity")]
    pub diversity: f64,
}

impl Default for ScoreWeights {
    fn default() -> Self {
        Self {
            recency: 0.3,
            volume: 0.3,
            diversity: 0.4,
        }
    }
}

fn default_recency() -> f64 { 0.3 }
fn default_volume() -> f64 { 0.3 }
fn default_diversity() -> f64 { 0.4 }

/// Output structure for reputation verification
#[derive(Debug, Clone, Serialize)]
pub struct ReputationOutput {
    /// Aggregated trust score (0-100)
    pub trust_score: f64,

    /// Confidence in the score (0-1)
    pub confidence: f64,

    /// Breakdown by platform
    pub components: HashMap<String, ComponentScore>,

    /// Unix timestamp of calculation
    pub timestamp: u64,

    /// Deterministic hash of calculation
    pub deterministic_hash: String,

    /// Version of scoring algorithm
    pub algorithm_version: String,
}

/// Individual platform component score
#[derive(Debug, Clone, Serialize)]
pub struct ComponentScore {
    pub score: f64,
    pub weight: f64,
    pub normalized_score: f64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_weights() {
        let weights = ScoreWeights::default();
        assert_eq!(weights.recency, 0.3);
        assert_eq!(weights.volume, 0.3);
        assert_eq!(weights.diversity, 0.4);

        // Weights should sum to 1.0
        assert!((weights.recency + weights.volume + weights.diversity - 1.0).abs() < 0.001);
    }

    #[test]
    fn test_deserialization() {
        let json = r#"{
            "user_id": "alice",
            "platforms": [
                {"platform_type": "github", "score": 85.0, "weight": 0.5, "metadata": {}}
            ],
            "timestamp": 1696348800
        }"#;

        let input: ReputationInput = serde_json::from_str(json).unwrap();
        assert_eq!(input.user_id, "alice");
        assert_eq!(input.platforms.len(), 1);
        assert_eq!(input.timestamp, 1696348800);
    }
}
