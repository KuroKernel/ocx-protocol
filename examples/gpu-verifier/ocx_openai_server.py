#!/usr/bin/env python3
"""OCX OpenAI-compatible inference server.

A drop-in replacement for `openai.ChatCompletion.create()` that returns the
completion AND a canonical OCX receipt in a response header
(`X-OCX-Receipt`, base64-encoded CBOR) plus the raw receipt hex in the
response body under the extension field `ocx_receipt_hex`.

Clients that know about OCX can verify every response. Clients that don't
care just get a standard OpenAI response.

Usage:
    python ocx_openai_server.py --model Qwen/Qwen2.5-72B-Instruct \
        --device-map auto --dtype bf16 --port 8000

    curl -X POST http://localhost:8000/v1/chat/completions \
        -H "Content-Type: application/json" \
        -d '{
            "model": "Qwen/Qwen2.5-72B-Instruct",
            "messages": [{"role": "user", "content": "What is the capital of France?"}],
            "max_tokens": 32,
            "temperature": 0.0
        }'

The response body is a standard OpenAI ChatCompletion with an extra
`ocx_receipt` object. The HTTP response also carries `X-OCX-Receipt-B64`
and `X-OCX-PublicKey-Hex` headers so lightweight clients that never touch
the body can still verify.

Verification from any client (no OCX library needed, just ctypes + the
canonical `libocx-verify.so`):

    receipt_cbor = base64.b64decode(response.headers['X-OCX-Receipt-B64'])
    pubkey = bytes.fromhex(response.headers['X-OCX-PublicKey-Hex'])
    lib.ocx_verify_receipt(receipt_cbor, len(receipt_cbor), pubkey)
"""
from __future__ import annotations

# Determinism flags MUST be set before importing torch
import os
os.environ.setdefault("CUBLAS_WORKSPACE_CONFIG", ":4096:8")
os.environ.setdefault("CUDA_LAUNCH_BLOCKING", "0")
os.environ.setdefault("TOKENIZERS_PARALLELISM", "false")

import argparse
import base64
import hashlib
import json
import time
import uuid
from contextlib import asynccontextmanager
from typing import Optional

import torch
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from fastapi import FastAPI, HTTPException, Response
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field
from transformers import AutoModelForCausalLM, AutoTokenizer
import uvicorn

from canonical_receipt import ReceiptCore, raw_public_key, sign_receipt
from ffi_verify import verify_receipt as ffi_verify_receipt

_DTYPE_MAP = {
    "float32": torch.float32,
    "fp32": torch.float32,
    "bfloat16": torch.bfloat16,
    "bf16": torch.bfloat16,
    "float16": torch.float16,
    "fp16": torch.float16,
}


class ChatMessage(BaseModel):
    role: str
    content: str


class ChatCompletionRequest(BaseModel):
    model: str
    messages: list[ChatMessage]
    max_tokens: Optional[int] = 128
    temperature: Optional[float] = 0.0
    top_p: Optional[float] = 1.0
    seed: Optional[int] = 42
    # OCX-specific extensions (ignored by OpenAI clients):
    ocx_issuer: Optional[str] = "ocx-openai-server-v0"


class InferenceState:
    """Holds the loaded model + signing key across requests."""
    def __init__(self):
        self.model = None
        self.tokenizer = None
        self.model_name: Optional[str] = None
        self.signer: Optional[Ed25519PrivateKey] = None
        self.pubkey_raw: Optional[bytes] = None
        self.dtype: str = "bf16"
        self.attn: str = "eager"
        self.device_map: str = "cuda:0"


STATE = InferenceState()


def configure_determinism(seed: int) -> None:
    torch.use_deterministic_algorithms(True, warn_only=True)
    torch.backends.cudnn.deterministic = True
    torch.backends.cudnn.benchmark = False
    torch.backends.cuda.matmul.allow_tf32 = False
    torch.backends.cudnn.allow_tf32 = False
    torch.manual_seed(seed)
    torch.cuda.manual_seed_all(seed)


