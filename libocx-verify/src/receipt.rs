//! The OCX Receipt data structure.

use crate::canonical_cbor::{CanonicalValue, CborParser};
use crate::VerificationError;
use std::collections::BTreeMap;
use std::time::{SystemTime, UNIX_EPOCH};
use serde_cbor;

/// A verified OCX Receipt containing all cryptographically protected fields.
#[derive(Debug, Clone, PartialEq, serde::Serialize, serde::Deserialize)]
pub struct OcxReceipt {
    /// Hash of the execution artifact (32 bytes).
    pub artifact_hash: [u8; 32],
    /// Hash of the input data (32 bytes).
    pub input_hash: [u8; 32],
    /// Hash of the output data (32 bytes).
    pub output_hash: [u8; 32],
    /// Computational cycles used during execution.
    pub cycles_used: u64,
    /// Unix timestamp when execution started (seconds since epoch).
    pub started_at: u64,
    /// Unix timestamp when execution finished (seconds since epoch).
    pub finished_at: u64,
    /// Public key identifier of the issuer.
    pub issuer_key_id: String,
    /// Ed25519 signature over the canonical CBOR of all fields except signature.
    pub signature: Vec<u8>,
    /// Optional: Hash of the previous receipt for chaining (v1.1 feature).
    pub prev_receipt_hash: Option<[u8; 32]>,
    /// Optional: Hash of the original request for binding (v1.1 feature).
    pub request_digest: Option<[u8; 32]>,
    /// Optional: Additional witness signatures for multi-party verification.
    pub witness_signatures: Vec<Vec<u8>>,
    /// Optional: VDF output y = x^(2^T) mod N (temporal proof, v1.2).
    pub vdf_output: Option<Vec<u8>>,
    /// Optional: Wesolowski proof π (temporal proof, v1.2).
    pub vdf_proof: Option<Vec<u8>>,
    /// Optional: VDF iterations T (temporal proof, v1.2).
    pub vdf_iterations: Option<u64>,
    /// Optional: VDF modulus identifier (temporal proof, v1.2).
    pub vdf_modulus_id: Option<String>,
}

/// Helper struct for unsigned receipt serialization
#[derive(Debug, Clone)]
pub struct UnsignedReceipt {
    pub artifact_hash: [u8; 32],
    pub input_hash: [u8; 32],
    pub output_hash: [u8; 32],
    pub cycles_used: u64,
    pub started_at: u64,
    pub finished_at: u64,
    pub issuer_key_id: String,
    pub prev_receipt_hash: Option<[u8; 32]>,
    pub request_digest: Option<[u8; 32]>,
    pub witness_signatures: Vec<Vec<u8>>,
    /// Optional: VDF output (v1.2 temporal proof).
    pub vdf_output: Option<Vec<u8>>,
    /// Optional: Wesolowski proof (v1.2 temporal proof).
    pub vdf_proof: Option<Vec<u8>>,
    /// Optional: VDF iterations (v1.2 temporal proof).
    pub vdf_iterations: Option<u64>,
    /// Optional: VDF modulus identifier (v1.2 temporal proof).
    pub vdf_modulus_id: Option<String>,
}

