#!/usr/bin/env python3
"""Qwen 7B determinism test - single fresh process run, prints hash.

Run this 3 times in fresh processes to compare cross-process determinism at 7B scale.
"""
import hashlib
import json
import sys
import time

from llama_cpp import Llama

# Point at first shard; llama.cpp loads the rest automatically
MODEL_FIRST_SHARD = "models/qwen2.5-7b-instruct-q4_k_m-00001-of-00002.gguf"
PROMPT = "What is the capital of France? Answer in one word:"


def sha256_json(obj) -> str:
    return hashlib.sha256(
        json.dumps(obj, sort_keys=True, separators=(",", ":")).encode()
    ).hexdigest()


def main():
    llm = Llama(
        model_path=MODEL_FIRST_SHARD,
        n_ctx=512,
        n_threads=1,
        seed=42,
        verbose=False,
    )
    # Warm up
    _ = llm(PROMPT, max_tokens=1, temperature=0.0)

    t0 = time.time()
    out = llm(
        PROMPT,
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
    print(f"OUTPUT_HASH={output_hash}")
    print(f"TEXT_LEN={len(text)}")
    print(f"ELAPSED_MS={elapsed}")
    print(f"FIRST_80={text[:80]!r}")


if __name__ == "__main__":
    main()
