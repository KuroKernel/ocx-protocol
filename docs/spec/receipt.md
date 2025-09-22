# OCX Receipt Specification v1.0

## Canonical Receipt Format

### CBOR Map Structure
All receipts MUST be encoded as CBOR maps with the following structure:

**Required Fields (integer keys, in order):**
- `1` → bstr (32 bytes, SHA-256 of program) - program_hash
- `2` → bstr (32 bytes, SHA-256 of input) - input_hash  
- `3` → bstr (32 bytes, SHA-256 of output) - output_hash
- `4` → integer (computational cycles used) - cycles
- `5` → integer (Unix timestamp when started) - started_at
- `6` → integer (Unix timestamp when finished) - finished_at
- `7` → text (issuer key identifier) - issuer_id
- `8` → bstr (64 bytes, Ed25519 signature) - signature

**Optional Fields (v1.1 extensions):**
- `9` → bstr (32 bytes, previous receipt hash) - prev_receipt_hash
- `10` → bstr (32 bytes, request digest) - request_digest
- `11` → array (witness signatures) - witness_signatures

### Canonical CBOR Rules (RFC 8949 Section 4.2)
- Definite lengths everywhere (no indefinite encoding)
- Integers use shortest-length encoding
- Map keys sorted numerically (1, 2, 3, ...)
- No duplicate keys
- No CBOR tags unless explicitly documented

### Signing Procedure
1. Create `receipt_core` by removing `signature` field from full receipt map
2. Compute message: `concat("OCXv1|receipt|", canonical_cbor(receipt_core))`
3. Sign with Ed25519 (NOT Ed25519ph) over message bytes
4. Store 64-byte signature in `signature` field

### Hash Fields
- All hash fields (`program_hash`, `input_hash`, `output_hash`) are raw SHA-256 bytes
- In CBOR: stored as bstr
- In JSON: base64-encoded for transport
