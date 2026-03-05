//! Foreign Function Interface (FFI) for C compatibility.
//!
//! This module provides a clean C interface to the OCX verification library,
//! enabling integration from any programming language that can call C functions.
//!
//! # Safety
//! This is the only module allowed to use `unsafe` code. All unsafe operations
//! are carefully validated to ensure memory safety.

use std::os::raw::c_char;
use std::ptr;
use std::slice;
use crate::{verify_receipt, verify_receipt_simple, VerificationError, OcxReceipt};
use crate::vdf;

/// Error codes for C API compatibility.
#[repr(C)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum OcxErrorCode {
    /// Operation succeeded.
    Success = 0,
    /// Invalid CBOR encoding.
    InvalidCbor = 1,
    /// CBOR data is not canonical.
    NonCanonicalCbor = 2,
    /// Required field is missing.
    MissingField = 3,
    /// Field has invalid value.
    InvalidFieldValue = 4,
    /// Cryptographic signature is invalid.
    InvalidSignature = 5,
    /// Hash mismatch detected.
    HashMismatch = 6,
    /// Timestamp is invalid.
    InvalidTimestamp = 7,
    /// Unexpected end of input.
    UnexpectedEof = 8,
    /// Integer overflow or underflow.
    IntegerOverflow = 9,
    /// Invalid UTF-8 sequence.
    InvalidUtf8 = 10,
    /// Invalid input parameters.
    InvalidInput = 11,
    /// Internal error.
    InternalError = 12,
}

impl From<VerificationError> for OcxErrorCode {
    fn from(error: VerificationError) -> Self {
        match error {
            VerificationError::InvalidCbor => OcxErrorCode::InvalidCbor,
            VerificationError::NonCanonicalCbor => OcxErrorCode::NonCanonicalCbor,
            VerificationError::MissingField(_) => OcxErrorCode::MissingField,
            VerificationError::InvalidFieldValue(_) => OcxErrorCode::InvalidFieldValue,
            VerificationError::InvalidSignature => OcxErrorCode::InvalidSignature,
            VerificationError::HashMismatch(_) => OcxErrorCode::HashMismatch,
            VerificationError::InvalidTimestamp => OcxErrorCode::InvalidTimestamp,
            VerificationError::UnexpectedEof => OcxErrorCode::UnexpectedEof,
            VerificationError::IntegerOverflow => OcxErrorCode::IntegerOverflow,
            VerificationError::InvalidUtf8 => OcxErrorCode::InvalidUtf8,
        }
    }
}

/// Simple verification function that returns only success/failure.
///
/// # Safety
/// - `cbor_data` must point to valid memory of at least `cbor_data_len` bytes
/// - `cbor_data` must remain valid for the duration of this function call
/// - `public_key` must point to exactly 32 bytes of valid memory
/// - `public_key` must remain valid for the duration of this function call
///
/// # Arguments
/// * `cbor_data` - Pointer to CBOR receipt data
/// * `cbor_data_len` - Length of CBOR data in bytes
/// * `public_key` - Pointer to 32-byte Ed25519 public key
///
/// # Returns
/// * `true` if verification succeeds
/// * `false` if verification fails
#[no_mangle]
pub unsafe extern "C" fn ocx_verify_receipt(
    cbor_data: *const u8,
    cbor_data_len: usize,
    public_key: *const u8,
) -> bool {
    // Validate input pointers
    if cbor_data.is_null() || public_key.is_null() {
        return false;
    }

    // Validate data length
    if cbor_data_len == 0 || cbor_data_len > 1024 * 1024 { // Max 1MB
        return false;
    }

    // Create safe slices from raw pointers
    let cbor_slice = slice::from_raw_parts(cbor_data, cbor_data_len);
    let key_slice = slice::from_raw_parts(public_key, 32);

    // Perform verification
    verify_receipt(cbor_slice, key_slice, false).is_ok()
}

