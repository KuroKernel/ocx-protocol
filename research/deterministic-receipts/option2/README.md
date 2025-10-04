# Option 2: Move Variable Fields Out of Signature

## Design Overview

Keep `cycles_used` (host-measured), `started_at`, `finished_at` **outside** the signed map.

**Signed core** = program/input/output hashes + issuer_id only.

## Implementation Strategy

### Receipt Structure
```go
type OCXReceipt struct {
    // Signed Core (deterministic)
    ProgramHash [32]byte `cbor:"1,keyasint"`
    InputHash   [32]byte `cbor:"2,keyasint"`
    OutputHash  [32]byte `cbor:"3,keyasint"`
    IssuerID    string   `cbor:"4,keyasint"`
    Signature   []byte   `cbor:"5,keyasint"`
    
    // Unsigned Metadata (variable)
    CyclesUsed  uint64   `cbor:"6,keyasint"`
    StartedAt   uint64   `cbor:"7,keyasint"`
    FinishedAt  uint64   `cbor:"8,keyasint"`
    HostInfo    HostInfo `cbor:"9,keyasint"`
}

type HostInfo struct {
    CPUModel    string `cbor:"1,keyasint"`
    KernelVer   string `cbor:"2,keyasint"`
    DurationMs  uint64 `cbor:"3,keyasint"`
}
```

### Benefits
- Each run produces a different receipt (expected)
- Verification doesn't depend on OS jitter
- Golden vectors must freeze one exemplar, not regenerate
- Clear separation between deterministic proof and variable metadata

### Drawbacks
- Receipts are not portable across runs
- Golden vectors become more complex to manage
- Less intuitive for users expecting identical receipts

## Implementation Notes

This approach requires:
1. Redefining the signed core to exclude variable fields
2. Updating verification logic to only verify the signed core
3. Modifying golden vector generation to freeze exemplars
4. Updating documentation to clarify receipt uniqueness

## Status: Research Only

This option is preserved for future consideration but not implemented in the main project.
