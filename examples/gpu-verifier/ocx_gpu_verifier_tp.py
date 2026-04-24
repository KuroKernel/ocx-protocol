#!/usr/bin/env python3
"""OCX GPU Verifier — Tensor Parallel edition.

Launch with:
    torchrun --nproc_per_node=2 --nnodes=1 --rdzv_backend=c10d \
        --rdzv_endpoint=localhost:29501 ocx_gpu_verifier_tp.py \
        --model Qwen/Qwen2.5-72B-Instruct --dtype bf16 --max-new-tokens 128

Uses HuggingFace Transformers 5.x native tensor parallelism (tp_plan="auto").
Each rank holds ~half of each layer's weights. Cross-rank communication is
NCCL all-reduce after each tensor-parallel GEMM.

The protocol-level question this answers: does NCCL all-reduce preserve
byte-identity across fresh Python processes? We record output_hash and
logits_hash per-rank; only rank 0 writes the receipt (all ranks agree on
the final logits because the TP-aware linear layer gathers the output
before the next layer boundary).
"""
from __future__ import annotations

import os

# Determinism flags — set BEFORE importing torch
os.environ.setdefault("CUBLAS_WORKSPACE_CONFIG", ":4096:8")
os.environ.setdefault("CUDA_LAUNCH_BLOCKING", "0")
os.environ.setdefault("TOKENIZERS_PARALLELISM", "false")
# NCCL: use ring algorithm explicitly (default, but pin it)
os.environ.setdefault("NCCL_ALGO", "Ring")
# Disable NCCL's async error handling which can change reduction timing
os.environ.setdefault("NCCL_ASYNC_ERROR_HANDLING", "1")
# Enable P2P so NCCL uses NVLink directly (NV18 between our GPUs)
os.environ.setdefault("NCCL_P2P_DISABLE", "0")

import argparse
import hashlib
import json
import sys
import time
from pathlib import Path
from typing import Tuple

import torch
import torch.distributed as dist
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from transformers import AutoModelForCausalLM, AutoTokenizer

from canonical_receipt import ReceiptCore, raw_public_key, sign_receipt
from ffi_verify import verify_receipt

DEFAULT_MODEL = "Qwen/Qwen2.5-72B-Instruct"
DEFAULT_PROMPT = "What is the capital of France? Answer in one word:"
DEFAULT_SEED = 42
DEFAULT_MAX_NEW_TOKENS = 32
ISSUER_ID = "ocx-gpu-verifier-tp-v0"

_DTYPE_MAP = {
    "bfloat16": torch.bfloat16,
    "bf16": torch.bfloat16,
    "float16": torch.float16,
    "fp16": torch.float16,
}


def sha256_bytes(data: bytes) -> bytes:
    return hashlib.sha256(data).digest()


def sha256_json(obj) -> bytes:
    return hashlib.sha256(
        json.dumps(obj, sort_keys=True, separators=(",", ":")).encode()
    ).hexdigest()