/// Detailed verification function that returns error codes.
///
/// # Safety
/// - `cbor_data` must point to valid memory of at least `cbor_data_len` bytes
/// - `public_key` must point to exactly 32 bytes of valid memory
/// - `error_code` must point to valid memory for writing an `OcxErrorCode`
///
/// # Arguments
/// * `cbor_data` - Pointer to CBOR receipt data
/// * `cbor_data_len` - Length of CBOR data in bytes
/// * `public_key` - Pointer to 32-byte Ed25519 public key
/// * `error_code` - Pointer to write error code (if verification fails)
///
/// # Returns
/// * `true` if verification succeeds
/// * `false` if verification fails (error code written to `error_code`)
#[no_mangle]
pub unsafe extern "C" fn ocx_verify_receipt_detailed(
    cbor_data: *const u8,
    cbor_data_len: usize,
    public_key: *const u8,
    error_code: *mut OcxErrorCode,
) -> bool {
    // Validate input pointers
    if cbor_data.is_null() || public_key.is_null() || error_code.is_null() {
        if !error_code.is_null() {
            *error_code = OcxErrorCode::InvalidInput;
        }
        return false;
    }

    // Validate data length
    if cbor_data_len == 0 || cbor_data_len > 1024 * 1024 {
        *error_code = OcxErrorCode::InvalidInput;
        return false;
    }

    // Create safe slices from raw pointers
    let cbor_slice = slice::from_raw_parts(cbor_data, cbor_data_len);
    let key_slice = slice::from_raw_parts(public_key, 32);

    // Perform verification
    match verify_receipt(cbor_slice, key_slice, false) {
        Ok(_) => {
            *error_code = OcxErrorCode::Success;
            true
        }
        Err(err) => {
            *error_code = err.into();
            false
        }
    }
}

/// Simple verification using key ID embedded in receipt.
///
/// # Safety
/// - `cbor_data` must point to valid memory of at least `cbor_data_len` bytes
///
/// # Arguments
/// * `cbor_data` - Pointer to CBOR receipt data
/// * `cbor_data_len` - Length of CBOR data in bytes
///
/// # Returns
/// * `true` if verification succeeds
/// * `false` if verification fails
#[no_mangle]
pub unsafe extern "C" fn ocx_verify_receipt_simple(
    cbor_data: *const u8,
    cbor_data_len: usize,
) -> bool {
    // Validate input pointers
    if cbor_data.is_null() || cbor_data_len == 0 || cbor_data_len > 1024 * 1024 {
        return false;
    }

    // Create safe slice from raw pointer
    let cbor_slice = slice::from_raw_parts(cbor_data, cbor_data_len);

    // Perform verification
    verify_receipt_simple(cbor_slice).is_ok()
}

/// Extract receipt fields for C applications.
///
/// # Safety
/// - All pointer parameters must be valid for writing
/// - String parameters must have sufficient buffer space
/// - Hash parameters must point to exactly 32 bytes of writable memory
///
/// # Returns
/// * `OcxErrorCode::Success` if extraction succeeds
/// * Error code if extraction fails
#[repr(C)]
#[derive(Debug)]
pub struct OcxReceiptFields {
    /// Artifact hash (32 bytes)
    pub artifact_hash: [u8; 32],
    /// Input hash (32 bytes)
    pub input_hash: [u8; 32],
    /// Output hash (32 bytes)
    pub output_hash: [u8; 32],
    /// Computational cycles used
    pub cycles_used: u64,
    /// Unix timestamp when execution started
    pub started_at: u64,
    /// Unix timestamp when execution finished
    pub finished_at: u64,
    /// Length of issuer key ID string
    pub issuer_key_id_len: usize,
    /// Length of signature
    pub signature_len: usize,
}

