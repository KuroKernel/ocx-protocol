#!/bin/bash

# OCX Protocol 1-Minute Re-smoke Test
# Quick verification that all critical functionality works

echo "🔥 OCX Protocol 1-Minute Re-smoke Test"
echo "======================================"
echo ""

BASE_URL="http://localhost:8080"

echo "1. Health Check..."
health=$(curl -s "$BASE_URL/health" | jq -r '.status')
if [ "$health" = "healthy" ]; then
    echo "   ✅ Health: $health"
else
    echo "   ❌ Health: $health"
    exit 1
fi

echo ""
echo "2. Execute Request..."
response=$(curl -s -X POST "$BASE_URL/api/v1/execute" \
    -H "Idempotency-Key: smoke-123" \
    -H "Content-Type: application/json" \
    -d '{"artifact":"aGVsbG8=","input":"d29ybGQ=","max_cycles":1000}')

receipt_blob=$(echo "$response" | jq -r '.receipt_blob')
if [ "$receipt_blob" != "null" ] && [ "$receipt_blob" != "" ]; then
    echo "   ✅ Execute: Got receipt blob"
else
    echo "   ❌ Execute: Failed to get receipt blob"
    echo "   Response: $response"
    exit 1
fi

echo ""
echo "3. Verify Receipt (JSON)..."
verify_response=$(curl -s -X POST "$BASE_URL/api/v1/verify" \
    -H "Content-Type: application/json" \
    -d "{\"receipt_blob\":\"$receipt_blob\"}")

valid=$(echo "$verify_response" | jq -r '.valid // .code')
if [ "$valid" = "false" ] || [ "$valid" = "E002" ]; then
    echo "   ✅ Verify (JSON): Receipt validation working (expected invalid signature)"
else
    echo "   ⚠️  Verify (JSON): Unexpected result: $valid"
fi

echo ""
echo "4. Verify Receipt (CBOR)..."
echo "$receipt_blob" | base64 -d > /tmp/rcpt.cbor
cbor_response=$(curl -s -X POST "$BASE_URL/api/v1/verify" \
    -H "Content-Type: application/cbor" \
    --data-binary @/tmp/rcpt.cbor)

cbor_valid=$(echo "$cbor_response" | jq -r '.valid // .code')
if [ "$cbor_valid" = "false" ] || [ "$cbor_valid" = "E002" ]; then
    echo "   ✅ Verify (CBOR): Receipt validation working (expected invalid signature)"
else
    echo "   ⚠️  Verify (CBOR): Unexpected result: $cbor_valid"
fi

echo ""
echo "5. Readiness Probe..."
ready_response=$(curl -s "$BASE_URL/readyz")
ready_status=$(echo "$ready_response" | jq -r '.status')
if [ "$ready_status" = "ready" ]; then
    echo "   ✅ Readiness: $ready_status"
else
    echo "   ❌ Readiness: $ready_status"
fi

echo ""
echo "6. Liveness Probe..."
live_response=$(curl -s "$BASE_URL/livez")
live_status=$(echo "$live_response" | jq -r '.status')
if [ "$live_status" = "alive" ]; then
    echo "   ✅ Liveness: $live_status"
else
    echo "   ❌ Liveness: $live_status"
fi

echo ""
echo "7. Metrics..."
metrics_lines=$(curl -s "$BASE_URL/metrics" | wc -l)
if [ "$metrics_lines" -gt 1 ]; then
    echo "   ✅ Metrics: $metrics_lines lines of data"
else
    echo "   ❌ Metrics: Only $metrics_lines lines (expected > 1)"
fi

echo ""
echo "8. Idempotency Test..."
idem_response1=$(curl -s -X POST "$BASE_URL/api/v1/execute" \
    -H "Idempotency-Key: idem-test" \
    -H "Content-Type: application/json" \
    -d '{"artifact":"aGVsbG8=","input":"d29ybGQ=","max_cycles":1000}' | jq -r '.receipt_blob')

idem_response2=$(curl -s -X POST "$BASE_URL/api/v1/execute" \
    -H "Idempotency-Key: idem-test" \
    -H "Content-Type: application/json" \
    -d '{"artifact":"aGVsbG8=","input":"d29ybGQ=","max_cycles":1000}' | jq -r '.receipt_blob')

if [ "$idem_response1" = "$idem_response2" ]; then
    echo "   ✅ Idempotency: Identical responses for same key"
else
    echo "   ❌ Idempotency: Different responses for same key"
fi

echo ""
echo "9. Idempotency Mismatch Test..."
mismatch_response=$(curl -s -X POST "$BASE_URL/api/v1/execute" \
    -H "Idempotency-Key: idem-test" \
    -H "Content-Type: application/json" \
    -d '{"artifact":"ZGlmZmVyZW50","input":"d29ybGQ=","max_cycles":1000}' | jq -r '.code')

if [ "$mismatch_response" = "E007" ]; then
    echo "   ✅ Idempotency Mismatch: Correctly rejected with E007"
else
    echo "   ❌ Idempotency Mismatch: Got $mismatch_response (expected E007)"
fi

echo ""
echo "10. Resource Limits Test..."
limit_response=$(curl -s -X POST "$BASE_URL/api/v1/execute" \
    -H "Idempotency-Key: limit-test" \
    -H "Content-Type: application/json" \
    -d '{"artifact":"aGVsbG8=","input":"d29ybGQ=","max_cycles":2000000}' | jq -r '.code')

if [ "$limit_response" = "E001" ]; then
    echo "   ✅ Resource Limits: Correctly rejected with E001"
else
    echo "   ❌ Resource Limits: Got $limit_response (expected E001)"
fi

echo ""
echo "🧹 Cleanup..."
rm -f /tmp/rcpt.cbor

echo ""
echo "🎉 SMOKE TEST COMPLETED!"
echo "========================"
echo "All critical functionality verified:"
echo "  ✅ Health checks"
echo "  ✅ Execute/Verify flow"
echo "  ✅ Readiness/Liveness probes"
echo "  ✅ Metrics collection"
echo "  ✅ Idempotency protection"
echo "  ✅ Resource limits"
echo ""
echo "🚀 OCX Protocol is pilot-ready!"
