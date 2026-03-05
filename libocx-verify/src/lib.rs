//! # libocx-verify
//! A high-assurance verifier for OCX cryptographic receipts.
//!
//! This library parses and verifies receipts encoded in canonical OCX-CBOR v1.1 format,
//! ensuring cryptographic integrity and adherence to the specification.

#![warn(missing_docs, missing_debug_implementations, unused_crate_dependencies)]

// Suppress unused dependency warning for ring (will be used in Part 2)
use ring as _;

pub mod canonical_cbor;
pub mod debug;
pub mod error;
pub mod receipt;
pub mod spec;
pub mod vdf;
pub mod verify;

// FFI module with unsafe code
#[cfg(feature = "ffi")]
pub mod ffi;

// Re-exports for library users
pub use error::VerificationError;
pub use receipt::OcxReceipt;
pub use verify::{verify_receipt, verify_receipt_simple, verify_receipts_batch, 
                 verify_receipt_trusted, verify_receipt_with_policy, VerificationPolicy};

// Re-export canonical CBOR types for testing
pub use canonical_cbor::{CanonicalValue, CborParser};

// Re-export FFI types when feature is enabled
#[cfg(feature = "ffi")]
pub use ffi::{OcxErrorCode, OcxReceiptFields, OcxReceiptData};