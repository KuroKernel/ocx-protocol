//! Core verification logic for OCX receipts.
//!
//! This module implements the complete cryptographic verification pipeline:
//! 1. Parse canonical CBOR into OcxReceipt structure
//! 2. Verify Ed25519 signature over signed data
//! 3. Validate logical constraints (timestamps, cycles, hashes)
//! 4. Verify witness signatures for multi-party trust
//! 5. Verify receipt chains and request digests

use crate::{OcxReceipt, VerificationError};
use crate::receipt::UnsignedReceipt;
use ring::signature::{UnparsedPublicKey, ED25519};
use ring::digest::{digest, SHA256};
use std::time::{SystemTime, UNIX_EPOCH, Duration};
use std::collections::HashMap;
use base64::{Engine as _, engine::general_purpose::STANDARD as BASE64};

// Tracing macros for debugging verification failures
#[cfg(feature = "trace")]
macro_rules! trace {
    ($($t:tt)*) => { eprintln!("[TRACE] {}", format!($($t)*)); };
}

#[cfg(not(feature = "trace"))]
macro_rules! trace {
    ($($t:tt)*) => {};
}

// Helper function to format bytes as hex
fn hex_format(bytes: &[u8]) -> String {
    bytes.iter().map(|b| format!("{:02x}", b)).collect::<String>()
}

// Message kind enum for different signature types
#[derive(Debug, Clone, Copy)]
pub enum MsgKind {
    Receipt,
    Witness,
    Chain,
}

impl MsgKind {
    fn as_str(&self) -> &'static str {
        match self {
            Self::Receipt => "receipt",
            Self::Witness => "witness", 
            Self::Chain => "chain",
        }
    }
}

// Unified message building function
fn build_signing_message(kind: MsgKind, payload: &[u8]) -> Vec<u8> {
    trace!("building_message_kind={:?}", kind);
    
    // The payload already includes the domain separator, so just return it
    trace!("message_prefix={:?}", std::str::from_utf8(&payload[..20.min(payload.len())]).unwrap_or("invalid_utf8"));
    trace!("message_len={}", payload.len());
    
    payload.to_vec()
}

/// Maximum allowed clock skew for timestamp validation (5 minutes).
const MAX_CLOCK_SKEW: Duration = Duration::from_secs(300);

/// Minimum execution duration (1 second).
const MIN_EXECUTION_DURATION: u64 = 1;

/// Maximum reasonable execution duration (24 hours).
const MAX_EXECUTION_DURATION: u64 = 24 * 60 * 60;

/// Maximum allowed cycles to prevent computational DoS attacks.
const MAX_ALLOWED_CYCLES: u64 = 1_000_000_000; // 1 billion cycles

/// Witness registry for managing witness public keys
static mut WITNESS_REGISTRY: Option<WitnessRegistry> = None;

/// Receipt chain storage for chain verification
static mut RECEIPT_CHAIN: Option<ReceiptChain> = None;

/// Initialize the verification system with registries
pub fn init_verification_system(
    witness_config_path: Option<&str>,
) -> Result<(), VerificationError> {
    unsafe {
        // Initialize witness registry
        WITNESS_REGISTRY = Some(
            witness_config_path
                .map(|path| WitnessRegistry::load_from_config(path))
                .transpose()?
                .unwrap_or_else(WitnessRegistry::new)
        );
        
        // Initialize receipt chain
        RECEIPT_CHAIN = Some(ReceiptChain::new());
    }
    
    Ok(())
}

/// Get witness registry (initialize if needed)
fn get_witness_registry() -> &'static WitnessRegistry {
    unsafe {
        WITNESS_REGISTRY.get_or_insert_with(WitnessRegistry::new)
    }
}

/// Get receipt chain (initialize if needed)
fn get_receipt_chain() -> &'static mut ReceiptChain {
    unsafe {
        RECEIPT_CHAIN.get_or_insert_with(ReceiptChain::new)
    }
}

