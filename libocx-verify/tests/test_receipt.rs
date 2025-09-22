use libocx_verify::{OcxReceipt, VerificationError};
use std::time::{SystemTime, UNIX_EPOCH};

/// Create a valid test receipt CBOR blob.
fn create_test_receipt_cbor() -> Vec<u8> {
    // This represents a valid OCX receipt in canonical CBOR format
    // Map with 8 required fields (integer keys 1-8)
    let mut cbor = Vec::new();
    
    // Map header: map(8)
    cbor.push(0xa8);
    
    // Key 1: program_hash (32 bytes)
    cbor.push(0x01);
    cbor.push(0x58); cbor.push(0x20); // bytes(32)
    cbor.extend_from_slice(&[0x01; 32]); // 32 bytes of 0x01
    
    // Key 2: input_hash (32 bytes)
    cbor.push(0x02);
    cbor.push(0x58); cbor.push(0x20); // bytes(32)
    cbor.extend_from_slice(&[0x02; 32]); // 32 bytes of 0x02
    
    // Key 3: output_hash (32 bytes)
    cbor.push(0x03);
    cbor.push(0x58); cbor.push(0x20); // bytes(32)
    cbor.extend_from_slice(&[0x03; 32]); // 32 bytes of 0x03
    
    // Key 4: cycles (uint64) - use minimal encoding
    cbor.push(0x04);
    cbor.push(0x19); cbor.push(0x03); cbor.push(0xe8); // 1000
    
    // Key 5: started_at (unix timestamp) - use minimal encoding
    cbor.push(0x05);
    let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() as u32;
    cbor.push(0x1a); // uint32
    cbor.extend_from_slice(&now.to_be_bytes());
    
    // Key 6: finished_at (unix timestamp) - use minimal encoding
    cbor.push(0x06);
    cbor.push(0x1a); // uint32
    cbor.extend_from_slice(&(now + 1).to_be_bytes());
    
    // Key 7: issuer_id (text string)
    cbor.push(0x07);
    cbor.push(0x64); // text(4)
    cbor.extend_from_slice(b"test");
    
    // Key 8: signature (64 bytes)
    cbor.push(0x08);
    cbor.push(0x58); cbor.push(0x40); // bytes(64)
    cbor.extend_from_slice(&[0x88; 64]); // 64 bytes of 0x88
    
    cbor
}

#[test]
fn test_parse_valid_receipt() {
    let cbor_data = create_test_receipt_cbor();
    let receipt = OcxReceipt::from_canonical_cbor(&cbor_data);
    assert!(receipt.is_ok(), "Failed to parse valid receipt: {:?}", receipt.err());
    
    let receipt = receipt.unwrap();
    assert_eq!(receipt.artifact_hash, [0x01; 32]);
    assert_eq!(receipt.input_hash, [0x02; 32]);
    assert_eq!(receipt.output_hash, [0x03; 32]);
    assert_eq!(receipt.cycles_used, 1000);
    assert_eq!(receipt.issuer_key_id, "test");
    assert_eq!(receipt.signature.len(), 64);
}

#[test]
fn test_missing_required_field() {
    // Create CBOR with missing artifact_hash (key 1)
    let mut cbor = Vec::new();
    cbor.push(0xa7); // map(7) instead of map(8)
    
    // Skip key 1, start with key 2
    cbor.push(0x02);
    cbor.push(0x58); cbor.push(0x20);
    cbor.extend_from_slice(&[0x02; 32]);
    
    // Key 3: output_hash
    cbor.push(0x03);
    cbor.push(0x58); cbor.push(0x20);
    cbor.extend_from_slice(&[0x03; 32]);
    
    // Key 4: cycles_used
    cbor.push(0x04);
    cbor.push(0x19); // uint16
    cbor.extend_from_slice(&10000u16.to_be_bytes());
    
    // Key 5: started_at
    cbor.push(0x05);
    cbor.push(0x1a); // uint32
    let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() as u32;
    cbor.extend_from_slice(&now.to_be_bytes());
    
    // Key 6: finished_at
    cbor.push(0x06);
    cbor.push(0x1a); // uint32
    cbor.extend_from_slice(&(now + 1).to_be_bytes());
    
    // Key 7: issuer_key_id
    cbor.push(0x07);
    cbor.push(0x64); // text(4)
    cbor.extend_from_slice(b"test");
    
    // Key 8: signature
    cbor.push(0x08);
    cbor.push(0x58); cbor.push(0x40);
    cbor.extend_from_slice(&[0x88; 64]);
    
    let result = OcxReceipt::from_canonical_cbor(&cbor);
    assert!(matches!(result, Err(VerificationError::MissingField("artifact_hash"))));
}

