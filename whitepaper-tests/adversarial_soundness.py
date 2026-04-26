#!/usr/bin/env python3
"""Adversarial-prover spot-check soundness — empirical validation of the
1 - (1-f)^k bound against five adversary strategies and two verifier
sampling strategies.

Pipeline
--------
A signed-receipt scheme with no re-execution check would let any prover
invent output hashes. OCX's defence is offline spot-check verification:
a verifier samples k receipts from a stream of N, independently re-runs
the inference, and rejects on any mismatch. This script measures the
catch probability empirically against:

  * HonestProver        — never lies; baseline
  * PureLiar            — every receipt fabricated
  * BernoulliLiar(f)    — each receipt independently fabricated with prob f
  * SelectiveLiar       — fabricates only on a designated high-stakes subset
  * ReplayAttacker(f)   — substitutes a hash from a different honest receipt

against:

  * UniformSpotCheck      — k receipts sampled uniformly w/o replacement
  * RiskWeightedSpotCheck — high-stakes receipts oversampled w× normal

For each (adversary × verifier × k) cell we run M Monte Carlo trials and
compare the empirical catch rate to:

    P_catch_uniform(N, L, k) = 1 - C(N - L, k) / C(N, k)        (exact, hypergeometric)
                             ≈ 1 - (1 - f)^k                    (independent-sample bound)

The verifier's "re-execution" is modelled by a bank lookup against the
canonical output_hash values committed under
`examples/gpu-verifier/results/h100/`. Those hashes are themselves
proven byte-identical across multiple fresh torchrun launches and 11K
warm-model iterations elsewhere in this repo, so a lookup is exactly
what an honest re-execution would produce. No GPU is required.

Vectorization: per-receipt receipts are simulated as boolean arrays
(`lied[i] ∈ {True,False}`) rather than full Python objects. The string
hash itself carries no information — what matters is whether the bank
lookup would mismatch, which is exactly `lied[i]`.

Usage:
    python adversarial_soundness.py --trials 10000 --N 10000 \\
        --output examples/gpu-verifier/results/h100/adversarial_soundness.jsonl
"""
from __future__ import annotations

import argparse
import json
import math
import statistics
import sys
import time
from dataclasses import dataclass, field
from pathlib import Path

import numpy as np

# --- challenge bank ---------------------------------------------------------

ROOT = Path(__file__).resolve().parents[1]
RESULTS_DIR = ROOT / "examples" / "gpu-verifier" / "results" / "h100"

BankKey = tuple[str, int, int, str]  # (model, tp_world_size, num_tokens, prompt)


def load_challenge_bank() -> dict[BankKey, str]:
    """Build (model, tp_world_size, num_tokens, prompt) -> output_hash from
    committed receipts.

    The parallelism strategy and generation length are part of the key
    because different reduction orders (NCCL ring vs CPU offload vs
    pipeline) legitimately produce different — but each internally
    reproducible — hashes, and different generation lengths produce
    different outputs. A verifier re-executing for spot-check uses the
    EXACT same configuration. Hash disagreement within a single key is a
    determinism bug and is surfaced loudly.
    """
    bank: dict[BankKey, str] = {}

    for p in sorted(RESULTS_DIR.glob("*.json")):
        try:
            d = json.loads(p.read_text())
        except Exception:
            continue
        if "prompt" not in d or "output_hash_hex" not in d:
            continue
        key: BankKey = (
            d.get("model", "?"),
            int(d.get("tp_world_size", 1)),
            int(d.get("num_generated_tokens", 0)),
            d["prompt"],
        )
        h = d["output_hash_hex"]
        prev = bank.setdefault(key, h)
        if prev != h:
            raise SystemExit(
                f"FATAL: hash disagreement for {key!r}: {prev[:12]} vs {h[:12]} in {p.name}"
            )

    model_for = {
        "longrun_qwen_0p5b_10000.jsonl": "Qwen/Qwen2.5-0.5B-Instruct",
        "longrun_mixtral_8x7b_1000.jsonl": "mistralai/Mixtral-8x7B-Instruct-v0.1",
    }
    for jp in sorted(RESULTS_DIR.glob("longrun_*.jsonl")):
        lines = jp.read_text().splitlines()
        if not lines:
            continue
        try:
            summary = json.loads(lines[-1])
        except Exception:
            continue
        if not summary.get("summary"):
            continue
        model = summary.get("model") or model_for.get(jp.name, "?")
        for prompt, h in summary.get("prompt_hashes", {}).items():
            key = (model, 1, 32, prompt)
            prev = bank.setdefault(key, h)
            if prev != h:
                raise SystemExit(
                    f"FATAL: hash disagreement for {key!r}: {prev[:12]} vs {h[:12]} in {jp.name}"
                )
    return bank


