# OCX Protocol

**Deterministic frontier-scale language model inference with signed receipts that verify offline in microseconds.**

OCX is a protocol for producing byte-identical outputs from large language models and binding each output to a portable, cryptographically signed canonical receipt. A verifier anywhere in the world can re-execute a sample of receipts on the same model and reject any mismatch in microseconds, without trusting the issuer, without a hardware-vendor enclave, and without a zero-knowledge proof system.

The full technical statement is in the whitepaper at [`whitepaper/paper.pdf`](whitepaper/paper.pdf). This README is the orientation document.

---

## What is proven

| Claim | Evidence | Source |
|---|---|---|
| Byte-identical inference at frontier scale | 22 fresh `torchrun` launches across 8 model × parallelism configurations on NVIDIA H100; every group internally byte-identical | [`examples/gpu-verifier/results/h100/`](examples/gpu-verifier/results/h100/) |
| **Cross-vendor byte-identity (AMD ↔ NVIDIA, single-GPU)** | Qwen 2.5-72B on AMD MI300X (CDNA3) produces the **same `output_hash` byte-for-byte** as NVIDIA H100 (Hopper) for single-GPU bf16 + eager attention, including a 51-token continuation. 9 fresh MI300X launches across 3 (model, length) groups, all match H100 baseline. | [`examples/gpu-verifier/results/mi300x/`](examples/gpu-verifier/results/mi300x/) |
| **Cross-vendor boundary (multi-GPU TP)** | 2× MI300X tensor-parallel produces hash `dd3fed4a...` (58 tokens) — **byte-identical within MI300X across 3 fresh launches, but differs from NVIDIA 2× H100 TP `f8dc73cb...`** as predicted by RCCL-fabric vs NCCL-ring all-reduce topology. The boundary is at the multi-GPU collective layer. | [`examples/gpu-verifier/results/mi300x_tp/`](examples/gpu-verifier/results/mi300x_tp/) |
| Mixture-of-experts is byte-deterministic under greedy decoding | Mixtral 8x7B on 2× H100 tensor-parallel, 3 fresh launches, identical `output_hash` and `logits_hash` | `mixtral_8x7b_tp_*.json` |
| Long-run warm-model stability | 11,000 sequential inferences (1K Mixtral 8x7B + 10K Qwen 2.5-0.5B), 0 byte-identity failures, 2 MiB peak memory drift | [`longrun_*.jsonl`](examples/gpu-verifier/results/h100/) |
| Cross-language receipt portability | Go ↔ Python ↔ Rust round-trip, byte-identical canonical CBOR, signature parity, 8/8 tests | [`whitepaper-tests/cross_language_roundtrip.{go,py}`](whitepaper-tests/) |
| **Cross-vendor receipt portability** | Receipts produced on AMD MI300X (single-GPU and 2× tensor-parallel) verify byte-for-byte through the Rust `libocx-verify` built on x86 NVIDIA hardware. 12/12 receipts pass `OCX_SUCCESS`. | [`examples/gpu-verifier/results/mi300x*/`](examples/gpu-verifier/) |
| Verification is sub-millisecond | 79.4 µs median, 114.6 µs p99, 12,392 receipts/sec on a single core | [`whitepaper-tests/bench_verify.py`](whitepaper-tests/bench_verify.py) |
| Spot-check soundness matches `1 − (1−f)^k` | 70 cells × 10,000 Monte Carlo trials, 0 deviations from theory at the 5σ envelope | [`whitepaper-tests/SOUNDNESS_PROOF.md`](whitepaper-tests/SOUNDNESS_PROOF.md) |
| Risk-weighted sampling beats uniform against targeted fraud | 9.76× higher catch rate at k=1 against a stake-targeting adversary | `adversarial_soundness.jsonl` |

Cumulative test record: **312 deterministic-protocol assertions + 31,000+ pass observations across 10K verification benchmarks, 11K warm-model iterations, and 700K spot-check trials. Zero failures.**

---

## Architecture

OCX is layered. The whitepaper covers Layers 1 and 2.