/// Extract receipt fields for C applications.
///
/// # Safety
/// - All pointer parameters must be valid for writing
/// - String parameters must have sufficient buffer space
/// - Hash parameters must point to exactly 32 bytes of writable memory
///
/// # Returns
/// * `OcxErrorCode::Success` if extraction succeeds
/// * Error code if extraction fails
#[no_mangle]
pub unsafe extern "C" fn ocx_extract_receipt_fields(
    cbor_data: *const u8,
    cbor_data_len: usize,
    fields: *mut OcxReceiptFields,
    issuer_key_id: *mut c_char,
    issuer_key_id_max_len: usize,
    signature: *mut u8,
    signature_max_len: usize,
) -> OcxErrorCode {
    // Validate input pointers
    if cbor_data.is_null() || fields.is_null() || 
       issuer_key_id.is_null() || signature.is_null() {
        return OcxErrorCode::InvalidInput;
    }

    // Validate data length
    if cbor_data_len == 0 || cbor_data_len > 1024 * 1024 {
        return OcxErrorCode::InvalidInput;
    }

    // Create safe slice from raw pointer
    let cbor_slice = slice::from_raw_parts(cbor_data, cbor_data_len);

    // Parse receipt
    let receipt = match OcxReceipt::from_canonical_cbor(cbor_slice) {
        Ok(r) => r,
        Err(err) => return err.into(),
    };

    // Check buffer sizes
    if receipt.issuer_key_id.len() >= issuer_key_id_max_len {
        return OcxErrorCode::InvalidInput;
    }
    if receipt.signature.len() > signature_max_len {
        return OcxErrorCode::InvalidInput;
    }

    // Fill fields structure
    (*fields).artifact_hash = receipt.artifact_hash;
    (*fields).input_hash = receipt.input_hash;
    (*fields).output_hash = receipt.output_hash;
    (*fields).cycles_used = receipt.cycles_used;
    (*fields).started_at = receipt.started_at;
    (*fields).finished_at = receipt.finished_at;
    (*fields).issuer_key_id_len = receipt.issuer_key_id.len();
    (*fields).signature_len = receipt.signature.len();

    // Copy string data
    let key_id_bytes = receipt.issuer_key_id.as_bytes();
    ptr::copy_nonoverlapping(
        key_id_bytes.as_ptr(),
        issuer_key_id as *mut u8,
        key_id_bytes.len(),
    );
    // Null-terminate the string
    *((issuer_key_id as *mut u8).add(key_id_bytes.len())) = 0;

    // Copy signature data
    ptr::copy_nonoverlapping(
        receipt.signature.as_ptr(),
        signature,
        receipt.signature.len(),
    );

    OcxErrorCode::Success
}

/// Get error message for an error code.
///
/// # Safety
/// - `buffer` must point to writable memory of at least `buffer_len` bytes
///
/// # Returns
/// * Number of bytes written to buffer (including null terminator)
/// * 0 if buffer is too small or invalid parameters
#[no_mangle]
pub unsafe extern "C" fn ocx_get_error_message(
    error_code: OcxErrorCode,
    buffer: *mut c_char,
    buffer_len: usize,
) -> usize {
    if buffer.is_null() || buffer_len == 0 {
        return 0;
    }

    let message = match error_code {
        OcxErrorCode::Success => "Success",
        OcxErrorCode::InvalidCbor => "Invalid CBOR encoding",
        OcxErrorCode::NonCanonicalCbor => "CBOR data is not canonical",
        OcxErrorCode::MissingField => "Required field is missing",
        OcxErrorCode::InvalidFieldValue => "Field has invalid value",
        OcxErrorCode::InvalidSignature => "Cryptographic signature is invalid",
        OcxErrorCode::HashMismatch => "Hash mismatch detected",
        OcxErrorCode::InvalidTimestamp => "Timestamp is invalid",
        OcxErrorCode::UnexpectedEof => "Unexpected end of input",
        OcxErrorCode::IntegerOverflow => "Integer overflow or underflow",
        OcxErrorCode::InvalidUtf8 => "Invalid UTF-8 sequence",
        OcxErrorCode::InvalidInput => "Invalid input parameters",
        OcxErrorCode::InternalError => "Internal error",
    };

    let message_bytes = message.as_bytes();
    let copy_len = std::cmp::min(message_bytes.len(), buffer_len - 1);

    // Copy message to buffer
    ptr::copy_nonoverlapping(
        message_bytes.as_ptr(),
        buffer as *mut u8,
        copy_len,
    );

    // Null-terminate
    *((buffer as *mut u8).add(copy_len)) = 0;

    copy_len + 1 // Include null terminator in count
}