#[test]
fn test_invalid_hash_length() {
    let mut cbor = Vec::new();
    cbor.push(0xa8); // map(8)
    
    // Key 1: artifact_hash with wrong length (31 bytes instead of 32)
    cbor.push(0x01);
    cbor.push(0x58); cbor.push(0x1f); // bytes(31)
    cbor.extend_from_slice(&[0x01; 31]); // 31 bytes
    
    // Key 2: input_hash (32 bytes)
    cbor.push(0x02);
    cbor.push(0x58); cbor.push(0x20);
    cbor.extend_from_slice(&[0x02; 32]);
    
    // Key 3: output_hash (32 bytes)
    cbor.push(0x03);
    cbor.push(0x58); cbor.push(0x20);
    cbor.extend_from_slice(&[0x03; 32]);
    
    // Key 4: cycles_used
    cbor.push(0x04);
    cbor.push(0x19); // uint16
    cbor.extend_from_slice(&10000u16.to_be_bytes());
    
    // Key 5: started_at
    cbor.push(0x05);
    cbor.push(0x1a); // uint32
    let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() as u32;
    cbor.extend_from_slice(&now.to_be_bytes());
    
    // Key 6: finished_at
    cbor.push(0x06);
    cbor.push(0x1a); // uint32
    cbor.extend_from_slice(&(now + 1).to_be_bytes());
    
    // Key 7: issuer_key_id
    cbor.push(0x07);
    cbor.push(0x64); // text(4)
    cbor.extend_from_slice(b"test");
    
    // Key 8: signature
    cbor.push(0x08);
    cbor.push(0x58); cbor.push(0x40);
    cbor.extend_from_slice(&[0x88; 64]);
    
    let result = OcxReceipt::from_canonical_cbor(&cbor);
    assert!(matches!(result, Err(VerificationError::InvalidFieldValue("artifact_hash"))));
}

#[test]
fn test_invalid_signature_length() {
    let mut cbor_data = create_test_receipt_cbor();
    
    // Modify the signature length byte to be 63 instead of 64
    // Find the signature length byte and modify it
    for i in 0..cbor_data.len() {
        if cbor_data[i] == 0x40 { // bytes(64) = 0x58 0x40
            if i > 0 && cbor_data[i-1] == 0x58 {
                cbor_data[i] = 0x3f; // Change to bytes(63)
                cbor_data.truncate(cbor_data.len() - 1); // Remove one byte
                break;
            }
        }
    }
    
    let result = OcxReceipt::from_canonical_cbor(&cbor_data);
    assert!(matches!(result, Err(VerificationError::InvalidSignature)));
}

#[test]
fn test_invalid_timestamps() {
    // Create receipt where finished_at < started_at by swapping timestamp values
    let mut cbor = Vec::new();
    cbor.push(0xa8); // map(8)
    
    // Key 1: artifact_hash (32 bytes)
    cbor.push(0x01);
    cbor.push(0x58); cbor.push(0x20);
    cbor.extend_from_slice(&[0x01; 32]);
    
    // Key 2: input_hash (32 bytes)
    cbor.push(0x02);
    cbor.push(0x58); cbor.push(0x20);
    cbor.extend_from_slice(&[0x02; 32]);
    
    // Key 3: output_hash (32 bytes)
    cbor.push(0x03);
    cbor.push(0x58); cbor.push(0x20);
    cbor.extend_from_slice(&[0x03; 32]);
    
    // Key 4: cycles_used
    cbor.push(0x04);
    cbor.push(0x19); // uint16 (minimal encoding for 10000)
    cbor.extend_from_slice(&10000u16.to_be_bytes());
    
    let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() as u32;
    
    // Key 5: started_at (later time)
    cbor.push(0x05);
    cbor.push(0x1a);
    cbor.extend_from_slice(&(now + 100).to_be_bytes());
    
    // Key 6: finished_at (earlier time)
    cbor.push(0x06);
    cbor.push(0x1a);
    cbor.extend_from_slice(&now.to_be_bytes());
    
    // Key 7: issuer_key_id
    cbor.push(0x07);
    cbor.push(0x64);
    cbor.extend_from_slice(b"test");
    
    // Key 8: signature
    cbor.push(0x08);
    cbor.push(0x58); cbor.push(0x40);
    cbor.extend_from_slice(&[0x88; 64]);
    
    let result = OcxReceipt::from_canonical_cbor(&cbor);
    assert!(matches!(result, Err(VerificationError::InvalidTimestamp)));
}

