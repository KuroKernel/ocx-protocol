//! Verifiable Delay Function (VDF) module for OCX temporal proofs.
//!
//! This module implements the Wesolowski VDF construction for proving that
//! a minimum amount of real wall-clock time elapsed between events.
//!
//! # Architecture
//!
//! - `params` — RSA modulus presets and configuration
//! - `wesolowski` — Core VDF evaluate/verify algorithms
//! - `error` — VDF-specific error types
//!
//! # Security Model
//!
//! The VDF uses an RSA-2048 modulus with unknown factorization (from the RSA
//! Factoring Challenge). Nobody — including the system operators — can compute
//! the VDF faster than sequential squaring.
//!
//! # Usage
//!
//! ```rust,ignore
//! use libocx_verify::vdf::{params, wesolowski};
//!
//! let params = params::default_params();
//! let receipt_hash: [u8; 32] = /* ... */;
//!
//! // Derive challenge from receipt hash
//! let challenge = wesolowski::derive_challenge(&receipt_hash, &params.modulus);
//!
//! // Evaluate (slow — ~1s for T=100,000)
//! let proof = wesolowski::evaluate(&challenge, 100_000, &params)?;
//!
//! // Verify (fast — <10ms)
//! let valid = wesolowski::verify_proof(&proof, &params)?;
//! assert!(valid);
//! ```

pub mod error;
pub mod params;
pub mod wesolowski;

// Re-exports for convenience
pub use error::VdfError;
pub use params::{VdfParams, default_params, get_params};
pub use wesolowski::{VdfProof, derive_challenge, evaluate, verify, verify_proof};
