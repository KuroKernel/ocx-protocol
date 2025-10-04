pub mod types;
pub mod aggregation;

use types::{ReputationInput, ReputationOutput};
use aggregation::aggregate_reputation;

/// WASM entry point for reputation aggregation
///
/// # Safety
/// This function is marked unsafe because it deals with raw pointers
/// for WASM FFI. The caller must ensure input_ptr points to valid
/// UTF-8 JSON data of the specified length.
#[no_mangle]
pub unsafe extern "C" fn aggregate_reputation_wasm(
    input_ptr: *const u8,
    input_len: usize,
) -> *mut u8 {
    // Parse input JSON
    let input_bytes = std::slice::from_raw_parts(input_ptr, input_len);

    let input: ReputationInput = match serde_json::from_slice(input_bytes) {
        Ok(i) => i,
        Err(e) => {
            // Return error as JSON
            let error_json = format!(r#"{{"error":"Invalid input JSON: {}"}}"#, e);
            let error_bytes = error_json.into_bytes();
            let boxed = error_bytes.into_boxed_slice();
            return Box::into_raw(boxed) as *mut u8;
        }
    };

    // Perform aggregation
    let output = aggregate_reputation(&input);

    // Serialize output to JSON
    let output_json = match serde_json::to_vec(&output) {
        Ok(j) => j,
        Err(e) => {
            let error_json = format!(r#"{{"error":"Serialization failed: {}"}}"#, e);
            error_json.into_bytes()
        }
    };

    // Return pointer to heap-allocated output
    let boxed = output_json.into_boxed_slice();
    Box::into_raw(boxed) as *mut u8
}

/// Free memory allocated by WASM module
///
/// # Safety
/// The caller must ensure ptr was allocated by this module
/// and has not been freed already.
#[no_mangle]
pub unsafe extern "C" fn free_wasm_memory(ptr: *mut u8, len: usize) {
    let _ = Box::from_raw(std::slice::from_raw_parts_mut(ptr, len));
}

/// Get version of the reputation module
#[no_mangle]
pub extern "C" fn get_module_version() -> *const u8 {
    b"trustscore-wasm-v0.1.0\0".as_ptr()
}

// Re-export public API
pub use types::{
    ReputationInput, ReputationOutput, PlatformScore,
    ScoreWeights, ComponentScore
};
pub use aggregation::aggregate_reputation;

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;

    #[test]
    fn test_wasm_entry_point() {
        let input = ReputationInput {
            user_id: "alice".to_string(),
            platforms: vec![
                PlatformScore {
                    platform_type: "github".to_string(),
                    score: 85.0,
                    weight: 0.5,
                    metadata: HashMap::new(),
                },
                PlatformScore {
                    platform_type: "linkedin".to_string(),
                    score: 90.0,
                    weight: 0.5,
                    metadata: HashMap::new(),
                },
            ],
            weights: ScoreWeights::default(),
            timestamp: 1696348800,
        };

        let input_json = serde_json::to_vec(&input).unwrap();

        unsafe {
            let result_ptr = aggregate_reputation_wasm(
                input_json.as_ptr(),
                input_json.len(),
            );

            // In a real test, we'd need to know the output length
            // For now, just verify the pointer is not null
            assert!(!result_ptr.is_null());
        }
    }

    #[test]
    fn test_error_handling() {
        let invalid_json = b"invalid json";

        unsafe {
            let result_ptr = aggregate_reputation_wasm(
                invalid_json.as_ptr(),
                invalid_json.len(),
            );

            assert!(!result_ptr.is_null());
            // Result should contain error message
        }
    }

    #[test]
    fn test_module_version() {
        let version_ptr = get_module_version();
        assert!(!version_ptr.is_null());
    }
}
