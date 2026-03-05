//! VDF parameters: RSA modulus presets and configuration.
//!
//! Security note: The RSA modulus N must have unknown factorization for VDF
//! security. We use the RSA-2048 challenge number from the RSA Factoring
//! Challenge, which is publicly known but has never been factored.
//! Nobody — including us — knows p and q, which is precisely the requirement.

use num_bigint::BigUint;
use super::error::VdfError;

/// Modulus identifier for the first OCX VDF modulus.
pub const OCX_VDF_MODULUS_ID_V1: &str = "ocx-vdf-v1";

/// RSA-2048 challenge number (decimal).
/// Source: RSA Factoring Challenge (RSA Laboratories, 1991).
/// Status: Unfactored as of 2025. 617 decimal digits, 2048 bits.
///
/// This number is the product of two unknown primes p and q, each ~1024 bits.
/// The factorization was never published and the challenge was withdrawn in 2007
/// without being solved. Using it ensures nobody can cheat the VDF.
const RSA_2048_DECIMAL: &str = "\
2519590847565789349402718324004839857142928212620403202777713783604366202070\
7595556264018525880784406918290641249515082189298559149176184502808489120072\
8449926873928072877767359714183472702618963750149718246911650776133798590957\
0009733045974880842840179742910064245869181719511874612151517265463228221686\
9987549182422433637259085141865462043576798423387184774447920739934236584823\
8242811981638150106748104516603773060562016196762561338441436038339044149526\
3443219011465754445417842402092461651572335077870774981712577246796292638635\
6373289912154831438167899885040445364023527381951378636564391212010397122822\
120720357\
";

/// Minimum allowed iterations for VDF computation.
pub const MIN_ITERATIONS: u64 = 1_000;

/// Maximum allowed iterations for VDF computation.
pub const MAX_ITERATIONS: u64 = 100_000_000;

/// Default iterations (~1 second on modern hardware).
pub const DEFAULT_ITERATIONS: u64 = 100_000;

/// VDF parameter set.
#[derive(Debug, Clone)]
pub struct VdfParams {
    /// The RSA modulus N = p * q.
    pub modulus: BigUint,
    /// Identifier for this modulus (for versioning/rotation).
    pub modulus_id: String,
    /// Minimum iterations floor.
    pub min_iterations: u64,
    /// Maximum iterations ceiling.
    pub max_iterations: u64,
}

impl VdfParams {
    /// Validate that iterations count is within bounds.
    pub fn validate_iterations(&self, iterations: u64) -> Result<(), VdfError> {
        if iterations < self.min_iterations {
            return Err(VdfError::IterationsTooLow(iterations));
        }
        if iterations > self.max_iterations {
            return Err(VdfError::IterationsTooHigh(iterations));
        }
        Ok(())
    }
}

/// Returns the default VDF parameters using the RSA-2048 challenge modulus.
pub fn default_params() -> VdfParams {
    let modulus = RSA_2048_DECIMAL
        .parse::<BigUint>()
        .expect("RSA-2048 constant must parse");

    VdfParams {
        modulus,
        modulus_id: OCX_VDF_MODULUS_ID_V1.to_string(),
        min_iterations: MIN_ITERATIONS,
        max_iterations: MAX_ITERATIONS,
    }
}

/// Lookup VDF parameters by modulus ID.
pub fn get_params(modulus_id: &str) -> Result<VdfParams, VdfError> {
    match modulus_id {
        OCX_VDF_MODULUS_ID_V1 => Ok(default_params()),
        _ => Err(VdfError::UnknownModulus(modulus_id.to_string())),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_rsa_2048_parses() {
        let params = default_params();
        // RSA-2048 is 2048 bits = 256 bytes
        let bytes = params.modulus.to_bytes_be();
        assert_eq!(bytes.len(), 256, "RSA-2048 must be exactly 256 bytes");
    }

    #[test]
    fn test_modulus_is_odd() {
        let params = default_params();
        // RSA modulus must be odd (product of two odd primes)
        assert_eq!(&params.modulus % 2u32, BigUint::from(1u32));
    }

    #[test]
    fn test_validate_iterations() {
        let params = default_params();
        assert!(params.validate_iterations(MIN_ITERATIONS).is_ok());
        assert!(params.validate_iterations(DEFAULT_ITERATIONS).is_ok());
        assert!(params.validate_iterations(MAX_ITERATIONS).is_ok());
        assert!(params.validate_iterations(MIN_ITERATIONS - 1).is_err());
        assert!(params.validate_iterations(MAX_ITERATIONS + 1).is_err());
    }

    #[test]
    fn test_get_params_known() {
        assert!(get_params(OCX_VDF_MODULUS_ID_V1).is_ok());
    }

    #[test]
    fn test_get_params_unknown() {
        assert!(get_params("unknown-modulus").is_err());
    }
}