impl OcxReceipt {
    /// Attempts to construct an `OcxReceipt` from canonical CBOR bytes.
    ///
    /// This function enforces the OCX-CBOR v1.1 specification:
    /// - Map with integer keys for compactness
    /// - Canonical field ordering
    /// - Strict type validation
    /// - Required vs optional field handling
    pub fn from_canonical_cbor(cbor_data: &[u8]) -> Result<Self, VerificationError> {
        let canonical_value = CborParser::new(cbor_data).parse_full()?;

        let map = match canonical_value {
            CanonicalValue::Map(m) => {
                println!("DEBUG: Parsed map with {} fields", m.len());
                for (key, value) in &m {
                    println!("DEBUG: Key: {:?}, Value type: {:?}", key, std::mem::discriminant(value));
                }
                m
            },
            _ => return Err(VerificationError::InvalidCbor),
        };

        // Extract required fields using integer keys (OCX-CBOR v1.1 spec)
        let artifact_hash = Self::extract_hash(&map, 1, "program_hash")?;
        let input_hash = Self::extract_hash(&map, 2, "input_hash")?;
        let output_hash = Self::extract_hash(&map, 3, "output_hash")?;
        let cycles_used = Self::extract_uint64(&map, 4, "cycles")?;
        let started_at = Self::extract_uint64(&map, 5, "started_at")?;
        let finished_at = Self::extract_uint64(&map, 6, "finished_at")?;
        let issuer_key_id = Self::extract_string(&map, 7, "issuer_id")?;
        let signature = Self::extract_bytes(&map, 8, "signature")?;

        // Extract optional fields (v1.1 extensions)
        let prev_receipt_hash = Self::extract_optional_hash(&map, 9)?;
        let request_digest = Self::extract_optional_hash(&map, 10)?;
        let witness_signatures = Self::extract_optional_signatures(&map, 11)?;

        // Extract optional VDF fields (v1.2 temporal proof)
        let vdf_output = Self::extract_optional_bytes(&map, 12)?;
        let vdf_proof = Self::extract_optional_bytes(&map, 13)?;
        let vdf_iterations = Self::extract_optional_uint64(&map, 14)?;
        let vdf_modulus_id = Self::extract_optional_string(&map, 15)?;

        // Validate field constraints
        Self::validate_timestamps(started_at, finished_at)?;
        Self::validate_cycles(cycles_used)?;
        Self::validate_signature(&signature)?;
        Self::validate_key_id(&issuer_key_id)?;

        Ok(OcxReceipt {
            artifact_hash,
            input_hash,
            output_hash,
            cycles_used,
            started_at,
            finished_at,
            issuer_key_id,
            signature,
            prev_receipt_hash,
            request_digest,
            witness_signatures,
            vdf_output,
            vdf_proof,
            vdf_iterations,
            vdf_modulus_id,
        })
    }

    /// Returns the serialized CBOR data that is covered by the signature.
    ///
    /// This reconstructs the canonical CBOR map containing all fields except
    /// the signature itself. The output must be byte-for-byte identical to
    /// what was originally signed.
    /// Generate the signed data for this receipt (canonical CBOR without signature)
    pub fn signed_data(&self) -> Result<Vec<u8>, VerificationError> {
        let mut map = BTreeMap::new();

        // Add all signed fields in canonical order (integer keys)
        map.insert(
            CanonicalValue::Integer(1),
            CanonicalValue::Bytes(self.artifact_hash.to_vec()),
        );
        map.insert(
            CanonicalValue::Integer(2),
            CanonicalValue::Bytes(self.input_hash.to_vec()),
        );
        map.insert(
            CanonicalValue::Integer(3),
            CanonicalValue::Bytes(self.output_hash.to_vec()),
        );
        map.insert(
            CanonicalValue::Integer(4),
            CanonicalValue::Integer(self.cycles_used),
        );
        map.insert(
            CanonicalValue::Integer(5),
            CanonicalValue::Integer(self.started_at),
        );
        map.insert(
            CanonicalValue::Integer(6),
            CanonicalValue::Integer(self.finished_at),
        );
        map.insert(
            CanonicalValue::Integer(7),
            CanonicalValue::Text(self.issuer_key_id.clone()),
        );

        // Add optional fields if present
        if let Some(prev_hash) = self.prev_receipt_hash {
            map.insert(
                CanonicalValue::Integer(9),
                CanonicalValue::Bytes(prev_hash.to_vec()),
            );
        }
        if let Some(request_digest) = self.request_digest {
            map.insert(
                CanonicalValue::Integer(10),
                CanonicalValue::Bytes(request_digest.to_vec()),
            );
        }
        if !self.witness_signatures.is_empty() {
            let witness_array: Vec<CanonicalValue> = self
                .witness_signatures
                .iter()
                .map(|sig| CanonicalValue::Bytes(sig.clone()))
                .collect();
            map.insert(
                CanonicalValue::Integer(11),
                CanonicalValue::Array(witness_array),
            );
        }

        // Add VDF fields if present (v1.2 temporal proof — signature covers these)
        if let Some(ref vdf_output) = self.vdf_output {
            map.insert(
                CanonicalValue::Integer(12),
                CanonicalValue::Bytes(vdf_output.clone()),
            );
        }
        if let Some(ref vdf_proof) = self.vdf_proof {
            map.insert(
                CanonicalValue::Integer(13),
                CanonicalValue::Bytes(vdf_proof.clone()),
            );
        }
        if let Some(vdf_iterations) = self.vdf_iterations {
            map.insert(
                CanonicalValue::Integer(14),
                CanonicalValue::Integer(vdf_iterations),
            );
        }
        if let Some(ref vdf_modulus_id) = self.vdf_modulus_id {
            map.insert(
                CanonicalValue::Integer(15),
                CanonicalValue::Text(vdf_modulus_id.clone()),
            );
        }

        // Serialize to canonical CBOR
        let cbor_data = Self::serialize_canonical_map(&map)?;

        // Create the complete signing message with domain separator
        let domain_separator = b"OCXv1|receipt|";
        let mut message = Vec::new();
        message.extend_from_slice(domain_separator);
        message.extend_from_slice(&cbor_data);
        
        Ok(message)
    }

