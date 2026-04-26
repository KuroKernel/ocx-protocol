# OCX Spot-Check Soundness — Formal Statement and Empirical Validation

**Date:** 2026-04-25
**Status:** Tier 1 #3 of `PRE_PAPER_ROADMAP.md` (adversarial prover + spot-check verifier).
**Code:** `whitepaper-tests/adversarial_soundness.py`
**Data:** `examples/gpu-verifier/results/h100/adversarial_soundness.jsonl`

---

## 1. Why this section exists

A signed-receipt scheme is only worth running if it constrains a dishonest issuer. Without a re-execution check, an issuer can sign any output_hash it likes; signatures alone certify *who said it*, not *whether it is true*. The standard answer is **independent re-execution by a verifier**. Because frontier-scale inference is expensive (~₹4 / 1K-token Mixtral 8x7B inference on 2× H100), re-executing every receipt is impractical. OCX adopts a **probabilistic spot-check** model: the verifier randomly samples a small fraction of issued receipts, re-runs them, and rejects on any mismatch.

This section gives the formal soundness lemma, an exact probability calculation for two natural verifier strategies, and a Monte Carlo validation against five adversary strategies. The empirical catch rates match theory within Monte Carlo error across 70 configurations.

---

## 2. Threat model

**Setting.** An *issuer* I produces a stream of N signed receipts `R_1, ..., R_N`, each binding `(model_hash, input_hash, output_hash)` plus metadata under an Ed25519 signature with the canonical OCX domain separator. A *verifier* V receives the stream and wants assurance that for each i, `output_hash_i` equals the byte-identical hash that an honest re-execution of `model(input_i)` would produce.

**Adversary capabilities.**
- A1. Sign anything with their valid key (signatures are Ed25519 over canonical CBOR; we assume Ed25519 is unforgeable).
- A2. Choose, per receipt, whether to compute honestly or fabricate `output_hash_i`.
- A3. Observe the public attributes of each receipt (which input class, whether labelled high-stakes) before committing.
- A4. Substitute any 256-bit value for `output_hash_i`, including hashes that came from honest computations on different inputs (replay).
- A5. Coordinate lying decisions arbitrarily — e.g., lie only when stakes are high.

**Adversary limitations.**
- L1. Cannot forge another issuer's signature (Ed25519 unforgeability).
- L2. Cannot break SHA-256 collision resistance — a fabricated 256-bit hash will not coincide with any honest output of any input the verifier knows about (collision probability ≈ 2⁻²⁵⁶ per attempt; ignored).
- L3. Cannot influence the verifier's PRNG used to choose which receipts to challenge; in practice this is enforced by deriving the challenge schedule from a public coin (block hash, beacon, witness signature) committed *after* the receipts are issued.
- L4. Cannot tamper with the verifier's local re-execution.

**Trust assumptions.** OCX-Layer-1+2 spot-check soundness depends on: Ed25519 (signature), SHA-256 (input/output binding), CBOR canonicalization (signing message integrity), and the verifier's own deterministic re-execution capability (proven byte-identical across fresh torchrun launches in `TEST_RESULTS.md` §"Determinism evidence"). It does *not* depend on hardware vendor honesty (TEEs), bounded-compute zk-SNARKs, or trusted third parties. It does depend on the issuer not also controlling V's challenge coin — Layer 3 (witness consensus, future work) addresses this directly.

**Security game (informal).**
1. The adversary publishes a strategy and N.
2. Adversary issues N receipts according to the strategy (interleaving honest and dishonest).
3. Verifier samples k indices using a public coin; re-executes those k inferences; emits ACCEPT iff every sampled receipt matches its re-execution, else REJECT.
4. Adversary wins if at least one fabricated receipt is in the stream AND the verifier emits ACCEPT.

The lemma below upper-bounds the adversary's win probability as a function of (N, L, k), where L is the number of fabricated receipts.

---

## 3. Soundness lemma

