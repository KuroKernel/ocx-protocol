# OCX AI Verifier

**Proving AI isn't bullshitting with cryptographic receipts.**

This example demonstrates how OCX can create verifiable proofs for AI inference, ensuring that AI systems can be held accountable for their outputs.

## The Problem

When an AI (LLM, memory system, chatbot) gives you an answer, you can't verify:
- What data did it actually use?
- Is the reasoning reproducible?
- Was the memory real, or hallucinated?

Users just have to... trust it.

## The Solution

OCX creates a cryptographic receipt for every AI inference:

```
Prompt → Model → Response → OCX Receipt

The receipt proves:
├── input_hash:  SHA256 of the exact prompt
├── model_hash:  SHA256 of the model file (proves which model)
├── output_hash: SHA256 of the exact response
├── timestamp:   When the inference happened
└── signature:   Ed25519 cryptographic proof
```

Anyone can verify by re-running the same inference and checking the hashes match.

## Quick Start

```bash
# Create virtual environment
python3 -m venv venv
source venv/bin/activate

# Install dependencies
pip install llama-cpp-python huggingface_hub cryptography

# Download a small model (469MB)
python -c "
from huggingface_hub import hf_hub_download
import os
os.makedirs('models', exist_ok=True)
hf_hub_download(
    repo_id='Qwen/Qwen2.5-0.5B-Instruct-GGUF',
    filename='qwen2.5-0.5b-instruct-q4_k_m.gguf',
    local_dir='./models'
)
"

# Test determinism first
python test_determinism.py

# Run the full demo
python ocx_ai_verifier.py
```

## Example Output

```
======================================================================
OCX VERIFIABLE AI DEMO
Proving AI isn't bullshitting with cryptographic receipts
======================================================================

[1/5] Initializing Verifiable AI...
Model hash: 74a4da8c9fdbcd15...
Public key: c0853a724344fc63...

[2/5] Running inference...
      Prompt: 'What is the capital of France? Answer in one word:'

[3/5] AI Response:
      'Paris...'

[4/5] OCX Receipt (Cryptographic Proof):
----------------------------------------------------------------------
  Receipt ID:  ocx-ai-a0e0483da314
  Input Hash:  e4c73ea2b4ced9cd...
  Output Hash: 1442c65fa823be6f...
  Model Hash:  74a4da8c9fdbcd15...
  Signature:   120a9dd098db0b27...
----------------------------------------------------------------------

[5/5] Verification:
  Model hash valid:  YES
  Input hash valid:  YES
  Output hash valid: YES
  Signature valid:   YES
----------------------------------------------------------------------
  OVERALL: VERIFIED - AI is NOT bullshitting!
======================================================================
```

## How Verification Works

1. **Model Hash**: SHA256 of the entire model file. Proves which exact model was used.
2. **Input Hash**: SHA256 of the prompt + parameters. Proves what was asked.
3. **Output Hash**: SHA256 of the response. Proves what was answered.
4. **Signature**: Ed25519 signature over the canonical receipt. Proves authenticity.

To verify, anyone can:
1. Download the same model (verify hash matches)
2. Run the same prompt with same parameters
3. Check output matches (deterministic inference)
4. Verify signature with public key

## Determinism

AI inference is typically non-deterministic (temperature, sampling). This demo achieves determinism by:

- `temperature=0` (greedy decoding)
- `top_k=1` (only pick top token)
- `seed=42` (fixed random seed)
- Single-threaded execution
- CPU-only (no GPU variance)

The `test_determinism.py` script verifies that 5 consecutive runs produce identical output.

## Use Cases

- **Memory/RAG Systems**: Prove which memories the AI accessed
- **Financial AI**: Audit trail for AI-driven decisions
- **Healthcare AI**: Verifiable diagnostic reasoning
- **Legal AI**: Non-repudiable AI-generated documents
- **Enterprise Chatbots**: Compliance and accountability

## Files

- `ocx_ai_verifier.py` - Main demo with receipt generation and verification
- `test_determinism.py` - Verifies model produces deterministic output

## Requirements

- Python 3.10+
- ~2GB RAM (for small quantized models)
- ~500MB disk (for model)

## License

MIT - Same as OCX Protocol
