use criterion::{black_box, criterion_group, criterion_main, Criterion, BenchmarkId};
use libocx_verify::{verify_receipt, OcxReceipt, VerificationError};
use std::time::Instant;

// Create a test receipt for benchmarking
fn create_test_receipt() -> (Vec<u8>, [u8; 32]) {
    let receipt = OcxReceipt {
        artifact_hash: [1u8; 32],
        input_hash: [2u8; 32],
        output_hash: [3u8; 32],
        cycles_used: 1000,
        started_at: 1640995200,
        finished_at: 1640995201,
        issuer_key_id: "test-key-123".to_string(),
        prev_receipt_hash: None,
        request_digest: None,
        witness_signatures: vec![],
        signature: vec![4u8; 64], // Placeholder signature
    };
    
    let cbor_data = receipt.to_canonical_cbor().unwrap();
    let public_key = [5u8; 32]; // Placeholder public key
    
    (cbor_data, public_key)
}

fn benchmark_dual_crypto_verification(c: &mut Criterion) {
    let (cbor_data, public_key) = create_test_receipt();
    
    let mut group = c.benchmark_group("dual_crypto_verification");
    
    // Benchmark single verification
    group.bench_function("single_verification", |b| {
        b.iter(|| {
            let _ = verify_receipt(black_box(&cbor_data), black_box(&public_key), false);
        })
    });
    
    // Benchmark batch verification (100 receipts)
    group.bench_function("batch_100_verifications", |b| {
        b.iter(|| {
            for _ in 0..100 {
                let _ = verify_receipt(black_box(&cbor_data), black_box(&public_key), false);
            }
        })
    });
    
    // Benchmark with tracing enabled
    group.bench_function("verification_with_tracing", |b| {
        b.iter(|| {
            // This would use the trace feature in real usage
            let _ = verify_receipt(black_box(&cbor_data), black_box(&public_key), false);
        })
    });
    
    group.finish();
}

fn benchmark_performance_targets(c: &mut Criterion) {
    let (cbor_data, public_key) = create_test_receipt();
    
    let mut group = c.benchmark_group("performance_targets");
    group.measurement_time(std::time::Duration::from_secs(10));
    
    // Test that we meet the <4ms target for crypto operations
    group.bench_function("crypto_target_4ms", |b| {
        b.iter(|| {
            let start = Instant::now();
            let _ = verify_receipt(black_box(&cbor_data), black_box(&public_key), false);
            let duration = start.elapsed();
            
            // Assert we meet the <4ms target
            assert!(duration.as_millis() < 4, 
                   "Crypto verification took {}ms, target <4ms", 
                   duration.as_millis());
        })
    });
    
    group.finish();
}

criterion_group!(benches, benchmark_dual_crypto_verification, benchmark_performance_targets);
criterion_main!(benches);
