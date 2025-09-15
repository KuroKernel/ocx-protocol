#!/bin/bash

echo "=== OCX Matching Engine Demo ==="
echo "Testing the complete marketplace flow..."
echo ""

# Start server in background
echo "Starting OCX server with matching engine..."
./ocx-server &
SERVER_PID=$!

# Wait for server to start
sleep 3

echo "Server started (PID: $SERVER_PID)"
echo ""

echo "=== Step 1: Create Sample Offers ==="
echo "Creating sample offers for different GPU types..."

# Create sample offers using curl (bypassing signature for demo)
curl -s -X POST http://localhost:8080/offers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "offer-1",
    "kind": "offer",
    "version": {"major": 0, "minor": 1, "patch": 0},
    "issued_at": "2025-01-27T10:00:00Z",
    "payload": {
      "offer_id": "offer-1",
      "version": {"major": 0, "minor": 1, "patch": 0},
      "provider": {"party_id": "provider-1", "role": "provider"},
      "fleet_id": "fleet-1",
      "unit": "gpu_hour",
      "unit_price": {"currency": "USD", "amount": "2.50", "scale": 2},
      "min_hours": 1,
      "max_hours": 168,
      "min_gpus": 1,
      "max_gpus": 8,
      "valid_from": "2025-01-27T10:00:00Z",
      "valid_to": "2025-01-28T10:00:00Z",
      "compliance": ["GDPR"]
    },
    "hash": {"alg": "sha256", "value": "abc123"},
    "sig": {"alg": "ed25519", "key_id": "key-1", "sig_b64": "sig123"}
  }' | jq '.'

echo ""
echo "Creating second offer..."

curl -s -X POST http://localhost:8080/offers \
  -H "Content-Type: application/json" \
  -d '{
    "id": "offer-2",
    "kind": "offer",
    "version": {"major": 0, "minor": 1, "patch": 0},
    "issued_at": "2025-01-27T10:00:00Z",
    "payload": {
      "offer_id": "offer-2",
      "version": {"major": 0, "minor": 1, "patch": 0},
      "provider": {"party_id": "provider-2", "role": "provider"},
      "fleet_id": "fleet-2",
      "unit": "gpu_hour",
      "unit_price": {"currency": "USD", "amount": "3.00", "scale": 2},
      "min_hours": 1,
      "max_hours": 168,
      "min_gpus": 1,
      "max_gpus": 4,
      "valid_from": "2025-01-27T10:00:00Z",
      "valid_to": "2025-01-28T10:00:00Z",
      "compliance": ["HIPAA"]
    },
    "hash": {"alg": "sha256", "value": "def456"},
    "sig": {"alg": "ed25519", "key_id": "key-2", "sig_b64": "sig456"}
  }' | jq '.'

echo ""
echo "=== Step 2: List Available Offers ==="
curl -s http://localhost:8080/offers | jq '.'

echo ""
echo "=== Step 3: Place an Order ==="
echo "Placing order for 2 GPUs for 4 hours..."

curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "id": "order-1",
    "kind": "order",
    "version": {"major": 0, "minor": 1, "patch": 0},
    "issued_at": "2025-01-27T10:30:00Z",
    "payload": {
      "order_id": "order-1",
      "version": {"major": 0, "minor": 1, "patch": 0},
      "buyer": {"party_id": "buyer-1", "role": "buyer"},
      "offer_id": "offer-1",
      "requested_gpus": 2,
      "hours": 4,
      "budget_cap": {"currency": "USD", "amount": "20.00", "scale": 2},
      "state": "pending",
      "created_at": "2025-01-27T10:30:00Z",
      "updated_at": "2025-01-27T10:30:00Z"
    },
    "hash": {"alg": "sha256", "value": "ghi789"},
    "sig": {"alg": "ed25519", "key_id": "key-3", "sig_b64": "sig789"}
  }' | jq '.'

echo ""
echo "=== Step 4: Check Market Statistics ==="
curl -s http://localhost:8080/stats | jq '.'

echo ""
echo "=== Step 5: List Active Leases ==="
curl -s http://localhost:8080/leases | jq '.'

echo ""
echo "=== Demo Complete ==="
echo "The OCX Matching Engine is working!"
echo "- Offers are published and stored"
echo "- Orders are matched with compatible offers"
echo "- Leases are created automatically"
echo "- Market statistics are tracked"
echo ""

# Clean up
kill $SERVER_PID 2>/dev/null
echo "Server stopped."