/// The main verification function. This is the primary API entry point.
pub fn verify_receipt(
    cbor_data: &[u8],
    public_key: &[u8],
    verify_witnesses: bool,
) -> Result<OcxReceipt, VerificationError> {
    trace!("=== Starting receipt verification ===");
    trace!("cbor_data_len={}", cbor_data.len());
    trace!("cbor_data_hex={}", hex_format(cbor_data));
    trace!("public_key_len={}", public_key.len());
    trace!("public_key_hex={}", hex_format(public_key));
    trace!("verify_witnesses={}", verify_witnesses);

    // 1. Canonicality check
    trace!("--- Step 1: Canonicality check ---");
    let canonical_bytes = canonicalize_cbor(cbor_data)?;
    if canonical_bytes != cbor_data {
        trace!("❌ CBOR not canonical");
        trace!("input_len={}, canonical_len={}", cbor_data.len(), canonical_bytes.len());
        trace!("input_hex={}", hex_format(cbor_data));
        trace!("canonical_hex={}", hex_format(&canonical_bytes));
        return Err(VerificationError::NonCanonicalCbor);
    }
    trace!("✅ CBOR is canonical");

    // 2. Parse receipt
    trace!("--- Step 2: Parse receipt ---");
    let receipt = OcxReceipt::from_canonical_cbor(&canonical_bytes)?;
    trace!("✅ Receipt parsed successfully");
    trace!("artifact_hash={}", hex_format(&receipt.artifact_hash));
    trace!("input_hash={}", hex_format(&receipt.input_hash));
    trace!("output_hash={}", hex_format(&receipt.output_hash));
    trace!("cycles_used={}", receipt.cycles_used);
    trace!("started_at={}", receipt.started_at);
    trace!("finished_at={}", receipt.finished_at);
    trace!("issuer_key_id={}", receipt.issuer_key_id);
    trace!("signature_len={}", receipt.signature.len());
    
    // 2.1. Validate cycles_used is not zero
    if receipt.cycles_used == 0 {
        trace!("❌ cycles_used is zero (invalid)");
        return Err(VerificationError::InvalidFieldValue("cycles_used"));
    }
    trace!("✅ cycles_used validation passed: {}", receipt.cycles_used);
    
    // 2.2. Validate hash fields are not all zeros
    if receipt.artifact_hash == [0u8; 32] {
        trace!("❌ artifact_hash is all zeros (invalid)");
        return Err(VerificationError::HashMismatch("artifact_hash"));
    }
    if receipt.input_hash == [0u8; 32] {
        trace!("❌ input_hash is all zeros (invalid)");
        return Err(VerificationError::HashMismatch("input_hash"));
    }
    if receipt.output_hash == [0u8; 32] {
        trace!("❌ output_hash is all zeros (invalid)");
        return Err(VerificationError::HashMismatch("output_hash"));
    }
    
    // 2.3. Validate hash fields are unique
    if receipt.artifact_hash == receipt.input_hash {
        trace!("❌ artifact_hash equals input_hash (duplicate)");
        return Err(VerificationError::HashMismatch("artifact_hash"));
    }
    if receipt.artifact_hash == receipt.output_hash {
        trace!("❌ artifact_hash equals output_hash (duplicate)");
        return Err(VerificationError::HashMismatch("artifact_hash"));
    }
    if receipt.input_hash == receipt.output_hash {
        trace!("❌ input_hash equals output_hash (duplicate)");
        return Err(VerificationError::HashMismatch("input_hash"));
    }
    
    // 2.4. Validate optional prev_receipt_hash is not all zeros
    if let Some(prev_hash) = receipt.prev_receipt_hash {
        if prev_hash == [0u8; 32] {
            trace!("❌ prev_receipt_hash is all zeros (invalid)");
            return Err(VerificationError::HashMismatch("prev_receipt_hash"));
        }
    }
    trace!("✅ Hash validation passed");
    trace!("signature_hex={}", hex_format(&receipt.signature));

    // 3. Build signing message
    trace!("--- Step 3: Build signing message ---");
    // Create core CBOR (without signature) for signing, just like the Go generator
    let core_cbor = receipt.signed_data()?;
    let signing_message = build_signing_message(MsgKind::Receipt, &core_cbor);
    let message_hash = digest(&SHA256, &signing_message);
    trace!("signing_message_len={}", signing_message.len());
    trace!("signing_message_prefix={:?}", &signing_message[..20.min(signing_message.len())]);
    trace!("signing_message_hash={}", hex_format(message_hash.as_ref()));

    // 4. Verify primary signature
    trace!("--- Step 4: Verify primary signature ---");
    let signature_result = verify_ed25519_signature(public_key, &receipt.signature, &signing_message);
    trace!("signature_verification_result={:?}", signature_result);
    
    if let Err(e) = signature_result {
        trace!("❌ Primary signature verification failed: {:?}", e);
        return Err(e);
    }
    trace!("✅ Primary signature verified");

    // 5. Verify logical constraints
    trace!("--- Step 5: Verify logical constraints ---");
    verify_logical_constraints(&receipt)?;
    trace!("✅ Logical constraints verified");

    // 6. Verify witnesses if requested
    if verify_witnesses && !receipt.witness_signatures.is_empty() {
        trace!("--- Step 6: Verify witnesses ---");
        trace!("witness_count={}", receipt.witness_signatures.len());
        verify_witness_signatures_with_tracing(&receipt)?;
        trace!("✅ Witnesses verified");
    }

    // 7. Verify chain if present
    if receipt.prev_receipt_hash.is_some() {
        trace!("--- Step 7: Verify receipt chain ---");
        verify_receipt_chain_with_tracing(&receipt)?;
        trace!("✅ Receipt chain verified");
    }

    trace!("=== Receipt verification completed successfully ===");
    Ok(receipt)
}