    /// Extract a required 32-byte hash from the map.
    fn extract_hash(
        map: &BTreeMap<CanonicalValue, CanonicalValue>,
        key: u64,
        field_name: &'static str,
    ) -> Result<[u8; 32], VerificationError> {
        let bytes = Self::extract_bytes(map, key, field_name)?;
        if bytes.len() != 32 {
            return Err(VerificationError::InvalidFieldValue(field_name));
        }
        let mut hash = [0u8; 32];
        hash.copy_from_slice(&bytes);
        Ok(hash)
    }

    /// Extract an optional 32-byte hash from the map.
    fn extract_optional_hash(
        map: &BTreeMap<CanonicalValue, CanonicalValue>,
        key: u64,
    ) -> Result<Option<[u8; 32]>, VerificationError> {
        match map.get(&CanonicalValue::Integer(key)) {
            Some(CanonicalValue::Bytes(bytes)) => {
                if bytes.len() != 32 {
                    return Err(VerificationError::InvalidFieldValue("optional_hash"));
                }
                let mut hash = [0u8; 32];
                hash.copy_from_slice(bytes);
                Ok(Some(hash))
            }
            None => Ok(None),
            _ => Err(VerificationError::InvalidFieldValue("optional_hash")),
        }
    }

    /// Extract required bytes from the map.
    fn extract_bytes(
        map: &BTreeMap<CanonicalValue, CanonicalValue>,
        key: u64,
        field_name: &'static str,
    ) -> Result<Vec<u8>, VerificationError> {
        match map.get(&CanonicalValue::Integer(key)) {
            Some(CanonicalValue::Bytes(bytes)) => Ok(bytes.clone()),
            None => Err(VerificationError::MissingField(field_name)),
            _ => Err(VerificationError::InvalidFieldValue(field_name)),
        }
    }

    /// Extract required uint64 from the map.
    fn extract_uint64(
        map: &BTreeMap<CanonicalValue, CanonicalValue>,
        key: u64,
        field_name: &'static str,
    ) -> Result<u64, VerificationError> {
        match map.get(&CanonicalValue::Integer(key)) {
            Some(CanonicalValue::Integer(value)) => Ok(*value),
            None => Err(VerificationError::MissingField(field_name)),
            _ => Err(VerificationError::InvalidFieldValue(field_name)),
        }
    }

    /// Extract required string from the map.
    fn extract_string(
        map: &BTreeMap<CanonicalValue, CanonicalValue>,
        key: u64,
        field_name: &'static str,
    ) -> Result<String, VerificationError> {
        match map.get(&CanonicalValue::Integer(key)) {
            Some(CanonicalValue::Text(text)) => Ok(text.clone()),
            None => Err(VerificationError::MissingField(field_name)),
            _ => Err(VerificationError::InvalidFieldValue(field_name)),
        }
    }

    /// Extract optional bytes from the map.
    fn extract_optional_bytes(
        map: &BTreeMap<CanonicalValue, CanonicalValue>,
        key: u64,
    ) -> Result<Option<Vec<u8>>, VerificationError> {
        match map.get(&CanonicalValue::Integer(key)) {
            Some(CanonicalValue::Bytes(bytes)) => Ok(Some(bytes.clone())),
            None => Ok(None),
            _ => Err(VerificationError::InvalidFieldValue("optional_bytes")),
        }
    }

    /// Extract optional uint64 from the map.
    fn extract_optional_uint64(
        map: &BTreeMap<CanonicalValue, CanonicalValue>,
        key: u64,
    ) -> Result<Option<u64>, VerificationError> {
        match map.get(&CanonicalValue::Integer(key)) {
            Some(CanonicalValue::Integer(value)) => Ok(Some(*value)),
            None => Ok(None),
            _ => Err(VerificationError::InvalidFieldValue("optional_uint64")),
        }
    }

    /// Extract optional string from the map.
    fn extract_optional_string(
        map: &BTreeMap<CanonicalValue, CanonicalValue>,
        key: u64,
    ) -> Result<Option<String>, VerificationError> {
        match map.get(&CanonicalValue::Integer(key)) {
            Some(CanonicalValue::Text(text)) => Ok(Some(text.clone())),
            None => Ok(None),
            _ => Err(VerificationError::InvalidFieldValue("optional_string")),
        }
    }

