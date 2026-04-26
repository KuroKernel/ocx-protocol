//! Browser-side OCX receipt verifier.
//!
//! This crate compiles the canonical Rust verification pipeline (canonical
//! CBOR parsing + signing-message construction + Ed25519 signature check)
//! to WebAssembly so that a browser can verify OCX receipts entirely
//! offline, with no server in the trust path.
//!
//! The native verifier in `libocx-verify` performs a dual-library check:
//! it confirms each signature with both `ring` and `ed25519-dalek` to
//! catch implementation divergence. This wasm build skips the ring leg
//! (ring doesn't cross-compile cleanly to `wasm32-unknown-unknown`) and
//! relies on `ed25519-dalek` alone — the same library `libocx-verify`
//! uses as its second cross-check, and the canonical reference Rust
//! Ed25519 implementation. The CBOR parser, signing-message construction,
//! receipt struct, and error enum are all imported directly from
//! `libocx-verify` so there is exactly one source of truth for receipt
//! structure and one for signature math.

use ed25519_dalek::{Signature, Verifier, VerifyingKey};
use libocx_verify::{OcxReceipt, VerificationError};
use serde::Serialize;
use wasm_bindgen::prelude::*;

/// Initializes the wasm module. Hooks the panic handler so any unwrap in
/// release builds surfaces in the browser console with a useful trace.
#[wasm_bindgen(start)]
pub fn init() {
    #[cfg(feature = "console_error_panic_hook")]
    console_error_panic_hook::set_once();
}

/// Discriminant codes for verification outcomes. These match the public
/// C ABI exposed by `libocx-verify::ffi::OcxErrorCode` so the same set
/// of strings is meaningful across native, FFI, and browser callers.
#[derive(Serialize, Debug, Clone, Copy)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum OcxStatusCode {
    /// Receipt parsed, signature valid against the supplied public key.
    Success,
    /// Input bytes failed CBOR parsing.
    InvalidCbor,
    /// CBOR is well-formed but not in canonical form (RFC 8949 § 4.2).
    NonCanonicalCbor,
    /// The receipt is missing a required CBOR field.
    MissingField,
    /// A field value is outside its valid range.
    InvalidFieldValue,
    /// Public key length wrong, signature length wrong, or Ed25519 verify failed.
    InvalidSignature,
    /// One of the embedded hashes does not match.
    HashMismatch,
    /// Receipt timestamp is outside acceptable bounds.
    InvalidTimestamp,
    /// Truncated CBOR input.
    UnexpectedEof,
    /// Integer overflow / underflow during parsing.
    IntegerOverflow,
    /// Invalid UTF-8 in a CBOR text-string field.
    InvalidUtf8,
    /// Public key was not 32 bytes.
    InvalidPublicKeyLength,
    /// Hex input did not parse.
    InvalidHexInput,
}

impl From<&VerificationError> for OcxStatusCode {
    fn from(err: &VerificationError) -> Self {
        match err {
            VerificationError::InvalidCbor => Self::InvalidCbor,
            VerificationError::NonCanonicalCbor => Self::NonCanonicalCbor,
            VerificationError::MissingField(_) => Self::MissingField,
            VerificationError::InvalidFieldValue(_) => Self::InvalidFieldValue,
            VerificationError::InvalidSignature => Self::InvalidSignature,
            VerificationError::HashMismatch(_) => Self::HashMismatch,
            VerificationError::InvalidTimestamp => Self::InvalidTimestamp,
            VerificationError::UnexpectedEof => Self::UnexpectedEof,
            VerificationError::IntegerOverflow => Self::IntegerOverflow,
            VerificationError::InvalidUtf8 => Self::InvalidUtf8,
        }
    }
}

