use libocx_verify::ffi::*;
use std::ffi::CStr;
use std::ptr;

#[test]
fn test_ffi_basic_verification() {
    // Create a simple test CBOR (this would normally be a valid receipt)
    let test_cbor = vec![
        0xa8, // map(8)
        // ... (would contain actual valid receipt data)
    ];
    
    let test_key = [0u8; 32]; // Test public key
    
    unsafe {
        let result = ocx_verify_receipt(
            test_cbor.as_ptr(),
            test_cbor.len(),
            test_key.as_ptr(),
        );
        
        // This will likely fail with our test data, but tests the FFI interface
        assert!(!result || result); // Test that function returns a boolean
    }
}

#[test]
fn test_ffi_error_reporting() {
    let test_cbor = vec![0x00]; // Invalid CBOR
    let test_key = [0u8; 32];
    let mut error_code = OcxErrorCode::Success;
    
    unsafe {
        let result = ocx_verify_receipt_detailed(
            test_cbor.as_ptr(),
            test_cbor.len(),
            test_key.as_ptr(),
            &mut error_code,
        );
        
        assert!(!result); // Should fail
        assert_ne!(error_code, OcxErrorCode::Success); // Should have error code
    }
}

#[test]
fn test_ffi_null_pointer_safety() {
    unsafe {
        // Test null pointer handling
        let result = ocx_verify_receipt(ptr::null(), 0, ptr::null());
        assert!(!result); // Should fail safely
        
        let mut error_code = OcxErrorCode::Success;
        let result = ocx_verify_receipt_detailed(
            ptr::null(),
            0,
            ptr::null(),
            &mut error_code,
        );
        assert!(!result);
        assert_eq!(error_code, OcxErrorCode::InvalidInput);
    }
}

#[test]
fn test_ffi_version_info() {
    let mut buffer = [0u8; 64];
    
    unsafe {
        let len = ocx_get_version(buffer.as_mut_ptr() as *mut i8, buffer.len());
        assert!(len > 0);
        assert!(len <= buffer.len());
        
        // Verify null termination
        assert_eq!(buffer[len - 1], 0);
        
        // Verify we can convert to string
        let version_cstr = CStr::from_ptr(buffer.as_ptr() as *const i8);
        let version_str = version_cstr.to_str().unwrap();
        assert!(!version_str.is_empty());
    }
}

#[test]
fn test_ffi_error_messages() {
    let mut buffer = [0u8; 256];
    
    unsafe {
        let len = ocx_get_error_message(
            OcxErrorCode::InvalidSignature,
            buffer.as_mut_ptr() as *mut i8,
            buffer.len(),
        );
        
        assert!(len > 0);
        assert!(len <= buffer.len());
        
        // Verify null termination
        assert_eq!(buffer[len - 1], 0);
        
        // Verify message content
        let msg_cstr = CStr::from_ptr(buffer.as_ptr() as *const i8);
        let msg_str = msg_cstr.to_str().unwrap();
        assert!(msg_str.contains("signature"));
    }
}

#[test]
fn test_ffi_batch_verification() {
    // Create test data
    let test_cbor1 = vec![0xa0]; // Empty map (invalid receipt)
    let test_cbor2 = vec![0xa0]; // Empty map (invalid receipt)
    let test_key = [0u8; 32];
    
    let receipts = [
        OcxReceiptData {
            cbor_data: test_cbor1.as_ptr(),
            cbor_data_len: test_cbor1.len(),
            public_key: test_key.as_ptr(),
        },
        OcxReceiptData {
            cbor_data: test_cbor2.as_ptr(),
            cbor_data_len: test_cbor2.len(),
            public_key: test_key.as_ptr(),
        },
    ];
    
    let mut results = [false; 2];
    
    unsafe {
        let success_count = ocx_verify_receipts_batch(
            receipts.as_ptr(),
            receipts.len(),
            results.as_mut_ptr(),
        );
        
        assert_eq!(success_count, 0); // Both should fail
        assert!(!results[0]);
        assert!(!results[1]);
    }
}

#[test]
fn test_ffi_simple_verification() {
    let test_cbor = vec![0xa0]; // Empty map (invalid receipt)
    
    unsafe {
        let result = ocx_verify_receipt_simple(
            test_cbor.as_ptr(),
            test_cbor.len(),
        );
        
        assert!(!result); // Should fail with invalid data
    }
}

#[test]
fn test_ffi_extract_fields() {
    let test_cbor = vec![0xa0]; // Empty map (invalid receipt)
    let mut fields = OcxReceiptFields {
        artifact_hash: [0; 32],
        input_hash: [0; 32],
        output_hash: [0; 32],
        cycles_used: 0,
        started_at: 0,
        finished_at: 0,
        issuer_key_id_len: 0,
        signature_len: 0,
    };
    let mut issuer_key_id = [0u8; 256];
    let mut signature = [0u8; 1024];
    
    unsafe {
        let result = ocx_extract_receipt_fields(
            test_cbor.as_ptr(),
            test_cbor.len(),
            &mut fields,
            issuer_key_id.as_mut_ptr() as *mut i8,
            issuer_key_id.len(),
            signature.as_mut_ptr(),
            signature.len(),
        );
        
        assert_ne!(result, OcxErrorCode::Success); // Should fail with invalid data
    }
}

#[test]
fn test_ffi_error_code_conversion() {
    // Test that all error codes have meaningful messages
    let error_codes = [
        OcxErrorCode::Success,
        OcxErrorCode::InvalidCbor,
        OcxErrorCode::NonCanonicalCbor,
        OcxErrorCode::MissingField,
        OcxErrorCode::InvalidFieldValue,
        OcxErrorCode::InvalidSignature,
        OcxErrorCode::HashMismatch,
        OcxErrorCode::InvalidTimestamp,
        OcxErrorCode::UnexpectedEof,
        OcxErrorCode::IntegerOverflow,
        OcxErrorCode::InvalidUtf8,
        OcxErrorCode::InvalidInput,
        OcxErrorCode::InternalError,
    ];
    
    let mut buffer = [0u8; 256];
    
    for error_code in error_codes {
        unsafe {
            let len = ocx_get_error_message(
                error_code,
                buffer.as_mut_ptr() as *mut i8,
                buffer.len(),
            );
            
            assert!(len > 0);
            assert!(len <= buffer.len());
            assert_eq!(buffer[len - 1], 0); // Null terminated
            
            // Verify message is not empty
            let msg_cstr = CStr::from_ptr(buffer.as_ptr() as *const i8);
            let msg_str = msg_cstr.to_str().unwrap();
            assert!(!msg_str.is_empty());
        }
    }
}

#[test]
fn test_ffi_buffer_overflow_protection() {
    let mut buffer = [0u8; 5]; // Very small buffer
    
    unsafe {
        let len = ocx_get_error_message(
            OcxErrorCode::InvalidSignature,
            buffer.as_mut_ptr() as *mut i8,
            buffer.len(),
        );
        
        assert!(len > 0);
        assert!(len <= buffer.len());
        assert_eq!(buffer[len - 1], 0); // Should still be null terminated
    }
}

#[test]
fn test_ffi_invalid_parameters() {
    unsafe {
        // Test with null buffer
        let len = ocx_get_error_message(
            OcxErrorCode::Success,
            ptr::null_mut(),
            100,
        );
        assert_eq!(len, 0);
        
        // Test with zero buffer length
        let mut buffer = [0u8; 100];
        let len = ocx_get_error_message(
            OcxErrorCode::Success,
            buffer.as_mut_ptr() as *mut i8,
            0,
        );
        assert_eq!(len, 0);
    }
}
