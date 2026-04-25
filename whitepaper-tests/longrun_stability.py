#!/usr/bin/env python3
"""Long-run warm-model determinism + drift test.

Loads a model ONCE, then runs N inferences in a tight loop. For each
distinct prompt, asserts that all output_hashes are byte-identical
across all iterations. Logs per-iteration timing, GPU memory, and GPU
temperature so any thermal or memory drift would be visible.

This complements the cross-process determinism test (which proves fresh-
state byte-identity via torchrun): this test proves that within a single
warm-loaded model session, accumulated state (CUDA streams, KV cache
allocator state, cudnn/cublas workspace) does not break byte-identity.

Usage:
    python longrun_stability.py --model mistralai/Mixtral-8x7B-Instruct-v0.1 \
        --iterations 1000 --max-new-tokens 32 --output /tmp/longrun.jsonl
"""
from __future__ import annotations

# Determinism flags — set BEFORE torch import
import os
os.environ.setdefault("CUBLAS_WORKSPACE_CONFIG", ":4096:8")
os.environ.setdefault("CUDA_LAUNCH_BLOCKING", "0")
os.environ.setdefault("TOKENIZERS_PARALLELISM", "false")

import argparse
import hashlib
import json
import statistics
import subprocess
import sys
import time
from pathlib import Path

import torch
from transformers import AutoModelForCausalLM, AutoTokenizer

# Rotating prompts (10 distinct ones; each one runs N/10 times)
PROMPTS = [
    "What is the capital of France? Answer in one word:",
    "What is 2 plus 2? Answer in one word:",
    "Name a color in three letters or fewer:",
    "What year did the Berlin Wall fall? Answer in digits only:",
    "What is the boiling point of water in Celsius? Answer in digits only:",
    "Name the largest planet in our solar system in one word:",
    "What language is spoken in Brazil? Answer in one word:",
    "What is the chemical symbol for gold? Answer in one or two letters:",
    "Who wrote Hamlet? Answer with last name only:",
    "What is the speed of light in m/s, in scientific notation? Answer briefly:",
]


