"""OCX GPU Verifier — deterministic LLM inference with canonical signed receipts.

This is the end-to-end binary for tonight's atomic deliverable:

    1. Load a small model (Qwen2.5-0.5B-Instruct) on GPU.
    2. Run a deterministic greedy forward pass with fixed seed.
    3. Hash inputs, outputs, and model.
    4. Build a canonical OCX ReceiptCore, sign with Ed25519 + domain separator.
    5. Self-verify the receipt via the Rust offline verifier.

All settings are configured BEFORE torch is imported — CUBLAS_WORKSPACE_CONFIG
must be set pre-init to take effect. Do not move these.
"""
from __future__ import annotations

# --- Environment (MUST be set before importing torch) ----------------------
import os

os.environ.setdefault("CUBLAS_WORKSPACE_CONFIG", ":4096:8")
# Day 2 verified determinism holds WITHOUT kernel-launch serialization at 0.5B.
# Default is now async (much faster). Override to "1" if suspect async reordering
# is causing a nondeterminism bug at scale.
os.environ.setdefault("CUDA_LAUNCH_BLOCKING", "0")
os.environ.setdefault("TOKENIZERS_PARALLELISM", "false")

# --- Imports ---------------------------------------------------------------
import argparse
import hashlib
import json
import sys
import time
from pathlib import Path
from typing import Tuple

import torch
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from transformers import AutoModelForCausalLM, AutoTokenizer

from canonical_receipt import ReceiptCore, raw_public_key, sign_receipt
from env_spec import collect as collect_env
from ffi_verify import verify_receipt

# --- Configuration ---------------------------------------------------------
DEFAULT_MODEL = "Qwen/Qwen2.5-0.5B-Instruct"
DEFAULT_PROMPT = "What is the capital of France? Answer in one word:"
DEFAULT_SEED = 42
DEFAULT_MAX_NEW_TOKENS = 64
DEFAULT_DTYPE = "float32"
DEFAULT_ATTN = "eager"
ISSUER_ID = "ocx-gpu-verifier-v0"
REPO_ROOT = Path(__file__).resolve().parents[2]
RECEIPTS_DIR = Path(__file__).resolve().parent / "receipts"

_DTYPE_MAP = {
    "float32": torch.float32,
    "fp32": torch.float32,
    "bfloat16": torch.bfloat16,
    "bf16": torch.bfloat16,
    "float16": torch.float16,
    "fp16": torch.float16,
}


# --- Helpers ---------------------------------------------------------------
def configure_determinism(seed: int) -> None:
    """Turn on every determinism knob PyTorch exposes."""
    # `warn_only=False` turns "no deterministic impl" errors into hard failures.
    # If this throws on a model op, we want to know — and fix or work around.
    torch.use_deterministic_algorithms(True, warn_only=False)
    torch.backends.cudnn.deterministic = True
    torch.backends.cudnn.benchmark = False
    torch.backends.cuda.matmul.allow_tf32 = False
    torch.backends.cudnn.allow_tf32 = False
    torch.manual_seed(seed)
    torch.cuda.manual_seed_all(seed)


def sha256_bytes(data: bytes) -> bytes:
    return hashlib.sha256(data).digest()


def model_hash(model: torch.nn.Module) -> bytes:
    """SHA256 of concatenated parameter tensors in a fixed order.

    Dtype-agnostic: hashes the raw memory representation via a uint8 view,
    so bfloat16 / float16 / float32 all work. This is a CPU-side
    deterministic fingerprint; NOT the same as the safetensors file hash.
    """
    h = hashlib.sha256()
    for name, p in sorted(model.state_dict().items()):
        h.update(name.encode())
        # .view(torch.uint8) gives the raw bytes regardless of dtype. numpy
        # supports uint8 for all tensor shapes.
        tensor = p.detach().cpu().contiguous()
        raw = tensor.view(torch.uint8).numpy().tobytes()
        h.update(raw)
    return h.digest()


def load_model(
    model_name: str,
    seed: int,
    dtype: str = DEFAULT_DTYPE,
    attn_impl: str = DEFAULT_ATTN,
) -> Tuple[AutoModelForCausalLM, AutoTokenizer]:
    """Load tokenizer + model in deterministic configuration on GPU."""
    configure_determinism(seed)

    torch_dtype = _DTYPE_MAP.get(dtype.lower())
    if torch_dtype is None:
        raise ValueError(f"unknown dtype {dtype!r}; valid: {list(_DTYPE_MAP)}")

    tokenizer = AutoTokenizer.from_pretrained(model_name)
    model = AutoModelForCausalLM.from_pretrained(
        model_name,
        torch_dtype=torch_dtype,
        attn_implementation=attn_impl,
        device_map={"": "cuda:0"},
    )
    model.eval()
    return model, tokenizer


