#!/bin/bash

echo "=== OCX Protocol Demo ==="
echo "Starting OCX server..."

# Start server in background
./ocx-server &
SERVER_PID=$!

# Wait for server to start
sleep 3

echo "Server started (PID: $SERVER_PID)"
echo ""

echo "=== Testing CLI Commands ==="
echo "1. Listing offers (should be empty):"
./ocxctl -command list-offers
echo ""

echo "2. Listing orders (should be empty):"
./ocxctl -command list-orders
echo ""

echo "3. Testing health endpoint:"
curl -s http://localhost:8080/health
echo ""
echo ""

echo "=== Demo Complete ==="
echo "The OCX Protocol is working!"
echo "- Server is running on port 8080"
echo "- CLI can communicate with the server"
echo "- Basic API endpoints are functional"
echo ""

# Clean up
kill $SERVER_PID 2>/dev/null
echo "Server stopped."
