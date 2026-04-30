"""Claude Code-style agent loop with OCX receipts on every step.

Boundary:
  • Every Claude API call → one receipt (model_call)
  • Every tool execution → one receipt (tool_call:<name>)
  • Every artifact written by the agent → one receipt (file_write:<path>)

The agent is given:
  • a system prompt (auditor persona, tool contract, stop conditions)
  • a user task ("audit this codebase for security bugs, write report.md")
  • the three sandboxed tools defined in tools.py

It runs until Claude returns a turn without `tool_use` blocks, or until
MAX_STEPS is exceeded. The final assistant message is treated as the
audit report and gets written to output/report.md (plus its own
file_write receipt).

This module returns the Chain object so the caller can persist it.
"""
from __future__ import annotations

import logging
from pathlib import Path
from typing import Any

from anthropic import Anthropic

from chain import Chain, Step, step_now
from tools import TOOL_SCHEMAS, Sandbox, dispatch

log = logging.getLogger("ocx.agent")

MODEL = "claude-opus-4-5"  # latest as of 2026-04-30
MAX_STEPS = 12
MAX_TOKENS_PER_TURN = 4096

SYSTEM = """\
You are a security auditor reviewing a Python codebase. You have three tools:
list_files, read_file, grep. You CANNOT modify files.

Workflow:
  1. List the top-level files first.
  2. Read each Python file you find.
  3. Identify likely security issues (SQL injection, command injection,
     hardcoded secrets, path traversal, open redirects, insecure
     deserialization, weak crypto, missing auth checks).
  4. When you have enough evidence, output your final report as plain
     markdown. Do NOT call any more tools after that turn — your final
     turn must be text only.

Report format (use this exact structure):
  # Security audit
  ## Summary
  one short paragraph.
  ## Findings
  N numbered findings, each with: severity (high|medium|low), file:line,
  the snippet, and a one-sentence fix.
  ## What I read
  list of files + lines of code reviewed.
"""

USER_TASK = """\
Audit the Python files under the target directory for likely security bugs.
Produce the report described in the system instructions. Be honest about
what you couldn't determine from static reading alone."""


def run_agent(
    *,
    api_key: str,
    sandbox: Sandbox,
    chain: Chain,
    output_dir: Path,
) -> str:
    """Run the agent loop. Returns the final report text."""
    client = Anthropic(api_key=api_key)
    messages: list[dict[str, Any]] = [{"role": "user", "content": USER_TASK}]

    final_text = ""
    for step_idx in range(MAX_STEPS):
        log.info("turn %d — calling model", step_idx)
        resp = client.messages.create(
            model=MODEL,
            max_tokens=MAX_TOKENS_PER_TURN,
            system=SYSTEM,
            tools=TOOL_SCHEMAS,
            messages=messages,
        )

        # Snapshot what the agent saw and produced for the receipt.
        model_inputs = {
            "model": MODEL,
            "system_sha256_prefix": _sha_prefix(SYSTEM),
            "turn": step_idx,
            "message_count": len(messages),
            # capture the last user/tool_result content so the receipt
            # carries an integrity tag for the inputs at this turn
            "last_message_role": messages[-1]["role"],
        }
        model_outputs = {
            "stop_reason": resp.stop_reason,
            "content": _serialize_content(resp.content),
            "usage": {
                "input_tokens": resp.usage.input_tokens,
                "output_tokens": resp.usage.output_tokens,
            },
        }
        chain.append(Step(
            kind=f"model_call:turn={step_idx}",
            inputs=model_inputs,
            outputs=model_outputs,
            cycles=resp.usage.input_tokens + resp.usage.output_tokens,
            started_at=int(_now()),
            finished_at=int(_now()),
        ))

        # Add the assistant turn to the conversation
        messages.append({"role": "assistant", "content": resp.content})

        # Are there any tool calls?
        tool_uses = [b for b in resp.content if b.type == "tool_use"]
        if not tool_uses:
            # Plain text turn — extract and we're done
            text_blocks = [b for b in resp.content if b.type == "text"]
            final_text = "\n".join(b.text for b in text_blocks).strip()
            log.info("agent finished after turn %d with stop_reason=%s",
                     step_idx, resp.stop_reason)
            break

        # Execute every tool call and append a tool_result for each
        tool_results: list[dict[str, Any]] = []
        for tu in tool_uses:
            log.info("  → tool_use %s args=%s", tu.name, tu.input)
            result = dispatch(sandbox, tu.name, dict(tu.input))
            chain.append(step_now(
                kind=f"tool_call:{tu.name}",
                inputs={"tool": tu.name, "args": dict(tu.input), "tool_use_id": tu.id},
                outputs={"result_preview": result[:400], "result_bytes": len(result)},
                cycles=max(1, len(result.encode("utf-8"))),
            ))
            tool_results.append({
                "type": "tool_result",
                "tool_use_id": tu.id,
                "content": result,
            })

        messages.append({"role": "user", "content": tool_results})
    else:
        log.warning("agent hit MAX_STEPS (%d) without producing a final text turn", MAX_STEPS)

    # Persist the report and emit a file_write receipt
    output_dir.mkdir(parents=True, exist_ok=True)
    report_path = output_dir / "report.md"
    report_bytes = (final_text or "(empty report — agent did not produce text)").encode("utf-8")
    report_path.write_bytes(report_bytes)
    chain.append(step_now(
        kind="file_write:report.md",
        inputs={"path": "report.md", "bytes": len(report_bytes)},
        outputs={"path": "report.md", "sha256_prefix": _sha_prefix(report_bytes)},
        cycles=max(1, len(report_bytes)),
    ))

    return final_text


# ---------- internals ----------

def _now() -> float:
    import time
    return time.time()


def _sha_prefix(s: str | bytes, n: int = 16) -> str:
    import hashlib
    if isinstance(s, str):
        s = s.encode("utf-8")
    return hashlib.sha256(s).hexdigest()[:n]


def _serialize_content(blocks: list[Any]) -> list[dict[str, Any]]:
    """Take Anthropic ContentBlock objects and produce JSON-friendly
    dicts — only the fields we need for the receipt fingerprint."""
    out: list[dict[str, Any]] = []
    for b in blocks:
        if b.type == "text":
            out.append({"type": "text", "text": b.text})
        elif b.type == "tool_use":
            out.append({
                "type": "tool_use",
                "id": b.id,
                "name": b.name,
                "input": dict(b.input),
            })
        else:
            out.append({"type": b.type})
    return out
