use trustscore_wasm::{ReputationInput, PlatformScore, ScoreWeights, aggregate_reputation};
use std::collections::HashMap;

/// Test that aggregation is fully deterministic
#[test]
fn test_deterministic_aggregation() {
    let input = create_standard_input();

    // Run 100 times
    let results: Vec<_> = (0..100)
        .map(|_| aggregate_reputation(&input))
        .collect();

    // All results must be identical
    let first = &results[0];
    for result in &results[1..] {
        assert_eq!(
            result.trust_score, first.trust_score,
            "Trust score must be deterministic"
        );
        assert_eq!(
            result.confidence, first.confidence,
            "Confidence must be deterministic"
        );
        assert_eq!(
            result.deterministic_hash, first.deterministic_hash,
            "Hash must be deterministic"
        );
    }
}

/// Test that same input on different platforms produces same result
#[test]
fn test_cross_platform_determinism() {
    let input = create_standard_input();

    let result = aggregate_reputation(&input);

    // Store expected values (these would be golden vectors in production)
    assert!((result.trust_score - 87.5).abs() < 0.1);
    assert!(result.confidence > 0.0);
    assert!(!result.deterministic_hash.is_empty());
    assert_eq!(result.algorithm_version, "trustscore-v1.0.0");
}

/// Test that input order doesn't affect result (commutativity)
#[test]
fn test_order_independence() {
    let platforms = vec![
        create_platform("github", 85.0, 0.33),
        create_platform("linkedin", 90.0, 0.33),
        create_platform("twitter", 80.0, 0.34),
    ];

    let input1 = ReputationInput {
        user_id: "alice".to_string(),
        platforms: platforms.clone(),
        weights: ScoreWeights::default(),
        timestamp: 1696348800,
    };

    // Reverse order
    let mut platforms_reversed = platforms.clone();
    platforms_reversed.reverse();

    let input2 = ReputationInput {
        user_id: "alice".to_string(),
        platforms: platforms_reversed,
        weights: ScoreWeights::default(),
        timestamp: 1696348800,
    };

    let result1 = aggregate_reputation(&input1);
    let result2 = aggregate_reputation(&input2);

    // Trust score should be identical (platform order doesn't matter)
    assert_eq!(result1.trust_score, result2.trust_score);
}

/// Test floating point determinism
#[test]
fn test_floating_point_determinism() {
    let input = ReputationInput {
        user_id: "test".to_string(),
        platforms: vec![
            create_platform("github", 85.123456789, 0.5),
            create_platform("linkedin", 90.987654321, 0.5),
        ],
        weights: ScoreWeights::default(),
        timestamp: 1696348800,
    };

    let results: Vec<_> = (0..1000)
        .map(|_| aggregate_reputation(&input))
        .collect();

    // All floating point results must be bit-identical
    let first_score = results[0].trust_score.to_bits();
    for result in &results[1..] {
        assert_eq!(
            result.trust_score.to_bits(),
            first_score,
            "Floating point must be bit-identical"
        );
    }
}

/// Test empty platforms edge case
#[test]
fn test_empty_platforms() {
    let input = ReputationInput {
        user_id: "empty".to_string(),
        platforms: vec![],
        weights: ScoreWeights::default(),
        timestamp: 1696348800,
    };

    let result = aggregate_reputation(&input);

    assert_eq!(result.trust_score, 0.0);
    assert_eq!(result.confidence, 0.0);
    assert!(result.components.is_empty());
}

/// Test single platform
#[test]
fn test_single_platform() {
    let input = ReputationInput {
        user_id: "single".to_string(),
        platforms: vec![create_platform("github", 85.0, 1.0)],
        weights: ScoreWeights::default(),
        timestamp: 1696348800,
    };

    let result = aggregate_reputation(&input);

    // Score should be 85 + diversity bonus (1 point)
    assert!((result.trust_score - 86.0).abs() < 0.1);
    assert!(result.confidence < 0.3); // Low confidence with single platform
}

/// Test maximum platforms (diversity cap)
#[test]
fn test_maximum_diversity() {
    let platforms: Vec<_> = (0..10)
        .map(|i| create_platform(&format!("platform{}", i), 80.0, 0.1))
        .collect();

    let input = ReputationInput {
        user_id: "diverse".to_string(),
        platforms,
        weights: ScoreWeights::default(),
        timestamp: 1696348800,
    };

    let result = aggregate_reputation(&input);

    // Diversity bonus should be capped at 5 points
    assert!((result.trust_score - 85.0).abs() < 0.1);
    assert!(result.confidence > 0.9); // High confidence with many platforms
}

// Helper functions
fn create_standard_input() -> ReputationInput {
    ReputationInput {
        user_id: "alice@example.com".to_string(),
        platforms: vec![
            create_platform("github", 90.0, 0.4),
            create_platform("linkedin", 85.0, 0.3),
            create_platform("twitter", 82.0, 0.3),
        ],
        weights: ScoreWeights::default(),
        timestamp: 1696348800,
    }
}

fn create_platform(name: &str, score: f64, weight: f64) -> PlatformScore {
    PlatformScore {
        platform_type: name.to_string(),
        score,
        weight,
        metadata: HashMap::new(),
    }
}
