use libocx_verify::canonical_cbor::{CanonicalValue, CborParser};
use libocx_verify::VerificationError;

#[test]
fn test_rejects_non_minimal_encoding() {
    // Encoded 0x00 (0) as 0x1800 (uint8) is non-minimal -> should error.
    let non_minimal = vec![0x18, 0x00];
    let result = CborParser::new(&non_minimal).parse_full();
    assert_eq!(result, Err(VerificationError::NonCanonicalCbor));
}

#[test]
fn test_rejects_non_sorted_map_keys() {
    // Map with keys {2: "b", 1: "a"} -> should error because keys are not sorted
    // CBOR: map(2) { 2: "b", 1: "a" }
    let non_sorted = vec![
        0xa2,       // map(2)
        0x02,       // key: 2
        0x61, 0x62, // value: "b"
        0x01,       // key: 1
        0x61, 0x61, // value: "a"
    ];
    let result = CborParser::new(&non_sorted).parse_full();
    assert_eq!(result, Err(VerificationError::NonCanonicalCbor));
}

#[test]
fn test_accepts_canonical_map() {
    // Map with keys {1: "a", 2: "b"} -> should succeed
    let canonical = vec![
        0xa2,       // map(2)
        0x01,       // key: 1
        0x61, 0x61, // value: "a"
        0x02,       // key: 2
        0x61, 0x62, // value: "b"
    ];
    let result = CborParser::new(&canonical).parse_full();
    assert!(result.is_ok());
}

#[test]
fn test_rejects_overlong_uint() {
    // Encoded 23 as 0x1817 (uint8) when it could be 0x17 -> should error
    let overlong = vec![0x18, 0x17];
    let result = CborParser::new(&overlong).parse_full();
    assert_eq!(result, Err(VerificationError::NonCanonicalCbor));
}

#[test]
fn test_accepts_minimal_uint() {
    // 24 encoded as 0x1818 is correct (24 requires uint8 encoding)
    let minimal = vec![0x18, 0x18];
    let result = CborParser::new(&minimal).parse_full();
    assert!(result.is_ok());
    
    if let Ok(CanonicalValue::Integer(value)) = result {
        assert_eq!(value, 24);
    } else {
        panic!("Expected integer value");
    }
}

#[test]
fn test_rejects_trailing_data() {
    // Valid integer followed by extra byte
    let trailing = vec![0x01, 0xff];
    let result = CborParser::new(&trailing).parse_full();
    assert_eq!(result, Err(VerificationError::NonCanonicalCbor));
}

#[test]
fn test_accepts_empty_array() {
    let empty_array = vec![0x80]; // array(0)
    let result = CborParser::new(&empty_array).parse_full();
    assert!(result.is_ok());
    
    if let Ok(CanonicalValue::Array(arr)) = result {
        assert_eq!(arr.len(), 0);
    } else {
        panic!("Expected array value");
    }
}

#[test]
fn test_accepts_empty_map() {
    let empty_map = vec![0xa0]; // map(0)
    let result = CborParser::new(&empty_map).parse_full();
    assert!(result.is_ok());
    
    if let Ok(CanonicalValue::Map(map)) = result {
        assert_eq!(map.len(), 0);
    } else {
        panic!("Expected map value");
    }
}

#[test]
fn test_rejects_invalid_utf8() {
    // Text string with invalid UTF-8
    let invalid_utf8 = vec![
        0x62,       // text(2)
        0xff, 0xfe, // invalid UTF-8 sequence
    ];
    let result = CborParser::new(&invalid_utf8).parse_full();
    assert_eq!(result, Err(VerificationError::InvalidUtf8));
}

#[test]
fn test_accepts_valid_utf8() {
    // Text string with valid UTF-8: "hello"
    let valid_utf8 = vec![
        0x65,                               // text(5)
        0x68, 0x65, 0x6c, 0x6c, 0x6f,       // "hello"
    ];
    let result = CborParser::new(&valid_utf8).parse_full();
    assert!(result.is_ok());
    
    if let Ok(CanonicalValue::Text(text)) = result {
        assert_eq!(text, "hello");
    } else {
        panic!("Expected text value");
    }
}

#[test]
fn test_rejects_duplicate_map_keys() {
    // Map with duplicate key "a"
    let duplicate_keys = vec![
        0xa2,       // map(2)
        0x61, 0x61, // key: "a"
        0x01,       // value: 1
        0x61, 0x61, // key: "a" (duplicate)
        0x02,       // value: 2
    ];
    let result = CborParser::new(&duplicate_keys).parse_full();
    assert_eq!(result, Err(VerificationError::NonCanonicalCbor));
}

#[test]
fn test_complex_canonical_structure() {
    // Complex structure: {1: [1, 2], "text": {"nested": 42}}
    let complex = vec![
        0xa2,                   // map(2)
        0x01,                   // key: 1
        0x82, 0x01, 0x02,       // value: [1, 2]
        0x64, 0x74, 0x65, 0x78, 0x74, // key: "text"
        0xa1,                   // value: map(1)
        0x66, 0x6e, 0x65, 0x73, 0x74, 0x65, 0x64, // key: "nested"
        0x18, 0x2a,             // value: 42
    ];
    let result = CborParser::new(&complex).parse_full();
    assert!(result.is_ok());
}
