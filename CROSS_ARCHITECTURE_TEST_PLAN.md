# OCX Protocol - Cross-Architecture Testing Plan

## Objective

Verify that OCX Protocol produces **identical results** on different CPU architectures:
- x86_64 (Intel/AMD) - Your current dev machine
- ARM64 (aarch64) - Raspberry Pi 4/5
- ARMv7 (32-bit ARM) - Raspberry Pi 3 (optional)

## Why This Matters

**Critical for OCX's value proposition:**
- Determinism claim: "Same input → Same output, **always**"
- If results differ between architectures → Trust broken
- Financial/legal applications require **mathematical certainty**

## Known Architecture Differences

### Potential Issues:

1. **Floating-Point Behavior**
   - IEEE 754 standard, but edge cases differ
   - NaN handling, denormals, rounding modes
   - **Our Code**: Uses f64 in WASM aggregator

2. **Endianness**
   - x86: Little-endian
   - ARM: Bi-endian (usually little)
   - **Our Code**: CBOR handles this, but need to verify

3. **Pointer Sizes**
   - x86_64: 64-bit pointers
   - ARM64: 64-bit pointers
   - ARMv7: 32-bit pointers
   - **Our Code**: Go handles this, but FFI with Rust needs checks

4. **SIMD Instructions**
   - x86: SSE, AVX
   - ARM: NEON
   - **Our Code**: Crypto libraries might use SIMD

5. **Memory Alignment**
   - x86: Relaxed (can access unaligned)
   - ARM: Strict (unaligned access = crash)
   - **Our Code**: Go and Rust handle this, but need to verify

## Test Strategy

### Phase 1: Build Verification

**Goal**: Ensure OCX compiles on ARM64

```bash
# On Raspberry Pi
cd ocx-protocol

# Install Go 1.21+ (ARM64)
wget https://go.dev/dl/go1.21.5.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-arm64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source ~/.cargo/env

# Build OCX
make build-go
make build-rust

# Check binaries
file bin/ocx-server
# Should say: ARM aarch64, dynamically linked
```

**Expected Issues**:
- ⚠️ Some dependencies might not have ARM builds
- ⚠️ Rust compilation slower on Pi (1 CPU vs 4-8)
- ⚠️ Memory constraints (need 2GB+ for Rust build)

### Phase 2: Receipt Determinism Test

**Goal**: Same receipt bytes on x86_64 and ARM64

**Test Case 1: Simple Execution**

```bash
# On x86_64 (your dev machine)
cat > /tmp/test_input.txt << 'EOF'
Hello, deterministic world!
EOF

./bin/ocx-server &
sleep 2

curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-key" \
  -d '{
    "program": "echo",
    "input": "SGVsbG8sIGRldGVybWluaXN0aWMgd29ybGQhCg=="
  }' | jq -r '.receipt_b64' > /tmp/receipt_x86.b64

# Copy to Pi
scp /tmp/receipt_x86.b64 pi@raspberrypi.local:/tmp/
```

```bash
# On ARM64 (Raspberry Pi)
./bin/ocx-server &
sleep 2

curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-key" \
  -d '{
    "program": "echo",
    "input": "SGVsbG8sIGRldGVybWluaXN0aWMgd29ybGQhCg=="
  }' | jq -r '.receipt_b64' > /tmp/receipt_arm.b64

# Compare
diff /tmp/receipt_x86.b64 /tmp/receipt_arm.b64
# Should be IDENTICAL (exit code 0)

# If different, debug:
base64 -d /tmp/receipt_x86.b64 | xxd > /tmp/receipt_x86.hex
base64 -d /tmp/receipt_arm.b64 | xxd > /tmp/receipt_arm.hex
diff /tmp/receipt_x86.hex /tmp/receipt_arm.hex
```

**Test Case 2: Reputation Computation**

```bash
# On both architectures
curl -X POST http://localhost:8080/api/v1/reputation/compute \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "alice",
    "platforms": {
      "github": 85.5,
      "linkedin": 72.3,
      "uber": 90.1
    }
  }' | jq '{trust_score, confidence, receipt_b64}' > /tmp/rep_result.json

# Compare trust_score (should be identical to 10+ decimal places)
jq '.trust_score' /tmp/rep_result.json
```

**Test Case 3: Cryptographic Operations**

```bash
# Generate key on both architectures
openssl genpkey -algorithm ed25519 -out /tmp/test_key.pem

# Sign same data
echo "test data" | openssl pkeyutl -sign \
  -inkey /tmp/test_key.pem -rawin -out /tmp/sig_x86.bin

# Compare signatures (SHOULD be identical with deterministic signing)
xxd /tmp/sig_x86.bin
xxd /tmp/sig_arm.bin
```

### Phase 3: Floating-Point Precision Test

**Goal**: Verify f64 arithmetic is identical

```go
// test_float_determinism.go
package main

import (
    "fmt"
    "math"
)

func main() {
    // Test cases from reputation system
    testCases := []struct{
        name string
        a, b float64
        op string
    }{
        {"github_weight", 85.5, 0.4, "multiply"},
        {"linkedin_weight", 72.3, 0.35, "multiply"},
        {"uber_weight", 90.1, 0.25, "multiply"},
        {"sum", 34.2, 25.305, "add"},
        {"division", 81.83, 1.0, "divide"},
        {"log_scale", 1000.0, 0, "log10"},
    }

    for _, tc := range testCases {
        var result float64
        switch tc.op {
        case "multiply":
            result = tc.a * tc.b
        case "add":
            result = tc.a + tc.b
        case "divide":
            result = tc.a / tc.b
        case "log10":
            result = math.Log10(tc.a)
        }

        fmt.Printf("%s: %.20f\n", tc.name, result)
    }
}
```

