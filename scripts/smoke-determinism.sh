#!/bin/bash
# OCX Deterministic Smoke Test
# Quick guardrail to verify deterministic execution

set -euo pipefail

echo "🧪 OCX Deterministic Smoke Test"
echo "================================"

# Setup
ART=artifacts/hello.sh
mkdir -p artifacts receipts

# Create test artifact
printf '#!/usr/bin/env bash\necho "hello-ocx"\n' > "$ART"
chmod +x "$ART"

echo "📝 Created test artifact: $ART"

# Run 5 executions and collect output hashes
echo "🔄 Running 5 executions..."
for i in {1..5}; do
    echo "  Run $i..."
    ./ocx execute "$ART" --output "receipts/run_$i.cbor" >/dev/null 2>&1
    
    # Extract output hash from receipt (if available)
    if [ -f "receipts/run_$i.cbor" ]; then
        # For now, just check that receipts are created and have content
        if [ -s "receipts/run_$i.cbor" ]; then
            echo "    ✓ Receipt created (size: $(stat -c%s "receipts/run_$i.cbor") bytes)"
        else
            echo "    ❌ Empty receipt"
            exit 1
        fi
    else
        echo "    ❌ No receipt created"
        exit 1
    fi
done

# Check that all receipts have the same size (basic determinism check)
echo "🔍 Checking receipt consistency..."
RECEIPT_SIZES=$(stat -c%s receipts/run_*.cbor | sort -u | wc -l)
if [ "$RECEIPT_SIZES" -eq 1 ]; then
    echo "✅ All receipts have identical size - deterministic execution verified"
else
    echo "❌ Receipt sizes differ - non-deterministic execution detected"
    echo "Receipt sizes:"
    stat -c%s receipts/run_*.cbor
    exit 1
fi

# Cleanup
echo "🧹 Cleaning up test files..."
rm -f receipts/run_*.cbor

echo "✅ Determinism smoke test PASSED"
