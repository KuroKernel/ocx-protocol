#ifndef OCX_VERIFY_H
#define OCX_VERIFY_H

#include <stdint.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

/// Verifies an OCX receipt in canonical CBOR format.
///
/// @param receipt_ptr Pointer to the receipt data
/// @param receipt_len Length of the receipt data in bytes
/// @param result Pointer to store the verification result (1 = valid, 0 = invalid)
/// @return 0 on success, non-zero error code on failure
int ocx_verify(const uint8_t* receipt_ptr, size_t receipt_len, int* result);

#ifdef __cplusplus
}
#endif

#endif // OCX_VERIFY_H
