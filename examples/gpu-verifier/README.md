# OCX GPU Verifier

**Deterministic GPU inference with canonical OCX receipts that verify offline in microseconds.**

This is the GPU-track counterpart to `examples/ai-verifier/` (CPU, llama-cpp, GGUF). Where the AI verifier measured same-arch cross-process determinism on CPU and documented a cross-arch boundary (SIMD reduction order), this module pushes into GPU territory: PyTorch + CUDA with deterministic flags, canonical CBOR receipts signed with Ed25519 and the protocol-level `OCXv1|receipt|` domain separator, and round-trip verification through the pre-built Rust `libocx-verify` shared library.

## Why this matters

Every AI-agent decision with financial or legal consequence needs a regulator-grade audit trail. Today's verifiable-inference landscape:

- **zkML** (ezkl, Giza) — can't scale past toy models.
- **TEEs** (NVIDIA H100 Confidential Computing) — hardware-vendor locked, attestation root is NVIDIA.
- **Risc Zero zkVM** — impractical for real inference throughput.

None of those ship production deterministic GPU inference with signed receipts. This module is the first move in that direction: prove the receipt pipeline plus a deterministic GPU path works at a small scale, then scale to 7B → 70B on H100.

## What's in this directory

- `canonical_receipt.py` — CBOR canonical encoder + domain-separated Ed25519 signer matching `pkg/receipt/types.go` byte-for-byte.
- `ffi_verify.py` — ctypes binding for the pre-built Rust `libocx-verify` shared library. Offline verification in ~80 microseconds.
- `env_spec.py` — environment fingerprint (GPU arch, driver, CUDA, torch) for Path 4 env-pinned receipts.
- `ocx_gpu_verifier.py` — end-to-end: load Qwen2.5-0.5B on GPU, deterministic forward pass, produce signed receipt.
- `test_determinism_gpu.py` — two-fresh-process byte-equality test + 100-receipt verify benchmark + tamper test.
- `parity/` — Go ↔ Python CBOR byte-parity fixtures (run once, confirmed identical encoding).
- `STATUS.md` — honest test results after running the harness.

## Setup

```bash
# 1) Create GPU venv (separate from ai-verifier's venv — PyTorch wheel is 3GB)
python3 -m venv venv-gpu

# 2) Install PyTorch. RTX 5060 is Blackwell (sm_120). Stable CUDA 12.4 wheels
# do NOT support sm_120. Use the nightly CUDA 12.8 wheels:
./venv-gpu/bin/pip install --pre --index-url https://download.pytorch.org/whl/nightly/cu128 torch

# 3) Other deps
./venv-gpu/bin/pip install transformers accelerate safetensors cbor2 cryptography
```

## Run

```bash
# Single inference + receipt + self-verify
./venv-gpu/bin/python ocx_gpu_verifier.py

# Two-process determinism test + verify bench
./venv-gpu/bin/python test_determinism_gpu.py
```

## Protocol plumbing notes

The receipt this module produces is a canonical CBOR map with integer keys. This matches the Go `pkg/receipt/types.go:ReceiptCore` byte-for-byte (verified with a fixture in `parity/`). Signing uses Ed25519 with the domain separator `OCXv1|receipt|` prepended to the canonical bytes, exactly as the protocol spec requires.

Concretely:

```
signed_map  = { 1: program_hash, 2: input_hash, 3: output_hash,
                4: cycles_used,  5: started_at, 6: finished_at,
                7: issuer_id }
signed_cbor = canonical_cbor(signed_map)
signature   = Ed25519.sign(priv_key, b"OCXv1|receipt|" + signed_cbor)
receipt     = canonical_cbor(signed_map | { 8: signature })
```

The Rust verifier reconstructs `signed_cbor` from the receipt, re-applies the domain separator, and validates the Ed25519 signature over that exact message. Byte-for-byte.

## Current scope

- Qwen2.5-0.5B-Instruct on RTX 5060 (sm_120, 8GB VRAM), float32, vanilla eager attention.
- Determinism mode: `torch.use_deterministic_algorithms(True)`, `CUBLAS_WORKSPACE_CONFIG=:4096:8`, `CUDA_LAUNCH_BLOCKING=1`.
- Greedy decoding (`do_sample=False, temperature=0`).

This is deliberately the minimum viable proof of the stack. Scale-up path (7B → 70B on H100, quantization, bf16, Flash Attention determinism) is documented in the plan file and in `STATUS.md`.

## Relation to the CPU ai-verifier

`examples/ai-verifier/ocx_ai_verifier.py` is the llama-cpp-based CPU demo. It has a known limitation: the Python side signs JSON with no domain separator, so its receipts **do not verify via canonical `libocx-verify`**. That code self-verifies with a handwritten Python path, masking the gap. This GPU module implements the canonical path correctly from day one and proves round-trip verification through the Rust lib. The CPU module's signing path will be ported to match in a follow-up.