    /// Extract optional witness signatures array.
    fn extract_optional_signatures(
        map: &BTreeMap<CanonicalValue, CanonicalValue>,
        key: u64,
    ) -> Result<Vec<Vec<u8>>, VerificationError> {
        match map.get(&CanonicalValue::Integer(key)) {
            Some(CanonicalValue::Array(array)) => {
                let mut signatures = Vec::new();
                for item in array {
                    match item {
                        CanonicalValue::Bytes(sig) => signatures.push(sig.clone()),
                        _ => return Err(VerificationError::InvalidFieldValue("witness_signatures")),
                    }
                }
                Ok(signatures)
            }
            None => Ok(Vec::new()),
            _ => Err(VerificationError::InvalidFieldValue("witness_signatures")),
        }
    }

    /// Validate timestamp constraints.
    fn validate_timestamps(started_at: u64, finished_at: u64) -> Result<(), VerificationError> {
        // Check that execution didn't finish before it started
        if finished_at < started_at {
            return Err(VerificationError::InvalidTimestamp);
        }

        // Check for reasonable execution duration (max 24 hours)
        const MAX_EXECUTION_DURATION: u64 = 24 * 60 * 60; // 24 hours in seconds
        if finished_at - started_at > MAX_EXECUTION_DURATION {
            return Err(VerificationError::InvalidTimestamp);
        }

        // Check that timestamps are not too far in the future (max 5 minutes clock skew)
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map_err(|_| VerificationError::InvalidTimestamp)?
            .as_secs();
        
        const MAX_CLOCK_SKEW: u64 = 5 * 60; // 5 minutes
        if finished_at > now + MAX_CLOCK_SKEW {
            return Err(VerificationError::InvalidTimestamp);
        }

        Ok(())
    }

    /// Validate computational cycles constraints.
    fn validate_cycles(cycles: u64) -> Result<(), VerificationError> {
        // Minimum cycles (must do some work)
        if cycles == 0 {
            return Err(VerificationError::InvalidFieldValue("cycles_used"));
        }

        // Maximum cycles (prevent overflow attacks)
        const MAX_CYCLES: u64 = 1_000_000_000; // 1 billion cycles max
        if cycles > MAX_CYCLES {
            return Err(VerificationError::InvalidFieldValue("cycles_used"));
        }

        Ok(())
    }

    /// Validate signature format.
    fn validate_signature(signature: &[u8]) -> Result<(), VerificationError> {
        // Ed25519 signatures are exactly 64 bytes
        if signature.len() != 64 {
            return Err(VerificationError::InvalidSignature);
        }
        Ok(())
    }

    /// Validate issuer key ID format.
    fn validate_key_id(key_id: &str) -> Result<(), VerificationError> {
        // Key ID must be non-empty and reasonable length
        if key_id.is_empty() || key_id.len() > 256 {
            return Err(VerificationError::InvalidFieldValue("issuer_key_id"));
        }

        // Key ID must be valid UTF-8 (already ensured by CBOR parser)
        // and contain only printable ASCII characters for safety
        if !key_id.chars().all(|c| c.is_ascii() && !c.is_control()) {
            return Err(VerificationError::InvalidFieldValue("issuer_key_id"));
        }

        Ok(())
    }

