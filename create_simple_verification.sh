#!/bin/bash
set -e

echo "🏢 OCX SIMPLE VERIFICATION PACKAGE"
echo "=================================="

# Create verification directory
VERIFICATION_DIR="ocx-simple-verification-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$VERIFICATION_DIR"
cd "$VERIFICATION_DIR"

echo "📁 Creating verification package in: $VERIFICATION_DIR"

# Create deterministic test
cat > test.sh << 'EOF2'
#!/bin/sh
echo "OCX Deterministic Execution Test"
echo "Result: SUCCESS"
exit 0
EOF2
chmod +x test.sh

# Copy working OCX binary
cp ../ocx ./ocx

# Create verification script
cat > verify.sh << 'EOF2'
#!/bin/bash
set -e

echo "🔬 OCX VERIFICATION"
echo "==================="

# Test determinism
echo "Testing determinism..."
for i in {1..5}; do
    ./ocx execute test.sh 2>/dev/null > "output_$i.txt"
    hash=$(sha256sum "output_$i.txt" | awk '{print $1}')
    echo "  Run $i: $hash"
done

unique_hashes=$(sha256sum output_*.txt | awk '{print $1}' | sort -u | wc -l)
if [ "$unique_hashes" -eq 1 ]; then
    echo "✅ DETERMINISTIC: All outputs identical"
else
    echo "❌ NON-DETERMINISTIC: Different outputs"
    exit 1
fi

# Test receipts
echo "Testing receipts..."
./ocx execute test.sh -o receipt.cbor 2>/dev/null > /dev/null
if [ -f "receipt.cbor.json" ]; then
    hash=$(jq -r '.OutputHash' receipt.cbor.json)
    echo "✅ RECEIPT: Hash = $hash"
else
    echo "❌ RECEIPT: Failed to generate"
    exit 1
fi

echo "🎉 VERIFICATION COMPLETE: OCX Protocol working!"
EOF2

chmod +x verify.sh

echo "✅ Package created: $VERIFICATION_DIR"
echo "📋 To test: cd $VERIFICATION_DIR && ./verify.sh"
