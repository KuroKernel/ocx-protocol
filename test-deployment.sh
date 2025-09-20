#!/bin/bash

echo "=== Testing OCX Deployment ==="

# Test API server
echo "Testing API server..."
if curl -f http://localhost:3001/health > /dev/null 2>&1; then
    echo "✅ API server health check passed"
    
    # Test execute endpoint
    echo "Testing execute endpoint..."
    RESULT=$(curl -s -X POST http://localhost:3001/api/v1/execute \
        -H "Content-Type: application/json" \
        -d '{"artifact":"dGVzdA==","input":"ZGF0YQ==","max_cycles":1000}')
    
    if echo "$RESULT" | grep -q "receipt_hash"; then
        echo "✅ Execute endpoint working"
    else
        echo "❌ Execute endpoint failed"
    fi
    
    # Test verify endpoint  
    echo "Testing verify endpoint..."
    VERIFY=$(curl -s -X POST http://localhost:3001/api/v1/verify \
        -H "Content-Type: application/json" \
        -d '{"receipt_blob":"dGVzdA=="}')
        
    if echo "$VERIFY" | grep -q "valid"; then
        echo "✅ Verify endpoint working"
    else
        echo "❌ Verify endpoint failed"
    fi
    
else
    echo "❌ API server not responding"
    echo "Make sure to start it with: ./cmd/api-server/api-server"
fi

# Test frontend
echo "Testing frontend..."
if curl -f http://localhost:3000 > /dev/null 2>&1; then
    echo "✅ Frontend serving"
else
    echo "❌ Frontend not responding"
    echo "Make sure to start it with: npm start or npm run serve"
fi

echo "=== Test complete ==="
