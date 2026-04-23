"""Canonical OCX receipt encoder + Ed25519 signer for Python.

Matches the Go `pkg/receipt/types.go:ReceiptCore` and the Rust
`libocx-verify/src/receipt.rs:OcxReceipt` byte-for-byte.

Receipt layout (canonical CBOR map with integer keys):
    key 1  bytes32  artifact_hash / program_hash
    key 2  bytes32  input_hash
    key 3  bytes32  output_hash
    key 4  uint64   cycles_used (Go: gas_used)
    key 5  uint64   started_at  (unix SECONDS, not nanos)
    key 6  uint64   finished_at (unix SECONDS)
    key 7  text     issuer_id
    key 8  bytes64  signature    (only in transmitted form)
    keys 9..11  optional v1.1 extensions (chain hash, request digest, witnesses)
    keys 12..15 optional v1.2 VDF fields

Signing message = b"OCXv1|receipt|" || canonical_cbor({1..7, 9..15})
Transmitted receipt = canonical_cbor({1..7, 8=signature, 9..15})

Rust verifier constraints (MUST be satisfied):
    cycles_used != 0
    all three hashes non-zero and mutually distinct
    finished_at >= started_at
    (finished_at - started_at) >= 1 second, <= 24 hours
"""
from __future__ import annotations

import cbor2
from dataclasses import dataclass, field
from typing import Optional

from cryptography.hazmat.primitives.asymmetric.ed25519 import (
    Ed25519PrivateKey,
    Ed25519PublicKey,
)
from cryptography.hazmat.primitives import serialization

# The exact domain separator used by Go generator + Rust verifier.
# Missing or wrong = signature cannot verify.
DOMAIN_SEPARATOR = b"OCXv1|receipt|"

SIGNATURE_LEN = 64


@dataclass
class ReceiptCore:
    """Signed fields. Keys 1-7 required. Optional keys 9-15 supported."""
    program_hash: bytes  # 32 bytes
    input_hash: bytes    # 32 bytes
    output_hash: bytes   # 32 bytes
    cycles_used: int     # non-zero
    started_at: int      # unix seconds
    finished_at: int     # unix seconds, >= started_at + 1
    issuer_id: str

    # v1.1 optional
    prev_receipt_hash: Optional[bytes] = None  # key 9
    request_digest: Optional[bytes] = None     # key 10
    witness_signatures: list[bytes] = field(default_factory=list)  # key 11

    # v1.2 optional (VDF temporal proof)
    vdf_output: Optional[bytes] = None     # key 12
    vdf_proof: Optional[bytes] = None      # key 13
    vdf_iterations: Optional[int] = None   # key 14
    vdf_modulus_id: Optional[str] = None   # key 15

    def _signed_map(self) -> dict[int, object]:
        """Build the canonical map that gets signed (keys 1-7 plus present optionals)."""
        self._validate_shapes()
        m: dict[int, object] = {
            1: self.program_hash,
            2: self.input_hash,
            3: self.output_hash,
            4: self.cycles_used,
            5: self.started_at,
            6: self.finished_at,
            7: self.issuer_id,
        }
        if self.prev_receipt_hash is not None:
            m[9] = self.prev_receipt_hash
        if self.request_digest is not None:
            m[10] = self.request_digest
        if self.witness_signatures:
            m[11] = list(self.witness_signatures)
        if self.vdf_output is not None:
            m[12] = self.vdf_output
        if self.vdf_proof is not None:
            m[13] = self.vdf_proof
        if self.vdf_iterations is not None:
            m[14] = self.vdf_iterations
        if self.vdf_modulus_id is not None:
            m[15] = self.vdf_modulus_id
        return m

    def _validate_shapes(self) -> None:
        if len(self.program_hash) != 32:
            raise ValueError(f"program_hash must be 32 bytes, got {len(self.program_hash)}")
        if len(self.input_hash) != 32:
            raise ValueError(f"input_hash must be 32 bytes, got {len(self.input_hash)}")
        if len(self.output_hash) != 32:
            raise ValueError(f"output_hash must be 32 bytes, got {len(self.output_hash)}")
        if self.cycles_used == 0:
            raise ValueError("cycles_used must be non-zero (Rust verifier rejects zero)")
        if self.finished_at < self.started_at:
            raise ValueError("finished_at must be >= started_at")
        duration = self.finished_at - self.started_at
        if duration < 1:
            raise ValueError(
                f"duration ({duration}s) must be >= 1 second; pad timestamps "
                "if wall-clock inference was faster"
            )
        if duration > 24 * 60 * 60:
            raise ValueError(f"duration ({duration}s) must be <= 24 hours")
        if (
            self.program_hash == bytes(32)
            or self.input_hash == bytes(32)
            or self.output_hash == bytes(32)
        ):
            raise ValueError("hashes must not be all zeros")
        if self.program_hash == self.input_hash:
            raise ValueError("program_hash and input_hash must differ")
        if self.program_hash == self.output_hash:
            raise ValueError("program_hash and output_hash must differ")
        if self.input_hash == self.output_hash:
            raise ValueError("input_hash and output_hash must differ")

    def canonical_signing_bytes(self) -> bytes:
        """Canonical CBOR of signed fields (no signature). Matches Go CanonicalizeCore."""
        return cbor2.dumps(self._signed_map(), canonical=True)

    def signing_message(self) -> bytes:
        """The exact bytes Ed25519 signs: domain_separator || canonical_cbor."""
        return DOMAIN_SEPARATOR + self.canonical_signing_bytes()

    def transmitted_cbor(self, signature: bytes) -> bytes:
        """Canonical CBOR of the transmitted receipt including key 8 = signature."""
        if len(signature) != SIGNATURE_LEN:
            raise ValueError(f"signature must be {SIGNATURE_LEN} bytes, got {len(signature)}")
        m = self._signed_map()
        m[8] = signature
        return cbor2.dumps(m, canonical=True)


def raw_public_key(pub: Ed25519PublicKey) -> bytes:
    """32-byte raw Ed25519 public key (what the Rust verifier expects)."""
    return pub.public_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PublicFormat.Raw,
    )


def sign_receipt(core: ReceiptCore, priv: Ed25519PrivateKey) -> tuple[bytes, bytes]:
    """Sign a ReceiptCore and return (signature, transmitted_cbor).

    Returns:
        (64-byte Ed25519 signature, canonical CBOR bytes of the full receipt)
    """
    msg = core.signing_message()
    signature = priv.sign(msg)
    receipt_bytes = core.transmitted_cbor(signature)
    return signature, receipt_bytes


if __name__ == "__main__":
    # Self-test: build a fixed receipt, sign it, print hex so a human can eyeball
    # against the Go parity dump.
    import hashlib

    key = Ed25519PrivateKey.generate()
    pub = key.public_key()

    core = ReceiptCore(
        program_hash=hashlib.sha256(b"program").digest(),
        input_hash=hashlib.sha256(b"input").digest(),
        output_hash=hashlib.sha256(b"output").digest(),
        cycles_used=1000,
        started_at=1640995200,
        finished_at=1640995201,
        issuer_id="ocx-gpu-verifier-demo",
    )
    sig, receipt_bytes = sign_receipt(core, key)
    print(f"signing_bytes_hex={core.canonical_signing_bytes().hex()}")
    print(f"signature_hex={sig.hex()}")
    print(f"transmitted_cbor_hex={receipt_bytes.hex()}")
    print(f"public_key_hex={raw_public_key(pub).hex()}")
    print(f"receipt_len_bytes={len(receipt_bytes)}")
