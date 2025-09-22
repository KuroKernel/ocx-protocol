use libocx_verify::canonical_cbor::{CanonicalValue, CborParser};
use libocx_verify::VerificationError;

fn main() {
    // Test with a simple CBOR map
    let test_cbor = vec![
        0xa2, // map with 2 pairs
        0x66, 0x63, 0x79, 0x63, 0x6c, 0x65, 0x73, // "cycles"
        0x19, 0x03, 0xe8, // 1000
        0x67, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, // "version"
        0x66, 0x6f, 0x63, 0x78, 0x2d, 0x31, // "ocx-1"
    ];
    
    println!("Testing CBOR: {:?}", test_cbor);
    
    match CborParser::new(&test_cbor).parse_full() {
        Ok(value) => {
            println!("✓ Parsed successfully: {:?}", value);
        }
        Err(e) => {
            println!("✗ Parsing failed: {:?}", e);
        }
    }
}