    /// Serialize receipt to canonical CBOR
    pub fn to_canonical_cbor(&self) -> Result<Vec<u8>, VerificationError> {
        let mut map = BTreeMap::new();

        // Add required fields with integer keys (OCX-CBOR v1.1 spec)
        map.insert(CanonicalValue::Integer(1), CanonicalValue::Bytes(self.artifact_hash.to_vec())); // program_hash
        map.insert(CanonicalValue::Integer(2), CanonicalValue::Bytes(self.input_hash.to_vec())); // input_hash
        map.insert(CanonicalValue::Integer(3), CanonicalValue::Bytes(self.output_hash.to_vec())); // output_hash
        map.insert(CanonicalValue::Integer(4), CanonicalValue::Integer(self.cycles_used)); // cycles
        map.insert(CanonicalValue::Integer(5), CanonicalValue::Integer(self.started_at)); // started_at
        map.insert(CanonicalValue::Integer(6), CanonicalValue::Integer(self.finished_at)); // finished_at
        map.insert(CanonicalValue::Integer(7), CanonicalValue::Text(self.issuer_key_id.clone())); // issuer_id
        map.insert(CanonicalValue::Integer(8), CanonicalValue::Bytes(self.signature.clone())); // signature

        // Add optional fields if present (v1.1 extensions)
        if let Some(prev_hash) = self.prev_receipt_hash {
            map.insert(CanonicalValue::Integer(9), CanonicalValue::Bytes(prev_hash.to_vec())); // prev_receipt_hash
        }
        if let Some(request_digest) = self.request_digest {
            map.insert(CanonicalValue::Integer(10), CanonicalValue::Bytes(request_digest.to_vec())); // request_digest
        }
        if !self.witness_signatures.is_empty() {
            let witness_array: Vec<CanonicalValue> = self
                .witness_signatures
                .iter()
                .map(|sig| CanonicalValue::Bytes(sig.clone()))
                .collect();
            map.insert(CanonicalValue::Integer(11), CanonicalValue::Array(witness_array)); // witness_signatures
        }

        // Add VDF fields if present (v1.2 temporal proof)
        if let Some(ref vdf_output) = self.vdf_output {
            map.insert(CanonicalValue::Integer(12), CanonicalValue::Bytes(vdf_output.clone()));
        }
        if let Some(ref vdf_proof) = self.vdf_proof {
            map.insert(CanonicalValue::Integer(13), CanonicalValue::Bytes(vdf_proof.clone()));
        }
        if let Some(vdf_iterations) = self.vdf_iterations {
            map.insert(CanonicalValue::Integer(14), CanonicalValue::Integer(vdf_iterations));
        }
        if let Some(ref vdf_modulus_id) = self.vdf_modulus_id {
            map.insert(CanonicalValue::Integer(15), CanonicalValue::Text(vdf_modulus_id.clone()));
        }

        // Serialize to canonical CBOR
        Self::serialize_canonical_map(&map)
    }

    /// Get signing message bytes
    pub fn get_signing_message(&self) -> Result<Vec<u8>, VerificationError> {
        // Create receipt without signature field
        let mut core_receipt = self.clone();
        core_receipt.signature = Vec::new();
        
        let core_cbor = core_receipt.to_canonical_cbor()?;
        Ok(crate::spec::create_signing_message(&core_cbor))
    }

    /// Public access to serialize_canonical_map for use by verify module.
    pub fn serialize_canonical_map_public(
        map: &BTreeMap<CanonicalValue, CanonicalValue>,
    ) -> Result<Vec<u8>, VerificationError> {
        Self::serialize_canonical_map(map)
    }

    /// Serialize a canonical CBOR map to bytes.
    fn serialize_canonical_map(
        map: &BTreeMap<CanonicalValue, CanonicalValue>,
    ) -> Result<Vec<u8>, VerificationError> {
        let mut output = Vec::new();

        // Encode map header
        Self::encode_map_header(&mut output, map.len())?;

        // Encode each key-value pair in canonical order
        for (key, value) in map {
            Self::encode_value(&mut output, key)?;
            Self::encode_value(&mut output, value)?;
        }

        Ok(output)
    }

    /// Encode a CBOR map header.
    fn encode_map_header(output: &mut Vec<u8>, len: usize) -> Result<(), VerificationError> {
        if len <= 23 {
            output.push(0xa0 | len as u8);
        } else if len <= 255 {
            output.push(0xb8);
            output.push(len as u8);
        } else if len <= 65535 {
            output.push(0xb9);
            output.extend_from_slice(&(len as u16).to_be_bytes());
        } else {
            return Err(VerificationError::InvalidCbor);
        }
        Ok(())
    }

    /// Encode a canonical CBOR value.
    fn encode_value(output: &mut Vec<u8>, value: &CanonicalValue) -> Result<(), VerificationError> {
        match value {
            CanonicalValue::Integer(n) => Self::encode_integer(output, *n),
            CanonicalValue::Bytes(bytes) => Self::encode_bytes(output, bytes),
            CanonicalValue::Text(text) => Self::encode_text(output, text),
            CanonicalValue::Array(array) => Self::encode_array(output, array),
            CanonicalValue::Map(map) => Self::serialize_canonical_map(map).map(|bytes| output.extend(bytes)),
        }
    }

    /// Encode a CBOR integer using minimal encoding.
    fn encode_integer(output: &mut Vec<u8>, value: u64) -> Result<(), VerificationError> {
        if value <= 23 {
            output.push(value as u8);
        } else if value <= 255 {
            output.push(0x18);
            output.push(value as u8);
        } else if value <= 65535 {
            output.push(0x19);
            output.extend_from_slice(&(value as u16).to_be_bytes());
        } else if value <= 4294967295 {
            output.push(0x1a);
            output.extend_from_slice(&(value as u32).to_be_bytes());
        } else {
            output.push(0x1b);
            output.extend_from_slice(&value.to_be_bytes());
        }
        Ok(())
    }

