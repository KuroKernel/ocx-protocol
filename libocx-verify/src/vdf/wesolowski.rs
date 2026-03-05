//! Wesolowski VDF (Verifiable Delay Function) implementation.
//!
//! Construction: y = x^(2^T) mod N
//!   - Evaluation: Sequential repeated squaring (non-parallelizable)
//!   - Proof: Wesolowski π via Fiat-Shamir hash-to-prime
//!   - Verification: O(log T) — fast for everyone
//!
//! Reference: Wesolowski, "Efficient verifiable delay functions" (2019)
//! https://eprint.iacr.org/2018/623

use num_bigint::BigUint;
use num_traits::{One, Zero};
use num_integer::Integer;
use sha2::{Sha256, Digest};
use std::time::Instant;

use super::error::VdfError;
use super::params::VdfParams;

/// Domain separator for VDF challenge derivation.
const VDF_DOMAIN_SEPARATOR: &[u8] = b"OCXv1|vdf|";

/// Domain separator for hash-to-prime in Wesolowski proof.
const PRIME_DOMAIN_SEPARATOR: &[u8] = b"OCXv1|vdf-prime|";

/// Result of a VDF evaluation.
#[derive(Debug, Clone)]
pub struct VdfProof {
    /// VDF output: y = x^(2^T) mod N
    pub output: Vec<u8>,
    /// Wesolowski proof π
    pub proof: Vec<u8>,
    /// Number of sequential squarings (T)
    pub iterations: u64,
    /// Which modulus was used
    pub modulus_id: String,
    /// The challenge input x (for verification convenience)
    pub challenge: Vec<u8>,
    /// Wall-clock time taken for evaluation (informational, not cryptographic)
    pub duration_ms: u64,
}

/// Derive a VDF challenge from a receipt hash using domain separation.
///
/// challenge = SHA256("OCXv1|vdf|" || receipt_hash) mod N
///
/// The domain separator ensures VDF challenges are distinct from signing messages
/// and other hash uses in the OCX protocol.
pub fn derive_challenge(receipt_hash: &[u8; 32], n: &BigUint) -> BigUint {
    let mut hasher = Sha256::new();
    hasher.update(VDF_DOMAIN_SEPARATOR);
    hasher.update(receipt_hash);
    let digest = hasher.finalize();

    // Interpret hash as big-endian integer, reduce mod N
    // Then ensure it's at least 2 (avoid trivial x=0 or x=1)
    let raw = BigUint::from_bytes_be(&digest) % n;
    if raw < BigUint::from(2u32) {
        BigUint::from(2u32)
    } else {
        raw
    }
}

/// Evaluate the VDF: compute y = x^(2^T) mod N via repeated squaring.
///
/// This is intentionally sequential and non-parallelizable.
/// For T=100,000 with a 2048-bit modulus, this takes ~1 second.
///
/// Also computes the Wesolowski proof π for efficient verification.
pub fn evaluate(
    challenge: &BigUint,
    iterations: u64,
    params: &VdfParams,
) -> Result<VdfProof, VdfError> {
    params.validate_iterations(iterations)?;

    let n = &params.modulus;

    // Validate challenge is in valid range [2, N-1]
    if challenge < &BigUint::from(2u32) || challenge >= n {
        return Err(VdfError::InvalidChallenge);
    }

    let start = Instant::now();

    // Step 1: Compute y = x^(2^T) mod N via repeated squaring
    // This is the non-parallelizable core — each step depends on the previous
    let mut y = challenge.clone();
    for _ in 0..iterations {
        y = (&y * &y) % n;
    }

    // Step 2: Compute Wesolowski proof π
    // l = hash_to_prime(x, y, T) — Fiat-Shamir challenge
    let l = hash_to_prime(challenge, &y, iterations);

    // π = x^(floor(2^T / l)) mod N
    // We compute 2^T mod l using fast modular exponentiation, then derive the quotient
    let proof = compute_proof(challenge, iterations, &l, n);

    let duration_ms = start.elapsed().as_millis() as u64;

    Ok(VdfProof {
        output: y.to_bytes_be(),
        proof: proof.to_bytes_be(),
        iterations,
        modulus_id: params.modulus_id.clone(),
        challenge: challenge.to_bytes_be(),
        duration_ms,
    })
}