```bash
# Run on both architectures
go run test_float_determinism.go > /tmp/float_x86.txt  # x86_64
go run test_float_determinism.go > /tmp/float_arm.txt  # ARM64

# Compare (should be IDENTICAL to 20 decimal places)
diff /tmp/float_x86.txt /tmp/float_arm.txt
```

### Phase 4: WASM Execution Test

**Goal**: WebAssembly produces identical results

```bash
# Install wasmtime on both architectures
curl https://wasmtime.dev/install.sh -sSf | bash

# Run WASM module
wasmtime artifacts/reputation-aggregator.wasm \
  --invoke compute_reputation \
  -- 5 alice 7 > /tmp/wasm_result.txt

# Compare outputs
diff /tmp/wasm_result_x86.txt /tmp/wasm_result_arm.txt
```

### Phase 5: Stress Test (1000 Iterations)

**Goal**: Verify consistency over many runs

```bash
#!/bin/bash
# run_determinism_stress_test.sh

for i in {1..1000}; do
    curl -s -X POST http://localhost:8080/api/v1/reputation/compute \
      -H "Content-Type: application/json" \
      -d '{
        "user_id": "user'$i'",
        "platforms": {
          "github": '$((50 + RANDOM % 50))',
          "linkedin": '$((50 + RANDOM % 50))',
          "uber": '$((50 + RANDOM % 50))'
        }
      }' | jq -r '.receipt_b64' >> /tmp/receipts_$ARCH.txt

    if [ $((i % 100)) -eq 0 ]; then
        echo "Completed $i iterations on $ARCH"
    fi
done

# Hash all receipts
cat /tmp/receipts_$ARCH.txt | sha256sum
```

Run on both architectures, compare final SHA256 hash.

## Expected Results

### ✅ Should Be Identical:

1. **Receipt Bytes** - Exact binary match
2. **Trust Scores** - Identical to 15+ decimal places
3. **Signatures** - Exact binary match (with same key)
4. **Output Hashes** - Exact match
5. **Gas Usage** - Identical values

### ⚠️ May Differ (Acceptable):

1. **Timestamps** - Will differ (execution time)
2. **Receipt IDs** - Include timestamp, so will differ
3. **Performance** - ARM64 may be slower
4. **Memory Usage** - May vary slightly

### ❌ Must NOT Differ:

1. **Computation Results** - Any difference = FAILURE
2. **CBOR Encoding** - Must be canonical
3. **Hash Values** - Input/output/program hashes must match

## Debugging Guide

### If Results Differ:

**Step 1: Isolate the Issue**
```bash
# Test each component separately
./verify-standalone /tmp/receipt.cbor /tmp/pubkey.b64  # Both architectures
go run test_float_determinism.go                       # Floating-point
wasmtime reputation-aggregator.wasm                    # WASM
```

**Step 2: Check Floating-Point**
```go
// Print raw bits
import "math"

x := 82.03
bits := math.Float64bits(x)
fmt.Printf("Float: %.20f\nBits: %064b\nHex: %016x\n", x, bits, bits)
```

**Step 3: Check CBOR Encoding**
```bash
# Decode and pretty-print
base64 -d receipt.cbor | cbor2diag.rb
```

**Step 4: Check Crypto Library**
```bash
# Verify Ed25519 implementation
go version         # Same Go version?
rustc --version    # Same Rust version?
ldd bin/ocx-server # Check linked libraries
```

## Mitigation Strategies

### If Determinism Fails:

**Option 1: Fix the Code**
- Use exact same floating-point operations
- Ensure canonical CBOR encoding
- Test edge cases (NaN, Infinity, -0.0)

**Option 2: Document Limitations**
```
⚠️ Cross-Architecture Notice:
OCX Protocol guarantees determinism within same architecture.
Cross-architecture determinism is best-effort.
For financial/legal use cases, verify on production architecture.
```

**Option 3: Reference Architecture**
```
✅ Certified Architectures:
- x86_64 (Linux, glibc 2.31+)
- ARM64 (Linux, glibc 2.31+)

⚠️ Other architectures: Test before production use
```

## Success Criteria

**Minimum Viable**:
- ✅ Builds on ARM64
- ✅ Basic execution works
- ✅ Receipts validate

**Strong**:
- ✅ Identical receipts (same input)
- ✅ Identical trust scores (< 0.0001% error)
- ✅ 1000+ iteration stress test passes

**Perfect**:
- ✅ Binary-identical receipts
- ✅ Exact floating-point match (15+ decimals)
- ✅ All tests pass 100%

## Timeline

1. **Raspberry Pi Setup**: 30 minutes
2. **Build OCX on ARM64**: 1 hour (slow compilation)
3. **Run Test Suite**: 1 hour
4. **Debug Issues**: 2-4 hours (if any)
5. **Documentation**: 1 hour

**Total**: 5-7 hours

## Next Steps

Once Raspberry Pi is working:
1. Run Phase 1 (build verification)
2. Run Phase 2 (receipt determinism)
3. Document results
4. Update PROJECT_DEEP_ANALYSIS.md with findings

---

**Note**: This is a CRITICAL gap in the current validation. Cross-architecture determinism is essential for the "provable computation" claim to hold universally.