    /// Encode CBOR byte string.
    fn encode_bytes(output: &mut Vec<u8>, bytes: &[u8]) -> Result<(), VerificationError> {
        let len = bytes.len();
        if len <= 23 {
            output.push(0x40 | len as u8);
        } else if len <= 255 {
            output.push(0x58);
            output.push(len as u8);
        } else if len <= 65535 {
            output.push(0x59);
            output.extend_from_slice(&(len as u16).to_be_bytes());
        } else {
            return Err(VerificationError::InvalidCbor);
        }
        output.extend_from_slice(bytes);
        Ok(())
    }

    /// Encode CBOR text string.
    fn encode_text(output: &mut Vec<u8>, text: &str) -> Result<(), VerificationError> {
        let bytes = text.as_bytes();
        let len = bytes.len();
        if len <= 23 {
            output.push(0x60 | len as u8);
        } else if len <= 255 {
            output.push(0x78);
            output.push(len as u8);
        } else if len <= 65535 {
            output.push(0x79);
            output.extend_from_slice(&(len as u16).to_be_bytes());
        } else {
            return Err(VerificationError::InvalidCbor);
        }
        output.extend_from_slice(bytes);
        Ok(())
    }

    /// Encode CBOR array.
    fn encode_array(output: &mut Vec<u8>, array: &[CanonicalValue]) -> Result<(), VerificationError> {
        let len = array.len();
        if len <= 23 {
            output.push(0x80 | len as u8);
        } else if len <= 255 {
            output.push(0x98);
            output.push(len as u8);
        } else if len <= 65535 {
            output.push(0x99);
            output.extend_from_slice(&(len as u16).to_be_bytes());
        } else {
            return Err(VerificationError::InvalidCbor);
        }

        for item in array {
            Self::encode_value(output, item)?;
        }
        Ok(())
    }

    /// Extracts a required hash field by text key.
    fn extract_hash_by_text_key(map: &BTreeMap<CanonicalValue, CanonicalValue>, field_name: &str) -> Result<[u8; 32], VerificationError> {
        let key_value = CanonicalValue::Text(field_name.to_string());
        let value = map.get(&key_value)
            .ok_or_else(|| VerificationError::MissingField("field"))?;
        
        match value {
            CanonicalValue::Bytes(bytes) => {
                if bytes.len() != 32 {
                    return Err(VerificationError::InvalidFieldValue("field"));
                }
                let mut hash = [0u8; 32];
                hash.copy_from_slice(bytes);
                Ok(hash)
            }
            _ => Err(VerificationError::InvalidFieldValue("field")),
        }
    }

    /// Extracts a required uint64 field by text key.
    fn extract_uint64_by_text_key(map: &BTreeMap<CanonicalValue, CanonicalValue>, field_name: &str) -> Result<u64, VerificationError> {
        let key_value = CanonicalValue::Text(field_name.to_string());
        let value = map.get(&key_value)
            .ok_or_else(|| VerificationError::MissingField("field"))?;
        
        match value {
            CanonicalValue::Integer(i) => Ok(*i),
            _ => Err(VerificationError::InvalidFieldValue("field")),
        }
    }

    /// Extracts a required string field by text key.
    fn extract_string_by_text_key(map: &BTreeMap<CanonicalValue, CanonicalValue>, field_name: &str) -> Result<String, VerificationError> {
        let key_value = CanonicalValue::Text(field_name.to_string());
        let value = map.get(&key_value)
            .ok_or_else(|| VerificationError::MissingField("field"))?;
        
        match value {
            CanonicalValue::Text(s) => Ok(s.clone()),
            _ => Err(VerificationError::InvalidFieldValue("field")),
        }
    }

    /// Extracts a required bytes field by text key.
    fn extract_bytes_by_text_key(map: &BTreeMap<CanonicalValue, CanonicalValue>, field_name: &str) -> Result<Vec<u8>, VerificationError> {
        let key_value = CanonicalValue::Text(field_name.to_string());
        let value = map.get(&key_value)
            .ok_or_else(|| VerificationError::MissingField("field"))?;
        
        match value {
            CanonicalValue::Bytes(b) => Ok(b.clone()),
            _ => Err(VerificationError::InvalidFieldValue("field")),
        }
    }