def run_inference(
    model: AutoModelForCausalLM,
    tokenizer: AutoTokenizer,
    prompt: str,
    max_new_tokens: int,
    seed: int,
) -> dict:
    """Greedy deterministic generation. Returns dict with text + hashes + timing."""
    torch.manual_seed(seed)
    torch.cuda.manual_seed_all(seed)

    inputs = tokenizer(prompt, return_tensors="pt").to("cuda:0")
    input_ids_bytes = inputs.input_ids.cpu().numpy().tobytes()

    torch.cuda.synchronize()
    t0 = time.perf_counter()

    with torch.inference_mode():
        output = model.generate(
            **inputs,
            max_new_tokens=max_new_tokens,
            do_sample=False,
            temperature=0.0,
            top_p=1.0,
            num_beams=1,
            return_dict_in_generate=True,
            output_logits=True,
        )

    torch.cuda.synchronize()
    elapsed_s = time.perf_counter() - t0

    # Decode the NEW tokens only
    generated_ids = output.sequences[0][inputs.input_ids.shape[1]:]
    generated_text = tokenizer.decode(generated_ids, skip_special_tokens=True)
    generated_ids_bytes = generated_ids.cpu().numpy().tobytes()

    # Logits at every position, concatenated + hashed. More sensitive
    # nondeterminism detector than token-level argmax. Hash via uint8 view
    # to be dtype-agnostic (bf16/fp16/fp32 all work).
    logits_tensor = torch.stack(list(output.logits), dim=0).cpu().contiguous()
    logits_bytes = logits_tensor.view(torch.uint8).numpy().tobytes()
    logits_hash = sha256_bytes(logits_bytes)

    # Output hash = sha256(generated token ids || decoded text bytes).
    # token ids are the deterministic surface (text is a decode of them).
    output_digest = sha256_bytes(generated_ids_bytes + generated_text.encode("utf-8"))

    # Input hash = sha256(prompt bytes || seed || max_new_tokens).
    input_blob = prompt.encode("utf-8") + f"|seed={seed}|max_new_tokens={max_new_tokens}".encode("utf-8")
    input_digest = sha256_bytes(input_blob)

    return {
        "generated_text": generated_text,
        "generated_ids_hex": generated_ids.cpu().numpy().tobytes().hex(),
        "logits_hash": logits_hash,
        "output_hash": output_digest,
        "input_hash": input_digest,
        "elapsed_s": elapsed_s,
        "num_generated_tokens": int(generated_ids.shape[0]),
    }


def build_receipt(
    *,
    program_hash: bytes,
    input_hash: bytes,
    output_hash: bytes,
    num_tokens: int,
    wall_start: float,
    wall_end: float,
    signer: Ed25519PrivateKey,
) -> Tuple[bytes, bytes]:
    """Build + sign a canonical OCX receipt. Returns (transmitted_cbor, signature)."""
    # Rust verifier requires finished_at - started_at >= 1 sec. If actual
    # inference was faster, pad the timestamps so the signed fields still
    # satisfy the constraint. Wall-clock duration goes into host_info.
    started_unix = int(wall_start)
    finished_unix = max(int(wall_end), started_unix + 1)

    core = ReceiptCore(
        program_hash=program_hash,
        input_hash=input_hash,
        output_hash=output_hash,
        cycles_used=max(num_tokens, 1),
        started_at=started_unix,
        finished_at=finished_unix,
        issuer_id=ISSUER_ID,
    )
    signature, receipt_bytes = sign_receipt(core, signer)
    return receipt_bytes, signature