/// A flat, JS-friendly view of the receipt's parsed fields. Hashes are
/// rendered as lowercase hex; byte fields whose presence is structural
/// (witnesses, VDF proof) are summarized as counts/booleans rather than
/// inlined as raw bytes.
#[derive(Serialize, Debug, Default)]
pub struct ReceiptView {
    pub artifact_hash: String,
    pub input_hash: String,
    pub output_hash: String,
    pub cycles_used: u64,
    pub started_at: u64,
    pub finished_at: u64,
    pub duration_seconds: u64,
    pub issuer_key_id: String,
    pub signature_hex: String,
    pub has_prev_receipt_hash: bool,
    pub has_request_digest: bool,
    pub witness_count: usize,
    pub has_vdf_proof: bool,
    pub vdf_iterations: Option<u64>,
}

impl From<&OcxReceipt> for ReceiptView {
    fn from(r: &OcxReceipt) -> Self {
        Self {
            artifact_hash: hex::encode(r.artifact_hash),
            input_hash: hex::encode(r.input_hash),
            output_hash: hex::encode(r.output_hash),
            cycles_used: r.cycles_used,
            started_at: r.started_at,
            finished_at: r.finished_at,
            duration_seconds: r.finished_at.saturating_sub(r.started_at),
            issuer_key_id: r.issuer_key_id.clone(),
            signature_hex: hex::encode(&r.signature),
            has_prev_receipt_hash: r.prev_receipt_hash.is_some(),
            has_request_digest: r.request_digest.is_some(),
            witness_count: r.witness_signatures.len(),
            has_vdf_proof: r.vdf_proof.is_some(),
            vdf_iterations: r.vdf_iterations,
        }
    }
}

/// The full result returned to JS — single object covering both happy
/// path and every named failure mode. JS code can switch on
/// `status_code` for the canonical OCX_* discriminant or read
/// `ok` for a boolean shortcut.
#[derive(Serialize, Debug)]
pub struct VerifyResult {
    pub ok: bool,
    pub status_code: OcxStatusCode,
    pub message: String,
    pub receipt: Option<ReceiptView>,
    pub bytes_total: usize,
    pub signed_message_bytes: usize,
}

/// Internal helper: take parsed bytes for the receipt CBOR and the
/// 32-byte public key, return a `VerifyResult`. All error branches
/// build a structured failure rather than panicking.
fn verify_inner(cbor: &[u8], pubkey: &[u8]) -> VerifyResult {
    let bytes_total = cbor.len();

    // 1. Parse canonical CBOR through the canonical receipt struct.
    let receipt = match OcxReceipt::from_canonical_cbor(cbor) {
        Ok(r) => r,
        Err(e) => {
            return VerifyResult {
                ok: false,
                status_code: (&e).into(),
                message: format!("{}", e),
                receipt: None,
                bytes_total,
                signed_message_bytes: 0,
            };
        }
    };
    let view = ReceiptView::from(&receipt);

    // 2. Public key must be exactly 32 bytes.
    if pubkey.len() != 32 {
        return VerifyResult {
            ok: false,
            status_code: OcxStatusCode::InvalidPublicKeyLength,
            message: format!(
                "public key must be 32 bytes; got {}",
                pubkey.len()
            ),
            receipt: Some(view),
            bytes_total,
            signed_message_bytes: 0,
        };
    }
    let mut pk_bytes = [0u8; 32];
    pk_bytes.copy_from_slice(pubkey);

    let pk = match VerifyingKey::from_bytes(&pk_bytes) {
        Ok(k) => k,
        Err(_) => {
            return VerifyResult {
                ok: false,
                status_code: OcxStatusCode::InvalidSignature,
                message: "public key bytes did not decode as a valid Ed25519 point".into(),
                receipt: Some(view),
                bytes_total,
                signed_message_bytes: 0,
            };
        }
    };

    // 3. Build the canonical signing message — same path the native
    //    verifier uses (domain separator + canonical CBOR of unsigned core).
    let signing_message = match receipt.get_signing_message() {
        Ok(m) => m,
        Err(e) => {
            return VerifyResult {
                ok: false,
                status_code: (&e).into(),
                message: format!("could not construct signing message: {}", e),
                receipt: Some(view),
                bytes_total,
                signed_message_bytes: 0,
            };
        }
    };
    let signed_message_bytes = signing_message.len();

    // 4. Signature must be exactly 64 bytes.
    if receipt.signature.len() != 64 {
        return VerifyResult {
            ok: false,
            status_code: OcxStatusCode::InvalidSignature,
            message: format!(
                "signature must be 64 bytes; got {}",
                receipt.signature.len()
            ),
            receipt: Some(view),
            bytes_total,
            signed_message_bytes,
        };
    }
    let mut sig_bytes = [0u8; 64];
    sig_bytes.copy_from_slice(&receipt.signature);
    let signature = Signature::from_bytes(&sig_bytes);

    // 5. The actual cryptographic check.
    match pk.verify(&signing_message, &signature) {
        Ok(()) => VerifyResult {
            ok: true,
            status_code: OcxStatusCode::Success,
            message: "OCX_SUCCESS — Ed25519 signature valid over canonical receipt".into(),
            receipt: Some(view),
            bytes_total,
            signed_message_bytes,
        },
        Err(_) => VerifyResult {
            ok: false,
            status_code: OcxStatusCode::InvalidSignature,
            message: "Ed25519 signature did not verify against the supplied public key".into(),
            receipt: Some(view),
            bytes_total,
            signed_message_bytes,
        },
    }
}

