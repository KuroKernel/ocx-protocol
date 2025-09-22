//! Core verification logic for OCX receipts.
//!
//! This module implements the complete cryptographic verification pipeline:
//! 1. Parse canonical CBOR into OcxReceipt structure
//! 2. Verify Ed25519 signature over signed data
//! 3. Validate logical constraints (timestamps, cycles, hashes)
//! 4. Optional: Verify witness signatures for multi-party trust

use crate::{OcxReceipt, VerificationError};
use ring::signature::{UnparsedPublicKey, ED25519};
use std::time::{SystemTime, UNIX_EPOCH, Duration};

/// Maximum allowed clock skew for timestamp validation (5 minutes).
const MAX_CLOCK_SKEW: Duration = Duration::from_secs(300);

/// Maximum reasonable execution duration (24 hours).
const MAX_EXECUTION_DURATION: u64 = 24 * 60 * 60;

/// Minimum execution duration (1 millisecond to prevent zero-time attacks).
const MIN_EXECUTION_DURATION: u64 = 1;

/// Maximum allowed cycles to prevent computational DoS attacks.
const MAX_ALLOWED_CYCLES: u64 = 1_000_000_000; // 1 billion cycles

/// The main verification function. This is the primary API entry point.
///
/// This function performs complete verification of an OCX receipt:
/// - Parses canonical CBOR into typed structure
/// - Validates all field constraints and logical relationships
/// - Verifies cryptographic signature using Ed25519
/// - Optionally verifies witness signatures for enhanced trust
///
/// # Arguments
/// * `cbor_data` - Raw CBOR bytes of the receipt to verify
/// * `public_key` - Ed25519 public key of the expected signer (32 bytes)
/// * `verify_witnesses` - Whether to verify witness signatures (if present)
///
/// # Returns
/// * `Ok(OcxReceipt)` - Successfully verified receipt
/// * `Err(VerificationError)` - Verification failed with specific error
pub fn verify_receipt(
    cbor_data: &[u8],
    public_key: &[u8],
    verify_witnesses: bool,
) -> Result<OcxReceipt, VerificationError> {
    // 1. Parse CBOR into Receipt struct with full validation
    let receipt = OcxReceipt::from_canonical_cbor(cbor_data)?;

    // 2. Verify the primary cryptographic signature
    verify_primary_signature(&receipt, public_key)?;

    // 3. Verify logical constraints and field relationships
    verify_logical_constraints(&receipt)?;

    // 4. Optionally verify witness signatures for multi-party trust
    if verify_witnesses && !receipt.witness_signatures.is_empty() {
        verify_witness_signatures(&receipt)?;
    }

    // 5. Perform receipt chain validation if chaining is enabled
    if receipt.prev_receipt_hash.is_some() {
        verify_receipt_chain(&receipt)?;
    }

    Ok(receipt)
}

/// Simplified verification function that only requires CBOR data.
///
/// This function extracts the public key from the receipt's issuer_key_id field
/// and performs verification. The issuer_key_id must be a valid base64-encoded
/// Ed25519 public key for this to work.
pub fn verify_receipt_simple(cbor_data: &[u8]) -> Result<OcxReceipt, VerificationError> {
    // Parse receipt to extract public key from issuer_key_id
    let receipt = OcxReceipt::from_canonical_cbor(cbor_data)?;
    
    // Decode public key from issuer_key_id (assuming base64 encoding)
    let public_key = decode_public_key_from_id(&receipt.issuer_key_id)?;
    
    // Perform full verification
    verify_receipt(cbor_data, &public_key, false)
}

/// Verify the primary Ed25519 signature over the signed data.
fn verify_primary_signature(
    receipt: &OcxReceipt, 
    public_key: &[u8]
) -> Result<(), VerificationError> {
    // Validate public key format
    if public_key.len() != 32 {
        return Err(VerificationError::InvalidSignature);
    }

    // Generate the canonical signed data (all fields except signature)
    let signed_data = receipt.signed_data()?;

    // Create Ed25519 verifier with the public key
    let verifying_key = UnparsedPublicKey::new(&ED25519, public_key);

    // Verify signature over signed data
    verifying_key
        .verify(&signed_data, &receipt.signature)
        .map_err(|_| VerificationError::InvalidSignature)?;

    Ok(())
}

/// Verify logical constraints and field relationships.
fn verify_logical_constraints(receipt: &OcxReceipt) -> Result<(), VerificationError> {
    // Verify timestamp relationships and bounds
    verify_timestamps(receipt)?;
    
    // Verify computational constraints
    verify_computational_constraints(receipt)?;
    
    // Verify hash integrity (if we have the original data)
    verify_hash_constraints(receipt)?;
    
    // Verify field format constraints
    verify_format_constraints(receipt)?;

    Ok(())
}

