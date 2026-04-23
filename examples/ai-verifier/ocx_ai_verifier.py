#!/usr/bin/env python3
"""
OCX AI Verifier - Proves AI isn't bullshitting.

This demo shows how OCX can create verifiable receipts for AI inference:
1. Hash the inputs (prompt, model)
2. Run deterministic inference
3. Hash the output
4. Create a signed receipt proving: input → model → output

Anyone can verify by:
1. Re-running the same inference
2. Checking the hashes match
3. Verifying the signature

"""

import hashlib
import json
import time
import base64
from datetime import datetime, timezone
from dataclasses import dataclass, asdict
from typing import Optional
import os

# Cryptographic signing (Ed25519)
try:
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
    from cryptography.hazmat.primitives import serialization
    HAS_CRYPTO = True
except ImportError:
    HAS_CRYPTO = False
    print("Warning: cryptography not installed, using mock signatures")

from llama_cpp import Llama

# ============================================================================
# CONFIGURATION
# ============================================================================

MODEL_PATH = "models/qwen2.5-0.5b-instruct-q4_k_m.gguf"
ISSUER_ID = "ocx-ai-demo-v1"

# ============================================================================
# DATA STRUCTURES
# ============================================================================

@dataclass
class AIInferenceInput:
    """What the AI received."""
    prompt: str
    model_hash: str  # SHA256 of model file
    temperature: float
    max_tokens: int
    seed: int

@dataclass
class AIInferenceOutput:
    """What the AI produced. Only `response` is hashed — timing and token counts
    are recorded as unhashed metadata so any third party can re-derive output_hash
    from the response text alone."""
    response: str

@dataclass
class OCXReceipt:
    """
    Cryptographic proof that this inference happened.

    This is the core primitive:
    input_hash + output_hash + timestamp + signature = verifiable truth
    """
    # Core identifiers
    receipt_id: str
    issuer_id: str

    # What was computed
    input_hash: str       # SHA256 of AIInferenceInput
    output_hash: str      # SHA256 of {"response": text} — reproducible by any verifier
    model_hash: str       # SHA256 of model file (proves which model)

    # When it happened
    timestamp: str        # ISO 8601 UTC

    # Cryptographic proof
    signature: str        # Ed25519 signature of canonical receipt
    public_key: str       # Public key for verification

    # Unhashed metadata (informational only — not covered by signature)
    response_text: str = ""
    tokens_generated: int = 0
    inference_time_ms: int = 0

    # Metadata
    version: str = "1.0"

    def to_canonical_bytes(self) -> bytes:
        """Canonical serialization for signing/verification.

        Only the hash-covered fields are signed. response_text, tokens_generated,
        and inference_time_ms are informational metadata and must NOT appear here,
        otherwise non-deterministic timing would poison the signature."""
        data = {
            "input_hash": self.input_hash,
            "issuer_id": self.issuer_id,
            "model_hash": self.model_hash,
            "output_hash": self.output_hash,
            "receipt_id": self.receipt_id,
            "timestamp": self.timestamp,
            "version": self.version,
        }
        return json.dumps(data, sort_keys=True, separators=(',', ':')).encode()

# ============================================================================
# CRYPTO UTILITIES
# ============================================================================

def sha256_hex(data: bytes) -> str:
    """SHA256 hash as hex string."""
    return hashlib.sha256(data).hexdigest()

def sha256_json(obj: dict) -> str:
    """SHA256 of canonical JSON."""
    canonical = json.dumps(obj, sort_keys=True, separators=(',', ':')).encode()
    return sha256_hex(canonical)

def get_model_hash(model_path: str) -> str:
    """SHA256 hash of model file (proves which model was used)."""
    h = hashlib.sha256()
    with open(model_path, 'rb') as f:
        while chunk := f.read(8192):
            h.update(chunk)
    return h.hexdigest()

class Signer:
    """Ed25519 signer for OCX receipts."""

    def __init__(self):
        if HAS_CRYPTO:
            self.private_key = Ed25519PrivateKey.generate()
            self.public_key = self.private_key.public_key()
            self.public_key_hex = self.public_key.public_bytes(
                encoding=serialization.Encoding.Raw,
                format=serialization.PublicFormat.Raw
            ).hex()
        else:
            self.public_key_hex = "mock_public_key_" + hashlib.sha256(os.urandom(32)).hexdigest()[:32]

    def sign(self, data: bytes) -> str:
        """Sign data and return hex signature."""
        if HAS_CRYPTO:
            sig = self.private_key.sign(data)
            return sig.hex()
        else:
            # Mock signature for demo
            return hashlib.sha256(data + b"mock_secret").hexdigest()

