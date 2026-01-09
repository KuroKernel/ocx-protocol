#!/usr/bin/env python3
"""
Test deterministic inference with llama-cpp-python.
Same input MUST produce same output for OCX verification to work.
"""

from llama_cpp import Llama
import hashlib
import json
import time

MODEL_PATH = "models/qwen2.5-0.5b-instruct-q4_k_m.gguf"

def get_hash(text: str) -> str:
    return hashlib.sha256(text.encode()).hexdigest()[:16]

def run_inference(llm, prompt: str, seed: int = 42) -> dict:
    """Run inference with deterministic settings."""
    start = time.time()

    output = llm(
        prompt,
        max_tokens=100,
        temperature=0.0,      # Greedy decoding - no randomness
        top_p=1.0,            # No nucleus sampling
        top_k=1,              # Only pick top token
        seed=seed,            # Fixed seed
        echo=False
    )

    elapsed = time.time() - start
    response_text = output['choices'][0]['text']

    return {
        "prompt": prompt,
        "prompt_hash": get_hash(prompt),
        "response": response_text,
        "response_hash": get_hash(response_text),
        "elapsed_ms": int(elapsed * 1000),
        "seed": seed
    }

def main():
    print("=" * 60)
    print("OCX AI DETERMINISM TEST")
    print("=" * 60)
    print()

    # Load model
    print("[1/5] Loading model...")
    llm = Llama(
        model_path=MODEL_PATH,
        n_ctx=512,            # Small context for speed
        n_threads=1,          # Single thread for maximum determinism
        seed=42,              # Fixed seed
        verbose=False
    )
    print("      Model loaded!")
    print()

    # Test prompt
    test_prompt = "What is 2 + 2? Answer in one word:"

    # Warm-up run with SAME prompt (first call can have variance)
    print("[2/5] Warm-up run with same prompt (discarded)...")
    _ = run_inference(llm, test_prompt)
    print("      Warm-up complete!")
    print()

    # Run same inference 5 times
    print(f"[3/5] Running same prompt 5 times...")
    print(f"      Prompt: '{test_prompt}'")
    print()

    results = []
    for i in range(5):
        result = run_inference(llm, test_prompt)
        results.append(result)
        print(f"      Run {i+1}: '{result['response'].strip()[:50]}...' (hash: {result['response_hash']})")

    print()

    # Check determinism
    print("[4/5] Checking determinism...")
    hashes = [r['response_hash'] for r in results]
    unique_hashes = set(hashes)

    if len(unique_hashes) == 1:
        print("      DETERMINISTIC! All 5 runs produced identical output.")
        print(f"      Output hash: {hashes[0]}")
        deterministic = True
    else:
        print(f"      NOT DETERMINISTIC! Found {len(unique_hashes)} different outputs:")
        for h in unique_hashes:
            print(f"        - {h}")
        deterministic = False

    print()

    # Summary
    print("[5/5] Summary")
    print("=" * 60)
    print(f"Model:        {MODEL_PATH}")
    print(f"Prompt hash:  {results[0]['prompt_hash']}")
    print(f"Output hash:  {results[0]['response_hash']}")
    print(f"Deterministic: {'YES' if deterministic else 'NO'}")
    print(f"Avg latency:  {sum(r['elapsed_ms'] for r in results) // 5}ms")
    print("=" * 60)

    if deterministic:
        print()
        print("This model CAN be used with OCX for verifiable AI!")
        print("Same prompt + same model = same output = verifiable")

    return deterministic

if __name__ == "__main__":
    main()