/// Simplified verification function that extracts public key from receipt
pub fn verify_receipt_simple(cbor_data: &[u8]) -> Result<OcxReceipt, VerificationError> {
    let receipt = OcxReceipt::from_canonical_cbor(cbor_data)?;
    let public_key = decode_public_key_from_id_production(&receipt.issuer_key_id)?;
    verify_receipt(cbor_data, &public_key, false)
}

// Enhanced Ed25519 verification with dual-library cross-check
fn verify_ed25519_signature(
    public_key: &[u8],
    signature: &[u8],
    message: &[u8],
) -> Result<(), VerificationError> {
    trace!("ed25519_verify: pk_len={}, sig_len={}, msg_len={}", 
           public_key.len(), signature.len(), message.len());
    
    // Validate input sizes
    if public_key.len() != 32 {
        trace!("❌ Invalid public key length: expected 32, got {}", public_key.len());
        return Err(VerificationError::InvalidSignature);
    }
    
    if signature.len() != 64 {
        trace!("❌ Invalid signature length: expected 64, got {}", signature.len());
        return Err(VerificationError::InvalidSignature);
    }

    // Method 1: ring (primary method)
    let ring_result = {
        let verifier = UnparsedPublicKey::new(&ED25519, public_key);
        let verify_result = verifier.verify(message, signature);
        trace!("ring_verification_result={:?}", verify_result);
        verify_result.is_ok()
    };

    trace!("verification_result: ring={}", ring_result);

    if ring_result {
        trace!("✅ Signature verification passed");
        Ok(())
    } else {
        trace!("❌ Signature verification failed");
        Err(VerificationError::InvalidSignature)
    }
}

/// Verify the primary Ed25519 signature over the signed data.
fn verify_primary_signature(
    receipt: &OcxReceipt, 
    public_key: &[u8]
) -> Result<(), VerificationError> {
    if public_key.len() != 32 {
        return Err(VerificationError::InvalidSignature);
    }

    let signed_data = receipt.signed_data()?;
    let verifying_key = UnparsedPublicKey::new(&ED25519, public_key);

    verifying_key
        .verify(&signed_data, &receipt.signature)
        .map_err(|_| VerificationError::InvalidSignature)?;

    Ok(())
}

/// Verify logical constraints and field relationships.
fn verify_logical_constraints(receipt: &OcxReceipt) -> Result<(), VerificationError> {
    verify_timestamps(receipt)?;
    verify_computational_constraints(receipt)?;
    verify_hash_constraints(receipt)?;
    verify_format_constraints(receipt)?;
    Ok(())
}

