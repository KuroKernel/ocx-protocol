#!/usr/bin/env python3
"""Whitepaper-grade cross-language test.

Reads the fixture written by cross_language_roundtrip.go, regenerates the
exact same canonical CBOR bytes from Python, signs with the same Ed25519
key, and asserts byte-equality with the Go-produced signing bytes,
signature, and full receipt. Then verifies BOTH the Go-produced and
Python-produced receipts via the canonical Rust libocx-verify.so FFI.

Result: a single test that proves Go, Python, and Rust agree on
1) canonical CBOR encoding of OCX ReceiptCore
2) Ed25519 deterministic signing with `OCXv1|receipt|` domain separator
3) the transmitted receipt format with the signature at integer key 8

If this passes, the protocol is cross-language verified.
"""
from __future__ import annotations

import hashlib
import json
import sys
from pathlib import Path

import cbor2
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

# Ensure import of the shared modules from gpu-verifier (they live under examples/)
HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent / "examples" / "gpu-verifier"))

from canonical_receipt import DOMAIN_SEPARATOR, ReceiptCore, raw_public_key, sign_receipt  # noqa: E402
from ffi_verify import verify_receipt as ffi_verify  # noqa: E402

OUT_DIR = Path("/tmp/ocx_xlang")


def red(s: str) -> str:
    return f"\033[31m{s}\033[0m"


def green(s: str) -> str:
    return f"\033[32m{s}\033[0m"


def main() -> int:
    fixture = json.loads((OUT_DIR / "fixture.json").read_text())
    keypair = json.loads((OUT_DIR / "keypair.json").read_text())

    go_signing_hex = (OUT_DIR / "go_signing_bytes.hex").read_text().strip()
    go_signature_hex = (OUT_DIR / "go_signature.hex").read_text().strip()
    go_receipt_cbor = (OUT_DIR / "go_receipt.cbor").read_bytes()

    # Reconstruct the same key Go used (deterministic from the seed)
    seed_bytes = bytes.fromhex(keypair["private_seed_hex"])
    priv = Ed25519PrivateKey.from_private_bytes(seed_bytes)
    pub_raw = raw_public_key(priv.public_key())

    if pub_raw.hex() != keypair["public_key_hex"]:
        print(red("FAIL: derived public key mismatches Go's"))
        return 1
    print(green("PASS: derived Ed25519 keypair matches Go (deterministic key derivation)"))

    # Reconstruct the same ReceiptCore the Go side built
    core = ReceiptCore(
        program_hash=bytes.fromhex(fixture["program_hash_hex"]),
        input_hash=bytes.fromhex(fixture["input_hash_hex"]),
        output_hash=bytes.fromhex(fixture["output_hash_hex"]),
        cycles_used=fixture["gas_used"],
        started_at=fixture["started_at"],
        finished_at=fixture["finished_at"],
        issuer_id=fixture["issuer_id"],
    )

    py_signing_bytes = core.canonical_signing_bytes()
    py_signing_hex = py_signing_bytes.hex()

    # Test 1: canonical signing bytes are byte-identical
    if py_signing_hex != go_signing_hex:
        print(red("FAIL: canonical signing bytes differ between Go and Python!"))
        print(f"  go : {go_signing_hex}")
        print(f"  py : {py_signing_hex}")
        return 2
    print(green(f"PASS: canonical CBOR signing bytes byte-identical Go ↔ Python ({len(py_signing_bytes)} bytes)"))

    # Test 2: Ed25519 deterministic — signing the same message yields same signature
    msg = DOMAIN_SEPARATOR + py_signing_bytes
    py_sig = priv.sign(msg)
    if py_sig.hex() != go_signature_hex:
        print(red("FAIL: signature differs between Go and Python!"))
        print(f"  go: {go_signature_hex}")
        print(f"  py: {py_sig.hex()}")
        return 3
    print(green("PASS: Ed25519 signature byte-identical Go ↔ Python (deterministic signing under same domain separator)"))

    # Test 3: full transmitted CBOR (signed map + key 8 = signature) byte-identical
    py_signature_bytes, py_receipt_cbor = sign_receipt(core, priv)
    if py_receipt_cbor != go_receipt_cbor:
        print(red("FAIL: transmitted receipt CBOR differs!"))
        print(f"  go : {go_receipt_cbor.hex()}")
        print(f"  py : {py_receipt_cbor.hex()}")
        return 4
    print(green(f"PASS: transmitted receipt CBOR byte-identical Go ↔ Python ({len(py_receipt_cbor)} bytes)"))

    # Test 4: Rust libocx-verify accepts both
    go_result = ffi_verify(go_receipt_cbor, pub_raw)
    py_result = ffi_verify(py_receipt_cbor, pub_raw)
    if not go_result.ok:
        print(red(f"FAIL: Rust verifier rejected Go-produced receipt: {go_result.error_name}"))
        return 5
    if not py_result.ok:
        print(red(f"FAIL: Rust verifier rejected Python-produced receipt: {py_result.error_name}"))
        return 6
    print(green(f"PASS: Rust libocx-verify accepts Go receipt   in {go_result.elapsed_us:.0f} µs"))
    print(green(f"PASS: Rust libocx-verify accepts Python receipt in {py_result.elapsed_us:.0f} µs"))

    # Test 5: tamper detection — flip one byte of signature, both should reject
    tampered = bytearray(go_receipt_cbor)
    tampered[-1] ^= 0x01
    tamper_result = ffi_verify(bytes(tampered), pub_raw)
    if tamper_result.ok:
        print(red("FAIL: Rust verifier accepted tampered receipt"))
        return 7
    print(green(f"PASS: tamper detection — flipped byte → {tamper_result.error_name}"))

    # Test 6: wrong public key is rejected
    other_priv = Ed25519PrivateKey.generate()
    other_pub = raw_public_key(other_priv.public_key())
    wrong_key_result = ffi_verify(go_receipt_cbor, other_pub)
    if wrong_key_result.ok:
        print(red("FAIL: Rust verifier accepted receipt under wrong public key"))
        return 8
    print(green(f"PASS: wrong-key detection — different pubkey → {wrong_key_result.error_name}"))

    print()
    print(green("ALL CROSS-LANGUAGE TESTS PASSED"))
    return 0


if __name__ == "__main__":
    sys.exit(main())