def gpu_temp_mem():
    """Return (temp_c, memory_used_mib) for GPU 0 via nvidia-smi."""
    try:
        out = subprocess.run(
            ["nvidia-smi",
             "--query-gpu=temperature.gpu,memory.used",
             "--format=csv,noheader,nounits", "-i", "0"],
            capture_output=True, text=True, timeout=5,
        )
        line = out.stdout.strip().split("\n")[0]
        t, m = [x.strip() for x in line.split(",")]
        return int(t), int(m)
    except Exception:
        return -1, -1


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--model", required=True)
    p.add_argument("--iterations", type=int, default=1000)
    p.add_argument("--max-new-tokens", type=int, default=32)
    p.add_argument("--seed", type=int, default=42)
    p.add_argument("--dtype", default="bf16", choices=["bf16", "fp16", "fp32"])
    p.add_argument("--device-map", default="auto",
                   help="cuda:0 to pin one GPU; auto to spread across all available")
    p.add_argument("--output", required=True, help="JSONL path for per-iteration log")
    p.add_argument("--temp-every", type=int, default=25, help="Log GPU temp every N iterations")
    args = p.parse_args()

    dtype = {"bf16": torch.bfloat16, "fp16": torch.float16, "fp32": torch.float32}[args.dtype]

    # Determinism setup
    torch.use_deterministic_algorithms(True, warn_only=True)
    torch.backends.cudnn.deterministic = True
    torch.backends.cudnn.benchmark = False
    torch.backends.cuda.matmul.allow_tf32 = False
    torch.backends.cudnn.allow_tf32 = False
    torch.manual_seed(args.seed)
    torch.cuda.manual_seed_all(args.seed)

    print(f"[longrun] loading {args.model} (dtype={args.dtype}, device_map={args.device_map})...")
    t_load_start = time.perf_counter()
    tok = AutoTokenizer.from_pretrained(args.model)
    device_map = args.device_map if args.device_map == "auto" else {"": args.device_map}
    # device_map="auto" + tp_plan are mutually exclusive in transformers. We use
    # device_map="auto" (pipeline-style across available GPUs) for the long-run
    # warm-model stability test. Cross-process tensor-parallel byte-identity is
    # already proven via torchrun in the cross-process tests.
    model = AutoModelForCausalLM.from_pretrained(
        args.model,
        torch_dtype=dtype,
        attn_implementation="eager",
        device_map=device_map,
    )
    model.eval()
    t_load = time.perf_counter() - t_load_start
    print(f"[longrun] loaded in {t_load:.1f}s. VRAM (rank 0): {torch.cuda.memory_allocated()/1e9:.2f} GB")

    # Get a deterministic device for inputs
    in_device = next(iter(model.parameters())).device

    # Per-prompt expected hash (set on first encounter); subsequent encounters MUST match
    expected_hash: dict[str, str] = {}
    fail_count = 0
    iter_times = []

    log_path = Path(args.output)
    log_path.parent.mkdir(parents=True, exist_ok=True)

    print(f"[longrun] starting {args.iterations} iterations across {len(PROMPTS)} prompt classes")
    print(f"[longrun] log path: {log_path}")

    t_start_all = time.perf_counter()
    with log_path.open("w") as logf:
        for i in range(args.iterations):
            prompt = PROMPTS[i % len(PROMPTS)]
            torch.manual_seed(args.seed)
            torch.cuda.manual_seed_all(args.seed)
            inputs = tok(prompt, return_tensors="pt").to(in_device)

            t0 = time.perf_counter()
            with torch.inference_mode():
                out = model.generate(
                    **inputs,
                    max_new_tokens=args.max_new_tokens,
                    do_sample=False,
                    temperature=0.0,
                    top_p=1.0,
                    num_beams=1,
                )
            t_iter = time.perf_counter() - t0
            iter_times.append(t_iter)

            generated = out[0][inputs.input_ids.shape[1]:]
            text = tok.decode(generated, skip_special_tokens=True)
            output_hash = hashlib.sha256(
                generated.cpu().numpy().tobytes() + text.encode("utf-8")
            ).hexdigest()

            # Check against the expected hash for this prompt
            prev = expected_hash.setdefault(prompt, output_hash)
            match = (output_hash == prev)
            if not match:
                fail_count += 1

            # Periodic temp/mem
            if i % args.temp_every == 0 or i == args.iterations - 1:
                temp, mem = gpu_temp_mem()
            else:
                temp, mem = None, None

            entry = {
                "iter": i,
                "prompt_idx": i % len(PROMPTS),
                "output_hash": output_hash,
                "match_expected": match,
                "iter_seconds": round(t_iter, 4),
            }
            if temp is not None:
                entry["gpu_temp_c"] = temp
                entry["gpu_mem_mib"] = mem
            logf.write(json.dumps(entry) + "\n")

            if i % 100 == 0 or i == args.iterations - 1:
                pct = 100 * (i + 1) / args.iterations
                msg = f"[longrun] iter {i+1}/{args.iterations} ({pct:.0f}%) - {t_iter*1000:.0f}ms"
                if temp is not None:
                    msg += f" - GPU {temp}C, {mem}MiB used"
                if fail_count:
                    msg += f" - FAILS={fail_count}"
                print(msg, flush=True)

    t_total = time.perf_counter() - t_start_all
    print()
    print("=" * 70)
    print(f"FINISHED — {args.iterations} iterations in {t_total:.1f}s ({args.iterations/t_total:.1f}/s)")
    print(f"  per-prompt distinct hashes: {len(expected_hash)} (expected {len(PROMPTS)})")
    print(f"  byte-identity failures: {fail_count}")
    print(f"  per-iter latency: median={statistics.median(iter_times)*1000:.0f}ms "
          f"mean={statistics.mean(iter_times)*1000:.0f}ms "
          f"max={max(iter_times)*1000:.0f}ms "
          f"min={min(iter_times)*1000:.0f}ms")
    print(f"  total wall time: {t_total/60:.1f} min")
    if fail_count == 0:
        print(f"  STATUS: PASS — zero byte-identity failures over {args.iterations} iterations")
    else:
        print(f"  STATUS: FAIL — {fail_count} iterations produced different output")
    print("=" * 70)
    print(f"  per-prompt expected hashes (each must reproduce {args.iterations//len(PROMPTS)} times):")
    for i, (p_text, h) in enumerate(expected_hash.items()):
        print(f"    [{i}] {h}  ← {p_text!r}")

    # Final summary appended to log
    with log_path.open("a") as logf:
        logf.write(json.dumps({
            "summary": True,
            "iterations": args.iterations,
            "duration_s": round(t_total, 2),
            "throughput_per_s": round(args.iterations / t_total, 2),
            "fails": fail_count,
            "prompt_hashes": expected_hash,
            "model": args.model,
            "dtype": args.dtype,
            "device_map": args.device_map,
        }) + "\n")

    sys.exit(0 if fail_count == 0 else 1)


if __name__ == "__main__":
    main()
