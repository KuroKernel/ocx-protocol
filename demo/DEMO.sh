#!/usr/bin/env bash
set -euo pipefail

PORT="${OCX_PORT:-8081}"
API="${OCX_API_KEY:-prod-ocx-key}"

pkill -f "/server$" 2>/dev/null || true
env -i PATH="$PATH" OCX_API_KEY="$API" OCX_DISABLE_DB=true OCX_PORT="$PORT" ./server --log-level info > /tmp/ocx_demo.log 2>&1 &
SP=$!
for i in {1..20}; do curl -fsS "http://127.0.0.1:$PORT/readyz" && break; sleep 0.3; done

test -f artifacts/hello.sh || printf '#!/usr/bin/env bash\necho "Deterministic output"\n' > artifacts/hello.sh
HASH=$(sha256sum artifacts/hello.sh | awk '{print $1}')
INHEX=$(printf "demo" | xxd -p -c 256)

REQ_A=$(jq -n --arg h "$HASH" --arg in "$INHEX" '{artifact_hash:$h, input:$in}')
A=$(curl -fsS -H "X-API-Key: $API" -H "Content-Type: application/json" -d "$REQ_A" "http://127.0.0.1:$PORT/api/v1/execute")
B=$(curl -fsS -H "X-API-Key: $API" -H "Content-Type: application/json" -d "$REQ_A" "http://127.0.0.1:$PORT/api/v1/execute")

echo "$A" > /tmp/a.json; echo "$B" > /tmp/b.json
jq -r '.stdout' /tmp/a.json | sha256sum
jq -r '.stdout' /tmp/b.json | sha256sum

jq -r '.receipt_b64' /tmp/a.json | base64 -d > /tmp/rcpt.cbor
PUBHEX=$(jq -r '.verification.public_key // empty' /tmp/a.json)
if [ -n "$PUBHEX" ]; then printf "%s" "$PUBHEX" | xxd -r -p | base64 > /tmp/pub.b64; else cp ops/keys/ocx_public.b64 /tmp/pub.b64; fi
./verify-standalone /tmp/rcpt.cbor /tmp/pub.b64

cp /tmp/rcpt.cbor /tmp/bad.cbor
printf '\x00' | dd of=/tmp/bad.cbor bs=1 seek=10 count=1 conv=notrunc 2>/dev/null
./verify-standalone /tmp/bad.cbor /tmp/pub.b64 && { echo "Expected failure"; kill $SP; exit 3; }

kill $SP
echo "DEMO ✅"
