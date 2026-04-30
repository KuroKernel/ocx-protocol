"""Sandboxed tool implementations for the agent loop.

Every tool call must be deterministic-friendly: bounded output size,
no live timestamps in results, no env-leaks, no network. The agent
sees results, the receipts capture them, and a third party can replay
the chain offline.

All paths are resolved against a fixed sandbox root passed at
construction time. Any path that escapes the root via `..` or symlink
traversal is rejected with `SandboxError`.
"""
from __future__ import annotations

import os
import re
from dataclasses import dataclass
from pathlib import Path

MAX_READ_BYTES = 50 * 1024     # 50 KB cap per read_file
MAX_GREP_MATCHES = 100
MAX_LIST_ENTRIES = 200


class SandboxError(Exception):
    """Raised when a tool call attempts to read or write outside the
    sandbox root, or violates a size/count cap."""


@dataclass
class Sandbox:
    """Resolves all paths against `root` and rejects escapes."""
    root: Path

    def __post_init__(self) -> None:
        self.root = self.root.resolve(strict=True)
        if not self.root.is_dir():
            raise SandboxError(f"root must be a directory: {self.root}")

    def resolve(self, rel_path: str) -> Path:
        """Resolve a relative path against the sandbox root, rejecting
        traversal."""
        if rel_path.startswith("/"):
            raise SandboxError(f"absolute paths not allowed: {rel_path}")
        candidate = (self.root / rel_path).resolve()
        try:
            candidate.relative_to(self.root)
        except ValueError as e:
            raise SandboxError(
                f"path {rel_path} escapes sandbox root {self.root}"
            ) from e
        return candidate


# ---------- tool implementations ----------

def read_file(sb: Sandbox, path: str) -> str:
    """Read up to MAX_READ_BYTES of a file, return as utf-8 (or replacement)."""
    p = sb.resolve(path)
    if not p.is_file():
        return f"ERROR: not a regular file: {path}"
    raw = p.read_bytes()[:MAX_READ_BYTES]
    truncated = "\n[... truncated]" if p.stat().st_size > MAX_READ_BYTES else ""
    return raw.decode("utf-8", errors="replace") + truncated


def list_files(sb: Sandbox, path: str = "") -> str:
    """List entries in a directory under the sandbox. Returns a
    newline-separated listing with `d` / `f` prefixes."""
    p = sb.resolve(path) if path else sb.root
    if not p.is_dir():
        return f"ERROR: not a directory: {path}"
    entries = []
    for child in sorted(p.iterdir()):
        kind = "d" if child.is_dir() else "f"
        size = child.stat().st_size if child.is_file() else 0
        rel = child.relative_to(sb.root)
        entries.append(f"{kind}  {size:>10}  {rel}")
        if len(entries) >= MAX_LIST_ENTRIES:
            entries.append("[... truncated]")
            break
    return "\n".join(entries) if entries else "(empty)"


def grep(sb: Sandbox, pattern: str, path: str = "") -> str:
    """Search files under `path` (file or directory) for `pattern`
    (Python regex). Returns up to MAX_GREP_MATCHES results in
    `relative/path:line:matched_line` format."""
    try:
        regex = re.compile(pattern)
    except re.error as e:
        return f"ERROR: invalid regex: {e}"
    p = sb.resolve(path) if path else sb.root
    targets: list[Path]
    if p.is_file():
        targets = [p]
    elif p.is_dir():
        targets = [c for c in p.rglob("*") if c.is_file()]
    else:
        return f"ERROR: not a file or directory: {path}"

    out: list[str] = []
    for fp in sorted(targets):
        try:
            text = fp.read_text(encoding="utf-8", errors="replace")
        except Exception:
            continue
        for lineno, line in enumerate(text.splitlines(), start=1):
            if regex.search(line):
                rel = fp.relative_to(sb.root)
                out.append(f"{rel}:{lineno}:{line.rstrip()}")
                if len(out) >= MAX_GREP_MATCHES:
                    out.append("[... truncated]")
                    return "\n".join(out)
    return "\n".join(out) if out else "(no matches)"


# ---------- tool registration for the Anthropic API ----------

# These are the JSON Schemas Claude sees. Keep deterministic-friendly:
# every parameter is a string, no enums on dynamic content.
TOOL_SCHEMAS: list[dict] = [
    {
        "name": "list_files",
        "description": "List files and subdirectories under a path within the audit target. Returns 'kind size relative/path' lines.",
        "input_schema": {
            "type": "object",
            "properties": {
                "path": {"type": "string", "description": "Relative path inside the target. Empty string for the root."},
            },
            "required": ["path"],
        },
    },
    {
        "name": "read_file",
        "description": f"Read up to {MAX_READ_BYTES} bytes of a file as UTF-8.",
        "input_schema": {
            "type": "object",
            "properties": {
                "path": {"type": "string", "description": "Relative path inside the target."},
            },
            "required": ["path"],
        },
    },
    {
        "name": "grep",
        "description": f"Search files for a Python regex. Returns up to {MAX_GREP_MATCHES} matches as 'relative/path:line:line_text'.",
        "input_schema": {
            "type": "object",
            "properties": {
                "pattern": {"type": "string", "description": "Python regex (re.search)."},
                "path": {"type": "string", "description": "File or directory under the target. Empty string for the root."},
            },
            "required": ["pattern", "path"],
        },
    },
]


def dispatch(sb: Sandbox, name: str, args: dict) -> str:
    """Run a tool by name. Returns the result string the agent sees."""
    if name == "read_file":
        return read_file(sb, args.get("path", ""))
    if name == "list_files":
        return list_files(sb, args.get("path", ""))
    if name == "grep":
        return grep(sb, args.get("pattern", ""), args.get("path", ""))
    return f"ERROR: unknown tool {name!r}"


# Used so the verifier can know which tool produced an output without
# re-running the call.
TOOL_NAMES = sorted(t["name"] for t in TOOL_SCHEMAS)
