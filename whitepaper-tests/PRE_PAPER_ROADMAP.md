# OCX Whitepaper — Pre-Submission Work Roadmap

**Date:** 2026-04-25
**Purpose:** Concrete list of additional work to do before writing the OCX whitepaper, ranked by impact-per-day-of-effort, with explicit reasoning for each item. Designed so the resulting paper is unignorable to AI-governance audiences (regulators, AI safety researchers, enterprise CTOs, NVIDIA, OpenAI/Anthropic, EU AI Office, RBI, etc.) rather than a strong-but-incremental engineering note.

---

## Honest assessment of where we are

We have:

- Deterministic GPU inference proven byte-identical across fresh torchrun launches for Qwen 2.5-72B-Instruct and Meta Llama-3.1-70B-Instruct
- Three parallelism configurations (1× CPU offload, 2× pipeline parallel, 2× tensor parallel) with NCCL ring all-reduce over NV18 NVLink
- Canonical CBOR receipts (Ed25519, OCXv1|receipt| domain separator) byte-identical across Go and Python implementations, accepted byte-for-byte by the canonical Rust libocx-verify FFI
- Verification latency ~80 µs median over 10K receipts, 12K verifications/sec/core
- 240+ tests passing (147 Go core + 79 Rust + 8 cross-language + 6 determinism evidence groups)
- OpenAI-compatible HTTP endpoint serving signed receipts as drop-in replacement
- All measurements reproducible from committed code at commit `c47c07c9`

This is a strong engineering result. **It is not yet world-stunning.** What separates "good engineering" from "stops the field" is the difference between proving a thing works in a few configurations and proving it works under the conditions sceptics will specifically demand.

---

## What separates "interesting" from "stunning"

Papers that genuinely shake their fields share a pattern: they solve something the field had explicitly said couldn't be done.

- Bitcoin (2008): solved Byzantine consensus, an open problem since the 1982 Lamport paper
- Attention Is All You Need (2017): removed every recurrence and convolution, beat all RNN baselines
- AlphaGo (2016): beat the world champion at a game considered AI-uncomputable

For OCX, the "field said it was impossible" claim is **production-grade deterministic GPU inference at frontier scale, with cheap offline verification, without zkML or TEEs**. We have early evidence. To stun, we need it under the conditions reviewers will specifically demand.

---

## Tier 1 — Must-haves before writing (one focused week)

These convert "neat experiment" into "irrefutable claim".

### 1. MoE determinism — Mixtral 8x7B / 8x22B / DeepSeek V3
The first sceptic's question will be "sure, dense models are byte-deterministic, but real production frontier is mixture-of-experts, and MoE has stochastic routing." If we prove byte-identity for an MoE model, that objection dies. **This single result moves the paper from "interesting" to "important".**

- Mixtral 8x7B Instruct (141 GB total, 47 GB active per token, 8 experts): standard MoE benchmark; fits on 2× H100 in bf16.
- Mixtral 8x22B Instruct (282 GB): the bigger flex; needs 4× H100 or 2× H100 with offload.
- DeepSeek V3 (671 B total, 37 B active): the actual frontier. Needs 8× H100. Headline if we can swing it.

Effort: 1 day on 2× H100 for Mixtral 8x7B, ~₹1 000 cloud spend.

### 2. Long-run stability — 100 000 continuous inferences
Sceptics will assume thermal drift, GPU clock variance, or accumulated state breaks determinism over hours of continuous load. We run 100 000 sequential signed inferences in a single session, prove 100 % byte-identity throughout, log throughput stability vs time, log GPU temperature vs time. Disproves the drift objection cleanly.

Effort: 6–12 hours of GPU time, fully scripted. ~₹2 500–5 000.

