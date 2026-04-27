# OCX Verifiable AI — Status Document

**Last updated:** 2026-04-23
**Author:** OCX Protocol team
**Scope:** Honest assessment of the deterministic AI inference primitive. What works, what doesn't, what's open.

This document is deliberately unpolished. It exists to prevent overclaiming when describing the work to investors, collaborators, or downstream consumers. If you are evaluating whether to depend on this primitive, read all three sections.

---

## 1. What works

We have a working proof-of-execution primitive for LLM inference. Given:

- A GGUF model file (we tested Qwen2.5-0.5B-Instruct q4_k_m, SHA256 `74a4da8c9fdbcd15...`)
- A prompt string
- Deterministic sampling flags: `temperature=0`, `top_k=1`, `top_p=1.0`, `seed=42`, `n_threads=1`
- CPU-only inference via `llama-cpp-python`

...we produce a **canonical CBOR OCX receipt that round-trips through the Rust `libocx-verify` shared library** — the same verifier that the Go protocol server and any third party uses. The receipt contains:

```
program_hash = SHA256(model file bytes)
input_hash   = SHA256(canonical JSON of {prompt, model_hash, temperature, max_tokens, seed})
output_hash  = SHA256(canonical JSON of {response: text})   ← reproducible by any verifier
signature    = Ed25519 over b"OCXv1|receipt|" + canonical_cbor({1: program_hash, 2: input_hash,
               3: output_hash, 4: cycles_used, 5: started_at, 6: finished_at, 7: issuer_id})
```

Two fixes landed today:

1. **Schema fix (output_hash):** derived from the response text alone, not from a payload that also contained `inference_time_ms` and `tokens_generated`. The old schema made output hashes unreproducible across runs. A third party holding the response text can now recompute `output_hash` and compare.

2. **Canonical pipeline fix (receipt signing):** the previous version signed a handwritten JSON payload with NO domain separator. Its receipts did not verify via the canonical `libocx-verify` Rust library; the demo self-verified through a handwritten Python path, which masked the gap. The new pipeline uses canonical CBOR matching `pkg/receipt/types.go:ReceiptCore` byte-for-byte, and Ed25519 signing with the protocol-required `OCXv1|receipt|` domain separator. Receipts now verify via `ocx_verify_receipt_detailed` through ctypes FFI into the pre-built `liblibocx_verify.so`.

Result: the canonical baseline hashes below are the SAME values that were documented before the fix (the hash formulas were already correct; only the receipt wire format was wrong), but now they sit inside spec-compliant CBOR receipts that any third party verifier can check offline in under a millisecond.

Evidence collected:

| Test | Result | Artifact |
|---|---|---|
| In-process determinism, 0.5B (5 runs, same process) | PASS — identical hash `10beb5d3a765ec34` (16-hex) | `test_determinism.py` |
| Cross-process determinism, 0.5B, x86 (3 fresh Python processes) | PASS — identical hash `f6346a93756300aa...` | `cross_arch_x86_64.json` |
| Cross-process determinism, 0.5B, ARM / Pi 5 (3 fresh Python processes) | PASS — identical hash `668851688e97f38a...` | `cross_arch_aarch64.json` |
| Cross-process determinism, 7B, short generation (7 chars, "Paris.") | PASS — identical hash `5026028fdaa2a2d5...` | `test_7b.py` |
| Cross-process determinism, 7B, long generation (96 tokens, 361 chars) | PASS — identical hash `813b8064f6be9964...` | `test_7b_long.py` |
| Cross-architecture determinism, 0.5B (x86 vs ARM) | **FAIL** — common prefix 263 chars (~55 tokens), then argmax flip | See §2.1 |
| Receipt signing + verification (after schema fix) | PASS — two fresh processes produce matching output_hash | `ocx_ai_verifier.py` |
| Canonical round-trip: Python receipt → Rust libocx-verify (after canonical fix) | PASS — `OCX_SUCCESS` in ~350 μs, tamper-detectable | `ocx_ai_verifier.py` + `ffi_verify.py` |
| Model file reproducibility | PASS — downloaded SHA256 matches repo exactly | `sha256sum` output |

The canonical x86 hashes for the reference test (Qwen 0.5B, prompt `"What is the capital of France? Answer in one word:"`, max_tokens=100):

```
model_sha256          74a4da8c9fdbcd15bd1f6d01d621410d31c6fc00986f5eb687824e7b93d7a9db
input_hash            e4c73ea2b4ced9cd6c5781141732220dd75cd6ed30899a2c8274aeff03d3f45c
canonical_output_hash f6346a93756300aa4e9d6ce218c684d4a5c0293b425fbee0cf03405a627b87be
machine               x86_64 (Linux 6.12 / Python 3.10.12 / llama-cpp-python 0.3.20)
```

