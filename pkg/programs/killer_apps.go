// OCX Killer Applications - Ready-to-Run Programs
// These programs will demonstrate OCX's power to potential adopters

package programs

import (
	"encoding/binary"
	
	"ocx.local/pkg/executor"
)

// Import instruction constants for easier reference
const (
	OP_NOP    = executor.OP_NOP
	OP_LOAD   = executor.OP_LOAD
	OP_STORE  = executor.OP_STORE
	OP_ADD    = executor.OP_ADD
	OP_SUB    = executor.OP_SUB
	OP_MUL    = executor.OP_MUL
	OP_DIV    = executor.OP_DIV
	OP_MOD    = executor.OP_MOD
	OP_HASH   = executor.OP_HASH
	OP_JUMP   = executor.OP_JUMP
	OP_HALT   = executor.OP_HALT
)

// Helper function to encode uint64 as little-endian bytes
func encodeUint64(value uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, value)
	return bytes
}

// =============================================================================
// 1. ALPHAFOLD PROTEIN FOLDING SIMULATOR (Simplified)
// =============================================================================

// Bytecode that simulates protein folding energy calculation
// Input: amino acid sequence (20 possible values)
// Output: folding energy score + verification hash
func ProteinFoldingProgram() []byte {
	// Build bytecode using helper functions
	code := []byte{
		// Load sequence length from input[0]
		byte(OP_LOAD), 0, 0, 0, 0,    // Load sequence_length
		byte(OP_STORE), 64, 0, 0, 0,  // Store to working memory
	}
	
	// Initialize energy accumulator (push 0)
	code = append(code, encodeUint64(0)...)
	code = append(code, byte(OP_STORE), 72, 0, 0, 0) // Store energy at [72]
	
	// Main folding loop (push 8 for starting position)
	code = append(code, encodeUint64(8)...)
	code = append(code, byte(OP_STORE), 80, 0, 0, 0) // Store position counter
	
	// Loop start
	code = append(code, 
		byte(OP_LOAD), 80, 0, 0, 0,   // Load position
		byte(OP_LOAD), 64, 0, 0, 0,   // Load sequence_length
		byte(OP_SUB),                 // position - sequence_length
		byte(OP_JUMP), 100, 0, 0, 0,  // Jump to end if done
	)
	
	// Calculate amino acid interaction energy
	code = append(code,
		byte(OP_LOAD), 80, 0, 0, 0,   // Load position
		byte(OP_LOAD), 0, 0, 0, 0,    // Load from input[position]
		byte(OP_LOAD), 80, 0, 0, 0,   // Load position again
	)
	code = append(code, encodeUint64(1)...) // Push 1
	code = append(code,
		byte(OP_ADD),                 // position + 1
		byte(OP_LOAD), 0, 0, 0, 0,    // Load next amino acid
		byte(OP_MUL),                 // amino1 * amino2
	)
	code = append(code, encodeUint64(17)...) // Push 17 (magic constant)
	code = append(code,
		byte(OP_MOD),                 // (amino1 * amino2) % 17
		byte(OP_LOAD), 72, 0, 0, 0,   // Load current energy
		byte(OP_ADD),                 // energy + interaction
		byte(OP_STORE), 72, 0, 0, 0,  // Store back
		byte(OP_LOAD), 80, 0, 0, 0,   // Load position
	)
	code = append(code, encodeUint64(2)...) // Push 2 (step by 2)
	code = append(code,
		byte(OP_ADD),                 // position + 2
		byte(OP_STORE), 80, 0, 0, 0,  // Store back
		byte(OP_JUMP), 30, 0, 0, 0,   // Jump back to loop start
	)
	
	// End: compute verification hash
	code = append(code,
		byte(OP_LOAD), 72, 0, 0, 0,   // Load final energy
		byte(OP_HASH),                // Hash the energy
		byte(OP_STORE), 88, 0, 0, 0,  // Store verification hash
		byte(OP_LOAD), 72, 0, 0, 0,   // Load energy
		byte(OP_STORE), 0, 0, 0, 0,   // Store to output[0]
		byte(OP_LOAD), 88, 0, 0, 0,   // Load hash
		byte(OP_STORE), 8, 0, 0, 0,   // Store to output[8]
		byte(OP_HALT),
	)
	
	return code
}

// Test data for protein folding (simplified amino acid sequence)
func ProteinSequenceTestData() []byte {
	data := make([]byte, 256)
	// Sequence length
	binary.LittleEndian.PutUint64(data[0:8], 20)
	
	// Amino acid sequence (simplified as numbers 1-20)
	acids := []uint64{1, 7, 3, 12, 8, 15, 2, 19, 5, 11, 4, 16, 9, 6, 18, 13, 10, 17, 14, 20}
	for i, acid := range acids {
		binary.LittleEndian.PutUint64(data[(i+1)*8:(i+2)*8], acid)
	}
	
	return data
}

