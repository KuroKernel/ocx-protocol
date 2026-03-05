#ifndef OCX_VERIFY_H
#define OCX_VERIFY_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/// Error codes for OCX verification operations.
typedef enum {
    OCX_SUCCESS = 0,
    OCX_INVALID_CBOR = 1,
    OCX_NON_CANONICAL_CBOR = 2,
    OCX_MISSING_FIELD = 3,
    OCX_INVALID_FIELD_VALUE = 4,
    OCX_INVALID_SIGNATURE = 5,
    OCX_HASH_MISMATCH = 6,
    OCX_INVALID_TIMESTAMP = 7,
    OCX_UNEXPECTED_EOF = 8,
    OCX_INTEGER_OVERFLOW = 9,
    OCX_INVALID_UTF8 = 10,
    OCX_INVALID_INPUT = 11,
    OCX_INTERNAL_ERROR = 12,
} OcxErrorCode;

/// Receipt fields structure for C applications.
typedef struct {
    uint8_t artifact_hash[32];
    uint8_t input_hash[32];
    uint8_t output_hash[32];
    uint64_t cycles_used;
    uint64_t started_at;
    uint64_t finished_at;
    size_t issuer_key_id_len;
    size_t signature_len;
} OcxReceiptFields;

/// Receipt data structure for batch processing.
typedef struct {
    const uint8_t* cbor_data;
    size_t cbor_data_len;
    const uint8_t* public_key;
} OcxReceiptData;

/// Simple verification function.
/// Returns true if verification succeeds, false otherwise.
bool ocx_verify_receipt(
    const uint8_t* cbor_data,
    size_t cbor_data_len,
    const uint8_t* public_key
);

/// Detailed verification function with error reporting.
/// Returns true if verification succeeds, false otherwise.
/// Error code is written to error_code parameter.
bool ocx_verify_receipt_detailed(
    const uint8_t* cbor_data,
    size_t cbor_data_len,
    const uint8_t* public_key,
    OcxErrorCode* error_code
);

/// Simple verification using embedded key ID.
/// Returns true if verification succeeds, false otherwise.
bool ocx_verify_receipt_simple(
    const uint8_t* cbor_data,
    size_t cbor_data_len
);

/// Extract receipt fields into C-compatible structure.
/// Returns error code (OCX_SUCCESS on success).
OcxErrorCode ocx_extract_receipt_fields(
    const uint8_t* cbor_data,
    size_t cbor_data_len,
    OcxReceiptFields* fields,
    char* issuer_key_id,
    size_t issuer_key_id_max_len,
    uint8_t* signature,
    size_t signature_max_len
);

/// Get error message for an error code.
/// Returns number of bytes written (including null terminator).
size_t ocx_get_error_message(
    OcxErrorCode error_code,
    char* buffer,
    size_t buffer_len
);

/// Get library version string.
/// Returns number of bytes written (including null terminator).
size_t ocx_get_version(
    char* buffer,
    size_t buffer_len
);

/// Batch verification function.
/// Returns number of successfully verified receipts.
size_t ocx_verify_receipts_batch(
    const OcxReceiptData* receipts,
    size_t receipt_count,
    bool* results
);

// ── VDF (Verifiable Delay Function) ─────────────────────────────────────

/// VDF proof structure for temporal proofs.
typedef struct {
    uint8_t output[256];        // VDF output y = x^(2^T) mod N (big-endian, right-aligned)
    uint32_t output_len;        // Actual length of output bytes
    uint8_t proof[256];         // Wesolowski proof π (big-endian, right-aligned)
    uint32_t proof_len;         // Actual length of proof bytes
    uint64_t iterations;        // Number of sequential squarings T
    uint8_t modulus_id[64];     // Null-terminated ASCII modulus identifier
    uint64_t duration_ms;       // Wall-clock time taken in milliseconds
} OcxVdfProof;

/// Evaluate VDF: compute temporal proof for a 32-byte receipt hash.
/// Returns error code (OCX_SUCCESS on success).
OcxErrorCode ocx_vdf_evaluate(
    const uint8_t* receipt_hash,    // 32-byte SHA-256 hash
    uint64_t iterations,            // Number of sequential squarings T
    OcxVdfProof* out_proof          // Output proof struct
);

/// Verify a VDF proof against a 32-byte receipt hash.
/// Returns true if the VDF proof is valid, false otherwise.
bool ocx_vdf_verify(
    const uint8_t* receipt_hash,    // 32-byte SHA-256 hash
    const OcxVdfProof* proof        // Proof struct to verify
);

#ifdef __cplusplus
}
#endif

#endif // OCX_VERIFY_H