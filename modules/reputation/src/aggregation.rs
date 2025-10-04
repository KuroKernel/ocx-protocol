use crate::types::*;
use std::collections::HashMap;

/// Version of the reputation scoring algorithm
const ALGORITHM_VERSION: &str = "trustscore-v1.0.0";

/// Aggregate reputation scores from multiple platforms
pub fn aggregate_reputation(input: &ReputationInput) -> ReputationOutput {
    let mut components = HashMap::new();
    let mut weighted_sum = 0.0;
    let mut total_weight = 0.0;

    // Calculate weighted score for each platform
    for platform in &input.platforms {
        let normalized_score = normalize_score(platform.score);
        weighted_sum += normalized_score * platform.weight;
        total_weight += platform.weight;

        components.insert(
            platform.platform_type.clone(),
            ComponentScore {
                score: platform.score,
                weight: platform.weight,
                normalized_score,
            },
        );
    }

    // Calculate final trust score
    let raw_trust_score = if total_weight > 0.0 {
        (weighted_sum / total_weight) * 100.0
    } else {
        0.0
    };

    // Apply diversity bonus
    let diversity_bonus = calculate_diversity_bonus(&input.platforms);
    let trust_score = (raw_trust_score + diversity_bonus).min(100.0);

    // Calculate confidence based on number of platforms
    let confidence = calculate_confidence(&input.platforms);

    // Generate deterministic hash
    let deterministic_hash = generate_deterministic_hash(input, trust_score);

    ReputationOutput {
        trust_score,
        confidence,
        components,
        timestamp: input.timestamp,
        deterministic_hash,
        algorithm_version: ALGORITHM_VERSION.to_string(),
    }
}

/// Normalize score to 0-1 range
fn normalize_score(score: f64) -> f64 {
    (score / 100.0).clamp(0.0, 1.0)
}

/// Calculate diversity bonus (up to 5 points for 5+ platforms)
fn calculate_diversity_bonus(platforms: &[PlatformScore]) -> f64 {
    let unique_platforms = platforms.len();
    (unique_platforms as f64).min(5.0)
}

/// Calculate confidence based on platform count and score variance
fn calculate_confidence(platforms: &[PlatformScore]) -> f64 {
    if platforms.is_empty() {
        return 0.0;
    }

    // Base confidence from platform count (max 0.7 at 5+ platforms)
    let count_confidence = (platforms.len() as f64 / 5.0).min(0.7);

    // Variance penalty (low variance = high confidence)
    let scores: Vec<f64> = platforms.iter().map(|p| p.score).collect();
    let mean = scores.iter().sum::<f64>() / scores.len() as f64;
    let variance = scores.iter().map(|s| (s - mean).powi(2)).sum::<f64>() / scores.len() as f64;
    let variance_confidence = (1.0 - (variance / 10000.0)).clamp(0.0, 0.3);

    (count_confidence + variance_confidence).min(1.0)
}

/// Generate deterministic hash for verification
fn generate_deterministic_hash(input: &ReputationInput, trust_score: f64) -> String {
    // Create deterministic string representation
    let mut hash_input = format!("user:{}", input.user_id);
    hash_input.push_str(&format!("|score:{:.2}", trust_score));
    hash_input.push_str(&format!("|ts:{}", input.timestamp));
    hash_input.push_str(&format!("|algo:{}", ALGORITHM_VERSION));

    // Sort platforms for determinism
    let mut platform_types: Vec<_> = input.platforms.iter()
        .map(|p| p.platform_type.as_str())
        .collect();
    platform_types.sort_unstable();

    for pt in platform_types {
        hash_input.push_str(&format!("|{}", pt));
    }

    // Simple deterministic hash (in production, use SHA256)
    format!("sha256:{:016x}", simple_hash(&hash_input))
}