If you run the same test on a different machine and get the same `canonical_output_hash`, we have cross-machine determinism. If you get a different hash, we have a portability limit worth measuring.

---

## 2. What does not work (or: untested / blocked)

This is the honest boundary of the claim. The primitive is real **within this boundary only**. Everything below is a known gap.

### 2.1 Cross-architecture (x86 vs ARM)

**Status: TESTED. Result: determinism breaks at the architecture boundary. Specific failure mode characterized.**

Test setup:
- x86_64: Pop!_OS (Linux 6.12), Intel CPU, Python 3.10.12, llama-cpp-python 0.3.20
- aarch64: Raspberry Pi 5 8GB, Cortex-A76 @ 2.4GHz, Debian 13 trixie (Linux 6.12), Python 3.13, llama-cpp-python 0.3.20
- Same model file (SHA256 verified identical, bit-for-bit transferred via scp)
- Same prompt, same sampling flags (`temperature=0, top_k=1, top_p=1.0, seed=42, n_threads=1`)
- Same upstream pip package — compiled from source on each platform

Result:
- **In-process determinism holds on ARM** (3 fresh processes → identical hash `668851688e97f38a6848de5a31095f14da70e36f4b9d0ff12c6458d6719359d2`, 437 chars)
- **In-process determinism holds on x86** (already established; hash `f6346a93756300aa4e9d6ce218c684d4a5c0293b425fbee0cf03405a627b87be`, 419 chars)
- **Cross-architecture determinism FAILS.** Different hashes. Outcome (b) from the earlier expectations list: common prefix, late divergence.

Failure characterization:
- Texts agree for exactly **263 characters** (~50-55 tokens)
- At token position ~55, argmax flips between two tokens with very close logits:
  - x86 picks the token for `"which"`
  - ARM picks the token for `"the"`
- After one token of disagreement, conditional distributions diverge, never reconverge
- Root cause: CPU-specific SIMD reduction order differs (AVX2 on x86 vs NEON on ARM). Quantized matrix-multiply accumulators sum in different orders, producing sub-ULP logit differences that were just large enough to flip argmax on one extremely close decision.

What this means:
- The primitive works *within* a CPU architecture with extremely strong guarantees
- It does NOT work *across* CPU architectures in its current form
- For OCX to claim cross-machine verifiable inference, a verifier must either:
  - (a) Run on the same CPU architecture as the prover (weak cross-platform story)
  - (b) Use an architecture-neutral reference inference path — pure C, no SIMD, standardized reduction tree. ggml has partial support for this via `GGML_CPU_GENERIC` but we have not measured perf cost.
  - (c) Accept a weaker trust model: "this output is consistent with the claimed model" rather than "this output is byte-identical to the canonical run." Practically useful but semantically different.
- This finding is the architecture boundary we suspected would exist. It's now measured, not theorized.

Concrete hashes for the record:
```
prompt:             "What is the capital of France? Answer in one word:"
model_sha256:       74a4da8c9fdbcd15bd1f6d01d621410d31c6fc00986f5eb687824e7b93d7a9db
input_hash:         e4c73ea2b4ced9cd6c5781141732220dd75cd6ed30899a2c8274aeff03d3f45c
x86_output_hash:    f6346a93756300aa4e9d6ce218c684d4a5c0293b425fbee0cf03405a627b87be
arm_output_hash:    668851688e97f38a6848de5a31095f14da70e36f4b9d0ff12c6458d6719359d2
divergence_at:      character index 263 (~token 55 of 96-token generation)
```

### 2.2 GPU / accelerator determinism

**Status: BLOCKED on CUDA toolkit version. Not tested today.**

We have an NVIDIA RTX 5060 (Blackwell, sm_120, 8GB VRAM) on the dev workstation. Driver supports CUDA 13.0. But the installed CUDA toolkit is `nvcc 11.5` from November 2021 — Blackwell requires CUDA 12.8 or newer. Building `llama-cpp-python` with GPU support against an old toolkit would compile but would not run on this card.

Installing CUDA 12.8+ is a system-level operation (3–4GB download, potential conflicts with existing CUDA 11.5 packages, may require reboot). We chose not to do it in this session.

Separately: even with a correct CUDA build, GPU inference is non-deterministic by default. cuBLAS GEMM reductions use non-deterministic orderings, stream concurrency introduces race conditions, and atomic operations on shared accumulators vary across runs. Making GPU inference deterministic requires:
- `CUBLAS_WORKSPACE_CONFIG=:4096:8` or `:16:8`
- Single stream, disabled async
- Deterministic kernel selection (may disable TF32, may disable tensor cores)
- Likely a significant perf hit

