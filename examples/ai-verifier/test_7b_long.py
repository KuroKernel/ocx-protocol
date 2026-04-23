#!/usr/bin/env python3
"""Qwen 7B determinism test with a long-generation prompt — exercises more tokens
of the numerical surface."""
import hashlib, json, time, sys
from llama_cpp import Llama

MODEL = "models/qwen2.5-7b-instruct-q4_k_m-00001-of-00002.gguf"
PROMPT = "Write a short poem about deterministic computation, exactly 8 lines. Do not include any explanation, just the poem."

def sha256_json(obj) -> str:
    return hashlib.sha256(json.dumps(obj, sort_keys=True, separators=(",", ":")).encode()).hexdigest()

llm = Llama(model_path=MODEL, n_ctx=512, n_threads=1, seed=42, verbose=False)
_ = llm(PROMPT, max_tokens=1, temperature=0.0)

t0 = time.time()
out = llm(PROMPT, max_tokens=200, temperature=0.0, top_p=1.0, top_k=1, seed=42, echo=False)
elapsed = int((time.time() - t0) * 1000)
text = out["choices"][0]["text"]
h = sha256_json({"response": text})
print(f"OUTPUT_HASH={h}")
print(f"TEXT_LEN={len(text)}")
print(f"TOKENS={out['usage']['completion_tokens']}")
print(f"ELAPSED_MS={elapsed}")
print(f"PREVIEW={text[:120]!r}")