    /// Extracts timestamp_ms and converts to seconds.
    fn extract_timestamp_ms(map: &BTreeMap<CanonicalValue, CanonicalValue>) -> Result<u64, VerificationError> {
        let key_value = CanonicalValue::Text("timestamp_ms".to_string());
        let value = map.get(&key_value)
            .ok_or_else(|| VerificationError::MissingField("timestamp_ms"))?;
        
        match value {
            CanonicalValue::Integer(ms) => Ok(ms / 1000), // Convert milliseconds to seconds
            _ => Err(VerificationError::InvalidFieldValue("timestamp_ms")),
        }
    }

    /// Extracts an optional hash field by text key.
    fn extract_optional_hash_by_text_key(map: &BTreeMap<CanonicalValue, CanonicalValue>, field_name: &str) -> Result<Option<[u8; 32]>, VerificationError> {
        let key_value = CanonicalValue::Text(field_name.to_string());
        match map.get(&key_value) {
            Some(CanonicalValue::Bytes(bytes)) => {
                if bytes.len() != 32 {
                    return Err(VerificationError::InvalidFieldValue("field"));
                }
                let mut hash = [0u8; 32];
                hash.copy_from_slice(bytes);
                Ok(Some(hash))
            }
            None => Ok(None),
            _ => Err(VerificationError::InvalidFieldValue("field")),
        }
    }

    /// Extracts an optional signatures array by text key.
    fn extract_optional_signatures_by_text_key(map: &BTreeMap<CanonicalValue, CanonicalValue>, field_name: &str) -> Result<Vec<Vec<u8>>, VerificationError> {
        let key_value = CanonicalValue::Text(field_name.to_string());
        match map.get(&key_value) {
            Some(CanonicalValue::Array(signatures)) => {
                let mut result = Vec::new();
                for sig in signatures {
                    match sig {
                        CanonicalValue::Bytes(bytes) => result.push(bytes.clone()),
                        _ => return Err(VerificationError::InvalidFieldValue("field")),
                    }
                }
                Ok(result)
            }
            None => Ok(Vec::new()),
            _ => Err(VerificationError::InvalidFieldValue("field")),
        }
    }

    /// Convert to canonical CBOR format (unsigned version for signing)
    fn to_canonical_cbor_unsigned(&self, unsigned: &UnsignedReceipt) -> Result<Vec<u8>, VerificationError> {
        // Use the same CBOR encoding as the Go side with integer keys
        let mut cbor_map = std::collections::BTreeMap::new();
        
        // Map fields to integer keys (matching OCX-CBOR v1.1 spec)
        cbor_map.insert(serde_cbor::Value::Integer(1), serde_cbor::Value::Bytes(unsigned.artifact_hash.to_vec()));
        cbor_map.insert(serde_cbor::Value::Integer(2), serde_cbor::Value::Bytes(unsigned.input_hash.to_vec()));
        cbor_map.insert(serde_cbor::Value::Integer(3), serde_cbor::Value::Bytes(unsigned.output_hash.to_vec()));
        cbor_map.insert(serde_cbor::Value::Integer(4), serde_cbor::Value::Integer(unsigned.cycles_used as i128));
        cbor_map.insert(serde_cbor::Value::Integer(5), serde_cbor::Value::Integer(unsigned.started_at as i128));
        cbor_map.insert(serde_cbor::Value::Integer(6), serde_cbor::Value::Integer(unsigned.finished_at as i128));
        cbor_map.insert(serde_cbor::Value::Integer(7), serde_cbor::Value::Text(unsigned.issuer_key_id.clone()));
        
        // Optional fields (only include if present)
        if let Some(prev_hash) = unsigned.prev_receipt_hash {
            cbor_map.insert(serde_cbor::Value::Integer(9), serde_cbor::Value::Bytes(prev_hash.to_vec()));
        }
        
        if let Some(request_digest) = unsigned.request_digest {
            cbor_map.insert(serde_cbor::Value::Integer(10), serde_cbor::Value::Bytes(request_digest.to_vec()));
        }
        
        if !unsigned.witness_signatures.is_empty() {
            let witness_array: Vec<serde_cbor::Value> = unsigned.witness_signatures
                .iter()
                .map(|sig| serde_cbor::Value::Bytes(sig.to_vec()))
                .collect();
            cbor_map.insert(serde_cbor::Value::Integer(11), serde_cbor::Value::Array(witness_array));
        }

        // VDF fields (v1.2 temporal proof — signature covers these)
        if let Some(ref vdf_output) = unsigned.vdf_output {
            cbor_map.insert(serde_cbor::Value::Integer(12), serde_cbor::Value::Bytes(vdf_output.clone()));
        }
        if let Some(ref vdf_proof) = unsigned.vdf_proof {
            cbor_map.insert(serde_cbor::Value::Integer(13), serde_cbor::Value::Bytes(vdf_proof.clone()));
        }
        if let Some(vdf_iterations) = unsigned.vdf_iterations {
            cbor_map.insert(serde_cbor::Value::Integer(14), serde_cbor::Value::Integer(vdf_iterations as i128));
        }
        if let Some(ref vdf_modulus_id) = unsigned.vdf_modulus_id {
            cbor_map.insert(serde_cbor::Value::Integer(15), serde_cbor::Value::Text(vdf_modulus_id.clone()));
        }

        // Note: Signature field (key 8) is intentionally omitted for signing

        let cbor_value = serde_cbor::Value::Map(cbor_map);

        // Serialize with deterministic encoding
        let mut buffer = Vec::new();
        serde_cbor::to_writer(&mut buffer, &cbor_value)
            .map_err(|_| VerificationError::InvalidCbor)?;

        Ok(buffer)
    }

