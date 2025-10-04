#!/bin/bash

echo "🔬 DETERMINISM VERIFICATION PROTOCOL"
echo "===================================="
echo ""

# Check if OCX binary exists
if [ ! -f "./ocx" ]; then
    echo "❌ OCX binary not found. Building..."
    make build
    if [ ! -f "./ocx" ]; then
        echo "❌ Failed to build OCX binary"
        exit 1
    fi
fi

echo "✅ OCX binary found: $(./ocx --version 2>/dev/null || echo 'version unknown')"
echo ""

# Test 1: Same Input → Same Output (Basic Determinism)
echo "🧪 TEST 1: Basic Determinism (Same Input → Same Output)"
echo "-------------------------------------------------------"

# Create test script
cat > /tmp/determinism_test.sh << 'EOF'
#!/bin/sh
echo "Test output"
echo "Deterministic test"
echo "Hello from OCX"
exit 0
EOF
chmod +x /tmp/determinism_test.sh

# Run it 5 times, capture outputs (stdout only for deterministic comparison)
echo "Running test script 5 times..."
for i in {1..5}; do
    echo "  Run $i..."
    ./ocx execute /tmp/determinism_test.sh > output_$i.txt 2>stderr_$i.txt
    echo "    Hash: $(sha256sum output_$i.txt | cut -d' ' -f1)"
done

# Compare: If deterministic, all hashes should be IDENTICAL
unique_hashes=$(sha256sum output_*.txt | awk '{print $1}' | sort -u | wc -l)
echo ""
echo "Unique hashes: $unique_hashes"
if [ "$unique_hashes" -eq 1 ]; then
    echo "✅ TEST 1 PASSED: Determinism verified"
else
    echo "❌ TEST 1 FAILED: Determinism broken ($unique_hashes different outputs)"
fi
echo ""

# Test 2: Environment Isolation
echo "🧪 TEST 2: Environment Isolation"
echo "--------------------------------"

# Test that environment variables DON'T leak
export RANDOM_VAR="should_not_appear_$(date +%s)"

cat > /tmp/env_test.sh << 'EOF'
#!/bin/sh
env | sort
EOF
chmod +x /tmp/env_test.sh

echo "Running with RANDOM_VAR=$RANDOM_VAR"
./ocx execute /tmp/env_test.sh > env_output_1.txt 2>env_stderr_1.txt

sleep 2
export RANDOM_VAR="different_value_$(date +%s)"
echo "Running with RANDOM_VAR=$RANDOM_VAR"
./ocx execute /tmp/env_test.sh > env_output_2.txt 2>env_stderr_2.txt

# If deterministic, outputs should be IDENTICAL despite different env
if diff env_output_1.txt env_output_2.txt > /dev/null; then
    echo "✅ TEST 2 PASSED: Environment isolated"
else
    echo "❌ TEST 2 FAILED: Environment not isolated"
    echo "Differences:"
    diff env_output_1.txt env_output_2.txt
fi
echo ""

# Test 3: Filesystem Determinism
echo "🧪 TEST 3: Filesystem Isolation"
echo "-------------------------------"

cat > /tmp/fs_test.sh << 'EOF'
#!/bin/sh
ls -la /tmp | wc -l
EOF
chmod +x /tmp/fs_test.sh

echo "Running filesystem test..."
./ocx execute /tmp/fs_test.sh > fs_output_1.txt 2>fs_stderr_1.txt

# Pollute filesystem
echo "Polluting filesystem with random files..."
touch /tmp/random_file_$(date +%s)_{1..100} 2>/dev/null || true

./ocx execute /tmp/fs_test.sh > fs_output_2.txt 2>fs_stderr_2.txt

# Should be identical (filesystem isolated)
if diff fs_output_1.txt fs_output_2.txt > /dev/null; then
    echo "✅ TEST 3 PASSED: Filesystem isolated"
else
    echo "❌ TEST 3 FAILED: Filesystem not isolated"
    echo "Differences:"
    diff fs_output_1.txt fs_output_2.txt
fi
echo ""

# Test 4: Seccomp Actually Working
echo "🧪 TEST 4: Seccomp Security"
echo "---------------------------"

cat > /tmp/network_test.sh << 'EOF'
#!/bin/sh
# Try to create a socket (should be blocked by seccomp)
nc -l 8080 &
sleep 1
kill $! 2>/dev/null || true
EOF
chmod +x /tmp/network_test.sh