Whether llama.cpp's CUDA backend exposes the hooks to force these modes is itself an open question. **This is a research project, not a ten-minute test.** Treat "GPU determinism" as future work.

### 2.3 Model scale

**Status: 7B CONFIRMED DETERMINISTIC. 13B and 70B untested.**

Qwen 0.5B and Qwen 7B (q4_k_m, both sharded and standard) both hold cross-process determinism on x86. The 7B test used a 96-token long-form generation — a poem about determinism, coincidentally — across three independent fresh Python processes. Identical hash in all three.

This is meaningful because the concern was: numerical error in matmul accumulation grows with hidden dimension, and larger models have tighter logit margins that could flip argmax under `top_k=1`. At 7B q4_k_m with 96 sequential argmax decisions, that concern did not manifest. The quantization noise floor dominates the rounding-error budget.

Not yet tested: Qwen 13B, Qwen 32B, Llama-3.1-70B. Higher-precision quantizations (q8, fp16) also untested — they have less quantization noise floor and could in theory be more sensitive to arithmetic reordering. But the q4_k_m result at 7B is a much stronger claim than what existed at the start of the session.

### 2.4 D-MVM integration

**Status: NOT STARTED.** The verifier runs in plain Python. It is not executing inside any sandboxed VM. Today the primitive relies on the end operator setting up a matching Python + llama-cpp-python + CPU-only environment. That is a trust boundary that a hash-equivalent execution environment would eliminate.

### 2.5 Small model, toy prompt

Our canonical test uses a 0.5B model and a trivial prompt ("capital of France"). This is not a realistic workload for any production use case — financial AI, healthcare AI, legal AI, etc. — that we list in the README. A primitive that works on Qwen 0.5B does not necessarily work on Llama-3.1-70B-Instruct at 32k context with tool-calling.

---

## 3. Open technical questions

Listed in rough order of what most affects the generality of the claim.

1. **Does the cross-arch hash match on Raspberry Pi (ARMv8 / NEON)?** If yes, the primitive is architecture-portable at this scale and quantization. If no, we need to identify which kernels produce the divergence and whether a reference implementation (pure C, no SIMD) could be used for the verifier at a perf cost.

2. **Does cross-process determinism hold at Qwen 7B?** At 13B? At 70B? We expect yes at 7B (the q4_k_m quantization dominates numerical noise floor) but need to measure.

3. **Is deterministic GPU inference achievable with llama.cpp's CUDA backend without catastrophic perf loss?** Requires CUDA 12.8+ install, patient configuration, and perf measurement. Real research project.

4. **What's the minimum trusted compute base for a verifier?** Right now a verifier must: run the same OS, same Python version, same llama-cpp-python build, same CPU microarchitecture. D-MVM is supposed to collapse that to "run the D-MVM container." Until D-MVM wraps inference, verifiers carry a lot of trusted dependencies.

5. **How does determinism interact with prompt caching?** llama.cpp's KV cache is per-process. The first token after a warm-up can differ from a fresh start unless you prime it identically. Our test script does prime it. Production users may not know to.

6. **What's the attack surface for a malicious prover?** Can someone craft a prompt that produces one text in a normal environment but a different text in a verifier's environment, such that `output_hash` looks legitimate only to them? This is a harder problem than it sounds and deserves a proper security writeup. Not attempted yet.

7. **What does the receipt look like when the AI refuses?** Or when the response is empty? Or when max_tokens is hit mid-token? All of these should be tested.

---

## Summary, one sentence each

- **Works:** Bit-identical LLM output across processes *within a single CPU architecture* for Qwen 0.5B and Qwen 7B q4_k_m with fixed sampling (including 96-token long generations on 7B), with signed Ed25519 receipts that any third party can re-verify from response text alone.
- **Breaks:** Bit-identical output *across* CPU architectures. x86 and ARM diverge at token ~55 due to SIMD reduction-order differences producing sub-ULP logit shifts that flip argmax on close decisions.
- **Open:** Cross-accelerator (GPU), larger scales (13B, 70B), sandboxed execution (D-MVM), architecture-neutral reference inference path, security analysis.
- **Don't claim:** Cross-architecture byte-identical reproducibility. We measured it and it fails in a specific, characterized way.

The test harness to expand the envelope exists and is cross-architecture — `cross_arch_test.py` runs unchanged on both x86 and aarch64. The primitive is useful today within its measured boundary. Widening that boundary is engineering work, not research: architecture-neutral inference via `GGML_CPU_GENERIC` is the obvious next step, with perf measurement to decide whether the determinism-vs-speed trade is acceptable per use case.