    /// Convert complete receipt to canonical CBOR (including signature) - serde_cbor version
    pub fn to_canonical_cbor_serde(&self) -> Result<Vec<u8>, VerificationError> {
        let mut cbor_map = std::collections::BTreeMap::new();
        
        // All fields including signature
        cbor_map.insert(serde_cbor::Value::Integer(1), serde_cbor::Value::Bytes(self.artifact_hash.to_vec()));
        cbor_map.insert(serde_cbor::Value::Integer(2), serde_cbor::Value::Bytes(self.input_hash.to_vec()));
        cbor_map.insert(serde_cbor::Value::Integer(3), serde_cbor::Value::Bytes(self.output_hash.to_vec()));
        cbor_map.insert(serde_cbor::Value::Integer(4), serde_cbor::Value::Integer(self.cycles_used as i128));
        cbor_map.insert(serde_cbor::Value::Integer(5), serde_cbor::Value::Integer(self.started_at as i128));
        cbor_map.insert(serde_cbor::Value::Integer(6), serde_cbor::Value::Integer(self.finished_at as i128));
        cbor_map.insert(serde_cbor::Value::Integer(7), serde_cbor::Value::Text(self.issuer_key_id.clone()));
        cbor_map.insert(serde_cbor::Value::Integer(8), serde_cbor::Value::Bytes(self.signature.to_vec()));
        
        // Optional fields
        if let Some(prev_hash) = self.prev_receipt_hash {
            cbor_map.insert(serde_cbor::Value::Integer(9), serde_cbor::Value::Bytes(prev_hash.to_vec()));
        }
        
        if let Some(request_digest) = self.request_digest {
            cbor_map.insert(serde_cbor::Value::Integer(10), serde_cbor::Value::Bytes(request_digest.to_vec()));
        }
        
        if !self.witness_signatures.is_empty() {
            let witness_array: Vec<serde_cbor::Value> = self.witness_signatures
                .iter()
                .map(|sig| serde_cbor::Value::Bytes(sig.to_vec()))
                .collect();
            cbor_map.insert(serde_cbor::Value::Integer(11), serde_cbor::Value::Array(witness_array));
        }

        // VDF fields (v1.2 temporal proof)
        if let Some(ref vdf_output) = self.vdf_output {
            cbor_map.insert(serde_cbor::Value::Integer(12), serde_cbor::Value::Bytes(vdf_output.clone()));
        }
        if let Some(ref vdf_proof) = self.vdf_proof {
            cbor_map.insert(serde_cbor::Value::Integer(13), serde_cbor::Value::Bytes(vdf_proof.clone()));
        }
        if let Some(vdf_iterations) = self.vdf_iterations {
            cbor_map.insert(serde_cbor::Value::Integer(14), serde_cbor::Value::Integer(vdf_iterations as i128));
        }
        if let Some(ref vdf_modulus_id) = self.vdf_modulus_id {
            cbor_map.insert(serde_cbor::Value::Integer(15), serde_cbor::Value::Text(vdf_modulus_id.clone()));
        }

        let cbor_value = serde_cbor::Value::Map(cbor_map);
        let mut buffer = Vec::new();
        serde_cbor::to_writer(&mut buffer, &cbor_value)
            .map_err(|_| VerificationError::InvalidCbor)?;

        Ok(buffer)
    }

    /// Returns true if this receipt contains VDF temporal proof fields.
    pub fn has_vdf_proof(&self) -> bool {
        self.vdf_output.is_some()
            && self.vdf_proof.is_some()
            && self.vdf_iterations.is_some()
            && self.vdf_modulus_id.is_some()
    }
}