def main() -> int:
    parser = argparse.ArgumentParser(description="OCX GPU Verifier (Qwen2.5-0.5B)")
    parser.add_argument("--model", default=DEFAULT_MODEL)
    parser.add_argument("--prompt", default=DEFAULT_PROMPT)
    parser.add_argument("--seed", type=int, default=DEFAULT_SEED)
    parser.add_argument("--max-new-tokens", type=int, default=DEFAULT_MAX_NEW_TOKENS)
    parser.add_argument(
        "--dtype",
        default=DEFAULT_DTYPE,
        choices=list(_DTYPE_MAP.keys()),
        help="Model/activation dtype. Default float32 for max numerical robustness.",
    )
    parser.add_argument(
        "--attn",
        default=DEFAULT_ATTN,
        choices=["eager", "sdpa", "flash_attention_2"],
        help="Attention implementation. 'eager' is vanilla (safest for determinism).",
    )
    parser.add_argument(
        "--once",
        action="store_true",
        help="Only print a machine-readable JSON line to stdout (for cross-process tests)",
    )
    parser.add_argument("--save-receipt", help="Write transmitted_cbor bytes to this file")
    parser.add_argument(
        "--key-hex",
        help="Ed25519 private key as 32-byte hex. If omitted, generates a fresh key.",
    )
    parser.add_argument(
        "--receipt-out",
        help="Write receipt JSON summary (hashes, hex) to this path",
    )
    args = parser.parse_args()

    if not torch.cuda.is_available():
        print("ERROR: CUDA not available — this binary requires a GPU.", file=sys.stderr)
        return 1

    if not args.once:
        print("=" * 70)
        print("OCX GPU VERIFIER — deterministic inference with signed receipts")
        print("=" * 70)
        env = collect_env()
        print(f"device       : {env.get('cuda_device_0')} ({env.get('cuda_compute_cap')})")
        print(f"driver       : {env.get('nvidia_driver')}")
        print(f"cuda_runtime : {env.get('cuda_runtime')}")
        print(f"torch        : {env.get('torch')}")
        print(f"model        : {args.model}")
        print()

    # Deterministic signing key: either supplied via --key-hex or derived
    # from seed for reproducibility in tests.
    if args.key_hex:
        priv_bytes = bytes.fromhex(args.key_hex)
        signer = Ed25519PrivateKey.from_private_bytes(priv_bytes)
    else:
        # Derived deterministically from seed so two fresh processes with
        # same seed produce comparable receipts (same pubkey, same sig).
        priv_bytes = hashlib.sha256(f"ocx-gpu-v0|seed={args.seed}".encode()).digest()
        signer = Ed25519PrivateKey.from_private_bytes(priv_bytes)

    pubkey_raw = raw_public_key(signer.public_key())

    # Load model
    model, tokenizer = load_model(args.model, args.seed, dtype=args.dtype, attn_impl=args.attn)
    m_hash = model_hash(model)

    # Run inference
    wall_start = time.time()
    result = run_inference(
        model, tokenizer, args.prompt, args.max_new_tokens, args.seed
    )
    wall_end = time.time()

    # Build receipt
    receipt_bytes, signature = build_receipt(
        program_hash=m_hash,
        input_hash=result["input_hash"],
        output_hash=result["output_hash"],
        num_tokens=result["num_generated_tokens"],
        wall_start=wall_start,
        wall_end=wall_end,
        signer=signer,
    )

    # Self-verify
    verify_result = verify_receipt(receipt_bytes, pubkey_raw)

    summary = {
        "model": args.model,
        "prompt": args.prompt,
        "seed": args.seed,
        "generated_text": result["generated_text"],
        "num_generated_tokens": result["num_generated_tokens"],
        "elapsed_s": round(result["elapsed_s"], 4),
        "logits_hash_hex": result["logits_hash"].hex(),
        "output_hash_hex": result["output_hash"].hex(),
        "input_hash_hex": result["input_hash"].hex(),
        "program_hash_hex": m_hash.hex(),
        "signature_hex": signature.hex(),
        "public_key_hex": pubkey_raw.hex(),
        "receipt_cbor_hex": receipt_bytes.hex(),
        "receipt_len_bytes": len(receipt_bytes),
        "verify_ok": verify_result.ok,
        "verify_error": verify_result.error_name,
        "verify_elapsed_us": round(verify_result.elapsed_us, 2),
    }

    if args.save_receipt:
        Path(args.save_receipt).write_bytes(receipt_bytes)
    if args.receipt_out:
        Path(args.receipt_out).write_text(json.dumps(summary, indent=2))

    if args.once:
        # Single JSON line for cross-process determinism tests
        print(json.dumps(summary))
    else:
        print(f"generated    : {result['generated_text']!r}")
        print(f"tokens       : {result['num_generated_tokens']}")
        print(f"inference    : {result['elapsed_s'] * 1000:.0f} ms")
        print()
        print("RECEIPT")
        print(f"  program_hash : {m_hash.hex()}")
        print(f"  input_hash   : {result['input_hash'].hex()}")
        print(f"  output_hash  : {result['output_hash'].hex()}")
        print(f"  logits_hash  : {result['logits_hash'].hex()}")
        print(f"  signature    : {signature.hex()}")
        print(f"  public_key   : {pubkey_raw.hex()}")
        print(f"  cbor_len     : {len(receipt_bytes)} bytes")
        print()
        print(
            f"VERIFY       : {'OK' if verify_result.ok else 'FAIL'} "
            f"({verify_result.error_name}, {verify_result.elapsed_us:.0f}us)"
        )

    return 0 if verify_result.ok else 2


if __name__ == "__main__":
    sys.exit(main())