/// Simple deterministic hash function (placeholder for SHA256)
fn simple_hash(s: &str) -> u64 {
    let mut hash: u64 = 5381;
    for byte in s.bytes() {
        hash = hash.wrapping_mul(33).wrapping_add(byte as u64);
    }
    hash
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_deterministic_aggregation() {
        let input = create_test_input();

        // Run aggregation multiple times
        let result1 = aggregate_reputation(&input);
        let result2 = aggregate_reputation(&input);
        let result3 = aggregate_reputation(&input);

        // Results must be identical
        assert_eq!(result1.trust_score, result2.trust_score);
        assert_eq!(result2.trust_score, result3.trust_score);
        assert_eq!(result1.deterministic_hash, result2.deterministic_hash);
        assert_eq!(result2.deterministic_hash, result3.deterministic_hash);
    }

    #[test]
    fn test_score_normalization() {
        assert_eq!(normalize_score(0.0), 0.0);
        assert_eq!(normalize_score(50.0), 0.5);
        assert_eq!(normalize_score(100.0), 1.0);
        assert_eq!(normalize_score(150.0), 1.0); // Clamped
    }

    #[test]
    fn test_diversity_bonus() {
        let platforms_2 = vec![
            create_platform("github", 80.0),
            create_platform("linkedin", 90.0),
        ];
        assert_eq!(calculate_diversity_bonus(&platforms_2), 2.0);

        let platforms_5 = vec![
            create_platform("github", 80.0),
            create_platform("linkedin", 90.0),
            create_platform("twitter", 85.0),
            create_platform("stackoverflow", 88.0),
            create_platform("medium", 82.0),
        ];
        assert_eq!(calculate_diversity_bonus(&platforms_5), 5.0);

        let platforms_7 = vec![
            create_platform("github", 80.0),
            create_platform("linkedin", 90.0),
            create_platform("twitter", 85.0),
            create_platform("stackoverflow", 88.0),
            create_platform("medium", 82.0),
            create_platform("dev.to", 86.0),
            create_platform("hashnode", 84.0),
        ];
        assert_eq!(calculate_diversity_bonus(&platforms_7), 5.0); // Capped at 5
    }

    #[test]
    fn test_confidence_calculation() {
        // Single platform - low confidence
        let platforms_1 = vec![create_platform("github", 80.0)];
        let conf1 = calculate_confidence(&platforms_1);
        assert!(conf1 < 0.3);

        // Three platforms - medium confidence
        let platforms_3 = vec![
            create_platform("github", 80.0),
            create_platform("linkedin", 82.0),
            create_platform("twitter", 81.0),
        ];
        let conf3 = calculate_confidence(&platforms_3);
        assert!(conf3 > 0.4 && conf3 < 0.7);

        // Five platforms with similar scores - high confidence
        let platforms_5 = vec![
            create_platform("github", 80.0),
            create_platform("linkedin", 81.0),
            create_platform("twitter", 80.5),
            create_platform("stackoverflow", 80.2),
            create_platform("medium", 80.8),
        ];
        let conf5 = calculate_confidence(&platforms_5);
        assert!(conf5 > 0.8);
    }

    #[test]
    fn test_weighted_aggregation() {
        let input = ReputationInput {
            user_id: "alice".to_string(),
            platforms: vec![
                PlatformScore {
                    platform_type: "github".to_string(),
                    score: 90.0,
                    weight: 0.5,
                    metadata: HashMap::new(),
                },
                PlatformScore {
                    platform_type: "linkedin".to_string(),
                    score: 70.0,
                    weight: 0.5,
                    metadata: HashMap::new(),
                },
            ],
            weights: ScoreWeights::default(),
            timestamp: 1696348800,
        };

        let result = aggregate_reputation(&input);

        // Expected: (90 * 0.5 + 70 * 0.5) / 1.0 = 80.0, plus diversity bonus
        assert!(result.trust_score >= 82.0 && result.trust_score <= 85.0);
    }

    // Helper functions
    fn create_test_input() -> ReputationInput {
        ReputationInput {
            user_id: "test_user".to_string(),
            platforms: vec![
                create_platform("github", 85.0),
                create_platform("linkedin", 90.0),
            ],
            weights: ScoreWeights::default(),
            timestamp: 1696348800,
        }
    }

    fn create_platform(name: &str, score: f64) -> PlatformScore {
        PlatformScore {
            platform_type: name.to_string(),
            score,
            weight: 1.0,
            metadata: HashMap::new(),
        }
    }
}