/// Get library version information.
///
/// # Safety
/// - `buffer` must point to writable memory of at least `buffer_len` bytes
///
/// # Returns
/// * Number of bytes written to buffer (including null terminator)
/// * 0 if buffer is too small or invalid parameters
#[no_mangle]
pub unsafe extern "C" fn ocx_get_version(
    buffer: *mut c_char,
    buffer_len: usize,
) -> usize {
    if buffer.is_null() || buffer_len == 0 {
        return 0;
    }

    let version = env!("CARGO_PKG_VERSION");
    let version_bytes = version.as_bytes();
    let copy_len = std::cmp::min(version_bytes.len(), buffer_len - 1);

    // Copy version to buffer
    ptr::copy_nonoverlapping(
        version_bytes.as_ptr(),
        buffer as *mut u8,
        copy_len,
    );

    // Null-terminate
    *((buffer as *mut u8).add(copy_len)) = 0;

    copy_len + 1
}

/// Receipt data structure for batch processing.
#[repr(C)]
#[derive(Debug)]
pub struct OcxReceiptData {
    /// Pointer to CBOR receipt data
    pub cbor_data: *const u8,
    /// Length of CBOR data in bytes
    pub cbor_data_len: usize,
    /// Pointer to 32-byte Ed25519 public key
    pub public_key: *const u8,
}

/// Batch verification function for C applications.
///
/// # Safety
/// - `receipts` must point to valid array of `receipt_count` elements
/// - Each receipt must have valid `cbor_data` and `public_key` pointers
/// - `results` must point to writable array of `receipt_count` elements
///
/// # Returns
/// * Number of successfully verified receipts
#[no_mangle]
pub unsafe extern "C" fn ocx_verify_receipts_batch(
    receipts: *const OcxReceiptData,
    receipt_count: usize,
    results: *mut bool,
) -> usize {
    if receipts.is_null() || results.is_null() || receipt_count == 0 {
        return 0;
    }

    let mut success_count = 0;

    for i in 0..receipt_count {
        let receipt = &*receipts.add(i);
        
        // Validate individual receipt
        if receipt.cbor_data.is_null() || receipt.public_key.is_null() ||
           receipt.cbor_data_len == 0 || receipt.cbor_data_len > 1024 * 1024 {
            *results.add(i) = false;
            continue;
        }

        // Create safe slices
        let cbor_slice = slice::from_raw_parts(receipt.cbor_data, receipt.cbor_data_len);
        let key_slice = slice::from_raw_parts(receipt.public_key, 32);

        // Verify receipt
        let is_valid = verify_receipt(cbor_slice, key_slice, false).is_ok();
        *results.add(i) = is_valid;
        
        if is_valid {
            success_count += 1;
        }
    }

    success_count
}

// ── VDF FFI ─────────────────────────────────────────────────────────────

/// VDF proof result for C API.
#[repr(C)]
#[derive(Debug)]
pub struct OcxVdfProof {
    /// VDF output y = x^(2^T) mod N (big-endian bytes).
    pub output: [u8; 256],
    /// Actual length of output bytes.
    pub output_len: u32,
    /// Wesolowski proof π (big-endian bytes).
    pub proof: [u8; 256],
    /// Actual length of proof bytes.
    pub proof_len: u32,
    /// Number of sequential squarings.
    pub iterations: u64,
    /// Modulus identifier (null-terminated ASCII).
    pub modulus_id: [u8; 64],
    /// Wall-clock time taken in milliseconds.
    pub duration_ms: u64,
}