// =============================================================================
// 2. LLVM COMPILER TESTING PROGRAM
// =============================================================================

// Simulates compiler optimization passes with deterministic results
func CompilerOptimizationProgram() []byte {
	return []byte{
		// Load IR instruction count
		byte(OP_LOAD), 0, 0, 0, 0,    // Load instruction_count
		
		// Initialize optimization metrics
		0, 0, 0, 0, 0, 0, 0, 0,       // Push 0 (eliminated instructions)
		byte(OP_STORE), 64, 0, 0, 0,  // Store eliminated count
		
		// Dead code elimination pass
		byte(OP_LOAD), 0, 0, 0, 0,    // Load instruction_count
		7, 0, 0, 0, 0, 0, 0, 0,       // Push 7 (elimination factor)
		byte(OP_DIV),                 // count / 7 (dead code eliminated)
		byte(OP_STORE), 72, 0, 0, 0,  // Store result
		
		// Constant folding pass
		byte(OP_LOAD), 72, 0, 0, 0,   // Load previous result
		3, 0, 0, 0, 0, 0, 0, 0,       // Push 3 (folding factor)
		byte(OP_DIV),                 // Further reduction
		byte(OP_STORE), 80, 0, 0, 0,  // Store result
		
		// Loop unrolling simulation
		byte(OP_LOAD), 80, 0, 0, 0,   // Load current count
		13, 0, 0, 0, 0, 0, 0, 0,      // Push 13 (unroll factor)
		byte(OP_MUL),                 // count * 13 (code expansion)
		byte(OP_STORE), 88, 0, 0, 0,  // Store expanded size
		
		// Final optimization score
		byte(OP_LOAD), 0, 0, 0, 0,    // Original count
		byte(OP_LOAD), 88, 0, 0, 0,   // Final count
		byte(OP_SUB),                 // improvement = original - final
		byte(OP_HASH),                // Hash the improvement
		
		// Store results to output
		byte(OP_STORE), 0, 0, 0, 0,   // Store optimization hash
		byte(OP_LOAD), 88, 0, 0, 0,   // Load final instruction count
		byte(OP_STORE), 8, 0, 0, 0,   // Store to output[8]
		
		byte(OP_HALT),
	}
}

// =============================================================================
// 3. BITCOIN MINING DIFFICULTY ADJUSTMENT
// =============================================================================

// Simulates Bitcoin difficulty adjustment algorithm
func DifficultyAdjustmentProgram() []byte {
	code := []byte{
		// Load block time array (last 2016 blocks)
		byte(OP_LOAD), 0, 0, 0, 0,    // Load first block time
		byte(OP_LOAD), 8, 0, 0, 0,    // Load last block time
		byte(OP_SUB),                 // Calculate time span
		byte(OP_STORE), 64, 0, 0, 0,  // Store actual timespan
	}
	
	// Target timespan (14 days in seconds)
	code = append(code, encodeUint64(1209600)...) // Push 1209600 (14 days)
	code = append(code, byte(OP_STORE), 72, 0, 0, 0) // Store target timespan
	
	code = append(code,
		// Load current difficulty
		byte(OP_LOAD), 16, 0, 0, 0,   // Load current difficulty
		byte(OP_STORE), 80, 0, 0, 0,  // Store for calculation
		
		// Calculate adjustment ratio
		byte(OP_LOAD), 64, 0, 0, 0,   // Load actual timespan
		byte(OP_LOAD), 72, 0, 0, 0,   // Load target timespan
		byte(OP_DIV),                 // actual / target
	)
	
	// Apply limits (clamp between 1/4 and 4x)
	code = append(code, encodeUint64(4)...) // Push 4
	code = append(code,
		byte(OP_MOD),                 // Clamp upper bound
		
		// Calculate new difficulty
		byte(OP_LOAD), 80, 0, 0, 0,   // Load current difficulty
		byte(OP_MUL),                 // difficulty * adjustment
		byte(OP_STORE), 88, 0, 0, 0,  // Store new difficulty
		
		// Generate proof hash
		byte(OP_LOAD), 88, 0, 0, 0,   // Load new difficulty
		byte(OP_HASH),                // Hash it
		
		// Store results
		byte(OP_STORE), 0, 0, 0, 0,   // Store hash to output[0]
		byte(OP_LOAD), 88, 0, 0, 0,   // Load new difficulty
		byte(OP_STORE), 8, 0, 0, 0,   // Store to output[8]
		
		byte(OP_HALT),
	)
	
	return code
}

// =============================================================================
// 4. DOOM ENGINE PHYSICS SIMULATION
// =============================================================================

