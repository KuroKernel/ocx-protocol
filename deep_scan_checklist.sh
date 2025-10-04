#!/bin/bash

echo "🔬 DEEP SURGICAL SCAN PROTOCOL"
echo "=============================="

for file in $(find pkg/ -name "*.go" -type f); do
    echo ""
    echo "📄 $file"
    
    # Check 1: Any TODOs/FIXMEs?
    if grep -q "TODO\|FIXME\|placeholder" "$file" 2>/dev/null; then
        echo "  ⚠️  Contains placeholders:"
        grep -n "TODO\|FIXME\|placeholder" "$file" | head -3
    fi
    
    # Check 2: Any panics?
    if grep -q "panic(" "$file" 2>/dev/null; then
        echo "  ❌ Contains panic():"
        grep -n "panic(" "$file"
    fi
    
    # Check 3: Error handling?
    if ! grep -q "if err != nil" "$file" 2>/dev/null; then
        echo "  ⚠️  Missing error handling?"
    fi
    
    # Check 4: Has tests?
    testfile="${file%%.go}_test.go"
    if [ ! -f "$testfile" ]; then
        echo "  ⚠️  No test file found"
    fi
done
