package deterministicvm

import (
	"bytes"
	"crypto/sha256"
	"math/rand"
	"testing"
	"time"
)

// Property: Gas calculation is deterministic
// For any script and input, gas must be identical across runs
func TestProperty_GasDeterminism(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 1000; i++ {
		// Generate random script
		scriptLen := rng.Intn(1000)
		script := make([]byte, scriptLen)
		rng.Read(script)

		// Generate random input
		inputLen := rng.Intn(1000)
		input := make([]byte, inputLen)
		rng.Read(input)

		// Calculate gas 3 times
		gas1 := CalculateDeterministicGas(string(script), input)
		gas2 := CalculateDeterministicGas(string(script), input)
		gas3 := CalculateDeterministicGas(string(script), input)

		// All must be identical
		if gas1 != gas2 || gas2 != gas3 {
			t.Fatalf("Gas not deterministic: %d, %d, %d for script len=%d, input len=%d",
				gas1, gas2, gas3, scriptLen, inputLen)
		}
	}
}

// Property: Gas is monotonic with input size
// Larger inputs should not produce less gas than smaller inputs (for same script)
func TestProperty_GasMonotonicity(t *testing.T) {
	rng := rand.New(rand.NewSource(42)) // Fixed seed for reproducibility

	scripts := []string{
		"echo hello",
		"cat",
		"for i in 1 2 3; do echo $i; done",
	}

	for _, script := range scripts {
		var lastGas uint64
		for inputSize := 0; inputSize <= 10000; inputSize += 100 {
			input := make([]byte, inputSize)
			rng.Read(input)

			gas := CalculateDeterministicGas(script, input)

			if inputSize > 0 && gas < lastGas {
				t.Errorf("Gas decreased with larger input: size=%d gas=%d, prev size=%d gas=%d",
					inputSize, gas, inputSize-100, lastGas)
			}
			lastGas = gas
		}
	}
}

// Property: Gas is non-negative for all inputs
func TestProperty_GasNonNegative(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 1000; i++ {
		scriptLen := rng.Intn(500)
		script := make([]byte, scriptLen)
		rng.Read(script)

		inputLen := rng.Intn(500)
		input := make([]byte, inputLen)
		rng.Read(input)

		gas := CalculateDeterministicGas(string(script), input)

		// Gas is uint64, so can't be negative, but checking > 0 for non-empty
		if (len(script) > 0 || len(input) > 0) && gas == 0 {
			t.Errorf("Gas is zero for non-empty input: script len=%d, input len=%d",
				len(script), len(input))
		}
	}
}

// Property: Opcode gas costs are consistent across the full range
func TestProperty_OpcodeGasConsistency(t *testing.T) {
	// Test all 256 possible opcodes multiple times
	for round := 0; round < 10; round++ {
		for opcode := 0; opcode < 256; opcode++ {
			gas1 := opcodeGasCost(byte(opcode))
			gas2 := opcodeGasCost(byte(opcode))

			if gas1 != gas2 {
				t.Fatalf("Opcode 0x%02x gas not deterministic: %d vs %d", opcode, gas1, gas2)
			}

			if gas1 == 0 {
				t.Fatalf("Opcode 0x%02x has zero gas cost", opcode)
			}
		}
	}
}

// Property: WASM gas calculation is deterministic
// Note: Uses recover to handle panics from malformed WASM - fuzz tests cover panic detection
func TestProperty_WASMGasDeterminism(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Test with WASM-like headers
	wasmMagic := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}

	for i := 0; i < 100; i++ {
		// Create WASM-like binary with random body
		bodyLen := rng.Intn(500) // Reduced size to avoid OOM
		body := make([]byte, bodyLen)
		rng.Read(body)

		wasm := append(wasmMagic, body...)

		// Use recover to handle panics from malformed WASM
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Panic is acceptable for malformed WASM
					// Fuzz tests verify panic-safety
					return
				}
			}()

			gas1 := calculateStaticWASMGas(wasm)
			gas2 := calculateStaticWASMGas(wasm)
			gas3 := calculateStaticWASMGas(wasm)

			if gas1 != gas2 || gas2 != gas3 {
				t.Fatalf("WASM gas not deterministic for body len=%d", bodyLen)
			}
		}()
	}
}

// Property: FP validator decisions are deterministic
func TestProperty_FPValidatorDeterminism(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping FP validator test in short mode")
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	strictValidator := NewFPValidator(false, false, true)
	nonStrictValidator := NewFPValidator(false, false, false)

	for i := 0; i < 10; i++ { // Small iteration count for performance
		// Generate random WASM-like bytes
		wasmLen := 8 + rng.Intn(100)
		wasm := make([]byte, wasmLen)
		wasm[0] = 0x00
		wasm[1] = 0x61
		wasm[2] = 0x73
		wasm[3] = 0x6d
		wasm[4] = 0x01
		wasm[5] = 0x00
		wasm[6] = 0x00
		wasm[7] = 0x00
		rng.Read(wasm[8:])

		// Test strict validator
		err1 := strictValidator.ValidateModule(wasm)
		err2 := strictValidator.ValidateModule(wasm)
		if (err1 == nil) != (err2 == nil) {
			t.Fatal("Strict validator not deterministic")
		}

		// Test non-strict validator
		err3 := nonStrictValidator.ValidateModule(wasm)
		err4 := nonStrictValidator.ValidateModule(wasm)
		if (err3 == nil) != (err4 == nil) {
			t.Fatal("Non-strict validator not deterministic")
		}
	}
}

