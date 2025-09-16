#!/bin/bash
set -e

echo "🚀 Starting OCX Database Server test..."

# Start server in background
./bin/ocx-db-server --port=8082 &
SERVER_PID=$!

# Wait for server to start
sleep 3

echo "📡 Testing database-connected server endpoints..."

# Test health endpoint
echo "Testing /health..."
curl -s http://localhost:8082/health | jq . || echo "Health check failed"

# Test API docs
echo "Testing /api..."
curl -s http://localhost:8082/api | jq . || echo "API docs failed"

# Test providers from database
echo "Testing /providers..."
curl -s http://localhost:8082/providers | jq . || echo "Providers failed"

# Test orders from database
echo "Testing /orders..."
curl -s http://localhost:8082/orders | jq . || echo "Orders failed"

# Test database stats
echo "Testing /stats..."
curl -s http://localhost:8082/stats | jq . || echo "Stats failed"

echo "✅ Database server test complete!"

# Kill server
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo "🛑 Server stopped"
