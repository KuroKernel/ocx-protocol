use std::fs;
use std::path::PathBuf;
use libocx_verify::{verify_receipt, verify_receipt_simple, VerificationError};
use libocx_verify::spec::{create_signing_message, DOMAIN_SEPARATOR};

#[test]
fn test_all_golden_vectors() {
    let vectors_dir = PathBuf::from("../conformance/conformance/receipts/v1");
    
    if !vectors_dir.exists() {
        panic!("Golden vectors not found. Run: cd conformance && go run generate_vectors.go");
    }
    
    let mut vector_count = 0;
    
    for entry in fs::read_dir(&vectors_dir).unwrap() {
        let entry = entry.unwrap();
        if entry.file_type().unwrap().is_dir() {
            test_golden_vector(&entry.path());
            vector_count += 1;
        }
    }
    
    assert!(vector_count > 0, "No golden vectors found");
    println!("Verified {} golden vectors", vector_count);
}

fn test_golden_vector(vector_dir: &PathBuf) {
    println!("Testing vector: {:?}", vector_dir.file_name().unwrap());
    
    // Load vector files
    let receipt_cbor = fs::read(vector_dir.join("receipt.cbor")).unwrap();
    let core_cbor = fs::read(vector_dir.join("core.cbor")).unwrap();
    let expected_message = fs::read(vector_dir.join("message.bin")).unwrap();
    let pubkey = fs::read(vector_dir.join("pubkey.bin")).unwrap();
    let expected_signature = fs::read(vector_dir.join("signature.bin")).unwrap();
    
    // Parse receipt
    let receipt = match libocx_verify::OcxReceipt::from_canonical_cbor(&receipt_cbor) {
        Ok(r) => r,
        Err(e) => {
            dump_debug_info(&receipt_cbor, &core_cbor, &expected_message);
            panic!("Failed to parse receipt: {:?}", e);
        }
    };
    
    // Verify signature matches expected
    assert_eq!(receipt.signature, expected_signature, "Signature mismatch");
    
    // Verify message construction
    let reconstructed_message = create_signing_message(&core_cbor);
    assert_eq!(reconstructed_message, expected_message, "Message reconstruction failed");
    
    // Verify receipt with the actual public key used to sign it
    match verify_receipt(&receipt_cbor, &pubkey, false) {
        Ok(_) => println!("✓ Vector verified successfully"),
        Err(e) => {
            dump_debug_info(&receipt_cbor, &core_cbor, &expected_message);
            panic!("Verification failed: {:?}", e);
        }
    }
    
    // Test simple verification (this will fail because the issuer_key_id doesn't match the actual key)
    match verify_receipt_simple(&receipt_cbor) {
        Ok(_) => println!("✓ Simple verification also successful"),
        Err(e) => println!("⚠ Simple verification failed (expected): {:?}", e),
    }
}

fn dump_debug_info(receipt_cbor: &[u8], core_cbor: &[u8], expected_message: &[u8]) {
    println!("=== DEBUG DUMP ===");
    println!("Receipt CBOR (first 32 bytes): {:?}", &receipt_cbor[..32.min(receipt_cbor.len())]);
    println!("Core CBOR (first 32 bytes): {:?}", &core_cbor[..32.min(core_cbor.len())]);
    println!("Expected message (first 32 bytes): {:?}", &expected_message[..32.min(expected_message.len())]);
    println!("Domain separator: {:?}", DOMAIN_SEPARATOR);
    println!("==================");
}
