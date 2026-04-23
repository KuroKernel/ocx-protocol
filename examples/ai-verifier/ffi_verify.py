"""ctypes wrapper around libocx-verify/target/release/liblibocx_verify.so.

Mirrored from examples/gpu-verifier/ffi_verify.py — keep in sync.
Duplicated rather than imported so each example directory is self-contained.

Provides a minimal Python API that calls the canonical Rust verifier. If a
Python-produced receipt round-trips through this module, it proves CBOR
canonicalization, domain separator usage, and Ed25519 signing all match the
protocol-level specification byte-for-byte.
"""
from __future__ import annotations

import ctypes
import os
import time
from dataclasses import dataclass
from pathlib import Path

# Repo-relative path to the pre-built shared library.
_DEFAULT_LIB = (
    Path(__file__).resolve().parents[2]  # ocx-protocol/
    / "libocx-verify"
    / "target"
    / "release"
    / "liblibocx_verify.so"
)

# Error codes mirrored from ocx_verify.h.
OCX_SUCCESS = 0
OCX_INVALID_CBOR = 1
OCX_NON_CANONICAL_CBOR = 2
OCX_MISSING_FIELD = 3
OCX_INVALID_FIELD_VALUE = 4
OCX_INVALID_SIGNATURE = 5
OCX_HASH_MISMATCH = 6
OCX_INVALID_TIMESTAMP = 7
OCX_UNEXPECTED_EOF = 8
OCX_INTEGER_OVERFLOW = 9
OCX_INVALID_UTF8 = 10
OCX_INVALID_INPUT = 11
OCX_INTERNAL_ERROR = 12

_ERROR_NAMES = {
    OCX_SUCCESS: "OCX_SUCCESS",
    OCX_INVALID_CBOR: "OCX_INVALID_CBOR",
    OCX_NON_CANONICAL_CBOR: "OCX_NON_CANONICAL_CBOR",
    OCX_MISSING_FIELD: "OCX_MISSING_FIELD",
    OCX_INVALID_FIELD_VALUE: "OCX_INVALID_FIELD_VALUE",
    OCX_INVALID_SIGNATURE: "OCX_INVALID_SIGNATURE",
    OCX_HASH_MISMATCH: "OCX_HASH_MISMATCH",
    OCX_INVALID_TIMESTAMP: "OCX_INVALID_TIMESTAMP",
    OCX_UNEXPECTED_EOF: "OCX_UNEXPECTED_EOF",
    OCX_INTEGER_OVERFLOW: "OCX_INTEGER_OVERFLOW",
    OCX_INVALID_UTF8: "OCX_INVALID_UTF8",
    OCX_INVALID_INPUT: "OCX_INVALID_INPUT",
    OCX_INTERNAL_ERROR: "OCX_INTERNAL_ERROR",
}


def error_name(code: int) -> str:
    return _ERROR_NAMES.get(code, f"UNKNOWN_{code}")


def _load_lib(path: Path | str | None = None) -> ctypes.CDLL:
    p = Path(path) if path else _DEFAULT_LIB
    if not p.exists():
        raise FileNotFoundError(
            f"libocx-verify shared library not found at {p}. "
            "Build it with: cd libocx-verify && cargo build --release"
        )
    lib = ctypes.CDLL(str(p))

    # bool ocx_verify_receipt_detailed(
    #     const uint8_t* cbor_data, size_t len, const uint8_t* pubkey,
    #     OcxErrorCode* err_out) -> bool
    lib.ocx_verify_receipt_detailed.argtypes = [
        ctypes.c_char_p,   # cbor_data (treated as byte buffer)
        ctypes.c_size_t,   # cbor_data_len
        ctypes.c_char_p,   # public_key (32 bytes)
        ctypes.POINTER(ctypes.c_int),  # OcxErrorCode* (enum is C int)
    ]
    lib.ocx_verify_receipt_detailed.restype = ctypes.c_bool

    lib.ocx_verify_receipt.argtypes = [
        ctypes.c_char_p,
        ctypes.c_size_t,
        ctypes.c_char_p,
    ]
    lib.ocx_verify_receipt.restype = ctypes.c_bool

    # size_t ocx_get_version(char* buf, size_t buf_len)
    lib.ocx_get_version.argtypes = [ctypes.c_char_p, ctypes.c_size_t]
    lib.ocx_get_version.restype = ctypes.c_size_t

    # size_t ocx_get_error_message(OcxErrorCode code, char* buf, size_t buf_len)
    lib.ocx_get_error_message.argtypes = [
        ctypes.c_int,
        ctypes.c_char_p,
        ctypes.c_size_t,
    ]
    lib.ocx_get_error_message.restype = ctypes.c_size_t

    return lib


