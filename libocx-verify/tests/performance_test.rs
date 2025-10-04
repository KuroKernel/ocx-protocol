use libocx_verify::{verify_receipt, OcxReceipt};
use std::time::Instant;

#[test]
fn test_dual_crypto_performance_target() {
    // Create a test receipt
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
    
    // Test performance over 1000 iterations
    let iterations = 1000;
    let start = Instant::now();
    
    for _ in 0..iterations {
        let _ = verify_receipt(&cbor_data, &public_key, false);
    }
    
    let total_duration = start.elapsed();
    let avg_duration = total_duration / iterations;
    
    println!("Dual-library crypto performance test:");
    println!("  Total time for {} verifications: {:?}", iterations, total_duration);
    println!("  Average time per verification: {:?}", avg_duration);
    println!("  Average time per verification: {}ms", avg_duration.as_millis());
    
    // Assert we meet the <4ms target
    assert!(avg_duration.as_millis() < 4, 
           "Dual-library verification took {}ms, target <4ms", 
           avg_duration.as_millis());
    
    println!("✅ Performance target met: {}ms < 4ms", avg_duration.as_millis());
}
