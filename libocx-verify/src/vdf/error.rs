//! VDF-specific error types.

use std::fmt;

/// All possible errors during VDF computation or verification.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum VdfError {
    /// The RSA modulus is invalid (too small, even, etc.).
    InvalidModulus,
    /// The VDF proof failed verification.
    InvalidProof,
    /// Iterations count is below the minimum threshold.
    IterationsTooLow(u64),
    /// Iterations count exceeds the maximum threshold.
    IterationsTooHigh(u64),
    /// VDF computation failed.
    ComputationFailed(String),
    /// VDF verification failed.
    VerificationFailed(String),
    /// The challenge input is invalid.
    InvalidChallenge,
    /// Unknown modulus ID — not in the registry.
    UnknownModulus(String),
}

impl fmt::Display for VdfError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            VdfError::InvalidModulus => write!(f, "invalid RSA modulus"),
            VdfError::InvalidProof => write!(f, "VDF proof verification failed"),
            VdfError::IterationsTooLow(t) => write!(f, "iterations {} below minimum", t),
            VdfError::IterationsTooHigh(t) => write!(f, "iterations {} above maximum", t),
            VdfError::ComputationFailed(msg) => write!(f, "VDF computation failed: {}", msg),
            VdfError::VerificationFailed(msg) => write!(f, "VDF verification failed: {}", msg),
            VdfError::InvalidChallenge => write!(f, "invalid VDF challenge input"),
            VdfError::UnknownModulus(id) => write!(f, "unknown modulus ID: {}", id),
        }
    }
}

impl std::error::Error for VdfError {}
