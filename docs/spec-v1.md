# OCX Protocol v1.0.0-rc.1 Specification

## Overview
This document defines the frozen specification for OCX Protocol v1.0.0-rc.1, establishing immutable API surfaces, determinism rules, and canonical data formats.

## 1. API Surface (IMMUTABLE)

### Core Functions
- `OCX_EXEC(artifact, input, max_cycles) -> OCXResult`
- `OCX_VERIFY(receipt_blob) -> VerificationResult`  
- `OCX_ACCOUNT(account_id) -> AccountInfo`

### OCXResult Structure
```go
type OCXResult struct {
    OutputHash  [32]byte `json:"output_hash"`
    CyclesUsed  uint64   `json:"cycles_used"`
    ReceiptHash [32]byte `json:"receipt_hash"`
    ReceiptBlob []byte   `json:"receipt_blob"`
}
```

### VerificationResult Structure
```go
type VerificationResult struct {
    Valid   bool   `json:"valid"`
    Reason  string `json:"reason,omitempty"`
}
```

### AccountInfo Structure
```go
type AccountInfo struct {
    AccountID    string `json:"account_id"`
    Balance      int64  `json:"balance"`
    CycleCount   uint64 `json:"cycle_count"`
    LastUpdated  string `json:"last_updated"`
}
```

## 2. Determinism Rules (IMMUTABLE)

### Execution Determinism
- **No System Calls**: Execution must not invoke any system calls
- **Fixed Byte Order**: All data must use little-endian byte order
- **Cycle-Accurate Metering**: Every instruction must consume exactly 1 cycle
- **Deterministic Randomness**: Use seeded PRNG with fixed seed from input hash
- **No External Dependencies**: Execution must not depend on external services

### Memory Layout
- **Fixed Stack Size**: 64KB maximum stack size
- **Fixed Heap Size**: 1MB maximum heap size
- **No Dynamic Allocation**: All memory must be pre-allocated
- **Deterministic Garbage Collection**: No GC during execution

### Instruction Set
- **Fixed Opcodes**: 256 instruction opcodes (0x00-0xFF)
- **No Undefined Instructions**: All opcodes must be defined
- **Deterministic Execution**: Same input always produces same output
- **No Side Effects**: Instructions must not modify external state

## 3. Canonical CBOR Receipt Format (IMMUTABLE)

### Receipt Structure
```go
type OCXReceipt struct {
    Version    uint8     `cbor:"1,keyasint"`
    Artifact   [32]byte  `cbor:"2,keyasint"`
    Input      [32]byte  `cbor:"3,keyasint"`
    Output     [32]byte  `cbor:"4,keyasint"`
    Cycles     uint64    `cbor:"5,keyasint"`
    Metering   Metering  `cbor:"6,keyasint"`
    Transcript [32]byte  `cbor:"7,keyasint"`
    Issuer     [32]byte  `cbor:"8,keyasint"`
    Signature  [64]byte  `cbor:"9,keyasint"`
}
```

### Metering Structure
```go
type Metering struct {
    Alpha uint64 `cbor:"1,keyasint"`
    Beta  uint64 `cbor:"2,keyasint"`
    Gamma uint64 `cbor:"3,keyasint"`
}
```

### CBOR Encoding Rules
- **Canonical Encoding**: Use CTAP2 canonical encoding
- **Sorted Keys**: All map keys must be sorted numerically
- **Deterministic Timestamps**: Use Unix timestamps (seconds since epoch)
- **Fixed Field Order**: Fields must appear in numerical order
- **No Optional Fields**: All fields are required

## 4. Pricing Formula (IMMUTABLE)

### Constants (FROZEN)
```go
const (
    ALPHA_COST_PER_CYCLE = 10  // Micro-units per cycle
    BETA_COST_PER_CYCLE  = 1   // Micro-units per cycle  
    GAMMA_COST_PER_CYCLE = 1   // Micro-units per cycle
)
```

### Pricing Calculation
```go
func CalculatePrice(cycles uint64) int64 {
    return int64(cycles * (ALPHA_COST_PER_CYCLE + BETA_COST_PER_CYCLE + GAMMA_COST_PER_CYCLE))
}
```

### Revenue Distribution
- **Alpha (70%)**: Core execution costs
- **Beta (20%)**: Infrastructure overhead
- **Gamma (10%)**: Protocol maintenance

## 5. Receipt Verification (IMMUTABLE)

### Verification Process
1. **Deserialize CBOR**: Parse receipt from canonical CBOR format
2. **Validate Structure**: Check all required fields are present
3. **Verify Signature**: Validate Ed25519 signature using issuer public key
4. **Check Metering**: Verify metering constants match frozen values
5. **Validate Cycles**: Ensure cycles > 0 and within limits
6. **Verify Hashes**: Validate artifact, input, output, and transcript hashes

### Signature Verification
- **Algorithm**: Ed25519
- **Public Key**: 32-byte issuer public key
- **Signature**: 64-byte Ed25519 signature
- **Message**: SHA-256 hash of all receipt fields except signature

## 6. Error Handling (IMMUTABLE)

### Error Codes
- **E001**: Invalid artifact format
- **E002**: Invalid input format
- **E003**: Cycle limit exceeded
- **E004**: Invalid receipt format
- **E005**: Signature verification failed
- **E006**: Metering validation failed

### Error Response Format
```go
type ErrorResponse struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}
```

## 7. Versioning (IMMUTABLE)

### Version Format
- **Major.Minor.Patch-PreRelease**
- **v1.0.0-rc.1**: First release candidate
- **v1.0.0**: Final release
- **v1.1.0**: Backward-compatible updates

### Compatibility Rules
- **Major Version**: Breaking changes require new major version
- **Minor Version**: New features, backward-compatible
- **Patch Version**: Bug fixes, backward-compatible
- **PreRelease**: Development/testing versions

## 8. Security Requirements (IMMUTABLE)

### Cryptographic Requirements
- **Hash Algorithm**: SHA-256 for all hashing operations
- **Signature Algorithm**: Ed25519 for all signatures
- **Random Number Generation**: Cryptographically secure PRNG
- **Key Management**: Secure key generation and storage

### Input Validation
- **Artifact Validation**: Must be valid bytecode
- **Input Validation**: Must be valid JSON or binary data
- **Cycle Validation**: Must be positive integer within limits
- **Receipt Validation**: Must be valid CBOR format

## 9. Performance Requirements (IMMUTABLE)

### Execution Limits
- **Maximum Cycles**: 1,000,000 per execution
- **Maximum Artifact Size**: 1MB
- **Maximum Input Size**: 10MB
- **Maximum Output Size**: 10MB
- **Execution Timeout**: 30 seconds

### Resource Limits
- **Memory Usage**: 128MB maximum
- **CPU Usage**: Single-threaded execution
- **Network Usage**: No network access allowed
- **File System**: No file system access allowed

## 10. Compliance Requirements (IMMUTABLE)

### Standards Compliance
- **RFC 7049**: CBOR specification compliance
- **RFC 8032**: Ed25519 signature compliance
- **FIPS 140-2**: Cryptographic module compliance
- **Common Criteria**: Security evaluation compliance

### Audit Requirements
- **Receipt Immutability**: Receipts cannot be modified after creation
- **Execution Traceability**: All executions must be traceable
- **Audit Logging**: All operations must be logged
- **Compliance Reporting**: Regular compliance reports required

---

**This specification is frozen for v1.0.0-rc.1 and cannot be modified without creating a new major version.**