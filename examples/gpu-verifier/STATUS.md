# OCX GPU Verifier — Status

**Last updated:** 2026-04-23
**Scope:** Honest assessment of deterministic GPU inference + canonical signed receipts. What works, what doesn't, what's next.

---

## 1. What works (end-to-end, measured tonight)

**Deterministic GPU inference of Qwen 2.5-0.5B on an RTX 5060 Blackwell (sm_120), with canonical OCX receipts that verify offline via the Rust `libocx-verify` shared library in ~80 microseconds, tamper-detectable by a one-bit flip.**

All three success criteria from the plan passed on first run:

| Criterion | Target | Measured | Result |
|---|---|---|---|
| **S1 — cross-process determinism** | 3 fresh processes → identical output_hash + logits_hash | hash `0fbfb8ec...` matches across all 3 runs; logits hash `a0d69f04...` also matches | **PASS** |
| **S2 — canonical round-trip** | Python-produced receipt verifies via Rust `ocx_verify_receipt_detailed` returning `OCX_SUCCESS` | 100/100 receipts verify, all `OCX_SUCCESS` | **PASS** |
| **S3 — verify latency** | per-receipt median < 5ms | median 79μs, p99 291μs (over 100 distinct receipts) | **PASS** (~60x margin) |

Tamper detection also confirmed: flipping one byte of the signature yields `OCX_INVALID_SIGNATURE` from the Rust verifier.

### Environment

```
arch          : x86_64
os            : Linux 6.12.10-76061203-generic (Pop!_OS 22.04)
gpu           : NVIDIA GeForce RTX 5060, sm_120 Blackwell, 8GB VRAM, UUID GPU-c283b517-6285-cf9b-71de-2738bd913a8d
nvidia_driver : 580.82.09
cuda_runtime  : 12.8 (bundled in PyTorch nightly wheels)
torch         : 2.12.0.dev20260408+cu128
transformers  : 5.6.0
python        : 3.10.12
```

### Determinism configuration

Everything that makes PyTorch reproducible, turned on:
- `torch.use_deterministic_algorithms(True, warn_only=False)`
- `torch.backends.cudnn.deterministic = True`
- `torch.backends.cudnn.benchmark = False`
- `torch.backends.cuda.matmul.allow_tf32 = False`
- `torch.backends.cudnn.allow_tf32 = False`
- `CUBLAS_WORKSPACE_CONFIG=:4096:8`
- `CUDA_LAUNCH_BLOCKING=1`
- `torch.manual_seed(42) + torch.cuda.manual_seed_all(42)`
- Model loaded as `torch_dtype=torch.float32, attn_implementation="eager"` (no Flash Attention)
- Greedy decoding (`do_sample=False, temperature=0.0, num_beams=1`)

### Canonical baseline (reproducible)

```
prompt              : "What is the capital of France? Answer in one word:"
max_new_tokens      : 64
seed                : 42
generated_text      : " Paris. The capital of France is Paris."
tokens              : 10
inference_latency   : ~810 ms per fresh process
program_hash        : 8f67cfdd66abbd1338639c042c2d9c645fa717f20899e89c59abc487707b6aed
input_hash          : 3c431e223966a26412fae474e84da6cf2d43a06f8ae987d64d0c0101d135e863
output_hash         : 0fbfb8ecf647e1758a0af6cca95e6e3a086f9d82e958ec85ee7d4422cf31cfa7
logits_hash         : a0d69f041d1a82f082328c17aa14913c25a5957a331761e86e660de41f94696c
receipt_cbor_size   : 208 bytes
verify_latency_p99  : 291 μs (Rust libocx-verify via ctypes)
```

Any verifier matching the environment above and running `test_determinism_gpu.py` should reproduce the `output_hash` and `logits_hash` exactly.

---

## 2. What was NOT tested (and is NOT claimed)

Being precise about the boundary of the claim:

### 2.1 Cross-GPU determinism (untested)
We only have one RTX 5060. **We have not tested** whether two different 5060s produce the same logits. Even within the same GPU arch, driver version, firmware version, and thermal/clock state can introduce tiny timing differences that occasionally affect kernel scheduling. Based on the CPU x86↔ARM experience (see `examples/ai-verifier/STATUS.md`), expect that cross-vendor cross-arch is almost certainly not byte-identical. Cross-vendor same-arch (two 5060s on two different machines) is the interesting measurement. Not done tonight.

### 2.2 Model scale (limited)
Qwen 2.5-0.5B is a toy. Only 290 weight tensors. Tiny hidden dim. Determinism here does not imply determinism at 7B, 13B, or 70B — numerical margins get tighter at scale and the argmax flip risk grows. 7B on 5060 (quantized) is the Day-4 milestone. 70B on H100 is the Day-6 milestone. Neither done.

### 2.3 Flash Attention (untested)
The demo uses `attn_implementation="eager"` (vanilla attention). This materializes the full `[L, L]` attention score matrix and is memory-infeasible at 70B (a single 70B forward pass at L=2048 would need ~40GB just for attention scores). Frontier-scale requires Flash Attention v2 with `deterministic=True`. FA2 was NOT tested tonight. Whether `deterministic=True` holds byte-identity across fresh processes on Blackwell is an open question.

### 2.4 Quantization (untested)
Currently using float32, which is the largest and slowest format but the most numerically robust for determinism testing. bf16 (half memory, 0.5B at bf16 ≈ 1GB) is Day-3. 4-bit quantization via GPTQ or bitsandbytes is Day-4. The concern: bitsandbytes 4-bit kernels use atomicAdd and are not on the `use_deterministic_algorithms` allowlist.