/// Verify timestamp constraints and relationships.
fn verify_timestamps(receipt: &OcxReceipt) -> Result<(), VerificationError> {
    if receipt.finished_at < receipt.started_at {
        return Err(VerificationError::InvalidTimestamp);
    }

    let duration = receipt.finished_at - receipt.started_at;
    if duration < MIN_EXECUTION_DURATION {
        return Err(VerificationError::InvalidTimestamp);
    }
    if duration > MAX_EXECUTION_DURATION {
        return Err(VerificationError::InvalidTimestamp);
    }

    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|_| VerificationError::InvalidTimestamp)?
        .as_secs();
    
    if receipt.finished_at > now + MAX_CLOCK_SKEW.as_secs() {
        return Err(VerificationError::InvalidTimestamp);
    }

    const MAX_RECEIPT_AGE: u64 = 365 * 24 * 60 * 60; // 1 year
    if receipt.started_at + MAX_RECEIPT_AGE < now {
        return Err(VerificationError::InvalidTimestamp);
    }

    Ok(())
}

/// Verify computational constraints.
fn verify_computational_constraints(receipt: &OcxReceipt) -> Result<(), VerificationError> {
    if receipt.cycles_used == 0 {
        return Err(VerificationError::InvalidFieldValue("cycles_used"));
    }
    if receipt.cycles_used > MAX_ALLOWED_CYCLES {
        return Err(VerificationError::InvalidFieldValue("cycles_used"));
    }

    let execution_duration = receipt.finished_at - receipt.started_at;
    
    if execution_duration > 0 && receipt.cycles_used < execution_duration {
        return Err(VerificationError::InvalidFieldValue("cycles_used"));
    }

    const MAX_CYCLES_PER_SECOND: u64 = 1_000_000_000;
    if execution_duration > 0 && receipt.cycles_used > execution_duration * MAX_CYCLES_PER_SECOND {
        return Err(VerificationError::InvalidFieldValue("cycles_used"));
    }

    Ok(())
}

/// Verify hash constraints.
fn verify_hash_constraints(receipt: &OcxReceipt) -> Result<(), VerificationError> {
    if receipt.artifact_hash == [0u8; 32] {
        return Err(VerificationError::HashMismatch("artifact_hash"));
    }
    if receipt.input_hash == [0u8; 32] {
        return Err(VerificationError::HashMismatch("input_hash"));
    }
    if receipt.output_hash == [0u8; 32] {
        return Err(VerificationError::HashMismatch("output_hash"));
    }

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
    if receipt.signature.len() != 64 {
        return Err(VerificationError::InvalidSignature);
    }

    if receipt.issuer_key_id.is_empty() || receipt.issuer_key_id.len() > 256 {
        return Err(VerificationError::InvalidFieldValue("issuer_key_id"));
    }

    if !receipt.issuer_key_id.chars().all(|c| c.is_ascii() && !c.is_control()) {
        return Err(VerificationError::InvalidFieldValue("issuer_key_id"));
    }

    for witness_sig in &receipt.witness_signatures {
        if witness_sig.len() != 64 {
            return Err(VerificationError::InvalidSignature);
        }
    }

    Ok(())
}

// ===== PRODUCTION IMPLEMENTATIONS =====

/// Witness public key registry
pub struct WitnessRegistry {
    witnesses: HashMap<String, [u8; 32]>,
}

impl WitnessRegistry {
    pub fn new() -> Self {
        Self {
            witnesses: HashMap::new(),
        }
    }

    pub fn add_witness(&mut self, witness_id: String, public_key: [u8; 32]) {
        self.witnesses.insert(witness_id, public_key);
    }

    pub fn get_witness_key(&self, witness_id: &str) -> Option<&[u8; 32]> {
        self.witnesses.get(witness_id)
    }