#[test]
fn test_zero_cycles() {
    let cbor_data = create_test_receipt_cbor();
    
    // Find and modify cycles_used to be 0
    // Create new CBOR with cycles = 0
    let mut cbor = Vec::new();
    cbor.push(0xa8); // map(8)
    
    // Keys 1-3: hashes
    for key in 1..=3 {
        cbor.push(key);
        cbor.push(0x58); cbor.push(0x20);
        cbor.extend_from_slice(&[key; 32]);
    }
    
    // Key 4: cycles_used = 0
    cbor.push(0x04);
    cbor.push(0x00); // integer(0)
    
    let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() as u32;
    
    // Keys 5-6: timestamps
    cbor.push(0x05);
    cbor.push(0x1a);
    cbor.extend_from_slice(&now.to_be_bytes());
    
    cbor.push(0x06);
    cbor.push(0x1a);
    cbor.extend_from_slice(&(now + 1).to_be_bytes());
    
    // Key 7: issuer_key_id
    cbor.push(0x07);
    cbor.push(0x64);
    cbor.extend_from_slice(b"test");
    
    // Key 8: signature
    cbor.push(0x08);
    cbor.push(0x58); cbor.push(0x40);
    cbor.extend_from_slice(&[0x88; 64]);
    
    let result = OcxReceipt::from_canonical_cbor(&cbor);
    assert!(matches!(result, Err(VerificationError::InvalidFieldValue("cycles_used"))));
}

#[test]
fn test_signed_data_generation() {
    let cbor_data = create_test_receipt_cbor();
    let receipt = OcxReceipt::from_canonical_cbor(&cbor_data).unwrap();
    
    let signed_data = receipt.signed_data();
    assert!(signed_data.is_ok(), "Failed to generate signed data: {:?}", signed_data.err());
    
    let signed_bytes = signed_data.unwrap();
    assert!(!signed_bytes.is_empty());
    
    // The signed data should be a valid CBOR map without the signature field
    // So it should be a map(7) instead of map(8)
    assert_eq!(signed_bytes[0], 0xa7); // map(7)
}

#[test]
fn test_roundtrip_signed_data() {
    let cbor_data = create_test_receipt_cbor();
    let receipt = OcxReceipt::from_canonical_cbor(&cbor_data).unwrap();
    
    // Generate signed data
    let signed_data = receipt.signed_data().unwrap();
    
    // Parse the signed data back (it should be valid CBOR)
    use libocx_verify::canonical_cbor::CborParser;
    let parsed = CborParser::new(&signed_data).parse_full();
    assert!(parsed.is_ok(), "Signed data is not valid canonical CBOR");
    
    // The parsed data should be a map with 7 fields (all except signature)
    if let Ok(libocx_verify::canonical_cbor::CanonicalValue::Map(map)) = parsed {
        assert_eq!(map.len(), 7); // All fields except signature
    } else {
        panic!("Signed data is not a map");
    }
}

#[test]
fn test_invalid_key_id() {
    let mut cbor = Vec::new();
    cbor.push(0xa8); // map(8)
    
    // Key 1: artifact_hash (32 bytes)
    cbor.push(0x01);
    cbor.push(0x58); cbor.push(0x20);
    cbor.extend_from_slice(&[0x01; 32]);
    
    // Key 2: input_hash (32 bytes)
    cbor.push(0x02);
    cbor.push(0x58); cbor.push(0x20);
    cbor.extend_from_slice(&[0x02; 32]);
    
    // Key 3: output_hash (32 bytes)
    cbor.push(0x03);
    cbor.push(0x58); cbor.push(0x20);
    cbor.extend_from_slice(&[0x03; 32]);
    
    // Key 4: cycles_used
    cbor.push(0x04);
    cbor.push(0x19); // uint16 (minimal encoding for 10000)
    cbor.extend_from_slice(&10000u16.to_be_bytes());
    
    let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs() as u32;
    
    // Key 5: started_at
    cbor.push(0x05);
    cbor.push(0x1a);
    cbor.extend_from_slice(&now.to_be_bytes());
    
    // Key 6: finished_at
    cbor.push(0x06);
    cbor.push(0x1a);
    cbor.extend_from_slice(&(now + 1).to_be_bytes());
    
    // Key 7: empty issuer_key_id (invalid)
    cbor.push(0x07);
    cbor.push(0x60); // text(0) - empty string
    
    // Key 8: signature
    cbor.push(0x08);
    cbor.push(0x58); cbor.push(0x40);
    cbor.extend_from_slice(&[0x88; 64]);
    
    let result = OcxReceipt::from_canonical_cbor(&cbor);
    assert!(matches!(result, Err(VerificationError::InvalidFieldValue("issuer_key_id"))));
}
