#!/bin/bash
# performance_test.sh

echo "OCX Protocol Verifier Performance Test"
echo "======================================"

# Build both versions
echo "Building verifiers..."
make build-rust
make build-go

# Start Rust server
echo "Starting Rust verifier server..."
OCX_USE_RUST_VERIFIER=true ./bin/ocx-server &
RUST_PID=$!
sleep 2

# Test Rust performance
echo "Testing Rust verifier performance..."
RUST_TIME=$(curl -s -X POST http://localhost:8080/verify \
  -H "Content-Type: application/json" \
  -d '{"receipt_data":"dGVzdA==","public_key":"dGVzdGtleXRlc3RrZXl0ZXN0a2V5dGVzdGs="}' \
  | jq -r '.duration')

# Stop Rust server
kill $RUST_PID

# Start Go server
echo "Starting Go verifier server..."
OCX_USE_RUST_VERIFIER=false ./bin/ocx-server &
GO_PID=$!
sleep 2

# Test Go performance
echo "Testing Go verifier performance..."
GO_TIME=$(curl -s -X POST http://localhost:8080/verify \
  -H "Content-Type: application/json" \
  -d '{"receipt_data":"dGVzdA==","public_key":"dGVzdGtleXRlc3RrZXl0ZXN0a2V5dGVzdGs="}' \
  | jq -r '.duration')

# Stop Go server
kill $GO_PID

# Calculate improvement
IMPROVEMENT=$(echo "scale=2; $GO_TIME / $RUST_TIME" | bc)

echo "Performance Results:"
echo "==================="
echo "Rust verifier: ${RUST_TIME}ns"
echo "Go verifier:   ${GO_TIME}ns"
echo "Improvement:   ${IMPROVEMENT}x faster"

if (( $(echo "$IMPROVEMENT > 10" | bc -l) )); then
    echo "🚀 Rust verifier is ${IMPROVEMENT}x faster!"
else
    echo "⚠️  Performance improvement is ${IMPROVEMENT}x"
fi
