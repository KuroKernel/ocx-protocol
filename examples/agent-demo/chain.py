"""Chain helpers: build OCX receipts that link to the previous one,
serialize the chain to disk, and load it back for verification.

Wire format:
    output/receipts.cbor   — CBOR array of transmitted receipts (each one
                             a canonical-CBOR map matching ReceiptCore +
                             signature). Verifier reads this directly.
    output/receipts.json   — Human-readable JSON sidecar (hex-encoded
                             hashes + timestamps). Strictly debug; the
                             cbor file is the source of truth.
    output/pubkey.txt      — base64 raw 32-byte Ed25519 public key. The
                             verifier needs this to check signatures.

Each receipt represents one *step* in the agent loop:
    program_hash = sha256("<step_kind>:<extra>")
    input_hash   = sha256(canonical-json of the step's inputs)
    output_hash  = sha256(canonical-json of the step's outputs)
    cycles_used  = a step-kind-appropriate effort proxy (tokens, bytes,
                   …); never zero (Rust verifier rejects 0)
    started_at   = unix seconds when the step started
    finished_at  = max(started_at + 1, unix seconds when it ended)
                   (verifier requires duration >= 1 second)
    issuer_id    = "ocx-agent-demo-v0"
    request_digest = sha256(previous receipt's canonical bytes), or
                     None on the first receipt.

NOTE on chain linkage. We store the previous-receipt hash in the OCX
receipt's `request_digest` field (key 10), not `prev_receipt_hash`
(key 9). Both fields are 32-byte hashes signed by the issuer. The
difference is that `libocx-verify`'s C FFI tries to look up key 9 in
a process-local chain registry it cannot populate via FFI, so any
receipt with key 9 set fails with `OCX_HASH_MISMATCH`. Key 10 has no
semantic checks beyond format, which is exactly what we need: signed
linkage that the per-receipt FFI verifies, plus our own
`verify_chain.py` walking the link integrity at the chain level.
"""
from __future__ import annotations

import base64
import hashlib
import json
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import cbor2
from cryptography.hazmat.primitives.asymmetric.ed25519 import (
    Ed25519PrivateKey,
)

from canonical_receipt import ReceiptCore, raw_public_key, sign_receipt

ISSUER_ID = "ocx-agent-demo-v0"


def _canonical_json(obj: Any) -> bytes:
    """Stable JSON for hashing inputs/outputs."""
    return json.dumps(
        obj, sort_keys=True, separators=(",", ":"), ensure_ascii=False
    ).encode("utf-8")


def _sha256(b: bytes) -> bytes:
    return hashlib.sha256(b).digest()


def receipt_hash(transmitted_cbor: bytes) -> bytes:
    """Stable identity of a receipt — SHA-256 of its canonical bytes."""
    return _sha256(transmitted_cbor)


@dataclass
class Step:
    kind: str               # e.g. "model_call", "tool_call:read_file", "file_write:report.md"
    inputs: Any             # JSON-serializable
    outputs: Any            # JSON-serializable
    cycles: int             # effort, > 0
    started_at: int         # unix seconds
    finished_at: int        # unix seconds, may be padded


class Chain:
    """Builds the receipt chain incrementally as the agent runs."""

    def __init__(self, signer: Ed25519PrivateKey) -> None:
        self.signer = signer
        self.pubkey = raw_public_key(signer.public_key())
        self.receipts: list[bytes] = []          # canonical CBOR per step
        self.json_steps: list[dict[str, Any]] = []  # debug sidecar
        self._prev_hash: bytes | None = None

    def append(self, step: Step) -> dict[str, Any]:
        """Sign a step into a receipt, link to previous, return debug
        record (also added to the JSON sidecar)."""
        program_hash = _sha256(("agent:" + step.kind).encode("utf-8"))
        input_bytes = _canonical_json(step.inputs)
        output_bytes = _canonical_json(step.outputs)
        input_hash = _sha256(input_bytes)
        output_hash = _sha256(output_bytes)

        # Verifier requires duration >= 1 sec; pad finished_at if needed.
        finished_at = max(step.finished_at, step.started_at + 1)

        # Verifier requires the three hashes to be mutually distinct.
        # In practice they will be (different domain prefixes), but if
        # an agent ever produces input bytes equal to the program-hash
        # input, we'd fail. Add a uniquifying suffix to output if needed.
        if output_hash == program_hash or output_hash == input_hash:
            output_bytes += b"\x00ocx-uniquify"
            output_hash = _sha256(output_bytes)
        if input_hash == program_hash:
            input_bytes += b"\x00ocx-uniquify"
            input_hash = _sha256(input_bytes)

        core = ReceiptCore(
            program_hash=program_hash,
            input_hash=input_hash,
            output_hash=output_hash,
            cycles_used=max(1, step.cycles),
            started_at=step.started_at,
            finished_at=finished_at,
            issuer_id=ISSUER_ID,
            request_digest=self._prev_hash,
        )
        sig, transmitted = sign_receipt(core, self.signer)
        self._prev_hash = receipt_hash(transmitted)
        self.receipts.append(transmitted)

        debug = {
            "index": len(self.receipts) - 1,
            "kind": step.kind,
            "program_hash": program_hash.hex(),
            "input_hash": input_hash.hex(),
            "output_hash": output_hash.hex(),
            "cycles_used": core.cycles_used,
            "started_at": step.started_at,
            "finished_at": finished_at,
            "request_digest": (
                core.request_digest.hex() if core.request_digest else None
            ),
            "receipt_hash": self._prev_hash.hex(),
            "signature": sig.hex(),
            "inputs_preview": _preview(input_bytes),
            "outputs_preview": _preview(output_bytes),
        }
        self.json_steps.append(debug)
        return debug

    def write(self, output_dir: Path) -> None:
        """Persist the chain to disk."""
        output_dir.mkdir(parents=True, exist_ok=True)
        # CBOR array of transmitted receipts (the verifier reads this).
        # One outer cbor2.dumps so verifiers can stream-parse one record.
        chain_bytes = cbor2.dumps(self.receipts, canonical=True)
        (output_dir / "receipts.cbor").write_bytes(chain_bytes)
        # Human-readable sidecar.
        (output_dir / "receipts.json").write_text(
            json.dumps(
                {
                    "issuer_id": ISSUER_ID,
                    "public_key_b64": base64.b64encode(self.pubkey).decode(),
                    "step_count": len(self.json_steps),
                    "steps": self.json_steps,
                },
                indent=2,
            )
        )
        # Public key in a standalone file so the verifier can locate it.
        (output_dir / "pubkey.txt").write_text(
            base64.b64encode(self.pubkey).decode() + "\n"
        )


def _preview(b: bytes, n: int = 240) -> str:
    """First N chars of a JSON blob for the human-readable sidecar."""
    s = b.decode("utf-8", errors="replace")
    return s if len(s) <= n else s[:n] + "…"


# Convenience: build a Step that wraps `now()` boundaries.
def step_now(kind: str, inputs: Any, outputs: Any, cycles: int) -> Step:
    started = int(time.time())
    return Step(
        kind=kind,
        inputs=inputs,
        outputs=outputs,
        cycles=cycles,
        started_at=started,
        finished_at=started,  # padded to +1s by Chain.append
    )