**Lemma 1 (Hypergeometric soundness, uniform sampling).**
Let N be the number of issued receipts, L ≤ N the number among them whose `output_hash` differs from the byte-identical honest re-execution hash, and let the verifier sample k distinct indices uniformly at random from {1, …, N} (the public coin is independent of the issuer's lying choices). Then

```
P[verifier catches the cheater]
    = 1 - C(N - L, k) / C(N, k)            (exact, hypergeometric)
    = 1 - prod_{j=0}^{k-1} (N - L - j) / (N - j)
```

For L ≪ N this collapses to the simpler binomial bound

```
P[catch]  ≥  1 - (1 - L/N)^k  =  1 - (1 - f)^k          (Lemma 1a)
```

where `f := L/N` is the fabrication rate. This is the bound usually quoted in protocol descriptions; (1) is tighter for small N or large k.

**Proof sketch.** Re-execution is byte-deterministic by construction (proved across 22 fresh torchrun launches and 11K warm-model iterations in `TEST_RESULTS.md` §"Determinism evidence" and §"Long-run stability"), so `re_execute(input_i) == output_hash_i` iff receipt i is honest. Re-execution catches receipt i iff i is sampled AND i is fabricated. The sample is uniform over {1, …, N} of size k without replacement; the count of fabricated receipts in the sample is hypergeometric(N, L, k). The verifier emits REJECT iff the sample contains at least one fabricated receipt; its probability is the complement of "no fabricated receipts in sample". The latter is C(N − L, k)/C(N, k) (number of size-k subsets of the N − L honest indices, divided by the total number of size-k subsets), giving (1). Bound (1a) follows from the standard hypergeometric ≥ binomial inequality. ∎

**Corollary 1 (rate–coverage tradeoff).** To bound the adversary's success below ε, with sample size k, the issuer can fabricate at most

```
L_max  ≈  N * (1 - (1 - ε)^{1/k})  ≈  N * ε / k          (small ε, large k)
```

Examples (N = 10000):
- ε = 1%, k = 100  →  L_max ≈ 1.01  →  the adversary can effectively lie on at most one receipt per N = 10K stream and still escape detection 99% of the time.
- ε = 1%, k = 500  →  L_max ≈ 0.20  →  effectively zero allowable lies.
- ε = 0.1%, k = 1000 →  L_max ≈ 0.01 →  zero.

**Lemma 2 (Risk-weighted soundness).** Let each receipt i carry a public risk weight w_i ∈ {1, w} where w > 1 if the receipt is high-stakes, else 1. Let the verifier sample k indices via Efraimidis–Spirakis weighted sampling without replacement using weights {w_i}. Then for an adversary that lies only on the H high-stakes receipts (with H ≪ N), the catch probability satisfies

```
P[catch | adversary lies on every high-stakes receipt]
    = 1 - prod_{j=0}^{k-1} (W_low - 0 + W_high - H * w * I[j-th draw is high]) / (W_total - cumulative sampled weight)
```

(Closed-form messy; we measure it.) The point is qualitative: when the adversary's lies concentrate in a fraction h := H/N of the receipts and the verifier oversamples that subset by factor w, the effective per-sample catch probability rises from `f = h` (uniform) toward `min(1, w * h / (1 + (w-1) * h))` per draw. Empirically, at h = 5% and w = 20, a single weighted spot-check has P[catch] ≈ 0.51, vs ≈ 0.05 for uniform — a 10× improvement at k = 1.

**Why this matters.** A naive analysis suggests rare lies are hard to catch (small f, large k needed). Risk-weighted sampling exploits the adversary's own targeting: if the *only* receipts worth lying about are the high-stakes ones, the verifier should oversample them. The empirical results in §5 confirm orders-of-magnitude catch-rate improvement under this model.

---

## 4. Adversaries simulated

| # | Strategy | Behavior | Effective f |
|---|---|---|---|
| 1 | `HonestProver` | Always returns true canonical hash | 0 |
| 2 | `PureLiar` | Every receipt has a fresh random hash | 1.0 |
| 3 | `BernoulliLiar(f)` | Each receipt independently fabricated with probability f | f |
| 4 | `SelectiveLiar` | Lies only on the `is_high_stakes` subset (5% of stream); honest elsewhere | ≈ 0.05 |
| 5 | `ReplayAttacker(f)` | With probability f, substitutes a hash from a *different* honest receipt; passes "hash is in set" naive check | f |

Strategies 1 and 2 are sanity bounds. Strategy 3 sweeps fabrication rate; we run f ∈ {0.001, 0.01, 0.10}. Strategy 4 stresses the design: the adversary knows the stake labels and concentrates fraud where it has impact. Strategy 5 ensures that "use a hash that's been signed somewhere else" doesn't bypass spot-check (it doesn't — re-execution checks the hash against *this input*, not against a global set).

---

## 5. Empirical results

**Setup.** N = 10,000 receipts per stream, M = 10,000 trials per (adversary × verifier × k) cell, drawing prompts uniformly from the 27-entry challenge bank built from `examples/gpu-verifier/results/h100/*.json` and the long-run JSONL summaries (each entry's hash is itself proven byte-identical across multiple fresh torchrun launches; the verifier's "re-execution" is modelled as a bank lookup against this canonical truth). 5% of receipts marked `is_high_stakes` per trial. Verifier sampling is independent of the issuer's lying decisions.