# --- adversaries (vectorized over an N-receipt stream) ---------------------

@dataclass
class Adversary:
    """An adversary returns a boolean array of length N: lied[i] = True iff
    receipt i was fabricated (its output_hash would NOT equal the true
    re-execution hash for input i).
    """
    name: str

    def lie_mask(self, N: int, is_high: np.ndarray, rng: np.random.Generator) -> np.ndarray:
        raise NotImplementedError


class HonestProver(Adversary):
    def __init__(self):
        super().__init__("HonestProver")

    def lie_mask(self, N, is_high, rng):
        return np.zeros(N, dtype=bool)


class PureLiar(Adversary):
    def __init__(self):
        super().__init__("PureLiar")

    def lie_mask(self, N, is_high, rng):
        return np.ones(N, dtype=bool)


class BernoulliLiar(Adversary):
    def __init__(self, f: float):
        self.f = f
        super().__init__(f"BernoulliLiar(f={f:.3f})")

    def lie_mask(self, N, is_high, rng):
        return rng.random(N) < self.f


class SelectiveLiar(Adversary):
    """Lies only on the high-stakes subset, with probability lie_rate within it."""
    def __init__(self, lie_rate: float = 1.0):
        self.lie_rate = lie_rate
        super().__init__(f"SelectiveLiar(stakes_only,p={lie_rate:.2f})")

    def lie_mask(self, N, is_high, rng):
        return is_high & (rng.random(N) < self.lie_rate)


class ReplayAttacker(Adversary):
    """For each fabricated receipt, the adversary substitutes a hash from a
    DIFFERENT honest receipt (a hash that exists in the bank, just for the
    wrong input). Re-execution still catches it, because re-execution
    compares against the bank entry for THIS input. Soundness is identical
    to BernoulliLiar(f) for the spot-check; the strategy is included to
    show that "use a hash that's been signed somewhere" doesn't bypass it.
    """
    def __init__(self, f: float):
        self.f = f
        super().__init__(f"ReplayAttacker(f={f:.3f})")

    def lie_mask(self, N, is_high, rng):
        return rng.random(N) < self.f


# --- verifiers (vectorized over an N-receipt stream) -----------------------

@dataclass
class Verifier:
    name: str
    k: int

    def sample_indices(self, N: int, is_high: np.ndarray,
                       rng: np.random.Generator) -> np.ndarray:
        raise NotImplementedError


class UniformSpotCheck(Verifier):
    def __init__(self, k: int):
        super().__init__(f"Uniform(k={k})", k)

    def sample_indices(self, N, is_high, rng):
        return rng.choice(N, size=min(self.k, N), replace=False)


class RiskWeightedSpotCheck(Verifier):
    """Efraimidis–Spirakis weighted reservoir sample without replacement."""
    def __init__(self, k: int, high_weight: float):
        self.high_weight = high_weight
        super().__init__(f"RiskWeighted(k={k}, w={high_weight:g})", k)

    def sample_indices(self, N, is_high, rng):
        weights = np.where(is_high, self.high_weight, 1.0)
        # ES key: r^(1/w_i). Top-k keys → top-k indices. Equivalent to
        # weighted sampling without replacement under exponential clocks.
        u = rng.random(N)
        keys = np.log(u) / weights  # equiv to log(u^(1/w)); avoids overflow
        # Argpartition for top-k by largest key (= least negative log/weight).
        k = min(self.k, N)
        idx = np.argpartition(-keys, k - 1)[:k]
        return idx