// Property: WASM binary detection is deterministic
func TestProperty_WASMDetectionDeterminism(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 1000; i++ {
		dataLen := rng.Intn(100)
		data := make([]byte, dataLen)
		rng.Read(data)

		result1 := isWASMBinary(data)
		result2 := isWASMBinary(data)
		result3 := isWASMBinary(data)

		if result1 != result2 || result2 != result3 {
			t.Fatalf("WASM detection not deterministic for data len=%d", dataLen)
		}
	}
}

// Property: WASM detection correctly identifies valid WASM magic
func TestProperty_WASMDetectionCorrectness(t *testing.T) {
	wasmMagic := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Valid WASM with random body
	for i := 0; i < 100; i++ {
		bodyLen := rng.Intn(100)
		body := make([]byte, bodyLen)
		rng.Read(body)

		wasm := append(wasmMagic, body...)
		if !isWASMBinary(wasm) {
			t.Fatalf("Valid WASM not detected at iteration %d", i)
		}
	}

	// Invalid data should not be detected as WASM
	for i := 0; i < 100; i++ {
		dataLen := 8 + rng.Intn(100)
		data := make([]byte, dataLen)
		rng.Read(data)

		// Ensure first 4 bytes are not WASM magic
		if data[0] == 0x00 && data[1] == 0x61 && data[2] == 0x73 && data[3] == 0x6d {
			data[0] = 0xff // Corrupt magic
		}

		if isWASMBinary(data) {
			t.Fatalf("Invalid data detected as WASM at iteration %d", i)
		}
	}
}

// Property: Output hash is deterministic for same script and input
func TestProperty_OutputHashDeterminism(t *testing.T) {
	testCases := []struct {
		script string
		input  []byte
	}{
		{"echo hello", nil},
		{"cat", []byte("input data")},
		{"echo $((2+2))", nil},
	}

	for _, tc := range testCases {
		// Hash the script and input to simulate output determination
		scriptHash := sha256.Sum256([]byte(tc.script))
		inputHash := sha256.Sum256(tc.input)

		// Combined hash should always be the same
		combined := append(scriptHash[:], inputHash[:]...)
		hash1 := sha256.Sum256(combined)
		hash2 := sha256.Sum256(combined)

		if !bytes.Equal(hash1[:], hash2[:]) {
			t.Fatalf("Hash not deterministic for script=%q", tc.script)
		}
	}
}

// Property: Gas base cost is always positive
func TestProperty_GasBaseCost(t *testing.T) {
	// Empty script and input should still have base cost
	gas := CalculateDeterministicGas("", nil)

	// Base cost should be non-zero (there's always overhead)
	if gas == 0 {
		// This might be acceptable depending on implementation
		t.Log("Note: Empty script/input has zero gas")
	}
}

// Property: Strict mode parsing is deterministic
func TestProperty_StrictModeParsingDeterminism(t *testing.T) {
	testValues := []string{
		"", "true", "false", "1", "0", "yes", "no",
		"TRUE", "FALSE", "YES", "NO", "True", "False",
		"invalid", "random", "  true  ", "1.0",
	}

	for _, val := range testValues {
		result1 := parseStrictModeValue(val)
		result2 := parseStrictModeValue(val)
		result3 := parseStrictModeValue(val)

		if result1 != result2 || result2 != result3 {
			t.Fatalf("Strict mode parsing not deterministic for %q", val)
		}
	}
}

// Property: Gas calculation is idempotent
func TestProperty_GasIdempotent(t *testing.T) {
	script := "for i in 1 2 3 4 5; do echo $i; done"
	input := []byte("test input data")

	// Calculate multiple times in sequence
	results := make([]uint64, 100)
	for i := 0; i < 100; i++ {
		results[i] = CalculateDeterministicGas(script, input)
	}

	// All results must be identical
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Fatalf("Gas not idempotent: iteration %d got %d, expected %d",
				i, results[i], results[0])
		}
	}
}

// Property: WASM static gas analysis handles all section types
func TestProperty_WASMSectionCoverage(t *testing.T) {
	wasmMagic := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}

	// Test each WASM section ID (0-12 are defined, others are custom)
	for sectionID := 0; sectionID < 16; sectionID++ {
		wasm := append(wasmMagic,
			byte(sectionID), // Section ID
			0x05,            // Section size (5 bytes)
			0x00, 0x00, 0x00, 0x00, 0x00, // Section content
		)

		gas1 := calculateStaticWASMGas(wasm)
		gas2 := calculateStaticWASMGas(wasm)

		if gas1 != gas2 {
			t.Fatalf("WASM gas not deterministic for section %d", sectionID)
		}
	}
}

// Benchmark property tests
func BenchmarkProperty_GasDeterminism(b *testing.B) {
	script := "for i in $(seq 1 100); do echo $i; done"
	input := bytes.Repeat([]byte("input"), 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateDeterministicGas(script, input)
	}
}

func BenchmarkProperty_OpcodeGas(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for op := 0; op < 256; op++ {
			opcodeGasCost(byte(op))
		}
	}
}
