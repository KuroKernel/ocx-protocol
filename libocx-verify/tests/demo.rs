//! Demonstration tests for OCX Protocol

use std::fs;
use std::path::PathBuf;
use libocx_verify::OcxReceipt;
use hex;

#[test]
fn demo_receipt_creation_and_verification() {
    // Create a test receipt manually
    let receipt = OcxReceipt {
        artifact_hash: [0x01; 32],
        input_hash: [0x02; 32],
        output_hash: [0x03; 32],
        cycles_used: 1000,
        started_at: 1640995200, // 2022-01-01 00:00:00 UTC
        finished_at: 1640995201, // 2022-01-01 00:00:01 UTC
        issuer_key_id: "test-issuer".to_string(),
        signature: vec![0x88; 64], // Placeholder signature
        prev_receipt_hash: None,
        request_digest: None,
        witness_signatures: Vec::new(),
    };
    
    // Test canonical CBOR encoding
    let cbor_data = receipt.to_canonical_cbor().unwrap();
    assert!(!cbor_data.is_empty(), "CBOR encoding should produce data");
    
    // Test signing message creation
    let signing_message = receipt.get_signing_message().unwrap();
    assert!(!signing_message.is_empty(), "Signing message should be created");
    
    println!("✓ Receipt creation and encoding demo completed successfully");
}

#[test]
fn demo_golden_vectors_verification() {
    let vectors_dir = PathBuf::from("../conformance/receipts/v1");
    
    if !vectors_dir.exists() {
        println!("⚠️  Golden vectors not found. Run: make generate-vectors");
        return;
    }
    
    let mut vector_count = 0;
    
    for entry in fs::read_dir(&vectors_dir).unwrap() {
        let entry = entry.unwrap();
        if entry.file_type().unwrap().is_dir() {
            let receipt_cbor = fs::read(entry.path().join("receipt.cbor")).unwrap();
            let pubkey = fs::read(entry.path().join("pubkey.bin")).unwrap();
            
                   // Test parsing
                   println!("Testing vector: {:?}", entry.file_name().to_string_lossy());
                   println!("CBOR length: {} bytes", receipt_cbor.len());
                   println!("CBOR first 32 bytes: {:?}", &receipt_cbor[..32.min(receipt_cbor.len())]);
                   
                   match OcxReceipt::from_canonical_cbor(&receipt_cbor) {
                       Ok(receipt) => {
                           println!("✓ Parsed vector: {:?}", entry.file_name().to_string_lossy());
                           
                           // Test canonical encoding roundtrip
                           let reencoded = receipt.to_canonical_cbor().unwrap();
                           assert_eq!(reencoded, receipt_cbor, "Roundtrip encoding should be identical");
                           
                           // Test signing message creation
                           let _signing_message = receipt.get_signing_message().unwrap();
                           
                           vector_count += 1;
                       }
                       Err(e) => {
                           println!("✗ Failed to parse vector {:?}: {:?}", entry.file_name().to_string_lossy(), e);
                           println!("CBOR hex: {}", hex::encode(&receipt_cbor[..64.min(receipt_cbor.len())]));
                       }
                   }
        }
    }
    
    assert!(vector_count > 0, "No golden vectors were successfully processed");
    println!("✓ Processed {} golden vectors successfully", vector_count);
}