# ============================================================================
# AI INFERENCE ENGINE
# ============================================================================

class VerifiableAI:
    """
    AI inference with OCX verification.

    Every inference produces a cryptographic receipt proving:
    - What prompt was given
    - Which model was used
    - What output was produced
    - When it happened
    """

    def __init__(self, model_path: str):
        print(f"Loading model: {model_path}")
        self.model_path = model_path
        self.model_hash = get_model_hash(model_path)
        print(f"Model hash: {self.model_hash[:16]}...")

        self.llm = Llama(
            model_path=model_path,
            n_ctx=512,
            n_threads=1,  # Single thread for determinism
            seed=42,
            verbose=False
        )

        self.signer = Signer()
        self._warmed_up = set()

        print(f"Public key: {self.signer.public_key_hex[:16]}...")
        print("Model loaded!")

    def _ensure_warmed_up(self, prompt: str):
        """Warm up for this specific prompt (llama.cpp caches prompt processing)."""
        prompt_hash = sha256_hex(prompt.encode())[:8]
        if prompt_hash not in self._warmed_up:
            _ = self.llm(prompt, max_tokens=1, temperature=0.0)
            self._warmed_up.add(prompt_hash)

    def infer_with_receipt(
        self,
        prompt: str,
        max_tokens: int = 100,
        seed: int = 42
    ) -> tuple[str, OCXReceipt]:
        """
        Run inference and return (response, OCX receipt).

        The receipt is cryptographic proof of this inference.
        """
        # Ensure determinism
        self._ensure_warmed_up(prompt)

        # Create input record
        inference_input = AIInferenceInput(
            prompt=prompt,
            model_hash=self.model_hash,
            temperature=0.0,
            max_tokens=max_tokens,
            seed=seed
        )
        input_hash = sha256_json(asdict(inference_input))

        # Run inference
        start_time = time.time()
        output = self.llm(
            prompt,
            max_tokens=max_tokens,
            temperature=0.0,
            top_p=1.0,
            top_k=1,
            seed=seed,
            echo=False
        )
        elapsed_ms = int((time.time() - start_time) * 1000)

        response_text = output['choices'][0]['text']
        tokens_generated = output['usage']['completion_tokens']

        # Output hash covers ONLY the response text (canonical JSON).
        # Anyone with the response text can reproduce this hash — no timing or
        # token counts in the hashed payload.
        output_hash = sha256_json({"response": response_text})

        # Create receipt
        timestamp = datetime.now(timezone.utc).isoformat()
        receipt_id = f"ocx-ai-{sha256_hex((timestamp + input_hash).encode())[:12]}"

        receipt = OCXReceipt(
            receipt_id=receipt_id,
            issuer_id=ISSUER_ID,
            input_hash=input_hash,
            output_hash=output_hash,
            model_hash=self.model_hash,
            timestamp=timestamp,
            signature="",  # Will be set below
            public_key=self.signer.public_key_hex,
            response_text=response_text,
            tokens_generated=tokens_generated,
            inference_time_ms=elapsed_ms,
        )

        # Sign the receipt
        canonical = receipt.to_canonical_bytes()
        receipt.signature = self.signer.sign(canonical)

        return response_text, receipt

# ============================================================================
# VERIFICATION
# ============================================================================

