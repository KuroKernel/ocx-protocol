//! Debug utilities for verification troubleshooting

use crate::{OcxReceipt, VerificationError};
use std::fmt;

pub struct VerificationDebugInfo {
    pub parsed_fields: Vec<String>,
    pub canonical_cbor_hex: String,
    pub message_prefix: String,
    pub signature_algorithm: String,
    pub ed25519_result: String,
}

impl fmt::Display for VerificationDebugInfo {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        writeln!(f, "=== VERIFICATION DEBUG INFO ===")?;
        writeln!(f, "Parsed fields: {:?}", self.parsed_fields)?;
        writeln!(f, "Canonical CBOR (first 64 hex): {}", &self.canonical_cbor_hex[..128.min(self.canonical_cbor_hex.len())])?;
        writeln!(f, "Message prefix: {}", self.message_prefix)?;
        writeln!(f, "Signature algorithm: {}", self.signature_algorithm)?;
        writeln!(f, "Ed25519 result: {}", self.ed25519_result)?;
        writeln!(f, "==============================")
    }
}

pub fn create_debug_info(
    receipt: &OcxReceipt, 
    verification_result: &Result<(), VerificationError>
) -> VerificationDebugInfo {
    
    let canonical_cbor = receipt.to_canonical_cbor().unwrap_or_default();
    let message = receipt.get_signing_message().unwrap_or_default();
    
    VerificationDebugInfo {
        parsed_fields: vec![
            "version".to_string(),
            "issuer_id".to_string(), 
            "timestamp_ms".to_string(),
            "program_hash".to_string(),
            "input_hash".to_string(),
            "output_hash".to_string(),
            "cycles".to_string(),
            "sig_alg".to_string(),
            "signature".to_string(),
        ],
        canonical_cbor_hex: hex::encode(&canonical_cbor),
        message_prefix: hex::encode(&message[..32.min(message.len())]),
        signature_algorithm: "ed25519".to_string(),
        ed25519_result: match verification_result {
            Ok(_) => "SUCCESS".to_string(),
            Err(e) => format!("FAILED: {:?}", e),
        },
    }
}