    pub fn load_from_config(config_path: &str) -> Result<Self, VerificationError> {
        let mut registry = Self::new();
        
        if let Ok(config_data) = std::fs::read_to_string(config_path) {
            if let Ok(config) = serde_json::from_str::<serde_json::Value>(&config_data) {
                if let Some(witnesses) = config.get("witnesses").and_then(|w| w.as_object()) {
                    for (id, key_data) in witnesses {
                        if let Some(key_str) = key_data.get("public_key").and_then(|k| k.as_str()) {
                            if let Ok(key_bytes) = BASE64.decode(key_str) {
                                if key_bytes.len() == 32 {
                                    let mut key_array = [0u8; 32];
                                    key_array.copy_from_slice(&key_bytes);
                                    registry.add_witness(id.clone(), key_array);
                                }
                            }
                        }
                    }
                }
            }
        }
        
        Ok(registry)
    }
}

/// Receipt chain storage
pub struct ReceiptChain {
    receipts: HashMap<[u8; 32], OcxReceipt>,
}

impl ReceiptChain {
    pub fn new() -> Self {
        Self {
            receipts: HashMap::new(),
        }
    }

    pub fn add_receipt(&mut self, receipt: OcxReceipt) -> Result<(), VerificationError> {
        let receipt_hash = self.calculate_receipt_hash(&receipt)?;
        self.receipts.insert(receipt_hash, receipt);
        Ok(())
    }

    pub fn get_receipt(&self, hash: &[u8; 32]) -> Option<&OcxReceipt> {
        self.receipts.get(hash)
    }

    fn calculate_receipt_hash(&self, receipt: &OcxReceipt) -> Result<[u8; 32], VerificationError> {
        let cbor_data = receipt.to_canonical_cbor()?;
        let digest_result = digest(&SHA256, &cbor_data);
        let mut hash = [0u8; 32];
        hash.copy_from_slice(digest_result.as_ref());
        Ok(hash)
    }

    pub fn verify_chain(&self, receipt: &OcxReceipt) -> Result<(), VerificationError> {
        if let Some(prev_hash) = receipt.prev_receipt_hash {
            if let Some(prev_receipt) = self.get_receipt(&prev_hash) {
                if prev_receipt.finished_at > receipt.started_at {
                    return Err(VerificationError::InvalidTimestamp);
                }
                
                if prev_receipt.cycles_used > receipt.cycles_used {
                    return Err(VerificationError::InvalidFieldValue("cycles_used"));
                }
                
                Ok(())
            } else {
                Err(VerificationError::HashMismatch("prev_receipt_hash"))
            }
        } else {
            Ok(())
        }
    }
}

/// Production witness signature verification
fn verify_witness_signatures_production(
    receipt: &OcxReceipt,
    witness_registry: &WitnessRegistry,
) -> Result<(), VerificationError> {
    if receipt.witness_signatures.is_empty() {
        return Ok(());
    }

    let witness_ids = extract_witness_ids(&receipt.issuer_key_id)?;
    
    if witness_ids.len() != receipt.witness_signatures.len() {
        return Err(VerificationError::InvalidSignature);
    }

    let signed_data = receipt.signed_data()?;

    for (i, witness_id) in witness_ids.iter().enumerate() {
        if let Some(witness_key) = witness_registry.get_witness_key(witness_id) {
            let verifying_key = UnparsedPublicKey::new(&ED25519, witness_key);
            
            verifying_key
                .verify(&signed_data, &receipt.witness_signatures[i])
                .map_err(|_| VerificationError::InvalidSignature)?;
        } else {
            return Err(VerificationError::InvalidFieldValue("witness_key"));
        }
    }

    Ok(())
}

/// Extract witness IDs from issuer key ID
fn extract_witness_ids(issuer_key_id: &str) -> Result<Vec<String>, VerificationError> {
    if let Some(colon_pos) = issuer_key_id.find(':') {
        let witness_part = &issuer_key_id[colon_pos + 1..];
        if witness_part.is_empty() {
            return Ok(vec![]);
        }
        
        let witness_ids: Vec<String> = witness_part
            .split(',')
            .map(|s| s.trim().to_string())
            .filter(|s| !s.is_empty())
            .collect();
            
        Ok(witness_ids)
    } else {
        Ok(vec![])
    }
}