def configure_determinism(seed: int) -> None:
    torch.use_deterministic_algorithms(True, warn_only=True)
    torch.backends.cudnn.deterministic = True
    torch.backends.cudnn.benchmark = False
    torch.backends.cuda.matmul.allow_tf32 = False
    torch.backends.cudnn.allow_tf32 = False
    torch.manual_seed(seed)
    torch.cuda.manual_seed_all(seed)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", default=DEFAULT_MODEL)
    parser.add_argument("--prompt", default=DEFAULT_PROMPT)
    parser.add_argument("--seed", type=int, default=DEFAULT_SEED)
    parser.add_argument("--max-new-tokens", type=int, default=DEFAULT_MAX_NEW_TOKENS)
    parser.add_argument("--dtype", default="bf16", choices=list(_DTYPE_MAP.keys()))
    parser.add_argument("--receipt-out", default=None)
    args = parser.parse_args()

    rank = int(os.environ.get("RANK", "0"))
    world_size = int(os.environ.get("WORLD_SIZE", "1"))
    local_rank = int(os.environ.get("LOCAL_RANK", "0"))

    torch.cuda.set_device(local_rank)
    configure_determinism(args.seed)

    if rank == 0:
        print(f"[rank 0] world_size={world_size}, local_rank={local_rank}, device=cuda:{local_rank}")
        print(f"[rank 0] loading model with tp_plan='auto' (tensor parallel)...")

    tokenizer = AutoTokenizer.from_pretrained(args.model)
    torch_dtype = _DTYPE_MAP[args.dtype.lower()]

    # HF Transformers 5.x native tensor parallelism
    model = AutoModelForCausalLM.from_pretrained(
        args.model,
        torch_dtype=torch_dtype,
        tp_plan="auto",
        attn_implementation="eager",
    )
    model.eval()

    if rank == 0:
        print(f"[rank 0] model loaded, VRAM per rank: {torch.cuda.memory_allocated()/1e9:.2f} GB")

    # Ensure every rank enters generation at the same barrier
    if dist.is_initialized():
        dist.barrier()

    torch.manual_seed(args.seed)
    torch.cuda.manual_seed_all(args.seed)

    inputs = tokenizer(args.prompt, return_tensors="pt").to(f"cuda:{local_rank}")

    torch.cuda.synchronize()
    t0 = time.perf_counter()
    with torch.inference_mode():
        output = model.generate(
            **inputs,
            max_new_tokens=args.max_new_tokens,
            do_sample=False,
            temperature=0.0,
            top_p=1.0,
            num_beams=1,
            return_dict_in_generate=True,
            output_logits=True,
        )
    torch.cuda.synchronize()
    elapsed_s = time.perf_counter() - t0

    generated_ids = output.sequences[0][inputs.input_ids.shape[1]:]
    generated_text = tokenizer.decode(generated_ids, skip_special_tokens=True)

    logits_tensor = torch.stack(list(output.logits), dim=0).cpu().contiguous()
    logits_hash = sha256_bytes(logits_tensor.view(torch.uint8).numpy().tobytes())
    output_hash = sha256_bytes(
        generated_ids.cpu().numpy().tobytes() + generated_text.encode("utf-8")
    )

    if rank == 0:
        print(f"[rank 0] text: {generated_text!r}")
        print(f"[rank 0] tokens: {generated_ids.shape[0]}")
        print(f"[rank 0] elapsed: {elapsed_s:.2f}s")
        print(f"[rank 0] output_hash_hex: {output_hash.hex()}")
        print(f"[rank 0] logits_hash_hex: {logits_hash.hex()}")

    # All ranks should have identical output — prove it via all-gather of
    # the raw hash bytes as uint8 tensors (int64 overflows on unsigned SHA bytes).
    if dist.is_initialized() and world_size > 1:
        hash_tensor = torch.tensor(
            list(output_hash), dtype=torch.uint8, device=f"cuda:{local_rank}",
        )
        gathered = [torch.zeros_like(hash_tensor) for _ in range(world_size)]
        dist.all_gather(gathered, hash_tensor)
        all_match = all(torch.equal(gathered[0], g) for g in gathered[1:])
        if rank == 0:
            print(f"[rank 0] all_ranks_output_hash_match: {all_match}")

    # Only rank 0 builds and signs the receipt.
    if rank == 0:
        priv_bytes = hashlib.sha256(f"ocx-gpu-tp|seed={args.seed}".encode()).digest()
        signer = Ed25519PrivateKey.from_private_bytes(priv_bytes)
        pubkey_raw = raw_public_key(signer.public_key())

        wall_start = int(time.time() - elapsed_s)
        wall_end = int(time.time())
        started_unix = wall_start
        finished_unix = max(wall_end, started_unix + 1)
        duration_s = max(1, finished_unix - started_unix)

        # Fingerprint is tp-specific — includes world_size and rank_id
        h = hashlib.sha256()
        h.update(f"TP_FINGERPRINT|model={args.model}|dtype={args.dtype}|world={world_size}\n".encode())
        program_hash = h.digest()

        input_blob = args.prompt.encode("utf-8") + f"|seed={args.seed}|max_new_tokens={args.max_new_tokens}|tp={world_size}".encode("utf-8")
        input_hash_bytes = sha256_bytes(input_blob)

        core = ReceiptCore(
            program_hash=program_hash,
            input_hash=input_hash_bytes,
            output_hash=output_hash,
            cycles_used=max(int(generated_ids.shape[0]), duration_s + 1, 1),
            started_at=started_unix,
            finished_at=finished_unix,
            issuer_id=ISSUER_ID,
        )
        _, receipt_cbor = sign_receipt(core, signer)
        vr = verify_receipt(receipt_cbor, pubkey_raw)
        print(f"[rank 0] receipt verify: {vr.error_name} in {vr.elapsed_us:.0f}us")

        if args.receipt_out:
            summary = {
                "model": args.model,
                "tp_world_size": world_size,
                "prompt": args.prompt,
                "seed": args.seed,
                "generated_text": generated_text,
                "num_generated_tokens": int(generated_ids.shape[0]),
                "elapsed_s": round(elapsed_s, 4),
                "output_hash_hex": output_hash.hex(),
                "logits_hash_hex": logits_hash.hex(),
                "input_hash_hex": input_hash_bytes.hex(),
                "program_hash_hex": program_hash.hex(),
                "public_key_hex": pubkey_raw.hex(),
                "receipt_cbor_hex": receipt_cbor.hex(),
                "verify_ok": vr.ok,
                "verify_error": vr.error_name,
                "verify_elapsed_us": round(vr.elapsed_us, 2),
            }
            Path(args.receipt_out).write_text(json.dumps(summary, indent=2))

    if dist.is_initialized():
        dist.barrier()
        dist.destroy_process_group()

    return 0


if __name__ == "__main__":
    sys.exit(main())