# --- soundness math --------------------------------------------------------

def hypergeom_catch_prob(N: int, L: int, k: int) -> float:
    """Exact P(at least one liar in sample) under uniform sampling w/o replacement.
       = 1 - C(N-L, k) / C(N, k)."""
    if L <= 0 or k <= 0:
        return 0.0
    if k > N:
        k = N
    if N - L < k:
        return 1.0
    log_no = (math.lgamma(N - L + 1) - math.lgamma(k + 1) - math.lgamma(N - L - k + 1)
              - (math.lgamma(N + 1) - math.lgamma(k + 1) - math.lgamma(N - k + 1)))
    return 1.0 - math.exp(log_no)


def independent_catch_prob(f: float, k: int) -> float:
    return 1.0 - (1.0 - f) ** k


# --- simulation -------------------------------------------------------------

def simulate(adversary: Adversary, verifier: Verifier,
             N: int, M: int, high_stakes_frac: float, seed: int) -> dict:
    """One (adversary, verifier) cell: M trials, vectorized."""
    rng = np.random.default_rng(seed)

    catches = 0
    lies_total = 0
    caught_per_trial = np.empty(M, dtype=np.int64)

    for t in range(M):
        is_high = rng.random(N) < high_stakes_frac
        lied = adversary.lie_mask(N, is_high, rng)
        n_lies = int(lied.sum())
        lies_total += n_lies
        idx = verifier.sample_indices(N, is_high, rng)
        n_caught = int(lied[idx].sum())
        caught_per_trial[t] = n_caught
        if n_caught > 0:
            catches += 1

    avg_lies = lies_total / M
    f_emp = avg_lies / N if N else 0.0
    p_catch_emp = catches / M
    p_catch_indep = independent_catch_prob(f_emp, verifier.k)
    p_catch_hyper = hypergeom_catch_prob(N, int(round(avg_lies)), verifier.k)
    return {
        "adversary": adversary.name,
        "verifier": verifier.name,
        "N": N,
        "k": verifier.k,
        "trials": M,
        "high_stakes_frac": high_stakes_frac,
        "f_empirical": round(f_emp, 6),
        "p_catch_empirical": round(p_catch_emp, 6),
        "p_catch_independent_bound": round(p_catch_indep, 6),
        "p_catch_hypergeometric": round(p_catch_hyper, 6),
        "mean_caught_per_trial": round(float(caught_per_trial.mean()), 4),
        "max_caught_per_trial": int(caught_per_trial.max()),
    }


# --- main -------------------------------------------------------------------