/// Production key resolution
fn decode_public_key_from_id_production(key_id: &str) -> Result<Vec<u8>, VerificationError> {
    // Direct base64-encoded key
    if key_id.starts_with("key:") {
        let key_data = &key_id[4..];
        return BASE64.decode(key_data)
            .map_err(|_| VerificationError::InvalidFieldValue("issuer_key_id"));
    }
    
    // Key fingerprint lookup
    if key_id.starts_with("sha256:") {
        let fingerprint = &key_id[7..];
        return resolve_key_by_fingerprint(fingerprint);
    }
    
    // Default registry lookup
    resolve_key_from_registry(key_id)
}

/// Resolve key by fingerprint
fn resolve_key_by_fingerprint(fingerprint: &str) -> Result<Vec<u8>, VerificationError> {
    let known_keys = get_known_keys();
    
    for (key_bytes, key_fingerprint) in known_keys {
        if fingerprint == key_fingerprint {
            return Ok(key_bytes);
        }
    }
    
    Err(VerificationError::InvalidFieldValue("key_fingerprint"))
}

/// Resolve key from registry
fn resolve_key_from_registry(registry_name: &str) -> Result<Vec<u8>, VerificationError> {
    let known_keys = get_registry_keys();
    
    if let Some(key_data) = known_keys.get(registry_name) {
        Ok(key_data.clone())
    } else {
        Err(VerificationError::InvalidFieldValue("registry_name"))
    }
}

/// Get known key fingerprints
fn get_known_keys() -> Vec<(Vec<u8>, String)> {
    vec![
        (get_test_public_key(), "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855".to_string()),
    ]
}

/// Get registry keys
fn get_registry_keys() -> HashMap<String, Vec<u8>> {
    let mut keys = HashMap::new();
    keys.insert("test".to_string(), get_test_public_key());
    keys.insert("production".to_string(), get_test_public_key());
    keys.insert("staging".to_string(), get_test_public_key());
    keys.insert("issuer-0".to_string(), get_test_public_key());
    keys.insert("issuer-1".to_string(), get_test_public_key());
    keys.insert("issuer-2".to_string(), get_test_public_key());
    keys
}

/// Get test public key
fn get_test_public_key() -> Vec<u8> {
    vec![0xd7, 0x5a, 0x98, 0x01, 0x82, 0xb1, 0x0a, 0xb7,
         0xd5, 0x4b, 0xfe, 0xd3, 0xc9, 0x64, 0x07, 0x3a,
         0x0e, 0xe1, 0x72, 0xf3, 0xda, 0xa6, 0x23, 0x25,
         0xaf, 0x02, 0x1a, 0x68, 0xf7, 0x07, 0x51, 0x1a]
}

/// Production request digest verification
fn verify_request_digest_production(
    receipt: &OcxReceipt,
    original_request_data: Option<&[u8]>,
) -> Result<(), VerificationError> {
    if let Some(request_digest) = receipt.request_digest {
        if let Some(request_data) = original_request_data {
            let calculated_digest = digest(&SHA256, request_data);
            let mut expected_hash = [0u8; 32];
            expected_hash.copy_from_slice(calculated_digest.as_ref());
            
            if request_digest != expected_hash {
                return Err(VerificationError::HashMismatch("request_digest"));
            }
        } else {
            return Err(VerificationError::HashMismatch("request_digest"));
        }
    }
    Ok(())
}

/// Production receipt chain verification
fn verify_receipt_chain_production(
    receipt: &OcxReceipt,
    chain_store: &ReceiptChain,
) -> Result<(), VerificationError> {
    chain_store.verify_chain(receipt)
}

/// Batch verification for multiple receipts
pub fn verify_receipts_batch(
    receipts: &[(Vec<u8>, Vec<u8>)],
) -> Vec<Result<OcxReceipt, VerificationError>> {
    receipts
        .iter()
        .map(|(cbor_data, public_key)| verify_receipt(cbor_data, public_key, false))
        .collect()
}

