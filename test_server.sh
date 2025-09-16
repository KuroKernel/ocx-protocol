#!/bin/bash
set -e

echo "🚀 Starting OCX Core Server test..."

# Start server in background
./bin/ocx-core-server --port=8081 &
SERVER_PID=$!

# Wait for server to start
sleep 3

echo "📡 Testing server endpoints..."

# Test health endpoint
echo "Testing /health..."
curl -s http://localhost:8081/health | jq . || echo "Health check failed"

# Test API docs
echo "Testing /api..."
curl -s http://localhost:8081/api | jq . || echo "API docs failed"

# Test providers
echo "Testing /providers..."
curl -s http://localhost:8081/providers | jq . || echo "Providers failed"

# Test orders
echo "Testing /orders..."
curl -s http://localhost:8081/orders | jq . || echo "Orders failed"

# Test GPU info
echo "Testing /gpu/info..."
curl -s http://localhost:8081/gpu/info | jq . || echo "GPU info failed"

echo "✅ Server test complete!"

# Kill server
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo "🛑 Server stopped"