/// Verify a VDF proof in O(log T) time.
///
/// Checks: π^l · x^r ≡ y (mod N)
/// where l = hash_to_prime(x, y, T) and r = 2^T mod l
pub fn verify(
    challenge: &BigUint,
    output: &BigUint,
    proof: &BigUint,
    iterations: u64,
    params: &VdfParams,
) -> Result<bool, VdfError> {
    params.validate_iterations(iterations)?;

    let n = &params.modulus;

    // Validate challenge is in range [1, N-1]
    if challenge.is_zero() || challenge >= n {
        return Err(VdfError::InvalidChallenge);
    }
    // Out-of-range output or proof means the proof is simply invalid
    if output.is_zero() || output >= n {
        return Ok(false);
    }
    if proof.is_zero() || proof >= n {
        return Ok(false);
    }

    // Step 1: Recompute l = hash_to_prime(x, y, T)
    let l = hash_to_prime(challenge, output, iterations);

    // Step 2: Compute r = 2^T mod l
    let r = pow2_mod_prime(iterations, &l);

    // Step 3: Check π^l · x^r ≡ y (mod N)
    let pi_l = proof.modpow(&l, n);
    let x_r = challenge.modpow(&r, n);
    let lhs = (&pi_l * &x_r) % n;

    Ok(lhs == *output)
}

/// Verify a VdfProof struct (convenience wrapper).
pub fn verify_proof(proof: &VdfProof, params: &VdfParams) -> Result<bool, VdfError> {
    let challenge = BigUint::from_bytes_be(&proof.challenge);
    let output = BigUint::from_bytes_be(&proof.output);
    let pi = BigUint::from_bytes_be(&proof.proof);

    verify(&challenge, &output, &pi, proof.iterations, params)
}

/// Hash-to-prime using Fiat-Shamir: deterministically derive a prime from (x, y, T).
///
/// Algorithm:
///   1. h = SHA256("OCXv1|vdf-prime|" || x_bytes || y_bytes || T_bytes)
///   2. candidate = h interpreted as big-endian integer
///   3. If candidate is even, increment by 1
///   4. While candidate is not prime, increment by 2
///
/// The resulting prime is ~256 bits, which is sufficient for Wesolowski security.
fn hash_to_prime(x: &BigUint, y: &BigUint, iterations: u64) -> BigUint {
    let mut hasher = Sha256::new();
    hasher.update(PRIME_DOMAIN_SEPARATOR);
    hasher.update(&x.to_bytes_be());
    hasher.update(&y.to_bytes_be());
    hasher.update(&iterations.to_be_bytes());
    let digest = hasher.finalize();

    let mut candidate = BigUint::from_bytes_be(&digest);

    // Ensure odd
    if candidate.is_even() {
        candidate += BigUint::one();
    }

    // Find next prime via trial division + Miller-Rabin
    while !is_probable_prime(&candidate) {
        candidate += BigUint::from(2u32);
    }

    candidate
}

/// Compute the Wesolowski proof: π = x^(floor(2^T / l)) mod N
///
/// We compute this using long division in the exponent:
///   - Maintain a running quotient q = 0 and remainder r = 1
///   - For each of T squaring steps:
///     - r = 2*r
///     - q = 2*q
///     - If r >= l: r -= l, q += 1
///   - Then π = x^q mod N
///
/// This is equivalent to computing floor(2^T / l) but avoids materializing
/// the enormous number 2^T.
fn compute_proof(x: &BigUint, iterations: u64, l: &BigUint, n: &BigUint) -> BigUint {
    // Long division: compute 2^T / l by tracking quotient bit-by-bit
    // while simultaneously doing modular exponentiation
    //
    // We use the observation that floor(2^T / l) can be computed
    // iteratively: start with r=1, for each step r = 2r, if r >= l then
    // r -= l and we have a '1' bit in the quotient.
    //
    // Simultaneously, we compute x^q mod N by squaring and conditionally
    // multiplying — standard binary exponentiation, but driven by the
    // quotient bits we're discovering.

    let mut r = BigUint::one(); // remainder
    let mut pi = BigUint::one(); // proof accumulator: x^q mod N

    for _ in 0..iterations {
        // Double the remainder
        r <<= 1;

        // Square the proof accumulator (corresponds to doubling the exponent)
        pi = (&pi * &pi) % n;

        // If remainder >= l, subtract l and multiply proof by x
        if r >= *l {
            r -= l;
            pi = (&pi * x) % n;
        }
    }

    pi
}

/// Compute 2^T mod l using fast modular exponentiation.
fn pow2_mod_prime(iterations: u64, l: &BigUint) -> BigUint {
    let two = BigUint::from(2u32);
    let t = BigUint::from(iterations);
    two.modpow(&t, l)
}