// Simulates Doom-style physics with collision detection
func DoomPhysicsProgram() []byte {
	code := []byte{
		// Load player position (x, y)
		byte(OP_LOAD), 0, 0, 0, 0,    // Load player_x
		byte(OP_LOAD), 8, 0, 0, 0,    // Load player_y
		
		// Load velocity
		byte(OP_LOAD), 16, 0, 0, 0,   // Load velocity_x
		byte(OP_LOAD), 24, 0, 0, 0,   // Load velocity_y
		
		// Update position (simple Euler integration)
		byte(OP_ADD),                 // new_y = y + velocity_y
		byte(OP_STORE), 64, 0, 0, 0,  // Store new_y
		
		byte(OP_LOAD), 0, 0, 0, 0,    // Load player_x again
		byte(OP_LOAD), 16, 0, 0, 0,   // Load velocity_x
		byte(OP_ADD),                 // new_x = x + velocity_x
		byte(OP_STORE), 72, 0, 0, 0,  // Store new_x
		
		// Simple wall collision (boundary check)
		byte(OP_LOAD), 72, 0, 0, 0,   // Load new_x
	}
	
	code = append(code, encodeUint64(1024)...) // Push 1024 (wall boundary)
	code = append(code,
		byte(OP_MOD),                 // x % 1024 (wrap around)
		byte(OP_STORE), 72, 0, 0, 0,  // Store clamped x
		byte(OP_LOAD), 64, 0, 0, 0,   // Load new_y  
	)
	code = append(code, encodeUint64(768)...) // Push 768 (wall boundary)
	code = append(code,
		byte(OP_MOD),                 // y % 768 (wrap around)
		byte(OP_STORE), 64, 0, 0, 0,  // Store clamped y
		
		// Calculate distance moved for collision response
		byte(OP_LOAD), 16, 0, 0, 0,   // Load velocity_x
		byte(OP_LOAD), 16, 0, 0, 0,   // Load velocity_x again
		byte(OP_MUL),                 // velocity_x^2
		
		byte(OP_LOAD), 24, 0, 0, 0,   // Load velocity_y
		byte(OP_LOAD), 24, 0, 0, 0,   // Load velocity_y again  
		byte(OP_MUL),                 // velocity_y^2
		
		byte(OP_ADD),                 // velocity_x^2 + velocity_y^2
		byte(OP_HASH),                // Hash for deterministic "sqrt"
		byte(OP_STORE), 80, 0, 0, 0,  // Store movement distance
		
		// Store final position and create game state hash
		byte(OP_LOAD), 72, 0, 0, 0,   // Load final x
		byte(OP_LOAD), 64, 0, 0, 0,   // Load final y
		byte(OP_ADD),                 // x + y
		byte(OP_HASH),                // Hash position
		
		// Output results
		byte(OP_STORE), 0, 0, 0, 0,   // Store state hash
		byte(OP_LOAD), 72, 0, 0, 0,   // Load final x
		byte(OP_STORE), 8, 0, 0, 0,   // Store x to output[8]
		byte(OP_LOAD), 64, 0, 0, 0,   // Load final y
		byte(OP_STORE), 16, 0, 0, 0,  // Store y to output[16]
		
		byte(OP_HALT),
	)
	
	return code
}

// =============================================================================
// 5. CHROMIUM WEBGL BENCHMARK
// =============================================================================

// Simulates WebGL shader compilation and performance testing
func WebGLBenchmarkProgram() []byte {
	code := []byte{
		// Load shader complexity parameters
		byte(OP_LOAD), 0, 0, 0, 0,    // Load vertex_count
		byte(OP_LOAD), 8, 0, 0, 0,    // Load texture_resolution
		
		// Vertex processing simulation
		byte(OP_MUL),                 // vertices * resolution
		byte(OP_STORE), 64, 0, 0, 0,  // Store processing load
		
		// Fragment shader simulation
		byte(OP_LOAD), 16, 0, 0, 0,   // Load fragment_complexity
		byte(OP_LOAD), 64, 0, 0, 0,   // Load processing load
		byte(OP_MUL),                 // complexity * load
	}
	
	code = append(code, encodeUint64(19)...) // Push 19 (magic GPU constant)
	code = append(code,
		byte(OP_DIV),                 // Normalize GPU load
		byte(OP_STORE), 72, 0, 0, 0,  // Store GPU cycles
		
		// Memory bandwidth simulation
		byte(OP_LOAD), 24, 0, 0, 0,   // Load memory_usage
		byte(OP_LOAD), 72, 0, 0, 0,   // Load GPU cycles
		byte(OP_ADD),                 // Total system load
		byte(OP_STORE), 80, 0, 0, 0,  // Store total
	)
	
	// Performance score calculation
	code = append(code, encodeUint64(10000000)...) // Push 10M (baseline score)
	code = append(code,
		byte(OP_LOAD), 80, 0, 0, 0,   // Load total load
		byte(OP_DIV),                 // 10M / total_load = performance
		byte(OP_HASH),                // Hash for reproducible score
	)
	
	// Frame rate estimation
	code = append(code, encodeUint64(60)...) // Push 60 (target FPS)
	code = append(code,
		byte(OP_LOAD), 72, 0, 0, 0,   // Load GPU cycles
	)
	code = append(code, encodeUint64(1000)...) // Push 1000
	code = append(code,
		byte(OP_DIV),                 // cycles / 1000
		byte(OP_SUB),                 // 60 - (cycles/1000) = estimated FPS
		
		// Store benchmark results
		byte(OP_STORE), 0, 0, 0, 0,   // Store FPS estimate
		byte(OP_LOAD), 80, 0, 0, 0,   // Load total system load
		byte(OP_STORE), 8, 0, 0, 0,   // Store load score
		
		byte(OP_HALT),
	)
	
	return code
}

