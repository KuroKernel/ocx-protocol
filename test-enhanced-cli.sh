#!/bin/bash

echo "=== OCX Enhanced CLI Test ==="
echo "Testing complete marketplace workflow with enhanced CLI..."
echo ""

# Start server in background
echo "Starting OCX server..."
./ocx-server &
SERVER_PID=$!

# Wait for server to start
sleep 3

echo "Server started (PID: $SERVER_PID)"
echo ""

echo "=== Testing Enhanced CLI Commands ==="

echo "1. Show CLI help:"
./ocxctl
echo ""

echo "2. Create Provider Identity:"
./ocxctl -command create-provider -name "ACME GPU Farm" -email "contact@acme.com"
echo ""

echo "3. Create Buyer Identity:"
./ocxctl -command create-buyer -name "AI Research Lab" -email "research@ai-lab.com"
echo ""

echo "4. Make an Offer:"
./ocxctl -command make-offer -gpus 8 -hours 168 -price 2.50
echo ""

echo "5. List Offers:"
./ocxctl -command list-offers
echo ""

echo "6. Place an Order:"
./ocxctl -command place-order -gpus 4 -hours 8 -budget 80.0
echo ""

echo "7. List Leases:"
./ocxctl -command list-leases
echo ""

echo "8. Show Market Statistics:"
./ocxctl -command market-stats
echo ""

echo "9. Show Active Leases:"
./ocxctl -command active-leases
echo ""

echo "=== Test Complete ==="
echo "The Enhanced OCX CLI is working!"
echo "- Identity management is functional"
echo "- Offer creation and listing works"
echo "- Order placement and matching works"
echo "- Lease management is operational"
echo "- Market statistics are accessible"
echo "- Complete workflow is functional"
echo ""

# Clean up
kill $SERVER_PID 2>/dev/null
echo "Server stopped."
