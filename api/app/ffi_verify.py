"""ctypes binding for libocx-verify. Loaded lazily; if the .so isn't
present (e.g. in a dev env without Rust), /v1/verify returns 503 and the
rest of the API still works."""
from __future__ import annotations

import ctypes
import logging
import os
from dataclasses import dataclass
from pathlib import Path

from .config import get_settings

log = logging.getLogger("ocx.ffi")

_lib: ctypes.CDLL | None = None
_load_attempted = False


def _try_load() -> ctypes.CDLL | None:
    global _lib, _load_attempted
    if _load_attempted:
        return _lib
    _load_attempted = True
    s = get_settings()
    candidate_paths = []
    if s.ocx_verify_lib:
        candidate_paths.append(s.ocx_verify_lib)
    # Common defaults
    candidate_paths += [
        "/opt/ocx/liblibocx_verify.so",
        "./liblibocx_verify.so",
        "../libocx-verify/target/release/liblibocx_verify.so",
    ]
    for p in candidate_paths:
        if not p or not Path(p).exists():
            continue
        try:
            lib = ctypes.CDLL(p)
            # Bind the one entry point we use:
            #   bool ocx_verify_receipt_detailed(
            #       const uint8_t* cbor, size_t len,
            #       const uint8_t* pubkey, int* err_out)
            lib.ocx_verify_receipt_detailed.argtypes = [
                ctypes.POINTER(ctypes.c_uint8),
                ctypes.c_size_t,
                ctypes.POINTER(ctypes.c_uint8),
                ctypes.POINTER(ctypes.c_int),
            ]
            lib.ocx_verify_receipt_detailed.restype = ctypes.c_bool
            _lib = lib
            log.info("libocx-verify loaded from %s", p)
            return _lib
        except Exception as e:
            log.warning("Failed to load %s: %s", p, e)
    log.error("libocx-verify not found in any candidate path; /v1/verify will be unavailable")
    return None


@dataclass
class VerifyResult:
    ok: bool
    error_code: int
    error_name: str


_ERROR_NAMES = {
    0: "OCX_SUCCESS",
    1: "OCX_INVALID_CBOR",
    2: "OCX_NON_CANONICAL_CBOR",
    3: "OCX_MISSING_FIELD",
    4: "OCX_INVALID_FIELD_VALUE",
    5: "OCX_INVALID_SIGNATURE",
    6: "OCX_HASH_MISMATCH",
    7: "OCX_INVALID_TIMESTAMP",
}


def verify(cbor_bytes: bytes, public_key_bytes: bytes) -> VerifyResult:
    lib = _try_load()
    if lib is None:
        raise RuntimeError("libocx-verify is not available on this server")
    if len(public_key_bytes) != 32:
        return VerifyResult(False, -1, "INVALID_PUBKEY_LEN")
    cbor_arr = (ctypes.c_uint8 * len(cbor_bytes)).from_buffer_copy(cbor_bytes)
    pk_arr = (ctypes.c_uint8 * 32).from_buffer_copy(public_key_bytes)
    err = ctypes.c_int(0)
    ok = lib.ocx_verify_receipt_detailed(cbor_arr, len(cbor_bytes), pk_arr, ctypes.byref(err))
    return VerifyResult(
        ok=bool(ok),
        error_code=err.value,
        error_name=_ERROR_NAMES.get(err.value, f"UNKNOWN_{err.value}"),
    )


def is_available() -> bool:
    return _try_load() is not None
