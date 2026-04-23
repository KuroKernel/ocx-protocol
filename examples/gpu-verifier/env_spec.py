"""Environment fingerprint for environment-pinned receipts (STRATEGY.md Path 4).

Collects a spec describing the execution environment so a verifier can decide
whether their environment matches — if not, bit-identical reproduction is out
of scope and the receipt's verification semantics become "trust-signer" rather
than "reproduce-locally".

The returned spec is a flat dict[str, str] suitable for the OCX receipt's
unsigned `host_info` field. NOT part of the signed payload, so a prover can't
lie about it without also controlling the signing key.
"""
from __future__ import annotations

import platform
import shutil
import subprocess
import sys
from typing import Optional


def _safe_run(cmd: list[str], timeout: float = 2.0) -> Optional[str]:
    try:
        out = subprocess.run(
            cmd, capture_output=True, text=True, timeout=timeout, check=False
        )
        return out.stdout.strip() or None
    except Exception:
        return None


def nvidia_smi_field(field: str) -> Optional[str]:
    """Query nvidia-smi for a single field (e.g. name, driver_version, uuid)."""
    if not shutil.which("nvidia-smi"):
        return None
    return _safe_run(
        ["nvidia-smi", f"--query-gpu={field}", "--format=csv,noheader,nounits"]
    )


def collect() -> dict[str, str]:
    """Collect the environment spec for the running process."""
    spec: dict[str, str] = {
        "arch": platform.machine(),
        "os": f"{platform.system()} {platform.release()}",
        "python": sys.version.split()[0],
    }

    # Torch + CUDA (if available)
    try:
        import torch  # type: ignore
        spec["torch"] = torch.__version__
        if torch.cuda.is_available():
            spec["cuda_runtime"] = torch.version.cuda or "unknown"
            spec["cuda_device_count"] = str(torch.cuda.device_count())
            spec["cuda_device_0"] = torch.cuda.get_device_name(0)
            # Compute capability (e.g. (12, 0) for Blackwell sm_120)
            cap = torch.cuda.get_device_capability(0)
            spec["cuda_compute_cap"] = f"sm_{cap[0]}{cap[1]}"
        else:
            spec["cuda_available"] = "false"
    except ImportError:
        spec["torch"] = "not_installed"

    # NVIDIA driver + GPU UUID (authoritative, not from torch)
    driver = nvidia_smi_field("driver_version")
    if driver:
        spec["nvidia_driver"] = driver
    uuid = nvidia_smi_field("uuid")
    if uuid:
        spec["gpu_uuid"] = uuid
    gpu_name = nvidia_smi_field("name")
    if gpu_name:
        spec["gpu_name"] = gpu_name

    # Model / inference deps (best effort)
    for mod in ("transformers", "cbor2", "cryptography", "numpy"):
        try:
            spec[f"py_{mod}"] = __import__(mod).__version__  # type: ignore
        except Exception:
            pass

    return spec


if __name__ == "__main__":
    import json
    print(json.dumps(collect(), indent=2, sort_keys=True))
