//! OCX Receipt specification constants and validation

use crate::VerificationError;

/// OCX Receipt specification version
pub const OCX_VERSION: &str = "ocx-1";

/// Domain separation prefix for signing
pub const DOMAIN_SEPARATOR: &[u8] = b"OCXv1|receipt|";

/// Ed25519 signature algorithm identifier
pub const SIG_ALG_ED25519: &str = "ed25519";

/// Required receipt field names (in lexical order)
pub const REQUIRED_FIELDS: &[&str] = &[
    "cycles",
    "input_hash", 
    "issuer_id",
    "output_hash",
    "program_hash",
    "sig_alg",
    "signature",
    "timestamp_ms",
    "version",
];

/// Validates that a receipt map contains all required fields
pub fn validate_receipt_fields(fields: &[&str]) -> Result<(), VerificationError> {
    for required_field in REQUIRED_FIELDS {
        if !fields.contains(required_field) {
            return Err(VerificationError::MissingField(required_field));
        }
    }
    Ok(())
}

/// Creates the canonical message bytes for signing
pub fn create_signing_message(receipt_core_cbor: &[u8]) -> Vec<u8> {
    let mut message = Vec::with_capacity(DOMAIN_SEPARATOR.len() + receipt_core_cbor.len());
    message.extend_from_slice(DOMAIN_SEPARATOR);
    message.extend_from_slice(receipt_core_cbor);
    message
}
