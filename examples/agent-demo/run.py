"""Entry point: run the OCX-signed agent on the audit-target sample.

Usage:
    export ANTHROPIC_API_KEY=sk-ant-…
    python3 run.py            # writes everything under output/

Output:
    output/receipts.cbor      — the signed chain (verifier reads this)
    output/receipts.json      — human-readable transcript with hashes
    output/pubkey.txt         — base64 raw 32-byte Ed25519 public key
    output/report.md          — the agent's audit report
"""
from __future__ import annotations

import logging
import os
import sys
from pathlib import Path

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

from agent import run_agent
from chain import Chain
from tools import Sandbox

HERE = Path(__file__).parent.resolve()
TARGET_DIR = HERE / "target"
OUTPUT_DIR = HERE / "output"


def main() -> int:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )

    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if not api_key:
        print("ERROR: set ANTHROPIC_API_KEY in env first.", file=sys.stderr)
        return 1

    if not TARGET_DIR.is_dir():
        print(f"ERROR: target dir missing: {TARGET_DIR}", file=sys.stderr)
        return 1

    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

    # Issuer key. For demo reproducibility we generate a fresh key each
    # run. In production this would be a long-lived agent identity key
    # registered in the OCX key registry. The pubkey lands in
    # output/pubkey.txt so the verifier knows what to check against.
    signer = Ed25519PrivateKey.generate()
    chain = Chain(signer)
    sandbox = Sandbox(root=TARGET_DIR)

    print(f"  sandbox: {TARGET_DIR}")
    print(f"  output : {OUTPUT_DIR}")
    print(f"  agent  : starting…\n")

    final = run_agent(
        api_key=api_key,
        sandbox=sandbox,
        chain=chain,
        output_dir=OUTPUT_DIR,
    )

    chain.write(OUTPUT_DIR)

    print(
        f"\n  done. {len(chain.receipts)} signed receipts written.\n"
        f"  next: python3 verify_chain.py output/"
    )
    if not final.strip():
        print("  WARNING: empty final report (agent may have hit MAX_STEPS)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