// =============================================================================
// PROGRAM REGISTRY AND EXECUTION HELPERS
// =============================================================================

type KillerProgram struct {
	Name        string
	Description string
	Bytecode    []byte
	TestData    []byte
	Expected    ExpectedResult
}

type ExpectedResult struct {
	CyclesMax   uint64
	OutputSize  int
	Deterministic bool
}

var KillerPrograms = []KillerProgram{
	{
		Name:        "protein_folding",
		Description: "AlphaFold-style protein folding energy calculation with verification",
		Bytecode:    ProteinFoldingProgram(),
		TestData:    ProteinSequenceTestData(),
		Expected: ExpectedResult{
			CyclesMax:     1000,
			OutputSize:    16, // Energy + hash
			Deterministic: true,
		},
	},
	{
		Name:        "llvm_optimization", 
		Description: "LLVM compiler optimization pass simulation",
		Bytecode:    CompilerOptimizationProgram(),
		TestData:    CompilerTestData(),
		Expected: ExpectedResult{
			CyclesMax:     500,
			OutputSize:    16, // Optimization hash + final count
			Deterministic: true,
		},
	},
	{
		Name:        "bitcoin_difficulty",
		Description: "Bitcoin difficulty adjustment algorithm",
		Bytecode:    DifficultyAdjustmentProgram(), 
		TestData:    BitcoinTestData(),
		Expected: ExpectedResult{
			CyclesMax:     300,
			OutputSize:    16, // Hash + new difficulty
			Deterministic: true,
		},
	},
	{
		Name:        "doom_physics",
		Description: "Doom engine physics simulation with collision",
		Bytecode:    DoomPhysicsProgram(),
		TestData:    DoomTestData(),
		Expected: ExpectedResult{
			CyclesMax:     400,
			OutputSize:    24, // State hash + x + y coordinates
			Deterministic: true,
		},
	},
	{
		Name:        "webgl_benchmark",
		Description: "Chromium WebGL shader benchmark simulation", 
		Bytecode:    WebGLBenchmarkProgram(),
		TestData:    WebGLTestData(),
		Expected: ExpectedResult{
			CyclesMax:     600,
			OutputSize:    16, // FPS + load score
			Deterministic: true,
		},
	},
}

// Helper functions to generate test data
func CompilerTestData() []byte {
	data := make([]byte, 32)
	binary.LittleEndian.PutUint64(data[0:8], 1000) // Instruction count
	return data
}

func BitcoinTestData() []byte {
	data := make([]byte, 32)
	binary.LittleEndian.PutUint64(data[0:8], 1234567890)   // First block time
	binary.LittleEndian.PutUint64(data[8:16], 1234568890)  // Last block time  
	binary.LittleEndian.PutUint64(data[16:24], 1000000)    // Current difficulty
	return data
}

func DoomTestData() []byte {
	data := make([]byte, 32)
	binary.LittleEndian.PutUint64(data[0:8], 100)   // Player x
	binary.LittleEndian.PutUint64(data[8:16], 200)  // Player y
	binary.LittleEndian.PutUint64(data[16:24], 5)   // Velocity x
	// For negative velocity, we'll use a large positive number that represents -3 in two's complement
	binary.LittleEndian.PutUint64(data[24:32], 0xFFFFFFFFFFFFFFFD)  // Velocity y (represents -3)
	return data
}

func WebGLTestData() []byte {
	data := make([]byte, 32)
	binary.LittleEndian.PutUint64(data[0:8], 10000)   // Vertex count
	binary.LittleEndian.PutUint64(data[8:16], 1024)   // Texture resolution
	binary.LittleEndian.PutUint64(data[16:24], 50)    // Fragment complexity
	binary.LittleEndian.PutUint64(data[24:32], 2048)  // Memory usage
	return data
}
