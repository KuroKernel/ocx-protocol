#!/usr/bin/env python3
"""Whitepaper-grade verifier latency + throughput benchmark.

Generates N distinct OCX receipts and times verification through the
Rust libocx-verify.so FFI. Reports mean / median / p99 / p999 / throughput.
"""
from __future__ import annotations

import hashlib
import statistics
import sys
import time
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent / "examples" / "gpu-verifier"))

from canonical_receipt import ReceiptCore, raw_public_key, sign_receipt  # noqa: E402
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey  # noqa: E402
from ffi_verify import verify_receipt  # noqa: E402


def gen_receipts(n: int):
    key = Ed25519PrivateKey.generate()
    pub = raw_public_key(key.public_key())
    now = int(time.time()) - 10
    receipts = []
    for i in range(n):
        suffix = f"-bench-{i}".encode()
        core = ReceiptCore(
            program_hash=hashlib.sha256(b"program" + suffix).digest(),
            input_hash=hashlib.sha256(b"input" + suffix).digest(),
            output_hash=hashlib.sha256(b"output" + suffix).digest(),
            cycles_used=1000 + i,
            started_at=now,
            finished_at=now + 2,
            issuer_id="ocx-bench",
        )
        _, cbor = sign_receipt(core, key)
        receipts.append((cbor, pub))
    return receipts


def main():
    n = int(sys.argv[1]) if len(sys.argv) > 1 else 10000
    print(f"Generating {n} receipts...")
    receipts = gen_receipts(n)
    print(f"  receipt size: {len(receipts[0][0])} bytes")

    # Warm-up
    for cbor, pub in receipts[:100]:
        verify_receipt(cbor, pub)

    # Measure
    t_start = time.perf_counter()
    times = []
    fails = 0
    for cbor, pub in receipts:
        r = verify_receipt(cbor, pub)
        if not r.ok:
            fails += 1
        times.append(r.elapsed_us)
    t_total = time.perf_counter() - t_start

    times.sort()
    mean = statistics.mean(times)
    median = statistics.median(times)
    p99 = times[int(len(times) * 0.99)]
    p999 = times[int(len(times) * 0.999)]
    p_max = times[-1]

    print()
    print(f"Verification benchmark over n={n} receipts:")
    print(f"  mean    : {mean:.1f} µs")
    print(f"  median  : {median:.1f} µs")
    print(f"  p99     : {p99:.1f} µs")
    print(f"  p999    : {p999:.1f} µs")
    print(f"  max     : {p_max:.1f} µs")
    print(f"  failures: {fails}")
    print(f"  throughput (incl Python overhead): {n / t_total:,.0f} receipts/sec")
    print(f"  throughput (FFI work only):        {1e6 / mean:,.0f} receipts/sec/core")


if __name__ == "__main__":
    main()