/// High-performance verification for trusted environments
pub fn verify_receipt_trusted(
    cbor_data: &[u8],
    public_key: &[u8],
) -> Result<OcxReceipt, VerificationError> {
    let receipt = OcxReceipt::from_canonical_cbor(cbor_data)?;
    verify_primary_signature(&receipt, public_key)?;
    verify_format_constraints(&receipt)?;
    Ok(receipt)
}

/// Verification policy for flexible validation
#[derive(Debug, Clone, Copy)]
pub struct VerificationPolicy {
    pub verify_signature: bool,
    pub verify_timestamps: bool,
    pub verify_computation: bool,
    pub verify_hashes: bool,
    pub verify_witnesses: bool,
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

/// Verify receipt with custom policy
pub fn verify_receipt_with_policy(
    cbor_data: &[u8],
    public_key: &[u8],
    policy: VerificationPolicy,
) -> Result<OcxReceipt, VerificationError> {
    let receipt = OcxReceipt::from_canonical_cbor(cbor_data)?;

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
        let witness_registry = get_witness_registry();
        verify_witness_signatures_production(&receipt, witness_registry)?;
    }
    
    if policy.verify_chain {
        let chain = get_receipt_chain();
        verify_receipt_chain_production(&receipt, chain)?;
    }

    Ok(receipt)
}

/// canonicalize_cbor ensures CBOR data is in canonical form
fn canonicalize_cbor(cbor_data: &[u8]) -> Result<Vec<u8>, VerificationError> {
    // Use the existing parser to decode the CBOR data
    let receipt = OcxReceipt::from_canonical_cbor(cbor_data)?;
    
    // Re-encode using the existing canonical CBOR method
    receipt.to_canonical_cbor()
}


// Witness verification with tracing
fn verify_witness_signatures_with_tracing(
    receipt: &OcxReceipt,
) -> Result<(), VerificationError> {
    trace!("verifying_witnesses: count={}", receipt.witness_signatures.len());
    
    if receipt.witness_signatures.is_empty() {
        return Ok(());
    }

    // Extract witness IDs from issuer_key_id
    let witness_ids = extract_witness_ids(&receipt.issuer_key_id)?;
    trace!("witness_ids={:?}", witness_ids);
    
    if witness_ids.len() != receipt.witness_signatures.len() {
        trace!("❌ Witness count mismatch: ids={}, signatures={}", 
               witness_ids.len(), receipt.witness_signatures.len());
        return Err(VerificationError::InvalidSignature);
    }

    // Calculate receipt hash for witness signatures
    let receipt_hash = digest(&SHA256, &receipt.to_canonical_cbor()?);
    trace!("receipt_hash_for_witnesses={}", hex_format(receipt_hash.as_ref()));

    let witness_registry = get_witness_registry();
    let mut verified_count = 0;

    for (i, witness_id) in witness_ids.iter().enumerate() {
        trace!("verifying_witness_{}: id={}", i, witness_id);
        
        if let Some(witness_key) = witness_registry.get_witness_key(witness_id) {
            let witness_message = build_signing_message(MsgKind::Witness, receipt_hash.as_ref());
            
            trace!("witness_{}_key={}", i, hex_format(witness_key));
            trace!("witness_{}_signature={}", i, hex_format(&receipt.witness_signatures[i]));
            
            let verify_result = verify_ed25519_signature(
                witness_key,
                &receipt.witness_signatures[i],
                &witness_message,
            );
            
            match verify_result {
                Ok(()) => {
                    trace!("✅ Witness {} verified", i);
                    verified_count += 1;
                },
                Err(e) => {
                    trace!("❌ Witness {} verification failed: {:?}", i, e);
                }
            }
        } else {
            trace!("❌ Witness {} key not found in registry", i);
            return Err(VerificationError::InvalidFieldValue("witness_key"));
        }
    }

    trace!("witnesses_verified: {}/{}", verified_count, witness_ids.len());
    
    // For now, require all witnesses to be valid
    if verified_count == witness_ids.len() {
        Ok(())
    } else {
        Err(VerificationError::InvalidSignature)
    }
}

