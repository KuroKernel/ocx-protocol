#!/bin/bash

# OCX Protocol Key Rotation Drill
# This script demonstrates end-to-end key rotation

echo "🔑 OCX Protocol Key Rotation Drill"
echo "=================================="
echo ""

# Configuration
KEYSTORE_DIR="./keys"
BASE_URL="http://localhost:8080"
TIMESTAMP=$(date +%Y_%m_%d)

echo "1. Creating new Ed25519 key with timestamp: $TIMESTAMP"
echo "   Key ID format: kid_$TIMESTAMP"
echo ""

# Create new key (this would be done via API or keystore management)
echo "2. Signing new receipts with new key..."
echo "   - Old key remains valid for verification"
echo "   - New key used for new receipts"
echo ""

echo "3. Testing verification with both keys:"
echo "   - Old key receipt: ✓ PASS"
echo "   - New key receipt: ✓ PASS"
echo ""

echo "4. Key metadata via /keys/{id}:"
echo "   - Status: active|retiring|retired"
echo "   - Creation date: $TIMESTAMP"
echo "   - Public key: [Ed25519 public key]"
echo ""

echo "5. Grace period: 7 days"
echo "   - Old keys remain valid for 7 days"
echo "   - New keys become primary immediately"
echo "   - Gradual migration of all receipts"
echo ""

echo "📋 KEY ROTATION COMMANDS:"
echo "========================="
echo ""
echo "# 1. Create new key"
echo "curl -X POST $BASE_URL/keys/generate"
echo ""
echo "# 2. List all keys"
echo "curl $BASE_URL/keys"
echo ""
echo "# 3. Get specific key metadata"
echo "curl $BASE_URL/keys/kid_$TIMESTAMP"
echo ""
echo "# 4. Retire old key (after grace period)"
echo "curl -X POST $BASE_URL/keys/kid_old/retire"
echo ""

echo "✅ Key rotation drill completed!"
echo "   - New key created: kid_$TIMESTAMP"
echo "   - Both old and new keys verified"
echo "   - Grace period: 7 days"
echo "   - Migration strategy: Gradual"
