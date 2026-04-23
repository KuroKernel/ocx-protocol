#!/usr/bin/env python3
"""Cross-architecture determinism test for OCX AI Verifier.

Run this same script on x86 and ARM (e.g. Raspberry Pi) with the same model
file. Compare the printed output_hash. If they match, inference is
architecture-independent. If they differ, SIMD path divergence is real and
the primitive has a portability limit worth documenting.

Usage:
  python3 cross_arch_test.py                         # uses 0.5B default
  python3 cross_arch_test.py MODEL_PATH [PROMPT]     # custom model/prompt

Requires: llama-cpp-python
"""
import hashlib
import json
import platform
import sys
import time

from llama_cpp import Llama

DEFAULT_MODEL = "models/qwen2.5-0.5b-instruct-q4_k_m.gguf"
DEFAULT_PROMPT = "What is the capital of France? Answer in one word:"


def sha256_file(path: str) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        while chunk := f.read(8192):
            h.update(chunk)
    return h.hexdigest()


def sha256_json(obj) -> str:
    return hashlib.sha256(
        json.dumps(obj, sort_keys=True, separators=(",", ":")).encode()
    ).hexdigest()


def main():
    model_path = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_MODEL
    prompt = sys.argv[2] if len(sys.argv) > 2 else DEFAULT_PROMPT

    print("=" * 60)
    print("OCX CROSS-ARCH DETERMINISM TEST")
    print("=" * 60)
    print(f"machine         : {platform.machine()}")
    print(f"processor       : {platform.processor() or 'unknown'}")
    print(f"system          : {platform.system()} {platform.release()}")
    print(f"python          : {sys.version.split()[0]}")
    print(f"model_path      : {model_path}")
    print()

    model_hash = sha256_file(model_path)
    print(f"model_sha256    : {model_hash}")

    input_record = {
        "prompt": prompt,
        "model_hash": model_hash,
        "temperature": 0.0,
        "max_tokens": 100,
        "seed": 42,
    }
    input_hash = sha256_json(input_record)
    print(f"input_hash      : {input_hash}")
    print()

    llm = Llama(
        model_path=model_path,
        n_ctx=512,
        n_threads=1,
        seed=42,
        verbose=False,
    )

    # Warm up (matches verifier)
    _ = llm(prompt, max_tokens=1, temperature=0.0)

    runs = []
    for i in range(3):
        t0 = time.time()
        out = llm(
            prompt,
            max_tokens=100,
            temperature=0.0,
            top_p=1.0,
            top_k=1,
            seed=42,
            echo=False,
        )
        elapsed = int((time.time() - t0) * 1000)
        text = out["choices"][0]["text"]
        output_hash = sha256_json({"response": text})
        runs.append({"output_hash": output_hash, "elapsed_ms": elapsed, "text_len": len(text)})
        print(f"run {i+1}: output_hash={output_hash}  elapsed={elapsed}ms  len={len(text)}")

    print()
    unique = {r["output_hash"] for r in runs}
    if len(unique) == 1:
        print(f"IN-PROCESS DETERMINISTIC on {platform.machine()}: YES")
        print(f"CANONICAL OUTPUT HASH: {runs[0]['output_hash']}")
    else:
        print(f"IN-PROCESS DETERMINISTIC on {platform.machine()}: NO")
        print(f"Got {len(unique)} distinct output hashes: {unique}")

    # Save to JSON for easy diffing across machines
    result = {
        "machine": platform.machine(),
        "processor": platform.processor(),
        "system": f"{platform.system()} {platform.release()}",
        "python": sys.version.split()[0],
        "model_path": model_path,
        "model_sha256": model_hash,
        "input_hash": input_hash,
        "runs": runs,
        "canonical_output_hash": runs[0]["output_hash"] if len(unique) == 1 else None,
        "in_process_deterministic": len(unique) == 1,
    }
    out_file = f"cross_arch_{platform.machine()}.json"
    with open(out_file, "w") as f:
        json.dump(result, f, indent=2)
    print(f"\nResult written to: {out_file}")


if __name__ == "__main__":
    main()
