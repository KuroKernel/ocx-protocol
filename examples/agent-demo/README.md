# OCX-signed agent demo

A Claude Code-style agent loop where every step — every model call,
every tool invocation, every file the agent writes — produces an OCX
receipt. The receipts form a hash-linked chain, signed end-to-end by
the agent's Ed25519 key. Anyone can run the verifier on the chain and
confirm: the agent did exactly what the chain claims, no more, no less.

This is the AI-accountability use case that the OCX whitepaper argues
for, made concrete.

---

## What gets signed

| Step kind | When | Inputs in receipt | Outputs in receipt |
|---|---|---|---|
| `model_call:turn=N` | every Claude API call | model name, system-prompt prefix, message count, last role | stop reason, content (text + tool_use blocks), token usage |
| `tool_call:read_file` / `:list_files` / `:grep` | every tool the agent invokes | tool name + args + the tool_use_id Claude sent | result preview + result byte count |
| `file_write:report.md` | the audit report at end | path + byte count | path + content sha256 prefix |

Each receipt's `prev_receipt_hash` points to the canonical-CBOR sha256
of the previous receipt, so a single byte changed anywhere in the
chain makes the next step's signature invalid against the recorded
hash. Issuer key id is verified to be consistent throughout.

The full receipt schema is the canonical OCX `ReceiptCore` (CBOR map
with integer keys 1–9, plus the issuer signature at key 8). It is
byte-for-byte the same as receipts produced by the GPU verifier and
verified by `libocx-verify`.

---

## Run it

```bash
# 1. Build the canonical Rust verifier (one-time)
cd ../../libocx-verify && cargo build --release && cd -

# 2. Set your Anthropic API key
export ANTHROPIC_API_KEY=sk-ant-...

# 3. Install the three runtime deps
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# 4. Run the agent on the audit-target sample
python3 run.py
```

`run.py` writes everything under `output/`:

```
output/
├── receipts.cbor   # the signed chain (verifier reads this)
├── receipts.json   # human-readable transcript with hex hashes
├── pubkey.txt      # base64 raw 32-byte Ed25519 public key
└── report.md       # the agent's audit report
```

---

## Verify the chain

```bash
python3 verify_chain.py output/
```

You'll see one line per receipt and a final verdict:

```
loaded 13 receipts from output/receipts.cbor
verifier  : ../../libocx-verify/target/release/liblibocx_verify.so
public key: 7c5d…

  [ 0] OK    sig=OCX_SUCCESS              hash=8a7d2e1c…  kind=model_call:turn=0
  [ 1] OK    sig=OCX_SUCCESS              hash=fde318f0…  kind=tool_call:list_files
  [ 2] OK    sig=OCX_SUCCESS              hash=4193b84d…  kind=tool_call:read_file
  …
  [12] OK    sig=OCX_SUCCESS              hash=2f9c01a4…  kind=file_write:report.md

OCX_CHAIN_VALID   len=13  issuer='ocx-agent-demo-v0'  first=8a7d2e1c…  last=2f9c01a4…
```

If anyone modifies any byte in `receipts.cbor` — flips a single character
in a tool result, swaps two receipts, drops the file_write step — the
chain breaks loudly:

```
OCX_CHAIN_BROKEN  at receipt 7: prev_receipt_hash mismatch:
  receipt[7].prev = 4f1a8b2c…  expected = 8d3c19af…
```

---

## What the demo proves (and what it does NOT)

Proves:
  • The agent's Ed25519 key signed every step.
  • The chain is internally consistent — nothing was inserted, removed,
    or reordered after issuance.
  • The model inputs, tool inputs, and tool outputs at each step are
    exactly what the receipts claim. Tampering is detectable.

Does NOT prove:
  • That Claude actually returned what's recorded. We trust our own
    process; a TEE-based or oracle-based attestation is a separate
    extension on top of this base. The OCX layer guarantees the
    transcript was recorded honestly by an agent holding the key.
  • That re-running the agent will produce the same output. LLM
    determinism is a separate problem (covered by the GPU-verifier
    track: deterministic-inference receipts).

The point of this demo is the **transcript layer**: a tamper-evident,
publicly replayable, third-party-verifiable record of what an AI agent
did. With the GPU verifier underneath, you can also bind each model
call to a deterministic-inference receipt and get an end-to-end
guarantee.

---

## Files

```
examples/agent-demo/
├── README.md
├── requirements.txt
├── canonical_receipt.py    # CBOR + Ed25519 signer (shared with gpu-verifier)
├── chain.py                # Step → ReceiptCore adapter + chain serialization
├── tools.py                # sandboxed list_files / read_file / grep
├── agent.py                # Claude tool-use loop with receipt emission
├── verify_chain.py         # walks the chain, calls libocx-verify per receipt
├── run.py                  # entry point
├── target/                 # the audit-target sample (deliberately vulnerable)
│   ├── app.py
│   └── utils.py
└── output/                 # produced by run.py (gitignored except for one
                            # committed reference run that anyone can verify)
```

---

## Issuer key

For demo reproducibility `run.py` generates a fresh Ed25519 key each
run and writes the matching public key to `output/pubkey.txt`. In
production this would be a long-lived agent identity registered in
the OCX key registry — every receipt from that agent verifiable
against a single pinned key, indefinitely.
