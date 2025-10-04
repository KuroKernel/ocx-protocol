#!/usr/bin/env bash
set -euo pipefail

PORT="${OCX_PORT:-9001}"
API="${OCX_API_KEY:-prod-ocx-key}"

wait_ready() {
  local url="http://127.0.0.1:$PORT/readyz"
  for i in {1..40}; do
    if curl -fsS "$url" >/dev/null; then return 0; fi
    sleep 0.25
  done
  echo "[ERROR] server never became ready at $url" >&2
  tail -n 50 /tmp/ocx_demo.log || true
  exit 1
}

cleanup() {
  test -f /tmp/ocx_pid && kill "$(cat /tmp/ocx_pid)" 2>/dev/null || true
}
trap cleanup EXIT

# Start fresh
pkill -f "/server$" 2>/dev/null || true
OCX_API_KEY="$API" OCX_DISABLE_DB=true OCX_PORT="$PORT" \
  ./server --log-level info > /tmp/ocx_demo.log 2>&1 & echo $! > /tmp/ocx_pid

wait_ready
echo "[OK] server ready on :$PORT"

# Deterministic artifact
mkdir -p artifacts
printf '#!/usr/bin/env bash\necho "Deterministic output"\n' > artifacts/hello.sh
chmod +x artifacts/hello.sh

HASH=$(sha256sum artifacts/hello.sh | awk '{print $1}')
INHEX=$(printf "demo" | xxd -p -c 256)
REQ=$(jq -n --arg h "$HASH" --arg in "$INHEX" '{artifact_hash:$h, input:$in}')

A=$(curl -fsS -H "X-API-Key: $API" -H "Content-Type: application/json" \
      -d "$REQ" "http://127.0.0.1:$PORT/api/v1/execute")
B=$(curl -fsS -H "X-API-Key: $API" -H "Content-Type: application/json" \
      -d "$REQ" "http://127.0.0.1:$PORT/api/v1/execute")
echo "$A" > /tmp/a.json; echo "$B" > /tmp/b.json

H1=$(jq -r '.stdout' /tmp/a.json | sha256sum | awk '{print $1}')
H2=$(jq -r '.stdout' /tmp/b.json | sha256sum | awk '{print $1}')
test "$H1" = "$H2" || { echo "[ERROR] non-deterministic stdout"; exit 2; }
echo "[OK] deterministic stdout: $H1"

jq -r '.receipt_b64' /tmp/a.json | base64 -d > /tmp/rcpt.cbor
PUBHEX=$(jq -r '.verification.public_key // empty' /tmp/a.json)
if [ -n "$PUBHEX" ]; then printf "%s" "$PUBHEX" | xxd -r -p | base64 > /tmp/pub.b64
else cp ops/keys/ocx_public.b64 /tmp/pub.b64
fi

echo "[OK] verifying..."
./verify-standalone /tmp/rcpt.cbor /tmp/pub.b64

cp /tmp/rcpt.cbor /tmp/bad.cbor
printf '\x00' | dd of=/tmp/bad.cbor bs=1 seek=10 count=1 conv=notrunc 2>/dev/null
if ./verify-standalone /tmp/bad.cbor /tmp/pub.b64; then
  echo "[ERROR] tamper should fail but succeeded"; exit 3
else
  echo "[OK] tamper detection works"
fi

echo "DEMO ✅"