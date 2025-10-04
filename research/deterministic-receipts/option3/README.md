# Option 3: Quantize + Cap

## Design Overview

If you must sign variable fields, **bucket** cycles and timing to coarse bins so small jitter doesn't change the receipt.

## Implementation Strategy

### Quantization Rules
```go
// Cycle quantization
func QuantizeCycles(cycles uint64) uint64 {
    // Round to nearest 10k-cycle bucket
    return ((cycles + 5000) / 10000) * 10000
}

// Time quantization  
func QuantizeTime(timestamp uint64) uint64 {
    // Round to nearest 100ms bucket
    return ((timestamp + 50) / 100) * 100
}
```

### Receipt Structure
```go
type OCXReceipt struct {
    ProgramHash [32]byte `cbor:"1,keyasint"`
    InputHash   [32]byte `cbor:"2,keyasint"`
    OutputHash  [32]byte `cbor:"3,keyasint"`
    CyclesUsed  uint64   `cbor:"4,keyasint"` // Quantized
    StartedAt   uint64   `cbor:"5,keyasint"` // Quantized
    FinishedAt  uint64   `cbor:"6,keyasint"` // Quantized
    IssuerID    string   `cbor:"7,keyasint"`
    Signature   []byte   `cbor:"8,keyasint"`
}
```

### Benefits
- Reduces jitter impact on receipts
- Still allows signing of timing information
- Simpler than moving fields out of signature

### Drawbacks
- Still inferior to true gas model
- Quantization introduces precision loss
- May not eliminate all jitter in high-variance environments
- Complex to tune quantization parameters

## Implementation Notes

This approach requires:
1. Implementing quantization functions for cycles and timestamps
2. Applying quantization before signing
3. Documenting quantization parameters in the spec
4. Testing across different system loads

## Status: Research Only

This option is preserved for future consideration but not implemented in the main project.