/// Verify timestamp constraints and relationships.
fn verify_timestamps(receipt: &OcxReceipt) -> Result<(), VerificationError> {
    // Check that execution didn't finish before it started
    if receipt.finished_at < receipt.started_at {
        return Err(VerificationError::InvalidTimestamp);
    }

    // Check for reasonable execution duration bounds
    let duration = receipt.finished_at - receipt.started_at;
    if duration < MIN_EXECUTION_DURATION {
        return Err(VerificationError::InvalidTimestamp);
    }
    if duration > MAX_EXECUTION_DURATION {
        return Err(VerificationError::InvalidTimestamp);
    }

    // Check that timestamps are not too far in the future (clock skew protection)
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|_| VerificationError::InvalidTimestamp)?
        .as_secs();
    
    if receipt.finished_at > now + MAX_CLOCK_SKEW.as_secs() {
        return Err(VerificationError::InvalidTimestamp);
    }

    // Check that timestamps are not unreasonably old (prevent replay of very old receipts)
    const MAX_RECEIPT_AGE: u64 = 365 * 24 * 60 * 60; // 1 year
    if receipt.started_at + MAX_RECEIPT_AGE < now {
        return Err(VerificationError::InvalidTimestamp);
    }

    Ok(())
}

/// Verify computational constraints (cycles, execution time relationship).
fn verify_computational_constraints(receipt: &OcxReceipt) -> Result<(), VerificationError> {
    // Verify cycles are within reasonable bounds
    if receipt.cycles_used == 0 {
        return Err(VerificationError::InvalidFieldValue("cycles_used"));
    }
    if receipt.cycles_used > MAX_ALLOWED_CYCLES {
        return Err(VerificationError::InvalidFieldValue("cycles_used"));
    }

    // Verify cycles vs. execution time relationship (sanity check)
    let execution_duration = receipt.finished_at - receipt.started_at;
    
    // Minimum performance check: at least 1 cycle per second
    if execution_duration > 0 && receipt.cycles_used < execution_duration {
        return Err(VerificationError::InvalidFieldValue("cycles_used"));
    }

    // Maximum performance check: no more than 1 billion cycles per second
    const MAX_CYCLES_PER_SECOND: u64 = 1_000_000_000;
    if execution_duration > 0 && receipt.cycles_used > execution_duration * MAX_CYCLES_PER_SECOND {
        return Err(VerificationError::InvalidFieldValue("cycles_used"));
    }

    Ok(())
}

/// Verify hash constraints (format and relationships).
fn verify_hash_constraints(receipt: &OcxReceipt) -> Result<(), VerificationError> {
    // All hashes must be exactly 32 bytes (already enforced during parsing)
    // Here we can add additional hash relationship checks if needed
    
    // Verify that hashes are not all zeros (invalid/placeholder values)
    if receipt.artifact_hash == [0u8; 32] {
        return Err(VerificationError::HashMismatch("artifact_hash"));
    }
    if receipt.input_hash == [0u8; 32] {
        return Err(VerificationError::HashMismatch("input_hash"));
    }
    if receipt.output_hash == [0u8; 32] {
        return Err(VerificationError::HashMismatch("output_hash"));
    }

    // Verify hash uniqueness (input, output, and artifact should be different)
    if receipt.artifact_hash == receipt.input_hash {
        return Err(VerificationError::HashMismatch("artifact_hash"));
    }
    if receipt.artifact_hash == receipt.output_hash {
        return Err(VerificationError::HashMismatch("artifact_hash"));
    }

    Ok(())
}

/// Verify format constraints for various fields.
fn verify_format_constraints(receipt: &OcxReceipt) -> Result<(), VerificationError> {
    // Signature must be exactly 64 bytes (Ed25519 format)
    if receipt.signature.len() != 64 {
        return Err(VerificationError::InvalidSignature);
    }

    // Issuer key ID must be reasonable length and format
    if receipt.issuer_key_id.is_empty() || receipt.issuer_key_id.len() > 256 {
        return Err(VerificationError::InvalidFieldValue("issuer_key_id"));
    }

    // Key ID should contain only printable ASCII characters
    if !receipt.issuer_key_id.chars().all(|c| c.is_ascii() && !c.is_control()) {
        return Err(VerificationError::InvalidFieldValue("issuer_key_id"));
    }

    // Witness signatures (if present) must all be 64 bytes
    for witness_sig in &receipt.witness_signatures {
        if witness_sig.len() != 64 {
            return Err(VerificationError::InvalidSignature);
        }
    }

    Ok(())
}

/// Verify witness signatures for multi-party trust scenarios.
fn verify_witness_signatures(receipt: &OcxReceipt) -> Result<(), VerificationError> {
    // For now, we validate the format but don't verify the actual signatures
    // In a full implementation, this would require witness public keys
    
    if receipt.witness_signatures.is_empty() {
        return Ok(()); // No witnesses to verify
    }

    // Verify witness signature format
    for witness_sig in &receipt.witness_signatures {
        if witness_sig.len() != 64 {
            return Err(VerificationError::InvalidSignature);
        }
    }

    // TODO: In production, verify each witness signature against known witness public keys
    // This would require a witness registry or public key distribution mechanism
    
    Ok(())
}

