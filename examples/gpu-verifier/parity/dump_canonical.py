#!/usr/bin/env python3
"""Python mirror of dump_canonical.go.

Produces canonical CBOR for the same fixture and prints hex.
The goal is byte-identity with Go's fxamacker/cbor/v2 canonical output.
"""
import cbor2

# Same fixture as dump_canonical.go
program_hash = bytes(range(1, 33))
input_hash = bytes(range(2, 34))
output_hash = bytes(range(3, 35))
gas_used = 1000
started_at = 1640995200
finished_at = 1640995201
issuer_id = "test-issuer"

signed_map = {
    1: program_hash,
    2: input_hash,
    3: output_hash,
    4: gas_used,
    5: started_at,
    6: finished_at,
    7: issuer_id,
}
signed_bytes = cbor2.dumps(signed_map, canonical=True)
print(f"SIGNED_HEX={signed_bytes.hex()}")

fake_sig = bytes(64)
transmitted_map = {**signed_map, 8: fake_sig}
transmitted_bytes = cbor2.dumps(transmitted_map, canonical=True)
print(f"TRANSMITTED_HEX={transmitted_bytes.hex()}")