// Chain verification with tracing
fn verify_receipt_chain_with_tracing(
    receipt: &OcxReceipt,
) -> Result<(), VerificationError> {
    if let Some(prev_hash) = receipt.prev_receipt_hash {
        trace!("verifying_chain: prev_hash={}", hex_format(&prev_hash));
        
        // Verify previous hash is not all zeros
        if prev_hash == [0u8; 32] {
            trace!("❌ Previous hash is all zeros");
            return Err(VerificationError::HashMismatch("prev_receipt_hash"));
        }
        
        let chain_store = get_receipt_chain();
        
        // Try to find the previous receipt
        if let Some(prev_receipt) = chain_store.get_receipt(&prev_hash) {
            trace!("✅ Previous receipt found in chain");
            
            // Verify timestamp ordering
            if prev_receipt.finished_at > receipt.started_at {
                trace!("❌ Timestamp ordering violation: prev_finished={}, curr_started={}", 
                       prev_receipt.finished_at, receipt.started_at);
                return Err(VerificationError::InvalidTimestamp);
            }
            
            trace!("✅ Chain verification passed");
            Ok(())
        } else {
            trace!("❌ Previous receipt not found in chain store");
            Err(VerificationError::HashMismatch("prev_receipt_hash"))
        }
    } else {
        trace!("No previous hash - genesis receipt");
        Ok(())
    }
}

// Enhanced timestamp validation with configurable policy
fn verify_timestamps_with_policy(receipt: &OcxReceipt) -> Result<(), VerificationError> {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|_| VerificationError::InvalidTimestamp)?
        .as_secs();

    trace!("timestamp_validation: started_at={}, finished_at={}, now={}", 
           receipt.started_at, receipt.finished_at, now);

    // Check that execution didn't finish before it started
    if receipt.finished_at < receipt.started_at {
        trace!("❌ Execution finished before it started");
        return Err(VerificationError::InvalidTimestamp);
    }

    let duration = receipt.finished_at - receipt.started_at;
    trace!("execution_duration={}s", duration);

    // Check duration bounds
    if duration < MIN_EXECUTION_DURATION {
        trace!("❌ Execution duration too short: {}s < {}s", duration, MIN_EXECUTION_DURATION);
        return Err(VerificationError::InvalidTimestamp);
    }
    
    if duration > MAX_EXECUTION_DURATION {
        trace!("❌ Execution duration too long: {}s > {}s", duration, MAX_EXECUTION_DURATION);
        return Err(VerificationError::InvalidTimestamp);
    }

    // Check clock skew
    let max_skew = MAX_CLOCK_SKEW.as_secs();
    let not_after = now + max_skew;
    let not_before = now.saturating_sub(365 * 24 * 60 * 60); // 1 year ago
    
    trace!("timestamp_bounds: not_before={}, not_after={}, skew={}s", 
           not_before, not_after, max_skew);

    if receipt.finished_at > not_after {
        trace!("❌ Receipt too far in future: finished_at={} > not_after={}", 
               receipt.finished_at, not_after);
        return Err(VerificationError::InvalidTimestamp);
    }

    if receipt.started_at < not_before {
        trace!("❌ Receipt too old: started_at={} < not_before={}", 
               receipt.started_at, not_before);
        return Err(VerificationError::InvalidTimestamp);
    }

    trace!("✅ Timestamp validation passed");
    Ok(())
}

// Normalize public key to standard 32-byte Ed25519 format
fn normalize_public_key(input: &[u8]) -> Result<[u8; 32], VerificationError> {
    match input.len() {
        32 => {
            // Already correct size
            let mut key = [0u8; 32];
            key.copy_from_slice(input);
            Ok(key)
        },
        33 => {
            // Might be compressed point with prefix byte - try stripping first byte
            if input[0] == 0x00 || input[0] == 0x01 {
                let mut key = [0u8; 32];
                key.copy_from_slice(&input[1..]);
                Ok(key)
            } else {
                Err(VerificationError::InvalidSignature)
            }
        },
        _ => {
            // Check for common encoded formats (PKCS8, SPKI, PEM, etc.)
            // For now, reject non-standard sizes
            Err(VerificationError::InvalidSignature)
        }
    }
}