_LIB: ctypes.CDLL | None = None


def lib() -> ctypes.CDLL:
    global _LIB
    if _LIB is None:
        _LIB = _load_lib(os.environ.get("OCX_VERIFY_LIB"))
    return _LIB


@dataclass
class VerifyResult:
    ok: bool
    error_code: int
    error_name: str
    elapsed_us: float

    def __bool__(self) -> bool:
        return self.ok


def verify_receipt(cbor_bytes: bytes, public_key: bytes) -> VerifyResult:
    """Verify a canonical OCX receipt. Returns a VerifyResult."""
    if len(public_key) != 32:
        raise ValueError(f"public_key must be 32 raw bytes, got {len(public_key)}")
    err = ctypes.c_int(0)
    t0 = time.perf_counter()
    ok = lib().ocx_verify_receipt_detailed(
        cbor_bytes,
        len(cbor_bytes),
        public_key,
        ctypes.byref(err),
    )
    elapsed = (time.perf_counter() - t0) * 1e6  # microseconds
    return VerifyResult(
        ok=bool(ok),
        error_code=err.value,
        error_name=error_name(err.value),
        elapsed_us=elapsed,
    )


def library_version() -> str:
    buf = ctypes.create_string_buffer(64)
    n = lib().ocx_get_version(buf, len(buf))
    return buf.value[: max(0, n - 1)].decode("utf-8", errors="replace") if n else "?"


if __name__ == "__main__":
    # Round-trip self-test: build a receipt in Python, sign it, verify via Rust.
    import hashlib
    import statistics

    from canonical_receipt import (
        ReceiptCore,
        raw_public_key,
        sign_receipt,
    )
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

    print(f"libocx-verify version: {library_version()}")

    key = Ed25519PrivateKey.generate()
    pub = raw_public_key(key.public_key())

    core = ReceiptCore(
        program_hash=hashlib.sha256(b"program").digest(),
        input_hash=hashlib.sha256(b"input").digest(),
        output_hash=hashlib.sha256(b"output").digest(),
        cycles_used=1000,
        started_at=int(time.time()) - 5,
        finished_at=int(time.time()) - 3,
        issuer_id="ocx-gpu-verifier-demo",
    )
    sig, receipt_bytes = sign_receipt(core, key)

    result = verify_receipt(receipt_bytes, pub)
    print(f"first-verify: ok={result.ok} err={result.error_name} elapsed={result.elapsed_us:.0f}us")

    # Timing: 100 verifications, median + p99
    timings: list[float] = []
    for _ in range(100):
        r = verify_receipt(receipt_bytes, pub)
        if not r.ok:
            raise RuntimeError(f"unexpected failure in benchmark: {r.error_name}")
        timings.append(r.elapsed_us)
    timings.sort()
    median = statistics.median(timings)
    p99 = timings[int(len(timings) * 0.99)]
    print(f"verify bench: n=100 median={median:.0f}us p99={p99:.0f}us")

    # Tamper test: flip one bit in the signature (last byte of receipt)
    tampered = bytearray(receipt_bytes)
    tampered[-1] ^= 0x01
    tampered_result = verify_receipt(bytes(tampered), pub)
    print(
        f"tamper test:  ok={tampered_result.ok} err={tampered_result.error_name} "
        f"(expected: ok=False, err=OCX_INVALID_SIGNATURE)"
    )

    # Summary
    if result.ok and not tampered_result.ok and tampered_result.error_code == OCX_INVALID_SIGNATURE:
        print("PASS: Python-produced receipts round-trip through Rust canonical verifier.")
    else:
        print("FAIL: round-trip broken.")
        raise SystemExit(1)