def load_model(model_name: str, dtype: str, attn: str, device_map: str, seed: int) -> None:
    """Populate the global STATE with a loaded model."""
    configure_determinism(seed)
    torch_dtype = _DTYPE_MAP[dtype.lower()]
    device_map_arg = {"": device_map} if device_map != "auto" else "auto"

    print(f"[ocx-server] loading {model_name} (dtype={dtype}, attn={attn}, device_map={device_map})...")
    tokenizer = AutoTokenizer.from_pretrained(model_name)
    model = AutoModelForCausalLM.from_pretrained(
        model_name,
        torch_dtype=torch_dtype,
        attn_implementation=attn,
        device_map=device_map_arg,
    )
    model.eval()

    # Deterministic server-side signing key derived from the server's
    # stable identity. In production this would come from an HSM; for the
    # demo we generate a fresh one on boot.
    signer = Ed25519PrivateKey.generate()

    STATE.model = model
    STATE.tokenizer = tokenizer
    STATE.model_name = model_name
    STATE.signer = signer
    STATE.pubkey_raw = raw_public_key(signer.public_key())
    STATE.dtype = dtype
    STATE.attn = attn
    STATE.device_map = device_map
    print(f"[ocx-server] model loaded. public_key: {STATE.pubkey_raw.hex()}")


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Load the model on startup."""
    opts = app.state.startup_opts
    load_model(opts["model"], opts["dtype"], opts["attn"], opts["device_map"], opts["seed"])
    yield


app = FastAPI(title="OCX Verifiable Chat Completions", lifespan=lifespan)


def build_prompt(messages: list[ChatMessage]) -> str:
    """Render messages via the tokenizer's chat template."""
    return STATE.tokenizer.apply_chat_template(
        [{"role": m.role, "content": m.content} for m in messages],
        tokenize=False,
        add_generation_prompt=True,
    )


def sha256_bytes(data: bytes) -> bytes:
    return hashlib.sha256(data).digest()


def sha256_json(obj) -> bytes:
    return hashlib.sha256(
        json.dumps(obj, sort_keys=True, separators=(",", ":")).encode()
    ).hexdigest().encode()  # return hex-bytes so callers can decode-and-compare


def fingerprint_model() -> bytes:
    """Stable per-load fingerprint for the loaded model + config.

    NOT the full weight hash — for large offloaded models that's infeasible
    per-request. Uses name + dtype + attn + device_map + a handful of
    parameter shape names. Still deterministic for a given loaded model.
    """
    h = hashlib.sha256()
    h.update(b"OCX_SERVER_MODEL_FP\n")
    h.update(f"model={STATE.model_name}\n".encode())
    h.update(f"dtype={STATE.dtype}\n".encode())
    h.update(f"attn={STATE.attn}\n".encode())
    h.update(f"device_map={STATE.device_map}\n".encode())
    # Top-k parameter fingerprint (stable, quick)
    for i, (name, p) in enumerate(sorted(STATE.model.state_dict().items())):
        h.update(f"{name}|{p.dtype}|{tuple(p.shape)}\n".encode())
        if i >= 20:  # cap — full walk is slow for 72B
            break
    return h.digest()