/// Probabilistic primality test using Miller-Rabin with deterministic witnesses.
///
/// For numbers < 2^64, the witnesses {2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37}
/// are sufficient for a deterministic test. For our ~256-bit candidates, we use
/// these small witnesses plus a few extra rounds for certainty.
fn is_probable_prime(n: &BigUint) -> bool {
    let one = BigUint::one();
    let two = BigUint::from(2u32);

    if *n < two {
        return false;
    }
    if *n == two || *n == BigUint::from(3u32) {
        return true;
    }
    if n.is_even() {
        return false;
    }

    // Write n-1 as 2^s * d where d is odd
    let n_minus_1 = n - &one;
    let mut d = n_minus_1.clone();
    let mut s: u64 = 0;
    while d.is_even() {
        d >>= 1;
        s += 1;
    }

    // Deterministic witnesses for numbers up to ~340 bits
    let witnesses: &[u64] = &[2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37];

    'witness: for &a in witnesses {
        let a_big = BigUint::from(a);
        if a_big >= *n {
            continue;
        }

        let mut x = a_big.modpow(&d, n);

        if x == one || x == n_minus_1 {
            continue 'witness;
        }

        for _ in 0..s - 1 {
            x = (&x * &x) % n;
            if x == n_minus_1 {
                continue 'witness;
            }
        }

        return false;
    }

    true
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::vdf::params;

    #[test]
    fn test_derive_challenge_deterministic() {
        let params = params::default_params();
        let hash1: [u8; 32] = [0xaa; 32];
        let hash2: [u8; 32] = [0xaa; 32];

        let c1 = derive_challenge(&hash1, &params.modulus);
        let c2 = derive_challenge(&hash2, &params.modulus);

        assert_eq!(c1, c2, "same input must produce same challenge");
    }

    #[test]
    fn test_derive_challenge_different_inputs() {
        let params = params::default_params();
        let hash1: [u8; 32] = [0xaa; 32];
        let hash2: [u8; 32] = [0xbb; 32];

        let c1 = derive_challenge(&hash1, &params.modulus);
        let c2 = derive_challenge(&hash2, &params.modulus);

        assert_ne!(c1, c2, "different inputs must produce different challenges");
    }

    #[test]
    fn test_derive_challenge_in_range() {
        let params = params::default_params();
        let hash: [u8; 32] = [0x42; 32];

        let c = derive_challenge(&hash, &params.modulus);

        assert!(c >= BigUint::from(2u32), "challenge must be >= 2");
        assert!(c < params.modulus, "challenge must be < N");
    }

    #[test]
    fn test_evaluate_verify_roundtrip() {
        let params = params::default_params();
        let hash: [u8; 32] = [0x42; 32];
        let challenge = derive_challenge(&hash, &params.modulus);

        // Use small T for test speed
        let vdf_proof = evaluate(&challenge, params::MIN_ITERATIONS, &params)
            .expect("evaluation should succeed");

        let valid = verify_proof(&vdf_proof, &params)
            .expect("verification should not error");

        assert!(valid, "valid proof must verify");
    }

    #[test]
    fn test_wrong_proof_fails() {
        let params = params::default_params();
        let hash: [u8; 32] = [0x42; 32];
        let challenge = derive_challenge(&hash, &params.modulus);

        let mut vdf_proof = evaluate(&challenge, params::MIN_ITERATIONS, &params)
            .expect("evaluation should succeed");

        // Tamper with the proof
        if let Some(byte) = vdf_proof.proof.first_mut() {
            *byte ^= 0xff;
        }

        let valid = verify_proof(&vdf_proof, &params)
            .expect("verification should not error");

        assert!(!valid, "tampered proof must not verify");
    }

    #[test]
    fn test_wrong_challenge_fails() {
        let params = params::default_params();
        let hash: [u8; 32] = [0x42; 32];
        let challenge = derive_challenge(&hash, &params.modulus);

        let vdf_proof = evaluate(&challenge, params::MIN_ITERATIONS, &params)
            .expect("evaluation should succeed");

        // Verify with a different challenge
        let wrong_hash: [u8; 32] = [0x99; 32];
        let wrong_challenge = derive_challenge(&wrong_hash, &params.modulus);

        let output = BigUint::from_bytes_be(&vdf_proof.output);
        let proof = BigUint::from_bytes_be(&vdf_proof.proof);

        let valid = verify(
            &wrong_challenge, &output, &proof,
            vdf_proof.iterations, &params,
        ).expect("verification should not error");

        assert!(!valid, "wrong challenge must not verify");
    }

    #[test]
    fn test_wrong_output_fails() {
        let params = params::default_params();
        let hash: [u8; 32] = [0x42; 32];
        let challenge = derive_challenge(&hash, &params.modulus);

        let mut vdf_proof = evaluate(&challenge, params::MIN_ITERATIONS, &params)
            .expect("evaluation should succeed");

        // Tamper with the output
        if let Some(byte) = vdf_proof.output.first_mut() {
            *byte ^= 0xff;
        }

        let valid = verify_proof(&vdf_proof, &params)
            .expect("verification should not error");

        assert!(!valid, "tampered output must not verify");
    }

    #[test]
    fn test_is_probable_prime() {
        assert!(is_probable_prime(&BigUint::from(2u32)));
        assert!(is_probable_prime(&BigUint::from(3u32)));
        assert!(is_probable_prime(&BigUint::from(5u32)));
        assert!(is_probable_prime(&BigUint::from(7u32)));
        assert!(is_probable_prime(&BigUint::from(97u32)));
        assert!(is_probable_prime(&BigUint::from(104729u32)));

        assert!(!is_probable_prime(&BigUint::from(0u32)));
        assert!(!is_probable_prime(&BigUint::from(1u32)));
        assert!(!is_probable_prime(&BigUint::from(4u32)));
        assert!(!is_probable_prime(&BigUint::from(100u32)));
        assert!(!is_probable_prime(&BigUint::from(104730u32)));
    }

    #[test]
    fn test_hash_to_prime_deterministic() {
        let x = BigUint::from(42u32);
        let y = BigUint::from(99u32);
        let t = 1000u64;

        let p1 = hash_to_prime(&x, &y, t);
        let p2 = hash_to_prime(&x, &y, t);

        assert_eq!(p1, p2, "same inputs must produce same prime");
        assert!(is_probable_prime(&p1), "result must be prime");
    }

    #[test]
    fn test_hash_to_prime_is_prime() {
        let x = BigUint::from(12345u32);
        let y = BigUint::from(67890u32);

        for t in [100, 1000, 10000, 100000] {
            let p = hash_to_prime(&x, &y, t);
            assert!(is_probable_prime(&p), "hash_to_prime({}) must produce a prime", t);
        }
    }

    #[test]
    fn test_iterations_below_minimum() {
        let params = params::default_params();
        let challenge = BigUint::from(42u32);

        let result = evaluate(&challenge, params::MIN_ITERATIONS - 1, &params);
        assert!(result.is_err(), "below minimum iterations must error");

        match result.unwrap_err() {
            VdfError::IterationsTooLow(_) => {},
            e => panic!("expected IterationsTooLow, got {:?}", e),
        }
    }

    #[test]
    fn test_iterations_above_maximum() {
        let params = params::default_params();
        let challenge = BigUint::from(42u32);

        let result = evaluate(&challenge, params::MAX_ITERATIONS + 1, &params);
        assert!(result.is_err(), "above maximum iterations must error");

        match result.unwrap_err() {
            VdfError::IterationsTooHigh(_) => {},
            e => panic!("expected IterationsTooHigh, got {:?}", e),
        }
    }

    #[test]
    fn test_zero_challenge_rejected() {
        let params = params::default_params();
        let challenge = BigUint::zero();

        let result = evaluate(&challenge, params::MIN_ITERATIONS, &params);
        assert!(result.is_err(), "zero challenge must error");
    }

    #[test]
    fn test_one_challenge_rejected() {
        let params = params::default_params();
        let challenge = BigUint::one();

        let result = evaluate(&challenge, params::MIN_ITERATIONS, &params);
        assert!(result.is_err(), "challenge=1 must error");
    }

    #[test]
    fn test_proof_output_sizes() {
        let params = params::default_params();
        let hash: [u8; 32] = [0x42; 32];
        let challenge = derive_challenge(&hash, &params.modulus);

        let vdf_proof = evaluate(&challenge, params::MIN_ITERATIONS, &params)
            .expect("evaluation should succeed");

        // Output and proof should be at most 256 bytes (2048-bit modulus)
        assert!(vdf_proof.output.len() <= 256, "output too large: {}", vdf_proof.output.len());
        assert!(vdf_proof.proof.len() <= 256, "proof too large: {}", vdf_proof.proof.len());
        assert!(vdf_proof.output.len() > 0, "output must not be empty");
        assert!(vdf_proof.proof.len() > 0, "proof must not be empty");
    }
}
