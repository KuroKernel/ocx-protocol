#!/usr/bin/env bash
# OCX cross-arch bootstrap for Raspberry Pi (aarch64).
# Idempotent — safe to re-run. Prints the canonical output hash at the end.
#
# Prerequisites:
#   - Pi has internet (ethernet or tethered)
#   - ~/ocx-arm/cross_arch_test.py exists (scp'd from workstation)
#   - ~/ocx-arm/models/qwen2.5-0.5b-instruct-q4_k_m.gguf exists (scp'd)
#
# Usage:
#   chmod +x pi_bootstrap.sh
#   ./pi_bootstrap.sh

set -euo pipefail

WORK_DIR="$HOME/ocx-arm"
MODEL_PATH="$WORK_DIR/models/qwen2.5-0.5b-instruct-q4_k_m.gguf"
EXPECTED_MODEL_SHA="74a4da8c9fdbcd15bd1f6d01d621410d31c6fc00986f5eb687824e7b93d7a9db"
X86_CANONICAL_HASH="f6346a93756300aa4e9d6ce218c684d4a5c0293b425fbee0cf03405a627b87be"

echo "=========================================="
echo "OCX Cross-Arch Bootstrap (Raspberry Pi)"
echo "=========================================="
echo "Arch:     $(uname -m)"
echo "Kernel:   $(uname -r)"
echo "CPU:      $(lscpu | grep 'Model name' | head -1 | sed 's/.*:\s*//')"
echo "RAM:      $(free -h | awk '/^Mem:/{print $2}')"
echo ""

# ---- 1. Preflight ----
cd "$WORK_DIR" || { echo "ERROR: $WORK_DIR does not exist. scp the files first."; exit 1; }

if [[ ! -f "$MODEL_PATH" ]]; then
  echo "ERROR: model file missing at $MODEL_PATH"
  echo "scp it from workstation:"
  echo "  scp workstation:/home/kurokernel/Desktop/AXIS/ocx-protocol/examples/ai-verifier/models/qwen2.5-0.5b-instruct-q4_k_m.gguf $MODEL_PATH"
  exit 1
fi

if [[ ! -f cross_arch_test.py ]]; then
  echo "ERROR: cross_arch_test.py missing in $WORK_DIR"
  exit 1
fi

# ---- 2. Verify model hash matches x86 ----
echo "[1/5] Verifying model hash matches x86..."
ACTUAL_SHA=$(sha256sum "$MODEL_PATH" | awk '{print $1}')
if [[ "$ACTUAL_SHA" != "$EXPECTED_MODEL_SHA" ]]; then
  echo "ERROR: model file hash mismatch!"
  echo "  expected: $EXPECTED_MODEL_SHA"
  echo "  got:      $ACTUAL_SHA"
  echo "File corrupted in transit. Re-scp from workstation."
  exit 1
fi
echo "      PASS: $ACTUAL_SHA"
echo ""

# ---- 3. Install system deps ----
echo "[2/5] Installing system build deps (if missing)..."
if ! dpkg -s build-essential cmake python3-venv python3-pip >/dev/null 2>&1; then
  sudo apt update
  sudo apt install -y build-essential cmake python3-venv python3-pip
fi
echo "      OK"
echo ""

# ---- 4. Set up venv + install llama-cpp-python ----
echo "[3/5] Setting up Python venv + llama-cpp-python..."
if [[ ! -d venv ]]; then
  python3 -m venv venv
fi
# shellcheck source=/dev/null
source venv/bin/activate
pip install --upgrade pip --quiet
if ! python -c "import llama_cpp" 2>/dev/null; then
  echo "      Compiling llama-cpp-python (this takes 10-30 min on Pi, be patient)..."
  pip install llama-cpp-python
fi
echo "      llama-cpp-python installed"
echo ""

# ---- 5. Run the determinism test ----
echo "[4/5] Running cross-arch determinism test..."
python3 cross_arch_test.py
echo ""

# ---- 6. Compare with x86 baseline ----
echo "[5/5] Comparing with x86 baseline..."
if [[ -f cross_arch_aarch64.json ]]; then
  PI_HASH=$(python3 -c "import json; print(json.load(open('cross_arch_aarch64.json'))['canonical_output_hash'])")
  echo ""
  echo "=========================================="
  echo "CROSS-ARCH DETERMINISM RESULT"
  echo "=========================================="
  echo "x86_64 canonical_output_hash:  $X86_CANONICAL_HASH"
  echo "aarch64 canonical_output_hash: $PI_HASH"
  echo ""
  if [[ "$PI_HASH" == "$X86_CANONICAL_HASH" ]]; then
    echo "*** MATCH — CROSS-ARCHITECTURE DETERMINISM CONFIRMED ***"
    echo ""
    echo "Same model + same prompt + same flags → identical bytes on x86 and ARM."
    echo "This is the strongest claim of the session."
  else
    echo "*** MISMATCH — ARCHITECTURE-DEPENDENT DIVERGENCE ***"
    echo ""
    echo "Hashes differ. This is also a valuable finding — it tells us llama.cpp's"
    echo "SIMD paths (AVX2 vs NEON) or reduction order produce different logits"
    echo "even with top_k=1 sampling. Next step: inspect response texts to see"
    echo "where generation diverges."
  fi
  echo "=========================================="
  echo ""
  echo "Send cross_arch_aarch64.json back to the workstation for analysis:"
  echo "  scp cross_arch_aarch64.json workstation:/home/kurokernel/Desktop/AXIS/ocx-protocol/examples/ai-verifier/"
fi