### 3. Adversarial prover proof-of-concept
A security paper without an adversary is incomplete. Build a mock "lying issuer" that returns fake receipts (claims to have run model X but didn't). Build a "spot-check verifier" that randomly re-runs ONE inference per N receipts. Demonstrate that the verifier catches a lying prover with probability `≥ 1 − (1 − p)^k`, where p is the per-spot-check fraud-detection probability and k is the number of independent challenges. State and prove a soundness lemma.

Effort: 2 days, ~200 lines of Python, no GPU needed for the spot-check protocol design.

**STATUS — DONE 2026-04-25.** Implemented in `whitepaper-tests/adversarial_soundness.py`. Results: 5 adversary strategies × 2 verifier strategies × 5 k-values = 70 cells × 10,000 Monte Carlo trials each, 0 / 70 deviations from theoretical hypergeometric prediction at the 5σ envelope. Risk-weighted sampling against a stake-targeting adversary at `k = 1` yields a 9.76× catch-rate improvement over uniform sampling. Formal soundness lemma + threat model + comparison-to-alternatives table written up in `whitepaper-tests/SOUNDNESS_PROOF.md`. Per-cell raw data committed at `examples/gpu-verifier/results/h100/adversarial_soundness.jsonl`.

### 4. Cross-vendor hardware — AMD MI300X round-trip
Kills the "you're NVIDIA-locked" objection. The OCX RECEIPT layer (canonical CBOR + Ed25519) is hardware-agnostic by construction; we prove this by running an AMD MI300X via RunPod / Lambda, producing receipts there, and confirming the canonical Rust libocx-verify accepts them byte-for-byte. We also note that the inference *outputs* will differ across vendors as expected (different SIMD reduction order across CDNA vs Hopper) — that boundary is identical to the cross-arch boundary we already documented for x86 ↔ ARM CPU.

Effort: 4–6 hours including setup. ~₹1 500–2 500.

**STATUS — DONE 2026-04-26 (with a positive surprise).** Provisioned 1× AMD MI300X on RunPod (₹165/hr). Installed PyTorch 2.4.1+rocm6.1, built libocx-verify with Rust 1.86 on the pod. Ran Qwen 2.5-0.5B and Qwen 2.5-72B at single-GPU bf16+eager attention. Headline finding contradicts the planning assumption: **for these configurations the inference outputs DO NOT differ across vendors — AMD MI300X produces byte-identical `output_hash` to NVIDIA H100 across 9 fresh launches in 3 (model, length) groups, including a 51-token continuation at frontier scale.** The 6 short-gen MI300X receipts also verify byte-for-byte through the local libocx-verify built on x86 NVIDIA hardware (cross-vendor receipt portability confirmed as expected). The receipt's environment binding still handles the general case where outputs differ; the empirical cross-vendor byte-identity is a useful surprise, not a guarantee. Receipts committed at `examples/gpu-verifier/results/mi300x/`. Paper §5.5 rewritten to lead with the cross-vendor finding; Table 1 expanded with three MI300X rows.

### 5. Formal threat model + theoretical framing section
Positions OCX in the academic literature. Without this, reviewers say "this is engineering, not research." Define:

- The adversary capabilities (what can a lying issuer do?)
- The trust assumptions (what does verification depend on at each layer?)
- The security game (what does it mean for the protocol to be secure?)
- Comparison table to zkML (different trust assumption — bounded compute), TEEs (different trust assumption — hardware vendor), audit logs (different trust assumption — issuer honesty), Certificate Transparency (closest neighbor)

Effort: 3 days of writing, no compute.

**Tier 1 total: ~7 days, ~₹6 000–10 000 cloud time. After this, the paper is unignorable.**

---

## Tier 2 — Force-multipliers if time permits

### 6. vLLM deterministic mode
Every production AI deployment uses vLLM. PagedAttention is atomic-heavy and known non-deterministic. If we make vLLM produce byte-identical receipts (or document precisely why it can't and what would need to change), we own the production-throughput conversation.

Effort: 4–6 days. High-effort, high-payoff.

### 7. Real production deployment — Kitaab in pilot mode
Nothing closes a regulator-facing argument like "here are N production receipts from a live deployment." Even low-volume Kitaab production data, exported as a public anonymised receipt corpus, becomes the strongest possible existence proof.

Effort: 3–5 days of integration work.

### 8. Closed-model audit proxy
We can't make GPT-4 or Claude deterministic, but we can demonstrate OCX wrapping a deterministic open model as a *stand-in for the closed-model audit problem*. Frame: "this is what audit would look like if Anthropic adopted the protocol on top of their existing deployment."

Effort: 1 day of writing + diagrams.

### 9. OCX-Bench — public benchmark suite
If "OCX-Bench" becomes the industry-standard way to measure whether an AI deployment is auditable (e.g. "this deployment scores 8/10 on OCX-Bench, that one scores 2/10"), you own the metric. Public, runnable, leaderboard-ready.

Effort: 1 week to design and ship publicly.

---

## Tier 3 — After-paper distribution

- Pre-read by 2–3 senior names in AI safety / cryptography (Boneh, Goodfellow, Karpathy, equivalents) — even one positive read amplifies more than the paper itself
- HN / Reddit r/MachineLearning posts with the punchy hashes + reproduction commands
- Twitter/X thread with a single screenshot of the byte-identical hashes
- arXiv submission
- Conference submission to USENIX Security / IEEE S&P / CCS for the following year

---

## What stops global giants in their tracks specifically

Each "global giant" reads any new paper with one question: *does this affect my exposure?*

| Giant | What makes them care |
|---|---|
| **NVIDIA** | If a software protocol achieves audit-grade trust without their hardware lock-in (H100 Confidential Computing premium), that's competitive pressure on margins. |
| **OpenAI / Anthropic** | Regulators will eventually require deterministic re-execution of consequential decisions. If their stack can't comply because it's non-deterministic, OCX becomes the "you should be doing this" reference architecture. |
| **EU AI Office** | Article 14 of the AI Act requires traceability for high-risk AI. OCX provides the concrete protocol-level answer they're hungry for. |
| **Big Indian banks (HDFC, ICICI, SBI)** | RBI is moving toward AI governance frameworks. OCX-on-Kitaab gives them a referenceable architecture before the regulator forces them to scramble. |
| **Google / DeepMind** | Less direct exposure, but the Trust & Safety teams genuinely care about audit primitives. |
| **Meta** | Has Llama. If OCX makes Llama auditable in a way GPT-4 isn't, that's a competitive feature. |

The specific combination that stops each of these is not 70B determinism in isolation. It's:

- Frontier-scale ✓ (already proven)
- Model-agnostic ✓ (already proven for Qwen + Llama)
- **MoE-capable** (Tier 1 item 1)
- **Hardware-vendor neutral** (Tier 1 item 4)
- **Adversarially robust** (Tier 1 item 3)
- **Long-run stable** (Tier 1 item 2)
- **Properly framed against alternatives** (Tier 1 item 5)
- Production-deployed (Tier 2 item 7)

The first six convert the paper from a single-result note into a multi-claim protocol paper that is uncomfortable to ignore.

---

## Concrete two-week plan

**Week 1 — empirical strengthening (cloud + experiments)**

- Day 1: Mixtral 8x7B Instruct on 2× H100 TP, 3 fresh torchruns, prove MoE byte-identity. (~₹1 000)
- Day 2: Mixtral 8x22B Instruct on 4× H100 TP if budget allows, OR Llama 3.3 70B + Qwen 2.5-32B as additional dense-frontier proofs on 2× H100. (~₹2 000)
- Day 3: 100 000 continuous inference long-run on 2× H100, log byte-identity + throughput + temperature over time. (~₹3 000)
- Day 4: Adversarial prover + spot-check verifier (no GPU, pure Python).
- Day 5: AMD MI300X round-trip test on RunPod. (~₹2 000)

**Week 2 — paper writing**

- Day 1: Outline + abstract + threat model section
- Day 2: Theoretical framing + comparison table (zkML / TEE / CT / audit logs)
- Day 3: Empirical sections with all measurements, citing TEST_RESULTS.md numbers verbatim
- Day 4: Discussion, future work, ethics, limitations
- Day 5: Polish, diagrams (architecture, threat model, parallelism configurations, latency CDF)
- Day 6: Pre-read with 1–2 trusted reviewers, incorporate feedback
- Day 7: Final pass, push to arXiv

**Total cloud spend for week 1: ~₹8 000–12 000.**

---

## What NOT to do before submitting

- **Do not build Layer 3** (VDF temporal proofs + peer-witness consensus). That is a separate paper. Trying to ship both simultaneously dilutes both.
- **Do not chase enterprise pilots** before the paper. Pilots take 3–6 months. Paper first, pilots later.
- **Do not invest in vLLM determinism** before the paper. It's a 4–6 day rabbit hole that may not work. After the paper.
- **Do not build a fancy demo or marketing site.** The paper IS the demo if it is well-written. After the paper.

---

## The single most important addition

If you do nothing else from this list, do **Tier 1 item 1: MoE determinism**.

The first question every reviewer and every senior researcher will ask is "does this work for MoE models?" — because MoE is the actual frontier (DeepSeek V3, GPT-4 itself is rumoured to be MoE, Mixtral, Qwen-MoE). Stochastic expert routing under tied tie-breaking is the natural objection.

If you have Mixtral 8x7B byte-identical receipts in the paper across three fresh torchrun launches, that question is answered before it's asked. The other Tier 1 items strengthen the result. The MoE one *defines* it.

One day on 2× H100 with Mixtral 8x7B Instruct, three fresh torchruns, save the receipts. Then write the paper with confidence that the result generalises across architectures.

---

## Reproducibility commitment

Every measurement in the resulting whitepaper should point at a specific commit hash of `https://github.com/KuroKernel/ocx-protocol`, a specific test command, and a specific receipt JSON committed under `examples/gpu-verifier/results/` or equivalent. No unverifiable claims. No measurements that aren't checked into the repo. This is what makes the paper genuinely citable rather than aspirational.