/// Primary entrypoint — accepts raw bytes from JS (`Uint8Array`).
///
/// Returns a JS object matching the `VerifyResult` shape above.
#[wasm_bindgen(js_name = ocxVerifyBytes)]
pub fn verify_bytes(cbor: &[u8], pubkey: &[u8]) -> JsValue {
    let result = verify_inner(cbor, pubkey);
    serde_wasm_bindgen::to_value(&result)
        .unwrap_or_else(|_| JsValue::NULL)
}

/// Hex-string convenience entrypoint — accepts the same inputs as the
/// website form (whitespace-tolerant lowercase hex). Useful when the JS
/// caller doesn't want to do the hex → Uint8Array conversion itself.
#[wasm_bindgen(js_name = ocxVerifyHex)]
pub fn verify_hex(cbor_hex: &str, pubkey_hex: &str) -> JsValue {
    let cbor_clean: String = cbor_hex.chars().filter(|c| !c.is_whitespace()).collect();
    let pubkey_clean: String = pubkey_hex.chars().filter(|c| !c.is_whitespace()).collect();

    let cbor = match hex::decode(&cbor_clean) {
        Ok(b) => b,
        Err(_) => {
            let result = VerifyResult {
                ok: false,
                status_code: OcxStatusCode::InvalidHexInput,
                message: "receipt is not valid hex".into(),
                receipt: None,
                bytes_total: 0,
                signed_message_bytes: 0,
            };
            return serde_wasm_bindgen::to_value(&result).unwrap_or(JsValue::NULL);
        }
    };
    let pubkey = match hex::decode(&pubkey_clean) {
        Ok(b) => b,
        Err(_) => {
            let result = VerifyResult {
                ok: false,
                status_code: OcxStatusCode::InvalidHexInput,
                message: "public key is not valid hex".into(),
                receipt: None,
                bytes_total: cbor.len(),
                signed_message_bytes: 0,
            };
            return serde_wasm_bindgen::to_value(&result).unwrap_or(JsValue::NULL);
        }
    };

    verify_bytes(&cbor, &pubkey)
}

/// Build / version metadata, exposed for the website to surface in the
/// "Run it locally" section so the user can confirm which exact verifier
/// build the browser is running.
#[wasm_bindgen(js_name = ocxVerifierVersion)]
pub fn verifier_version() -> String {
    format!(
        "libocx-verify-wasm {} (libocx-verify {})",
        env!("CARGO_PKG_VERSION"),
        // libocx-verify's version is read from its own Cargo.toml at build
        // time via the cargo:: rerun-if-changed metadata exposed through env!.
        // Fall back to a literal if the env var is absent.
        option_env!("LIBOCX_VERIFY_VERSION").unwrap_or("0.1.0"),
    )
}