/// Evaluate VDF: compute temporal proof for a 32-byte receipt hash.
///
/// # Safety
/// - `receipt_hash` must point to exactly 32 bytes of valid memory
/// - `out_proof` must point to a writable `OcxVdfProof` struct
///
/// # Arguments
/// * `receipt_hash` - 32-byte SHA-256 hash of the receipt core
/// * `iterations` - Number of sequential squarings T
/// * `out_proof` - Output proof struct
///
/// # Returns
/// * `OcxErrorCode::Success` on success
/// * Error code on failure
#[no_mangle]
pub unsafe extern "C" fn ocx_vdf_evaluate(
    receipt_hash: *const u8,
    iterations: u64,
    out_proof: *mut OcxVdfProof,
) -> OcxErrorCode {
    if receipt_hash.is_null() || out_proof.is_null() {
        return OcxErrorCode::InvalidInput;
    }

    // Read the 32-byte hash
    let hash_slice = slice::from_raw_parts(receipt_hash, 32);
    let mut hash = [0u8; 32];
    hash.copy_from_slice(hash_slice);

    // Get default VDF parameters
    let params = vdf::default_params();

    // Validate iterations
    if let Err(_) = params.validate_iterations(iterations) {
        return OcxErrorCode::InvalidFieldValue;
    }

    // Derive challenge from receipt hash
    let challenge = vdf::derive_challenge(&hash, &params.modulus);

    // Evaluate VDF
    let proof = match vdf::evaluate(&challenge, iterations, &params) {
        Ok(p) => p,
        Err(_) => return OcxErrorCode::InternalError,
    };

    // Copy output into FFI struct
    let output_bytes = &proof.output;
    let proof_bytes = &proof.proof;

    if output_bytes.len() > 256 || proof_bytes.len() > 256 {
        return OcxErrorCode::InternalError;
    }

    // Zero-initialize the output struct
    ptr::write_bytes(out_proof, 0, 1);

    // Copy output
    ptr::copy_nonoverlapping(
        output_bytes.as_ptr(),
        (*out_proof).output.as_mut_ptr().add(256 - output_bytes.len()),
        output_bytes.len(),
    );
    (*out_proof).output_len = output_bytes.len() as u32;

    // Copy proof
    ptr::copy_nonoverlapping(
        proof_bytes.as_ptr(),
        (*out_proof).proof.as_mut_ptr().add(256 - proof_bytes.len()),
        proof_bytes.len(),
    );
    (*out_proof).proof_len = proof_bytes.len() as u32;

    (*out_proof).iterations = iterations;
    (*out_proof).duration_ms = proof.duration_ms;

    // Copy modulus ID
    let modulus_id = params.modulus_id.as_bytes();
    let copy_len = std::cmp::min(modulus_id.len(), 63);
    ptr::copy_nonoverlapping(
        modulus_id.as_ptr(),
        (*out_proof).modulus_id.as_mut_ptr(),
        copy_len,
    );

    OcxErrorCode::Success
}

/// Verify a VDF proof against a 32-byte receipt hash.
///
/// # Safety
/// - `receipt_hash` must point to exactly 32 bytes of valid memory
/// - `proof` must point to a valid `OcxVdfProof` struct
///
/// # Returns
/// * `true` if the VDF proof is valid
/// * `false` if invalid or on error
#[no_mangle]
pub unsafe extern "C" fn ocx_vdf_verify(
    receipt_hash: *const u8,
    proof: *const OcxVdfProof,
) -> bool {
    if receipt_hash.is_null() || proof.is_null() {
        return false;
    }

    // Read the 32-byte hash
    let hash_slice = slice::from_raw_parts(receipt_hash, 32);
    let mut hash = [0u8; 32];
    hash.copy_from_slice(hash_slice);

    // Extract modulus ID from proof
    let modulus_id_bytes = &(*proof).modulus_id;
    let modulus_id_len = modulus_id_bytes.iter().position(|&b| b == 0).unwrap_or(64);
    let modulus_id = match std::str::from_utf8(&modulus_id_bytes[..modulus_id_len]) {
        Ok(s) => s,
        Err(_) => return false,
    };

    // Look up VDF parameters
    let params = match vdf::get_params(modulus_id) {
        Ok(p) => p,
        Err(_) => return false,
    };

    // Derive challenge
    let challenge = vdf::derive_challenge(&hash, &params.modulus);

    // Extract output and proof bytes (right-aligned in 256-byte buffer)
    use num_bigint::BigUint;
    let output_len = (*proof).output_len as usize;
    let proof_len = (*proof).proof_len as usize;

    if output_len == 0 || output_len > 256 || proof_len == 0 || proof_len > 256 {
        return false;
    }

    let output = BigUint::from_bytes_be(&(*proof).output[256 - output_len..]);
    let proof_val = BigUint::from_bytes_be(&(*proof).proof[256 - proof_len..]);

    // Verify
    match vdf::verify(&challenge, &output, &proof_val, (*proof).iterations, &params) {
        Ok(valid) => valid,
        Err(_) => false,
    }
}