echo "Testing seccomp with network syscall..."
./ocx execute --seccomp /tmp/network_test.sh 2>&1 | tee seccomp_test.txt

# Should contain "bad system call" or "operation not permitted"
if grep -q "bad system call\|operation not permitted\|SIGSYS" seccomp_test.txt; then
    echo "✅ TEST 4 PASSED: Seccomp is blocking syscalls"
else
    echo "❌ TEST 4 FAILED: Seccomp not working - security compromised!"
    echo "Output:"
    cat seccomp_test.txt
fi
echo ""

# Test 5: Receipt Verification (if available)
echo "🧪 TEST 5: Receipt Verification"
echo "-------------------------------"

if ./ocx --help 2>&1 | grep -q "generate-receipt"; then
    echo "Testing receipt generation..."
    ./ocx execute /tmp/determinism_test.sh --generate-receipt > receipt_1.json 2>&1
    ./ocx execute /tmp/determinism_test.sh --generate-receipt > receipt_2.json 2>&1
    
    if [ -f receipt_1.json ] && [ -f receipt_2.json ]; then
        # Extract execution hashes if possible
        hash1=$(jq -r '.output_hash' receipt_1.json 2>/dev/null || echo "unknown")
        hash2=$(jq -r '.output_hash' receipt_2.json 2>/dev/null || echo "unknown")
        
        if [ "$hash1" = "$hash2" ] && [ "$hash1" != "unknown" ]; then
            echo "✅ TEST 5 PASSED: Receipt hashes match: $hash1"
        else
            echo "❌ TEST 5 FAILED: Receipt hashes differ or unavailable"
            echo "Hash 1: $hash1"
            echo "Hash 2: $hash2"
        fi
    else
        echo "⚠️  TEST 5 SKIPPED: Receipt generation not available"
    fi
else
    echo "⚠️  TEST 5 SKIPPED: Receipt generation not available in CLI"
fi
echo ""

# Test 6: Cross-Architecture (if Docker available)
echo "🧪 TEST 6: Cross-Architecture Determinism"
echo "----------------------------------------"

if command -v docker >/dev/null 2>&1; then
    echo "Testing cross-architecture determinism with Docker..."
    
    # Test on current architecture
    ./ocx execute /tmp/determinism_test.sh > output_native.txt 2>&1
    native_hash=$(sha256sum output_native.txt | cut -d' ' -f1)
    echo "Native architecture hash: $native_hash"
    
    # Try to test on different architecture if possible
    if docker run --platform linux/amd64 --rm -v $(pwd):/work ubuntu:22.04 bash -c "cd /work && ./ocx execute /tmp/determinism_test.sh" > output_docker.txt 2>&1; then
        docker_hash=$(sha256sum output_docker.txt | cut -d' ' -f1)
        echo "Docker architecture hash: $docker_hash"
        
        if [ "$native_hash" = "$docker_hash" ]; then
            echo "✅ TEST 6 PASSED: Cross-architecture determinism verified"
        else
            echo "❌ TEST 6 FAILED: Cross-architecture determinism broken"
        fi
    else
        echo "⚠️  TEST 6 SKIPPED: Docker cross-architecture test failed"
    fi
else
    echo "⚠️  TEST 6 SKIPPED: Docker not available"
fi
echo ""

# Summary
echo "📊 VERIFICATION SUMMARY"
echo "======================="
echo "Test 1 (Basic Determinism): $([ "$unique_hashes" -eq 1 ] && echo "✅ PASS" || echo "❌ FAIL")"
echo "Test 2 (Environment Isolation): $(diff env_output_1.txt env_output_2.txt >/dev/null && echo "✅ PASS" || echo "❌ FAIL")"
echo "Test 3 (Filesystem Isolation): $(diff fs_output_1.txt fs_output_2.txt >/dev/null && echo "✅ PASS" || echo "❌ FAIL")"
echo "Test 4 (Seccomp Security): $(grep -q "bad system call\|operation not permitted\|SIGSYS" seccomp_test.txt && echo "✅ PASS" || echo "❌ FAIL")"
echo ""

# Cleanup
echo "🧹 Cleaning up test files..."
rm -f output_*.txt env_output_*.txt fs_output_*.txt seccomp_test.txt receipt_*.json /tmp/determinism_test.sh /tmp/env_test.sh /tmp/fs_test.sh /tmp/network_test.sh

echo "Verification complete!"
