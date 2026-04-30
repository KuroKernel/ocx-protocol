"""Walk the receipt chain produced by run.py and verify:

  1. Every individual receipt's Ed25519 signature, via the canonical
     Rust verifier exposed by libocx-verify (C FFI).
  2. The chain link itself: receipt[i].prev_receipt_hash equals
     sha256(receipt[i-1] canonical bytes).
  3. The issuer key id is consistent through the chain.

Output: one line per receipt and a final "OCX_CHAIN_VALID" or
"OCX_CHAIN_BROKEN at receipt N: <reason>".

Usage:
    python3 verify_chain.py output/
"""
from __future__ import annotations

import base64
import ctypes
import hashlib
import json
import sys
from ctypes import POINTER, c_bool, c_int, c_size_t, c_ubyte
from pathlib import Path

import cbor2

# ---------- locate libocx-verify ----------

_LIB_CANDIDATES = [
    Path(__file__).parent.parent.parent / "libocx-verify" / "target" / "release" / "liblibocx_verify.so",
    Path("/opt/ocx/liblibocx_verify.so"),
]


def _load_lib() -> ctypes.CDLL:
    for p in _LIB_CANDIDATES:
        if p.exists():
            return ctypes.CDLL(str(p))
    raise FileNotFoundError(
        "libocx-verify shared library not found. Build it with:\n"
        "    cd libocx-verify && cargo build --release"
    )


_lib = _load_lib()
# Signature from libocx-verify/ocx_verify.h:
#   bool ocx_verify_receipt_detailed(
#       const uint8_t *receipt_cbor, size_t cbor_len,
#       const uint8_t *pubkey32,
#       int *error_code_out
#   );
_lib.ocx_verify_receipt_detailed.argtypes = [
    POINTER(c_ubyte), c_size_t,
    POINTER(c_ubyte),
    POINTER(c_int),
]
_lib.ocx_verify_receipt_detailed.restype = c_bool


def verify_one(cbor_bytes: bytes, pubkey: bytes) -> tuple[bool, int]:
    """Return (ok, error_code). 0 == OCX_SUCCESS."""
    err = c_int(0)
    cbor_arr = (c_ubyte * len(cbor_bytes)).from_buffer_copy(cbor_bytes)
    pk_arr = (c_ubyte * len(pubkey)).from_buffer_copy(pubkey)
    ok = _lib.ocx_verify_receipt_detailed(
        cbor_arr, len(cbor_bytes),
        pk_arr,
        ctypes.byref(err),
    )
    return bool(ok), int(err.value)


# ---------- chain walk ----------

ERROR_NAMES = {
    0:  "OCX_SUCCESS",
    1:  "OCX_INVALID_CBOR",
    2:  "OCX_NON_CANONICAL_CBOR",
    3:  "OCX_MISSING_FIELD",
    4:  "OCX_INVALID_FIELD_VALUE",
    5:  "OCX_INVALID_SIGNATURE",
    6:  "OCX_HASH_MISMATCH",
    7:  "OCX_INVALID_TIMESTAMP",
}


def main(out_dir: str) -> int:
    out = Path(out_dir).resolve()
    receipts_path = out / "receipts.cbor"
    pubkey_path = out / "pubkey.txt"
    if not receipts_path.exists() or not pubkey_path.exists():
        print(f"ERROR: missing receipts.cbor or pubkey.txt in {out}", file=sys.stderr)
        return 1

    raw = receipts_path.read_bytes()
    chain: list[bytes] = cbor2.loads(raw)
    pubkey = base64.b64decode(pubkey_path.read_text().strip())
    if len(pubkey) != 32:
        print(f"ERROR: pubkey must be 32 bytes, got {len(pubkey)}", file=sys.stderr)
        return 1

    # Optional: cross-check the JSON sidecar's metadata if present
    sidecar: dict | None = None
    sidecar_path = out / "receipts.json"
    if sidecar_path.exists():
        sidecar = json.loads(sidecar_path.read_text())

    print(f"loaded {len(chain)} receipts from {receipts_path}")
    print(f"verifier  : {next(p for p in _LIB_CANDIDATES if p.exists())}")
    print(f"public key: {pubkey.hex()}")
    print()

    issuer_seen: str | None = None
    prev_hash: bytes | None = None
    fail_at: int | None = None
    fail_reason: str | None = None

    for i, cbor_bytes in enumerate(chain):
        # Parse the receipt body so we can check the chain field +
        # issuer (the C FFI doesn't surface them). Chain linkage lives
        # in `request_digest` (key 10): see chain.py for why key 10
        # rather than key 9.
        record = cbor2.loads(cbor_bytes)
        recorded_prev = record.get(10)
        issuer = record.get(7)
        rec_hash = hashlib.sha256(cbor_bytes).digest()

        ok, err = verify_one(cbor_bytes, pubkey)
        err_name = ERROR_NAMES.get(err, f"UNKNOWN({err})")

        # Chain checks
        chain_ok = True
        if i == 0:
            if recorded_prev is not None:
                chain_ok = False
                fail_reason = "first receipt has request_digest but should be None"
        else:
            if recorded_prev != prev_hash:
                chain_ok = False
                fail_reason = (
                    f"chain link mismatch: receipt[{i}].request_digest = "
                    f"{(recorded_prev or b'').hex()[:16]}…  expected = "
                    f"{prev_hash.hex()[:16]}…"
                )

        # Issuer consistency
        if issuer_seen is None:
            issuer_seen = issuer
        elif issuer != issuer_seen:
            chain_ok = False
            fail_reason = f"issuer changed: was {issuer_seen!r}, now {issuer!r}"

        verdict = "OK" if (ok and chain_ok) else "FAIL"
        sidecar_kind = (
            sidecar["steps"][i]["kind"]
            if sidecar and i < len(sidecar.get("steps", []))
            else "(unknown)"
        )
        print(
            f"  [{i:>2}] {verdict:4s}  sig={err_name:24s}  "
            f"hash={rec_hash.hex()[:12]}…  kind={sidecar_kind}"
        )

        if not ok or not chain_ok:
            fail_at = i
            if fail_reason is None:
                fail_reason = f"signature: {err_name}"
            break
        prev_hash = rec_hash

    print()
    if fail_at is not None:
        print(f"OCX_CHAIN_BROKEN  at receipt {fail_at}: {fail_reason}")
        return 2

    print(
        f"OCX_CHAIN_VALID   "
        f"len={len(chain)}  "
        f"issuer={issuer_seen!r}  "
        f"first={hashlib.sha256(chain[0]).hexdigest()[:12]}…  "
        f"last={hashlib.sha256(chain[-1]).hexdigest()[:12]}…"
    )
    return 0


if __name__ == "__main__":
    out_dir = sys.argv[1] if len(sys.argv) > 1 else "output"
    sys.exit(main(out_dir))