```
Layer 3 — Witness consensus + temporal proofs    [DESIGNED, NOT YET IMPLEMENTED]
              ↑ removes adaptive challenge-coin assumption

Layer 2 — Signed receipt protocol                [PRODUCTION]
              ↑ canonical CBOR + Ed25519 + domain separator
              ↑ cross-language byte parity (Go / Python / Rust)
              ↑ offline verification via libocx-verify FFI

Layer 1 — Deterministic computation              [PRODUCTION for HF Transformers + NCCL]
              ↑ byte-identical LLM inference at 70B+ scale
              ↑ tensor-parallel, pipeline-parallel, CPU-offload
```

### Components

| Path | Language | Purpose |
|---|---|---|
| [`pkg/receipt/`](pkg/receipt/) | Go | Canonical CBOR encoder, `ReceiptCore` schema, signing path |
| [`pkg/keystore/`](pkg/keystore/) | Go | Ed25519 keystore; defines the `OCXv1\|receipt\|` domain separator |
| [`libocx-verify/`](libocx-verify/) | Rust | Canonical offline verifier; C FFI; the only verifier that signs in this codebase |
| [`examples/gpu-verifier/`](examples/gpu-verifier/) | Python | Deterministic GPU inference + receipt generation; OpenAI-compatible HTTP endpoint |
| [`whitepaper-tests/`](whitepaper-tests/) | Python + Go | Test plan, results, cross-language round-trip, verification benchmark, adversarial soundness simulator, long-run stability |
| [`whitepaper/`](whitepaper/) | LaTeX | The whitepaper source and rendered PDF |
| [`cmd/server/`](cmd/server/) | Go | Reference HTTP server for the deterministic-VM execution path (Layer-1 wrapper) |

The receipt schema, canonical encoding, and verifier together comprise about 2,500 lines of cross-language code. The deterministic GPU inference setup — the load-bearing part for AI-specific applications — is about 400 lines of Python configuration on top of HuggingFace Transformers.

---

## Quick start

### Verify an existing receipt (sub-millisecond, no GPU required)

```bash
git clone https://github.com/KuroKernel/ocx-protocol.git
cd ocx-protocol

# Build the canonical Rust verifier (one-time)
cd libocx-verify && cargo build --release && cd ..

# Run the verification benchmark on 10,000 distinct receipts
python3 whitepaper-tests/bench_verify.py 10000
# Expected: median ≈ 80 µs, p99 ≈ 115 µs, throughput ≈ 12K receipts/sec/core
```

### Reproduce the determinism evidence

```bash
# Aggregate every committed receipt into the byte-identity matrix
python3 whitepaper-tests/aggregate_determinism_evidence.py
# Expected: 12 / 12 model × hardware × parallelism groups byte-identical
```

The full reproducible test methodology — hardware matrix, exact prompts, expected hashes, pass criteria — is specified in [`whitepaper-tests/CROSS_VENDOR_DETERMINISM_BENCHMARK.md`](whitepaper-tests/CROSS_VENDOR_DETERMINISM_BENCHMARK.md). Anyone with comparable hardware can run the listed commands and observe the same hashes.

### Reproduce the spot-check soundness simulation

```bash
# 70 (adversary × verifier × k) cells, 10,000 Monte Carlo trials each
python3 whitepaper-tests/adversarial_soundness.py \
    --trials 10000 --N 10000 \
    --output examples/gpu-verifier/results/h100/adversarial_soundness.jsonl
# Expected: PASS — all 70 cells within 5σ of theoretical hypergeometric prediction
# Wall time: ~90 seconds on a single CPU core
```

### Run deterministic inference yourself (requires NVIDIA GPU + the right software stack)

See [`examples/gpu-verifier/README.md`](examples/gpu-verifier/README.md) for the full setup. In short: PyTorch 2.10+ with CUDA 12.4+, NVIDIA driver 560+, an H100 (or for small models, any recent NVIDIA GPU including RTX 4090 / 5060). Tensor-parallel runs require two GPUs with NVLink for the byte-identity claim to hold.

```bash
cd examples/gpu-verifier
torchrun --nproc_per_node=2 ocx_gpu_verifier_tp.py \
    --model Qwen/Qwen2.5-72B-Instruct \
    --prompt "What is the capital of France? Answer in one word:" \
    --max-new-tokens 32 \
    --output qwen72b_run.json
```

---

## Whitepaper

[`whitepaper/paper.pdf`](whitepaper/paper.pdf) — 10 pages, 4,851 words. Sections:

1. Introduction: the AI audit-trail problem and why existing answers do not solve it
2. Related work: zkML, TEEs, Certificate Transparency, ML reproducibility — and where OCX sits
3. Protocol: receipt schema, canonical encoding, signature scheme, environment binding
4. Implementation: deterministic inference substrate, verification stack, deployment
5. Empirical results: cross-language parity, verification performance, byte-identical inference matrix, long-run stability, cross-session reproducibility
6. Soundness: threat model, hypergeometric soundness lemma, replay-irrelevance lemma, Monte Carlo validation, comparison table
7. Limitations: explicit list of what the protocol does not provide
8. Conclusion: open questions

LaTeX source is [`whitepaper/paper.tex`](whitepaper/paper.tex). Build instructions: [`whitepaper/README.md`](whitepaper/README.md).

---

## What this protocol does NOT do

OCX is a primitive, not a complete trust system. The limitations are explicit:

- **Not zero knowledge.** A spot-check verifier sees the input and output of any sampled inference. Confidential workloads need a separate confidentiality layer.
- **Not vendor-portable inference.** Output hashes from NVIDIA H100 will not equal output hashes from AMD MI300X for the same weights and inputs, by the physics of floating-point reduction order. Receipts and verifiers are vendor-portable; the underlying computation is not. The receipt's environment binding makes the configuration explicit.
- **Not a defence against hardware-vendor compromise.** If the GPU vendor ships a driver that produces different outputs on alternate Tuesdays, no software-level audit catches it. We mitigate weakly by binding driver versions; full mitigation requires multi-vendor cross-verification.
- **Not yet implemented for vLLM or PagedAttention.** The Hugging Face Transformers + NCCL ring all-reduce path is byte-deterministic. Production-throughput stacks are not yet measured.
- **Not applicable to closed-source models.** Reproducibility requires the verifier to re-execute the model. GPT-4, Claude, and Gemini cannot be made auditable without provider participation.
- **Layer 3 (witness consensus, adaptive-challenge protection) is designed but not implemented.** The current spot-check soundness assumes the verifier's challenge coin is independent of the issuer's lying decisions. In production this requires a public beacon committed after receipts are issued.

The full limitations section is [`whitepaper/paper.tex` §7](whitepaper/paper.tex) and [`whitepaper-tests/TEST_RESULTS.md`](whitepaper-tests/TEST_RESULTS.md) "What is NOT proven by these tests".

---

## Repository structure

```
ocx-protocol/
├── whitepaper/                The whitepaper (LaTeX source + rendered PDF + build README)
├── whitepaper-tests/          Test plan, results, soundness proof, all paper-cited tests
├── pkg/                       Go: core protocol (receipt, keystore, verify, chain, executor)
├── libocx-verify/             Rust: canonical offline verifier with C FFI
├── examples/
│   ├── gpu-verifier/          Python: deterministic GPU inference + receipts (the AI-specific path)
│   ├── ai-verifier/           Python: earlier CPU-side LLM determinism evidence (Pi 5 + x86)
│   └── ...                    Other examples and reference clients
├── cmd/server/                Go: reference HTTP server for the deterministic-VM execution path
├── docs/                      Operational and design documentation
└── LICENSE                    MIT
```

---

## Status

Research-grade, not a turnkey product. The protocol primitive (Layers 1 + 2) is complete and tested at frontier scale. Layer 3 (witness consensus) is designed but not yet implemented. There is no managed hosted service. There are no commercial pilots.

The whitepaper is being prepared for arXiv submission. The repository is currently private and will be made public once trademark filings on "OCX" are complete. After that, every commit referenced in the paper is fixed and reproducible.

---

## Citing

```
@misc{ocx2026,
  title  = {Deterministic Frontier-Scale Language Model Inference with Signed Receipts},
  author = {Singh, Aishwary},
  year   = {2026},
  note   = {OCX Protocol whitepaper},
  url    = {https://github.com/KuroKernel/ocx-protocol}
}
```

---

## License

MIT — see [`LICENSE`](LICENSE).

The MIT license covers code. The name "OCX" and the OCX visual identity are subject to a separate trademark filing; the trademark exists to prevent confusion (a "verified by OCX" claim should mean the protocol described here, not a fork that diverges silently). The code itself is freely reusable under MIT.

---

## Contact

Aishwary Singh — `hhaishwary@gmail.com`

Security issues: please disclose privately to the email above before opening a public issue.