### 2.5 Perf (unoptimized by design)
`CUDA_LAUNCH_BLOCKING=1` serializes every kernel launch. With float32 + eager attention + fresh-process load overhead, fresh-process wall time is ~8.5s of which ~7.5s is model load and ~1s is inference. Real deployment needs:
- `CUDA_LAUNCH_BLOCKING=0` (need to confirm determinism holds)
- Warm model pooled across requests
- bf16 or lower precision
These are Week-1 experiments, not tonight.

### 2.6 Environment field on receipt (not yet implemented)
STRATEGY.md Path 4 proposes `environment` in the receipt itself — arch, driver, CUDA, torch, GPU UUID, model SHA. Right now we capture those via `env_spec.py` but they go only into `host_info` (unsigned metadata). Promoting them to a signed receipt field is a schema change across Go + Rust + Python. Not done tonight; Week-1 work.

### 2.7 Known Transformers warning
```
The following generation flags are not valid and may be ignored: ['temperature', 'top_k'].
```
Transformers v5 deprecated these kwargs when `do_sample=False` because they're not used in greedy decoding. Harmless. Will clean up when porting to stable transformers v4.x or silencing the warning.

---

## 3. Open technical questions

Ordered by what most affects the primitive's reach:

1. **Does determinism hold across two RTX 5060s on two different machines?** Needs a second 5060 or a cloud provider with matching consumer GPUs.

2. **Does determinism hold when `CUDA_LAUNCH_BLOCKING` is dropped?** If yes, throughput doubles without losing correctness. Single biggest perf win if it works.

3. **Does 7B hold determinism at 4-bit quantization on 5060?** Most likely path to fitting real models on consumer hardware. Requires GPTQ (exllama deterministic) or custom deterministic NF4 kernels.

4. **Does Flash Attention v2 `deterministic=True` actually hold cross-process at 70B on H100?** This is THE load-bearing question for frontier-scale. If yes, full scale works. If no, we fall back to SDPA math backend at shorter context, and 70B on one H100 at decent speed becomes much harder.

5. **Can the receipt protocol be extended with a signed `environment` field without breaking existing Go/Rust verifiers?** Needs a schema bump (v1.3?). The Rust verifier's `from_canonical_cbor` currently rejects maps with unknown integer keys in the signed region, so new optional fields are gated on updating the verifier first.

6. **Is verification latency stable under adversarial load?** 80μs median is great but assumes a warm CPU path. Cold-start, deep-recursion, and resource-exhaustion attacks on the Rust verifier are not tested. Security review needed before production.

7. **Do we need NVIDIA H100 Confidential Computing mode (attested enclave) for the commercial story, or is same-environment determinism sufficient?** Enterprise buyers may want hardware attestation regardless of determinism. Path 4 receipts + NVIDIA attestation would be the belt-and-suspenders product offering.

---

## 4. The critical bug we did NOT propagate

During planning we identified that `examples/ai-verifier/ocx_ai_verifier.py` signs **JSON without the OCX domain separator**. Its receipts do not verify via canonical `libocx-verify` — the existing demo only self-verifies via a handwritten Python verifier, which masks the gap.

This new GPU module was written from day one to use:
- Canonical CBOR matching `pkg/receipt/types.go` byte-for-byte (verified with a parity fixture in `parity/`)
- Domain separator `b"OCXv1|receipt|"` prepended before Ed25519 signing
- Receipt format as transmitted: map with integer keys 1-8 where key 8 is the signature
- Signing message format: `domain_separator || canonical_cbor(signed_map)` where `signed_map` contains keys 1-7 (and optionals 9-15 if present), but NOT key 8

All verified end-to-end by having Python-produced receipts round-trip through the Rust `libocx-verify.so` `ocx_verify_receipt_detailed` FFI call, returning `OCX_SUCCESS`.

The CPU ai-verifier will get the same fix in a follow-up, but tonight's work isolated the new GPU code from that bug entirely.

---

## 5. File map

| File | Purpose |
|---|---|
| `canonical_receipt.py` | Python mirror of Go `ReceiptCore` with canonical CBOR + domain-separated Ed25519 signing. |
| `ffi_verify.py` | ctypes binding for `libocx-verify.so`. Offline verification in ~80μs. |
| `env_spec.py` | Environment fingerprint for Path 4 env-pinned receipts (host_info today, signed receipt field pending schema bump). |
| `ocx_gpu_verifier.py` | End-to-end binary: load Qwen2.5-0.5B on GPU, deterministic forward pass, produce signed receipt, self-verify. |
| `test_determinism_gpu.py` | 3-fresh-process byte-equality test (S1) + 100-receipt verify bench (S2/S3) + tamper test. |
| `parity/dump_canonical.go` | Go reference: dumps canonical CBOR hex for a known fixture. |
| `parity/dump_canonical.py` | Python mirror: produces identical bytes. Confirmed byte-equal. |
| `requirements.txt` | Python deps (excluding torch which installs from PyTorch nightly index). |
| `README.md` | Setup + run instructions. |
| `STATUS.md` | This file. |

---

## 6. One-sentence summary

**Deterministic GPU inference with offline-verifiable Ed25519-signed canonical OCX receipts now works end-to-end on RTX 5060 Blackwell for Qwen 2.5-0.5B, passing byte-identical cross-process determinism, canonical round-trip verification through the Rust library in 79μs median, and tamper detection; the envelope is deliberately narrow (one GPU, small model, float32, vanilla attention, no perf optimization) so scale-up and productization work is measurement, not architecture.**