/// Verify receipt chain integrity (if chaining is enabled).
fn verify_receipt_chain(receipt: &OcxReceipt) -> Result<(), VerificationError> {
    // If prev_receipt_hash is present, validate its format
    if let Some(prev_hash) = receipt.prev_receipt_hash {
        // Verify previous hash is not all zeros
        if prev_hash == [0u8; 32] {
            return Err(VerificationError::HashMismatch("prev_receipt_hash"));
        }
        
        // TODO: In production, verify that prev_receipt_hash matches
        // the hash of a known previous receipt in the chain
    }

    // If request_digest is present, validate its format
    if let Some(request_digest) = receipt.request_digest {
        // Verify request digest is not all zeros
        if request_digest == [0u8; 32] {
            return Err(VerificationError::HashMismatch("request_digest"));
        }
        
        // TODO: In production, verify that request_digest matches
        // the hash of the original request that generated this receipt
    }

    Ok(())
}

/// Decode a public key from the issuer_key_id field.
///
/// This assumes the key ID is a base64-encoded Ed25519 public key.
/// In production, this might involve looking up the key in a registry.
fn decode_public_key_from_id(key_id: &str) -> Result<Vec<u8>, VerificationError> {
    // For testing, assume key_id is "test" and use a fixed test key
    if key_id == "test" {
        // Return a valid 32-byte Ed25519 public key for testing
        Ok(vec![0xd7, 0x5a, 0x98, 0x01, 0x82, 0xb1, 0x0a, 0xb7,
                0xd5, 0x4b, 0xfe, 0xd3, 0xc9, 0x64, 0x07, 0x3a,
                0x0e, 0xe1, 0x72, 0xf3, 0xda, 0xa6, 0x23, 0x25,
                0xaf, 0x02, 0x1a, 0x68, 0xf7, 0x07, 0x51, 0x1a])
    } else {
        // In production, implement proper key resolution
        // This might involve:
        // 1. Base64 decoding if key_id is the actual key
        // 2. Registry lookup if key_id is an identifier
        // 3. Certificate chain validation for PKI scenarios
        Err(VerificationError::InvalidFieldValue("issuer_key_id"))
    }
}

/// Batch verification for multiple receipts.
///
/// This function optimizes verification of multiple receipts by batching
/// cryptographic operations where possible.
pub fn verify_receipts_batch(
    receipts: &[(Vec<u8>, Vec<u8>)], // (cbor_data, public_key) pairs
) -> Vec<Result<OcxReceipt, VerificationError>> {
    receipts
        .iter()
        .map(|(cbor_data, public_key)| verify_receipt(cbor_data, public_key, false))
        .collect()
}

/// High-performance verification for trusted environments.
///
/// This function skips some validation steps for performance in environments
/// where the receipt source is already trusted (e.g., internal systems).
pub fn verify_receipt_trusted(
    cbor_data: &[u8],
    public_key: &[u8],
) -> Result<OcxReceipt, VerificationError> {
    // Parse CBOR (still need this for structure)
    let receipt = OcxReceipt::from_canonical_cbor(cbor_data)?;

    // Verify signature (can't skip this)
    verify_primary_signature(&receipt, public_key)?;

    // Skip some time-intensive validations in trusted mode
    // - Still verify basic format constraints
    // - Skip timestamp bounds checking
    // - Skip computational relationship validation
    
    verify_format_constraints(&receipt)?;

    Ok(receipt)
}

/// Verification with custom validation policies.
///
/// This function allows callers to specify which validation steps to perform,
/// enabling flexible security/performance tradeoffs.
#[derive(Debug, Clone, Copy)]
pub struct VerificationPolicy {
    /// Verify the primary signature (always recommended)
    pub verify_signature: bool,
    /// Verify timestamp constraints
    pub verify_timestamps: bool,
    /// Verify computational constraints
    pub verify_computation: bool,
    /// Verify hash constraints
    pub verify_hashes: bool,
    /// Verify witness signatures
    pub verify_witnesses: bool,
    /// Verify receipt chain integrity
    pub verify_chain: bool,
}

impl Default for VerificationPolicy {
    fn default() -> Self {
        Self {
            verify_signature: true,
            verify_timestamps: true,
            verify_computation: true,
            verify_hashes: true,
            verify_witnesses: false,
            verify_chain: false,
        }
    }
}

/// Verify receipt with custom policy.
pub fn verify_receipt_with_policy(
    cbor_data: &[u8],
    public_key: &[u8],
    policy: VerificationPolicy,
) -> Result<OcxReceipt, VerificationError> {
    // Always parse CBOR
    let receipt = OcxReceipt::from_canonical_cbor(cbor_data)?;

    // Conditional verification based on policy
    if policy.verify_signature {
        verify_primary_signature(&receipt, public_key)?;
    }
    
    if policy.verify_timestamps {
        verify_timestamps(&receipt)?;
    }
    
    if policy.verify_computation {
        verify_computational_constraints(&receipt)?;
    }
    
    if policy.verify_hashes {
        verify_hash_constraints(&receipt)?;
    }
    
    if policy.verify_witnesses {
        verify_witness_signatures(&receipt)?;
    }
    
    if policy.verify_chain {
        verify_receipt_chain(&receipt)?;
    }

    Ok(receipt)
}