@app.post("/v1/chat/completions")
async def chat_completions(req: ChatCompletionRequest):
    if STATE.model is None:
        raise HTTPException(503, "Model not loaded yet")

    # Render + tokenize
    prompt_text = build_prompt(req.messages)
    inputs = STATE.tokenizer(prompt_text, return_tensors="pt").to(next(STATE.model.parameters()).device)

    # Deterministic decoding config
    torch.manual_seed(req.seed)
    torch.cuda.manual_seed_all(req.seed)

    t0 = time.perf_counter()
    with torch.inference_mode():
        output = STATE.model.generate(
            **inputs,
            max_new_tokens=req.max_tokens,
            do_sample=False,
            temperature=0.0,
            top_p=1.0,
            num_beams=1,
            return_dict_in_generate=True,
            output_logits=False,  # don't allocate logits tensors for server throughput
        )
    elapsed_s = time.perf_counter() - t0

    generated_ids = output.sequences[0][inputs.input_ids.shape[1]:]
    text = STATE.tokenizer.decode(generated_ids, skip_special_tokens=True)
    num_tokens = int(generated_ids.shape[0])

    # Build receipt
    program_hash = fingerprint_model()
    input_blob = (
        prompt_text.encode("utf-8")
        + f"|seed={req.seed}|max_tokens={req.max_tokens}|temp={req.temperature}|top_p={req.top_p}".encode()
    )
    input_hash = sha256_bytes(input_blob)
    output_hash = sha256_bytes(
        generated_ids.cpu().numpy().tobytes() + text.encode("utf-8")
    )

    now = int(time.time())
    started_unix = now - max(1, int(elapsed_s))
    finished_unix = now
    duration_s = max(1, finished_unix - started_unix)

    core = ReceiptCore(
        program_hash=program_hash,
        input_hash=input_hash,
        output_hash=output_hash,
        cycles_used=max(num_tokens, duration_s + 1, 1),
        started_at=started_unix,
        finished_at=finished_unix,
        issuer_id=req.ocx_issuer,
    )
    signature, receipt_cbor = sign_receipt(core, STATE.signer)

    # Self-verify before returning — catches misconfiguration immediately
    vr = ffi_verify_receipt(receipt_cbor, STATE.pubkey_raw)

    # OpenAI-format response body + OCX extension
    completion_id = f"chatcmpl-{uuid.uuid4().hex[:24]}"
    body = {
        "id": completion_id,
        "object": "chat.completion",
        "created": finished_unix,
        "model": req.model,
        "choices": [{
            "index": 0,
            "message": {"role": "assistant", "content": text},
            "finish_reason": "stop",
        }],
        "usage": {
            "prompt_tokens": int(inputs.input_ids.shape[1]),
            "completion_tokens": num_tokens,
            "total_tokens": int(inputs.input_ids.shape[1]) + num_tokens,
        },
        # OCX extension (ignored by OpenAI SDK, used by OCX-aware clients)
        "ocx_receipt": {
            "receipt_cbor_hex": receipt_cbor.hex(),
            "signature_hex": signature.hex(),
            "public_key_hex": STATE.pubkey_raw.hex(),
            "program_hash_hex": program_hash.hex(),
            "input_hash_hex": input_hash.hex(),
            "output_hash_hex": output_hash.hex(),
            "verify_ok": vr.ok,
            "verify_error": vr.error_name,
            "verify_elapsed_us": round(vr.elapsed_us, 2),
        },
    }

    resp = JSONResponse(body)
    resp.headers["X-OCX-Receipt-B64"] = base64.b64encode(receipt_cbor).decode()
    resp.headers["X-OCX-PublicKey-Hex"] = STATE.pubkey_raw.hex()
    resp.headers["X-OCX-Verify"] = "OCX_SUCCESS" if vr.ok else vr.error_name
    resp.headers["X-OCX-Verify-Us"] = str(int(vr.elapsed_us))
    return resp


@app.get("/v1/models")
async def list_models():
    """Minimal OpenAI /v1/models endpoint so SDK probes don't 404."""
    return {
        "object": "list",
        "data": [{
            "id": STATE.model_name,
            "object": "model",
            "owned_by": "ocx",
            "ocx_public_key_hex": STATE.pubkey_raw.hex() if STATE.pubkey_raw else None,
        }],
    }


@app.get("/healthz")
async def health():
    return {
        "status": "ok" if STATE.model is not None else "loading",
        "model": STATE.model_name,
        "public_key_hex": STATE.pubkey_raw.hex() if STATE.pubkey_raw else None,
    }


@app.get("/ocx/public_key")
async def public_key():
    """Return the server's signing public key for clients to configure."""
    if STATE.pubkey_raw is None:
        raise HTTPException(503, "Server not ready")
    return {"public_key_hex": STATE.pubkey_raw.hex()}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", default="Qwen/Qwen2.5-72B-Instruct")
    parser.add_argument("--dtype", default="bf16", choices=list(_DTYPE_MAP.keys()))
    parser.add_argument("--attn", default="eager", choices=["eager", "sdpa", "flash_attention_2"])
    parser.add_argument("--device-map", default="auto")
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--host", default="0.0.0.0")
    parser.add_argument("--port", type=int, default=8000)
    args = parser.parse_args()

    app.state.startup_opts = {
        "model": args.model,
        "dtype": args.dtype,
        "attn": args.attn,
        "device_map": args.device_map,
        "seed": args.seed,
    }

    uvicorn.run(app, host=args.host, port=args.port)


if __name__ == "__main__":
    main()
