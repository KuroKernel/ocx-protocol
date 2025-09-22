//! Error types for OCX receipt verification.

use std::fmt;

/// All possible errors that can occur during receipt verification.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum VerificationError {
    /// The input data is not valid CBOR.
    InvalidCbor,
    /// The CBOR is not in canonical form (wrong field order, non-minimal encoding, etc.).
    NonCanonicalCbor,
    /// A required field is missing from the receipt.
    MissingField(&'static str),
    /// A field has a value that is out of its valid range.
    InvalidFieldValue(&'static str),
    /// The cryptographic signature is invalid.
    InvalidSignature,
    /// The hash of a field does not match the computed value.
    HashMismatch(&'static str),
    /// The receipt's timestamp is invalid or outside acceptable bounds.
    InvalidTimestamp,
    /// Unexpected end of input while parsing.
    UnexpectedEof,
    /// Integer overflow or underflow.
    IntegerOverflow,
    /// Invalid UTF-8 sequence in text string.
    InvalidUtf8,
}

impl fmt::Display for VerificationError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            VerificationError::InvalidCbor => write!(f, "invalid CBOR encoding"),
            VerificationError::NonCanonicalCbor => write!(f, "CBOR data is not canonical"),
            VerificationError::MissingField(field) => write!(f, "missing required field: {}", field),
            VerificationError::InvalidFieldValue(field) => write!(f, "field '{}' has an invalid value", field),
            VerificationError::InvalidSignature => write!(f, "invalid signature"),
            VerificationError::HashMismatch(field) => write!(f, "hash mismatch for field: {}", field),
            VerificationError::InvalidTimestamp => write!(f, "invalid timestamp"),
            VerificationError::UnexpectedEof => write!(f, "unexpected end of input"),
            VerificationError::IntegerOverflow => write!(f, "integer overflow or underflow"),
            VerificationError::InvalidUtf8 => write!(f, "invalid UTF-8 sequence"),
        }
    }
}

impl std::error::Error for VerificationError {}