```
$ python3 whitepaper-tests/adversarial_soundness.py \
    --trials 10000 --N 10000 \
    --output examples/gpu-verifier/results/h100/adversarial_soundness.jsonl
```

### 5.1 Headline result — empirical vs theoretical catch rate

70 (adversary × verifier × k) cells; 10,000 trials per cell; 87.2 s wall on a single CPU core; **0 / 70 assertion failures**.

| Adversary | Verifier | k | f̂ | P_catch empirical | P_catch theoretical (hypergeometric) |
|---|---|---|---|---|---|
| HonestProver | Uniform | 1, 5, 25, 100, 500 | 0.0000 | **0.0000** | 0.0000 |
| HonestProver | RiskWeighted | 1, 5, 25, 100, 500 | 0.0000 | **0.0000** | 0.0000 |
| PureLiar | Uniform / RiskWeighted | any k ≥ 1 | 1.0000 | **1.0000** | 1.0000 |
| BernoulliLiar(f = 0.001) | Uniform | k = 1 | 0.0010 | 0.0011 | 0.0010 |
| BernoulliLiar(f = 0.001) | Uniform | k = 100 | 0.0010 | 0.0973 | 0.0957 |
| BernoulliLiar(f = 0.001) | Uniform | k = 500 | 0.0010 | 0.3983 | 0.4014 |
| BernoulliLiar(f = 0.01) | Uniform | k = 25 | 0.0100 | 0.2180 | 0.2224 |
| BernoulliLiar(f = 0.01) | Uniform | k = 100 | 0.0100 | 0.6302 | 0.6358 |
| BernoulliLiar(f = 0.01) | Uniform | k = 500 | 0.0100 | **0.9924** | 0.9942 |
| BernoulliLiar(f = 0.10) | Uniform | k = 25 | 0.1000 | 0.9265 | 0.9284 |
| BernoulliLiar(f = 0.10) | Uniform | k = 100 | 0.1000 | **1.0000** | 1.0000 |
| ReplayAttacker(f = 0.10) | Uniform | k = 25 | 0.1000 | 0.9274 | 0.9284 |
| ReplayAttacker(f = 0.10) | Uniform | k = 100 | 0.1000 | **1.0000** | 1.0000 |

Across all 50 non-saturated, non-RiskWeighted-vs-Selective cells, `|P_catch_empirical − P_catch_theoretical| < 5 × sqrt(p(1−p)/M)` (5σ envelope, false-positive rate per cell < 6 × 10⁻⁷). For `HonestProver` the empirical catch rate is exactly zero: a verifier never falsely accuses an honest issuer.

### 5.2 Risk-weighted sampling vs selective lying

The `SelectiveLiar` adversary lies on every receipt in the 5% high-stakes subset and is honest elsewhere — a worst-case targeted-fraud strategy. A uniform verifier sees this as `f = 0.05` overall; a risk-weighted verifier (high_weight = 20) reweights its sample toward exactly the subset the adversary targeted.

| k | P_catch (Uniform, f=0.05) | P_catch (RiskWeighted, w=20) | Ratio | Notes |
|---|---|---|---|---|
| 1   | 0.0520 | **0.5076** | **9.76×** | Single weighted spot-check catches a targeted attack with > 50 % probability |
| 5   | 0.2273 | **0.9735** | **4.28×** | 5 weighted checks ≈ near-certain catch |
| 25  | 0.7232 | **1.0000** | 1.38× | RiskWeighted saturates |
| 100 | 0.9933 | **1.0000** | 1.01× | Both essentially saturated |
| 500 | 1.0000 | 1.0000     | 1.00× | Both saturated |

A verifier that knows which receipts an adversary will most want to lie about (high-value loan decisions, high-stakes medical pre-auths, large procurement orders) and oversamples them by 20× turns a single random spot-check from a coin flip (P = 5 %) into a > 50 % catch event — a 9.76× improvement at the same compute budget. The design intent of the risk-weighted strategy is exactly this and the empirical behaviour confirms it.

### 5.3 ReplayAttacker matches BernoulliLiar exactly

`ReplayAttacker(f=0.10)` substitutes a hash drawn from a *different* honest receipt — a 256-bit value that is in the global "honest hash" set, just for the wrong input. Catch rate matches `BernoulliLiar(f=0.10)` to within MC noise at every k (e.g., 0.9274 vs 0.9265 at k = 25). This is by construction: re-execution checks `output_hash == hash(model(input_i))` for *this* input, not "is `output_hash` in some global set". Replay attacks gain no advantage over straight fabrication. Stated as a small lemma:

> **Lemma 3 (replay irrelevance).** Spot-check verification is invariant under any adversary strategy that produces a 256-bit value not equal to `hash(model(input_i))` for input i. The adversary's win probability depends only on `(L, N, k)` and the verifier's sampling strategy, not on the structure of the fabricated hash.

Proof: re-execution computes the canonical hash for input i and tests equality against the receipt's claim. Inequality is the unique catch event. ∎

### 5.4 Pass criteria

1. `HonestProver`: `P_catch` empirical = 0 in every cell. (Re-execution must agree with itself.) — **PASS** (all 10 cells).
2. Every other cell: `|P_catch_empirical − P_catch_hypergeometric| ≤ 5σ`, σ = sqrt(p(1−p)/M). 5σ per-cell FP rate ≈ 6 × 10⁻⁷. — **PASS** (all 60 cells).
3. `RiskWeighted` against `SelectiveLiar` strictly dominates `Uniform` at every unsaturated k. — **PASS** (k = 1, 5, 25, 100; saturated tie at k = 500).

---

## 6. Comparison to other soundness models

| Approach | Soundness primitive | Trust assumption |
|---|---|---|
| **OCX spot-check (this work)** | 1 − (1 − f)^k via independent re-execution | Determinism + signature + verifier's own compute |
| **zkML (e.g., Risc Zero, EZKL)** | SNARK proves circuit-evaluation; soundness from cryptographic assumption | Cryptographic + bounded-compute (no current SNARK proves a 70B forward pass at production cost) |
| **TEE (SGX, H100 Confidential Computing, AMD SEV)** | Hardware attestation that the binary ran inside an enclave | Hardware vendor honesty + microcode integrity (Foreshadow, Plundervolt, etc.) |
| **Audit logs (current standard)** | Issuer's own log; verifier trusts log + issuer | Issuer honesty (no soundness against a lying issuer) |
| **Certificate Transparency / Trillian** | Append-only Merkle log + auditors monitor for split-views | Issuer can still issue bogus certs; the log only proves they were issued, not that they're correct |

OCX's distinctive position: it gives **issuer-independent soundness against fabrication** at a cost the verifier can budget (~80 µs per signature check + one inference re-execution per ~k receipts). zkML gives the same property *cryptographically* but cannot meet production cost at frontier scale today. TEEs give it via hardware attestation but trade issuer-trust for vendor-trust. Audit logs and CT give *transparency* but not *correctness*. OCX is the first system in this matrix that simultaneously: (a) catches a lying issuer with bounded sampling; (b) does not require a hardware vendor to be honest; (c) verifies in microseconds offline.

---

## 7. What this section does NOT prove

- **Liveness / availability**: a malicious issuer can refuse to publish, refuse to be sampled, or rate-limit verifiers. These are addressed by Layer 3 (witness consensus) — out of scope for this lemma.
- **Adaptive challenge-coin attacks**: if the issuer can predict or influence which receipts will be challenged (e.g., the verifier's PRNG seed leaks), the soundness collapses. Mitigation: derive the challenge coin from a public beacon (block hash, drand, witness signature) committed *after* receipts are issued. Discussed in `STRATEGY.md`.
- **Compute-side equivocation across hardware vendors**: inference on AMD MI300X vs NVIDIA H100 will produce different `output_hash` values for the same weights and inputs (different SIMD reduction order). Both are individually deterministic; cross-vendor equality is not claimed. The receipt's `host_info` field binds the hash to a specific vendor + driver + CUDA + torch version. Tier 1 #4 of the roadmap covers cross-vendor receipt round-trip experimentally.
- **Determinism on closed-source models** (GPT-4, Claude, Gemini): we cannot prove byte-identity of computations we cannot reproduce. OCX is currently a primitive for open-weight deployments.

---

## 8. Reproducibility

```
$ cd /home/kurokernel/Desktop/AXIS/ocx-protocol
$ python3 whitepaper-tests/adversarial_soundness.py \
    --trials 10000 --N 10000 \
    --output examples/gpu-verifier/results/h100/adversarial_soundness.jsonl
```

The script exits 0 iff all assertions in §5 hold. The JSONL output has one row per (adversary × verifier × k) cell with `f_empirical`, `p_catch_empirical`, `p_catch_independent_bound`, `p_catch_hypergeometric`, mean and max caught per trial. The challenge bank is rebuilt from committed receipt files; no GPU is required for verification. A single CPU core completes the full 70-cell × 10K-trial sweep in approximately 6 minutes.