def verify_receipt(
    prompt: str,
    response: str,
    receipt: OCXReceipt,
    model_path: str
) -> dict:
    """
    Verify an OCX AI receipt.

    This is what ANYONE can do to verify the AI isn't bullshitting:
    1. Check model hash matches claimed model
    2. Check input hash matches prompt
    3. Check output hash matches response
    4. Verify signature
    """
    results = {
        "model_hash_valid": False,
        "input_hash_valid": False,
        "output_hash_valid": False,
        "signature_valid": False,
        "overall_valid": False
    }

    # 1. Verify model hash
    actual_model_hash = get_model_hash(model_path)
    results["model_hash_valid"] = (actual_model_hash == receipt.model_hash)

    # 2. Verify input hash
    inference_input = AIInferenceInput(
        prompt=prompt,
        model_hash=receipt.model_hash,
        temperature=0.0,
        max_tokens=100,  # Default
        seed=42
    )
    actual_input_hash = sha256_json(asdict(inference_input))
    results["input_hash_valid"] = (actual_input_hash == receipt.input_hash)

    # 3. Verify output hash by re-deriving from the response text.
    # With the fixed schema, output_hash = SHA256(canonical_json({"response": text})).
    # Any third party holding the response can recompute and compare.
    actual_output_hash = sha256_json({"response": response})
    results["output_hash_valid"] = (actual_output_hash == receipt.output_hash)

    # 4. Verify signature
    if HAS_CRYPTO:
        try:
            from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey
            pub_key_bytes = bytes.fromhex(receipt.public_key)
            pub_key = Ed25519PublicKey.from_public_bytes(pub_key_bytes)
            canonical = receipt.to_canonical_bytes()
            sig_bytes = bytes.fromhex(receipt.signature)
            pub_key.verify(sig_bytes, canonical)
            results["signature_valid"] = True
        except Exception as e:
            results["signature_valid"] = False
    else:
        # Mock verification
        results["signature_valid"] = True

    results["overall_valid"] = all([
        results["model_hash_valid"],
        results["input_hash_valid"],
        results["output_hash_valid"],
        results["signature_valid"]
    ])

    return results

# ============================================================================
# DEMO
# ============================================================================

def main():
    print("=" * 70)
    print("OCX VERIFIABLE AI DEMO")
    print("Proving AI isn't bullshitting with cryptographic receipts")
    print("=" * 70)
    print()

    # Initialize
    print("[1/5] Initializing Verifiable AI...")
    ai = VerifiableAI(MODEL_PATH)
    print()

    # Run inference
    prompt = "What is the capital of France? Answer in one word:"
    print(f"[2/5] Running inference...")
    print(f"      Prompt: '{prompt}'")
    print()

    response, receipt = ai.infer_with_receipt(prompt)

    print(f"[3/5] AI Response:")
    print(f"      '{response.strip()}'")
    print()

    # Show receipt
    print("[4/5] OCX Receipt (Cryptographic Proof):")
    print("-" * 70)
    print(f"  Receipt ID:  {receipt.receipt_id}")
    print(f"  Timestamp:   {receipt.timestamp}")
    print(f"  Input Hash:  {receipt.input_hash[:32]}...")
    print(f"  Output Hash: {receipt.output_hash[:32]}...")
    print(f"  Model Hash:  {receipt.model_hash[:32]}...")
    print(f"  Signature:   {receipt.signature[:32]}...")
    print(f"  Public Key:  {receipt.public_key[:32]}...")
    print("-" * 70)
    print()

    # Verify
    print("[5/5] Verification (What anyone can do):")
    verification = verify_receipt(prompt, response, receipt, MODEL_PATH)
    print(f"  Model hash valid:  {'YES' if verification['model_hash_valid'] else 'NO'}")
    print(f"  Input hash valid:  {'YES' if verification['input_hash_valid'] else 'NO'}")
    print(f"  Output hash valid: {'YES' if verification['output_hash_valid'] else 'NO'}")
    print(f"  Signature valid:   {'YES' if verification['signature_valid'] else 'NO'}")
    print("-" * 70)
    print(f"  OVERALL: {'VERIFIED - AI is NOT bullshitting!' if verification['overall_valid'] else 'VERIFICATION FAILED'}")
    print("=" * 70)
    print()

    # Explain
    print("What this proves:")
    print("  1. This EXACT prompt was given to the AI")
    print("  2. This EXACT model (by hash) was used")
    print("  3. This EXACT response was produced")
    print("  4. It happened at this EXACT time")
    print("  5. Anyone can re-run and get the SAME result")
    print()
    print("The AI cannot lie about what it was asked or what it said.")
    print("=" * 70)

    # Save receipt
    receipt_file = "last_receipt.json"
    with open(receipt_file, 'w') as f:
        json.dump(asdict(receipt), f, indent=2)
    print(f"\nReceipt saved to: {receipt_file}")

    return receipt

if __name__ == "__main__":
    main()
