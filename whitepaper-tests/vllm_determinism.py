"""vLLM determinism test on H100. Five identical greedy requests within a single
LLM instance, then dump output_hash for each.

Run as: python3 vllm_determinism.py [--model M] [--prompt P] [--max-tokens N]

Output: JSON line per request with output_hash + decoded text.
"""
import argparse, hashlib, json, os, sys, time

os.environ.setdefault("HF_HUB_ENABLE_HF_TRANSFER", "1")

from vllm import LLM, SamplingParams

ap = argparse.ArgumentParser()
ap.add_argument("--model", default="Qwen/Qwen2.5-7B-Instruct")
ap.add_argument("--prompt", default="Write a short poem about deterministic computation, exactly 4 lines. Just the poem:")
ap.add_argument("--max-tokens", type=int, default=128)
ap.add_argument("--n-requests", type=int, default=5)
ap.add_argument("--enforce-eager", action="store_true")
ap.add_argument("--output", default=None, help="Write JSONL to this path")
args = ap.parse_args()

print(f"[vllm-det] model={args.model}", flush=True)
print(f"[vllm-det] prompt={args.prompt!r}", flush=True)
print(f"[vllm-det] max_tokens={args.max_tokens} n_requests={args.n_requests} enforce_eager={args.enforce_eager}", flush=True)

t0 = time.perf_counter()
llm = LLM(
    model=args.model,
    dtype="bfloat16",
    seed=42,
    enforce_eager=args.enforce_eager,
    gpu_memory_utilization=0.9,
)
t_load = time.perf_counter() - t0
print(f"[vllm-det] load took {t_load:.1f}s", flush=True)

sp = SamplingParams(
    temperature=0.0,
    top_p=1.0,
    top_k=-1,
    max_tokens=args.max_tokens,
    seed=42,
)

results = []
for i in range(args.n_requests):
    t1 = time.perf_counter()
    out = llm.generate([args.prompt], sp)
    t_gen = time.perf_counter() - t1
    o = out[0]
    text = o.outputs[0].text
    token_ids = list(o.outputs[0].token_ids)
    h = hashlib.sha256()
    for tid in token_ids:
        h.update(tid.to_bytes(8, "little"))
    h.update(text.encode("utf-8"))
    output_hash = h.hexdigest()
    rec = {
        "iter": i,
        "n_tokens": len(token_ids),
        "elapsed_s": round(t_gen, 4),
        "output_hash": output_hash,
        "text": text,
    }
    results.append(rec)
    print(f"[vllm-det] iter {i}: tokens={len(token_ids)} hash={output_hash[:16]}... t={t_gen:.2f}s", flush=True)

if args.output:
    with open(args.output, "w") as f:
        for r in results:
            f.write(json.dumps(r) + "\n")
    print(f"[vllm-det] wrote {len(results)} records to {args.output}", flush=True)

unique = set(r["output_hash"] for r in results)
print(f"\n[vllm-det] WITHIN-LAUNCH RESULT: {len(unique)} unique hash among {len(results)} requests")
print(f"  hashes: {sorted(unique)}")
sys.exit(0 if len(unique) == 1 else 1)
