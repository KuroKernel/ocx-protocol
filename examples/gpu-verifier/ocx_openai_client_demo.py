#!/usr/bin/env python3
"""Client demo for the OCX OpenAI-compatible server.

Shows three things:
1. Standard OpenAI SDK usage works unchanged — `client.chat.completions.create()`
   returns the normal response object. The OCX receipt is available as an
   extra field on the response.
2. An OCX-aware client can pull the receipt from the response header or body
   and verify it locally via the canonical Rust `libocx-verify.so` in under
   a millisecond.
3. The same endpoint works with `curl` for zero-SDK integration.

Usage:
    python ocx_openai_client_demo.py --server http://localhost:8000 \
        --prompt "What is the capital of France?"
"""
from __future__ import annotations

import argparse
import base64
import json
import time
from typing import Any

import requests

from ffi_verify import verify_receipt


def call_server(server: str, prompt: str, max_tokens: int = 32) -> dict[str, Any]:
    """Plain HTTP call — works from any language, any SDK."""
    t0 = time.perf_counter()
    resp = requests.post(
        f"{server}/v1/chat/completions",
        json={
            "model": "auto",
            "messages": [{"role": "user", "content": prompt}],
            "max_tokens": max_tokens,
            "temperature": 0.0,
            "seed": 42,
        },
        timeout=300,
    )
    rtt_ms = (time.perf_counter() - t0) * 1000
    resp.raise_for_status()
    body = resp.json()
    return {
        "body": body,
        "headers": dict(resp.headers),
        "rtt_ms": rtt_ms,
    }


def verify_response(response: dict[str, Any]) -> None:
    """Two ways to verify — from header (lightweight) or body (rich)."""
    headers = response["headers"]
    body = response["body"]

    print(f"HTTP round-trip: {response['rtt_ms']:.1f} ms")
    print(f"Response completion: {body['choices'][0]['message']['content']!r}")
    print(f"Tokens generated:    {body['usage']['completion_tokens']}")
    print()

    # --- Method A: header-based verification (lightweight clients) ---
    receipt_b64 = headers.get("x-ocx-receipt-b64") or headers.get("X-OCX-Receipt-B64")
    pubkey_hex = headers.get("x-ocx-publickey-hex") or headers.get("X-OCX-PublicKey-Hex")
    if not receipt_b64 or not pubkey_hex:
        print("No OCX receipt headers present.")
        return

    receipt_cbor = base64.b64decode(receipt_b64)
    pubkey = bytes.fromhex(pubkey_hex)

    vr = verify_receipt(receipt_cbor, pubkey)
    print(f"Header-based verify: {vr.error_name} in {vr.elapsed_us:.0f}us")

    # --- Method B: body-based verification (rich — access hashes etc.) ---
    ocx = body.get("ocx_receipt")
    if ocx:
        print(f"Receipt fields from body:")
        print(f"  program_hash : {ocx['program_hash_hex']}")
        print(f"  input_hash   : {ocx['input_hash_hex']}")
        print(f"  output_hash  : {ocx['output_hash_hex']}")
        print(f"  signature    : {ocx['signature_hex'][:32]}...")
        print(f"  public_key   : {ocx['public_key_hex']}")
        print(f"  server-side verify: {ocx['verify_error']} in {ocx['verify_elapsed_us']}us")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--server", default="http://localhost:8000")
    parser.add_argument("--prompt", default="What is the capital of France? Answer in one word.")
    parser.add_argument("--max-tokens", type=int, default=32)
    parser.add_argument("--runs", type=int, default=3, help="Repeat to test same-server byte-identity")
    args = parser.parse_args()

    hashes = []
    for i in range(args.runs):
        print("=" * 70)
        print(f"RUN {i+1}/{args.runs}")
        print("=" * 70)
        response = call_server(args.server, args.prompt, args.max_tokens)
        verify_response(response)
        ocx = response["body"].get("ocx_receipt", {})
        hashes.append(ocx.get("output_hash_hex"))
        print()

    if len(set(hashes)) == 1:
        print(f"SAME-SERVER DETERMINISM: PASS — all {args.runs} calls produced output_hash {hashes[0]}")
    else:
        print(f"SAME-SERVER DETERMINISM: FAIL — {len(set(hashes))} distinct hashes")
        for h in set(hashes):
            print(f"  {h}")


if __name__ == "__main__":
    main()
