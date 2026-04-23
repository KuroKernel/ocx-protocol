"""Cross-process determinism + receipt-verify bench for ocx_gpu_verifier.py.

Runs the GPU verifier in N fresh subprocesses with the same prompt + seed,
compares output_hash and logits_hash byte-for-byte. Then benchmarks the Rust
offline verifier over a hundred receipts.

Success criteria (from the plan):
    S1: output_hash + logits_hash byte-identical across fresh processes
    S2: every receipt verifies via ocx_verify_receipt_detailed (OCX_SUCCESS)
    S3: median verify time < 5ms, p99 < 5ms
"""
from __future__ import annotations

import argparse
import json
import statistics
import subprocess
import sys
import time
from pathlib import Path

from canonical_receipt import raw_public_key
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from ffi_verify import OCX_INVALID_SIGNATURE, OCX_SUCCESS, verify_receipt

SCRIPT = Path(__file__).resolve().parent / "ocx_gpu_verifier.py"
PY = sys.executable  # same interpreter as this process


def run_once(
    prompt: str,
    seed: int,
    max_new_tokens: int,
    dtype: str = "float32",
    attn: str = "eager",
) -> dict:
    """Spawn a fresh subprocess of ocx_gpu_verifier.py --once and parse JSON."""
    out = subprocess.run(
        [
            PY, str(SCRIPT),
            "--once",
            "--prompt", prompt,
            "--seed", str(seed),
            "--max-new-tokens", str(max_new_tokens),
            "--dtype", dtype,
            "--attn", attn,
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    if out.returncode != 0:
        sys.stderr.write(out.stderr)
        raise SystemExit(f"subprocess failed (rc={out.returncode})")
    # Last line is the JSON summary
    lines = [l for l in out.stdout.strip().splitlines() if l.strip().startswith("{")]
    if not lines:
        raise SystemExit(f"no JSON in subprocess stdout: {out.stdout[:200]}")
    return json.loads(lines[-1])


def test_determinism(
    prompt: str,
    seed: int,
    max_new_tokens: int,
    n: int,
    dtype: str = "float32",
    attn: str = "eager",
) -> bool:
    print(f"Running {n} fresh subprocesses of ocx_gpu_verifier.py...")
    print(f"  prompt      : {prompt!r}")
    print(f"  seed        : {seed}")
    print(f"  max_new     : {max_new_tokens}")
    print(f"  dtype       : {dtype}")
    print(f"  attn        : {attn}")
    print()

    runs = []
    for i in range(n):
        t0 = time.perf_counter()
        r = run_once(prompt, seed, max_new_tokens, dtype=dtype, attn=attn)
        elapsed = time.perf_counter() - t0
        print(
            f"  run {i+1}: output={r['output_hash_hex'][:16]}... "
            f"logits={r['logits_hash_hex'][:16]}... "
            f"verify={r['verify_ok']} ({r['verify_elapsed_us']}us) "
            f"wall={elapsed:.1f}s"
        )
        runs.append(r)

    print()
    output_hashes = {r["output_hash_hex"] for r in runs}
    logits_hashes = {r["logits_hash_hex"] for r in runs}
    texts = {r["generated_text"] for r in runs}
    program_hashes = {r["program_hash_hex"] for r in runs}

    if len(program_hashes) != 1:
        print(f"WARN: model weights differ across runs ({len(program_hashes)} unique)")

    deterministic_output = len(output_hashes) == 1
    deterministic_logits = len(logits_hashes) == 1

    if deterministic_output and deterministic_logits:
        print("S1 PASS — output_hash AND logits_hash byte-identical across processes")
        print(f"  canonical output_hash : {runs[0]['output_hash_hex']}")
        print(f"  canonical logits_hash : {runs[0]['logits_hash_hex']}")
        print(f"  generated text        : {runs[0]['generated_text']!r}")
        return True

    print("S1 FAIL — nondeterminism detected across fresh processes")
    if not deterministic_output:
        print(f"  distinct output_hash : {len(output_hashes)}")
        for h in output_hashes:
            print(f"    {h}")
    if not deterministic_logits:
        print(f"  distinct logits_hash : {len(logits_hashes)}")
    if len(texts) > 1:
        print(f"  distinct texts       : {len(texts)}")
        for t in texts:
            print(f"    {t!r}")
    else:
        print("  (texts identical but logits differ — subtle float drift)")
    return False


def test_verify_bench(receipts: list[tuple[bytes, bytes]]) -> bool:
    """Time 100 verifications of distinct receipts. Tamper-test one."""
    print(f"Benchmarking Rust offline verifier over {len(receipts)} receipts...")

    all_ok = True
    timings: list[float] = []
    for cbor_bytes, pubkey in receipts:
        r = verify_receipt(cbor_bytes, pubkey)
        if not r.ok:
            all_ok = False
            print(f"  unexpected failure: {r.error_name}")
        timings.append(r.elapsed_us)

    timings.sort()
    median = statistics.median(timings)
    p99 = timings[int(len(timings) * 0.99)] if len(timings) >= 100 else timings[-1]
    mean = statistics.mean(timings)
    print(f"  n={len(timings)} mean={mean:.0f}us median={median:.0f}us p99={p99:.0f}us")

    s3 = p99 < 5000  # 5ms in microseconds
    if all_ok and s3:
        print("S2 + S3 PASS — all verifies OK and p99 < 5ms")
    elif not all_ok:
        print("S2 FAIL — at least one receipt did not verify")
        return False
    elif not s3:
        print(f"S3 FAIL — p99 {p99:.0f}us >= 5000us")
        return False

    # Tamper: flip last byte of first receipt's signature, should fail cleanly.
    cbor0, pub0 = receipts[0]
    tampered = bytearray(cbor0)
    tampered[-1] ^= 0x01
    tres = verify_receipt(bytes(tampered), pub0)
    if not tres.ok and tres.error_code == OCX_INVALID_SIGNATURE:
        print("TAMPER PASS — flipped byte detected (OCX_INVALID_SIGNATURE)")
    else:
        print(f"TAMPER FAIL — got ok={tres.ok} err={tres.error_name}")
        return False

    return True


def generate_test_receipts(n: int) -> list[tuple[bytes, bytes]]:
    """Generate N distinct synthetic receipts for the verify-bench.

    NOTE: these are synthetic (no GPU inference). The determinism test covers
    the inference-generated receipts separately. Here we just want to
    benchmark verify() over many distinct inputs.
    """
    import hashlib
    from canonical_receipt import ReceiptCore, sign_receipt

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
        _, cbor_bytes = sign_receipt(core, key)
        receipts.append((cbor_bytes, pub))
    return receipts


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--mode",
        choices=["all", "determinism", "verify"],
        default="all",
        help="Which tests to run",
    )
    parser.add_argument("--prompt", default="What is the capital of France? Answer in one word:")
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--max-new-tokens", type=int, default=64)
    parser.add_argument("--runs", type=int, default=3, help="Number of fresh subprocesses for determinism test")
    parser.add_argument("--bench-n", type=int, default=100, help="Number of receipts for verify bench")
    parser.add_argument("--dtype", default="float32", help="Model dtype for determinism test")
    parser.add_argument("--attn", default="eager", help="Attention implementation")
    args = parser.parse_args()

    overall = True

    if args.mode in ("all", "determinism"):
        print("=" * 70)
        print("TEST 1/2 — DETERMINISM (S1)")
        print("=" * 70)
        overall &= test_determinism(
            args.prompt, args.seed, args.max_new_tokens, args.runs,
            dtype=args.dtype, attn=args.attn,
        )
        print()

    if args.mode in ("all", "verify"):
        print("=" * 70)
        print("TEST 2/2 — VERIFY BENCHMARK (S2 + S3)")
        print("=" * 70)
        receipts = generate_test_receipts(args.bench_n)
        overall &= test_verify_bench(receipts)
        print()

    print("=" * 70)
    if overall:
        print("ALL TESTS PASSED")
        return 0
    print("ONE OR MORE TESTS FAILED")
    return 1


if __name__ == "__main__":
    sys.exit(main())