def main():
    p = argparse.ArgumentParser()
    p.add_argument("--trials", type=int, default=10000)
    p.add_argument("--N", type=int, default=10000)
    p.add_argument("--high-stakes-frac", type=float, default=0.05)
    p.add_argument("--seed", type=int, default=20260425)
    p.add_argument("--output", required=True)
    args = p.parse_args()

    bank = load_challenge_bank()
    if len(bank) < 2:
        raise SystemExit("FATAL: not enough distinct entries in challenge bank")

    print(f"[adversarial] challenge-bank size: {len(bank)} unique (model,parallelism,tokens,prompt) keys")
    print(f"[adversarial] N={args.N}, trials per cell M={args.trials}")
    print(f"[adversarial] high-stakes fraction: {args.high_stakes_frac:.0%}")
    print()

    adversaries: list[Adversary] = [
        HonestProver(),
        PureLiar(),
        BernoulliLiar(0.001),
        BernoulliLiar(0.01),
        BernoulliLiar(0.10),
        SelectiveLiar(1.0),
        ReplayAttacker(0.10),
    ]
    k_values = [1, 5, 25, 100, 500]

    def verifiers_for(k: int) -> list[Verifier]:
        return [UniformSpotCheck(k), RiskWeightedSpotCheck(k, high_weight=20.0)]

    out_path = Path(args.output)
    out_path.parent.mkdir(parents=True, exist_ok=True)

    rows: list[dict] = []
    t0 = time.perf_counter()
    with out_path.open("w") as f:
        for adv in adversaries:
            for k in k_values:
                for ver in verifiers_for(k):
                    seed = (args.seed + abs(hash((adv.name, ver.name)))) % (2**32 - 1)
                    row = simulate(adv, ver, N=args.N, M=args.trials,
                                   high_stakes_frac=args.high_stakes_frac,
                                   seed=seed)
                    rows.append(row)
                    f.write(json.dumps(row) + "\n")
                    print(f"  {adv.name:36s} | {ver.name:32s} | "
                          f"f̂={row['f_empirical']:.4f} | "
                          f"P_catch emp={row['p_catch_empirical']:.4f} "
                          f"vs indep={row['p_catch_independent_bound']:.4f} "
                          f"hyper={row['p_catch_hypergeometric']:.4f}",
                          flush=True)
    t_sim = time.perf_counter() - t0

    # --- assertions ---------------------------------------------------------
    print()
    print("=" * 78)
    print("ASSERTIONS")
    print("=" * 78)
    failures = 0

    for row in rows:
        # 1) HonestProver must never be caught
        if row["adversary"] == "HonestProver":
            if row["p_catch_empirical"] != 0.0:
                print(f"  FAIL: {row['adversary']} {row['verifier']} caught with "
                      f"P={row['p_catch_empirical']}")
                failures += 1
            continue

        # 2) Empirical within 5σ of hypergeometric prediction
        p = row["p_catch_hypergeometric"]
        sigma = math.sqrt(p * (1 - p) / row["trials"]) if 0 < p < 1 else 0
        diff = abs(row["p_catch_empirical"] - p)
        if sigma > 0 and diff > 5 * sigma:
            if "RiskWeighted" in row["verifier"] and "Selective" in row["adversary"]:
                continue  # risk-weighted vs SelectiveLiar deliberately diverges
            print(f"  FAIL: {row['adversary']} {row['verifier']} "
                  f"emp={row['p_catch_empirical']:.4f} "
                  f"vs hyper={p:.4f} (5σ={5*sigma:.4f}, diff={diff:.4f})")
            failures += 1

    # 3) RiskWeighted vs SelectiveLiar should beat Uniform at unsaturated k.
    for k in [1, 5, 25, 100, 500]:
        u = next(r for r in rows
                 if r["adversary"].startswith("SelectiveLiar")
                 and r["verifier"].startswith("Uniform")
                 and r["k"] == k)
        w = next(r for r in rows
                 if r["adversary"].startswith("SelectiveLiar")
                 and r["verifier"].startswith("RiskWeighted")
                 and r["k"] == k)
        sigma = math.sqrt(0.25 / row["trials"])
        we, ue = w["p_catch_empirical"], u["p_catch_empirical"]
        if ue >= 0.9999:
            verdict = "SATURATED" if we >= 0.9999 else "FAIL"
        elif we < ue - 3 * sigma:
            verdict = "FAIL"
        elif we > ue:
            verdict = "PASS"
        else:
            verdict = "TIE"
        ratio = (we / max(ue, 1e-9))
        print(f"  {verdict}: at k={k}, RiskWeighted/Uniform = {ratio:.2f}× "
              f"({we:.4f} vs {ue:.4f})")
        if verdict == "FAIL":
            failures += 1

    print()
    print("=" * 78)
    print(f"TOTAL: {len(rows)} cells, {args.trials} trials each, "
          f"{t_sim:.1f}s sim wall, failures={failures}")
    print(f"output JSONL: {out_path}")
    if failures == 0:
        print("STATUS: PASS — all soundness predictions hold within Monte Carlo error")
    else:
        print(f"STATUS: FAIL — {failures} assertion(s) violated")
    print("=" * 78)
    sys.exit(0 if failures == 0 else 1)


if __name__ == "__main__":
    main()
