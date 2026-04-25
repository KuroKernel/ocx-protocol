#!/usr/bin/env python3
"""Pull every committed receipt JSON in examples/gpu-verifier/results/ and
print a consolidated determinism evidence table for the whitepaper.

For each (model, parallelism, dtype, prompt-class) group, list the
output_hash, logits_hash, and assert byte-equality across the runs in
that group.
"""
from __future__ import annotations

import json
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RESULTS_DIR = ROOT / "examples" / "gpu-verifier" / "results" / "h100"


def classify(filename: str) -> tuple[str, str]:
    """Return (group_key, run_id) for grouping."""
    f = filename
    if f.startswith("r72b_long"):
        return ("Qwen2.5-72B-Instruct / 1×H100 CPU-offload (device_map=auto) / bf16 / long-gen 128t", f)
    if f.startswith("pp_2gpu"):
        return ("Qwen2.5-72B-Instruct / 2×H100 pipeline-parallel (device_map=auto, no offload) / bf16 / long-gen 128t", f)
    if f.startswith("tp_2gpu_short"):
        return ("Qwen2.5-72B-Instruct / 2×H100 tensor-parallel (tp_plan=auto) / bf16 / short-gen 32t", f)
    if f.startswith("tp_2gpu_long"):
        return ("Qwen2.5-72B-Instruct / 2×H100 tensor-parallel (tp_plan=auto) / bf16 / long-gen 128t", f)
    if f.startswith("llama31_70b_tp_short"):
        return ("Meta Llama-3.1-70B-Instruct / 2×H100 tensor-parallel / bf16 / short-gen 32t", f)
    if f.startswith("llama31_70b_tp_long"):
        return ("Meta Llama-3.1-70B-Instruct / 2×H100 tensor-parallel / bf16 / long-gen 128t", f)
    return ("OTHER", f)


def main():
    groups: dict[str, list[dict]] = defaultdict(list)
    for path in sorted(RESULTS_DIR.glob("*.json")):
        try:
            data = json.loads(path.read_text())
        except Exception:
            continue
        key, run_id = classify(path.name)
        groups[key].append({"run_id": run_id, **data})

    print("=" * 78)
    print("OCX DETERMINISM EVIDENCE (committed receipts on disk)")
    print("=" * 78)

    overall_pass = True
    for group_key, runs in groups.items():
        if group_key == "OTHER":
            continue
        print(f"\n## {group_key}")
        print(f"   runs: {len(runs)}")
        out_hashes = {r["output_hash_hex"] for r in runs}
        log_hashes = {r["logits_hash_hex"] for r in runs}
        texts = {r["generated_text"] for r in runs}
        all_verify = all(r.get("verify_ok", False) for r in runs)
        elapsed_list = [r.get("elapsed_s", 0.0) for r in runs]
        verify_us_list = [r.get("verify_elapsed_us", 0.0) for r in runs]

        out_match = len(out_hashes) == 1
        log_match = len(log_hashes) == 1
        text_match = len(texts) == 1

        status = "PASS" if (out_match and log_match and text_match and all_verify) else "FAIL"
        if status == "FAIL":
            overall_pass = False

        print(f"   output_hash : {next(iter(out_hashes))[:32]}...  match={out_match}")
        print(f"   logits_hash : {next(iter(log_hashes))[:32]}...  match={log_match}")
        print(f"   text        : {repr(next(iter(texts)))[:80]}...  match={text_match}")
        print(f"   verify_ok   : {all_verify}  (mean {sum(verify_us_list)/len(verify_us_list):.0f} µs)")
        print(f"   inference_s : min={min(elapsed_list):.2f}, max={max(elapsed_list):.2f}")
        print(f"   STATUS      : {status}")

    print()
    print("=" * 78)
    print(f"OVERALL: {'PASS' if overall_pass else 'FAIL'} across {len(groups)-1 if 'OTHER' in groups else len(groups)} test groups")
    print("=" * 78)
    return 0 if overall_pass else 1


if __name__ == "__main__":
    raise SystemExit(main())
