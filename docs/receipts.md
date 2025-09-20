# OCX Receipt Format Specification

## CBOR Schema (Canonical)

```
receipt := {
  v: uint8,                    // Profile version (1 for v1-min)
  artifact: bytes(32),         // SHA256 hash of executed code
  input: bytes(32),            // SHA256 hash of input data
  output: bytes(32),           // SHA256 hash of computation output
  cycles: uint64,              // Actual CPU cycles consumed
  metering: {                  // Pricing constants (frozen in v1)
    alpha: uint64,             // Cost per cycle (10)
    beta: uint64,              // Cost per I/O byte (1) 
    gamma: uint64              // Cost per memory page (100)
  },
  transcript_root: bytes(32),  // Merkle root of execution trace
  issuer: bytes(32),           // Ed25519 public key
  sig: bytes(64)               // Ed25519 signature of receipt body
}
```

## Example Receipt (Hex Encoded)

```
a7                           // CBOR map with 7 key-value pairs
  01                         // Key: v (version)
  01                         // Value: 1 (v1-min profile)
  02                         // Key: artifact
  58 20                      // 32-byte string follows
  abc123def456...            // SHA256 hash of artifact
  03                         // Key: input  
  58 20                      // 32-byte string follows
  def456abc123...            // SHA256 hash of input
  ...
```
