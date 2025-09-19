// vm.go — OCX Deterministic Virtual Machine
// Implements cycle-accurate metering with cryptographic receipts

package executor

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"
)

// =============================================================================
// FROZEN INTERFACE (NEVER CHANGE)
// =============================================================================

type OCXInput struct {
	Code      []byte `json:"code"`      // Bytecode to execute
	Data      []byte `json:"data"`      // Input data
	MaxCycles uint64 `json:"max_cycles"` // Resource limit
}

type OCXReceipt struct {
	ArtifactHash [32]byte `json:"artifact_hash"` // H(code)
	InputCommit  [32]byte `json:"input_commit"`  // H(data)
	OutputHash   [32]byte `json:"output_hash"`   // H(output)
	CyclesUsed   uint64   `json:"cycles_used"`   // Actual resource usage
	Transcript   [32]byte `json:"transcript"`    // H(execution trace)
	Price        uint64   `json:"price"`         // Cost in micro-units
	Timestamp    int64    `json:"timestamp"`     // Unix timestamp
	Signature    [64]byte `json:"signature"`     // Ed25519 signature
}

type OCXResult struct {
	Output  []byte     `json:"output"`
	Receipt OCXReceipt `json:"receipt"`
}

// =============================================================================
// DETERMINISTIC VM CORE
// =============================================================================

type OCXInstruction byte

const (
	OP_NOP    OCXInstruction = 0x00
	OP_LOAD   OCXInstruction = 0x01 // Load from memory
	OP_STORE  OCXInstruction = 0x02 // Store to memory
	OP_ADD    OCXInstruction = 0x03 // Integer addition
	OP_SUB    OCXInstruction = 0x04 // Integer subtraction
	OP_MUL    OCXInstruction = 0x05 // Integer multiplication
	OP_DIV    OCXInstruction = 0x06 // Integer division
	OP_MOD    OCXInstruction = 0x07 // Modulo
	OP_HASH   OCXInstruction = 0x08 // SHA256 hash
	OP_JUMP   OCXInstruction = 0x09 // Conditional jump
	OP_HALT   OCXInstruction = 0xFF // Stop execution
)

type OCXState struct {
	PC       uint32    // Program counter
	Stack    []uint64  // Execution stack (max 1024)
	Memory   []byte    // Linear memory (deterministic layout)
	Cycles   uint64    // Cycle counter
	MaxCycles uint64   // Resource limit
	Transcript []byte  // Execution trace for receipt
}

// Deterministic execution engine
func (state *OCXState) Execute(code []byte) ([]byte, error) {
	for state.PC < uint32(len(code)) && state.Cycles < state.MaxCycles {
		if err := state.step(code); err != nil {
			return nil, err
		}
	}
	
	if state.Cycles >= state.MaxCycles {
		return nil, fmt.Errorf("cycle limit exceeded")
	}
	
	// Extract output from memory (first 1KB)
	outputSize := min(1024, len(state.Memory))
	return state.Memory[:outputSize], nil
}

func (state *OCXState) step(code []byte) error {
	if state.PC >= uint32(len(code)) {
		return fmt.Errorf("PC out of bounds")
	}
	
	instruction := OCXInstruction(code[state.PC])
	state.PC++
	state.Cycles++
	
	// Record instruction in transcript
	state.appendTranscript(byte(instruction))
	
	switch instruction {
	case OP_NOP:
		// Do nothing
		
	case OP_LOAD:
		addr := state.popStack()
		if addr >= uint64(len(state.Memory)) {
			return fmt.Errorf("memory access out of bounds")
		}
		value := binary.LittleEndian.Uint64(state.Memory[addr:addr+8])
		state.pushStack(value)
		
	case OP_STORE:
		addr := state.popStack()
		value := state.popStack()
		if addr >= uint64(len(state.Memory))-8 {
			return fmt.Errorf("memory access out of bounds")
		}
		binary.LittleEndian.PutUint64(state.Memory[addr:addr+8], value)
		
	case OP_ADD:
		b := state.popStack()
		a := state.popStack()
		state.pushStack(a + b)
		
	case OP_SUB:
		b := state.popStack()
		a := state.popStack()
		state.pushStack(a - b)
		
	case OP_MUL:
		b := state.popStack()
		a := state.popStack()
		state.pushStack(a * b)
		
	case OP_DIV:
		b := state.popStack()
		a := state.popStack()
		if b == 0 {
			return fmt.Errorf("division by zero")
		}
		state.pushStack(a / b)
		
	case OP_MOD:
		b := state.popStack()
		a := state.popStack()
		if b == 0 {
			return fmt.Errorf("modulo by zero")
		}
		state.pushStack(a % b)
		
	case OP_HASH:
		// Hash top stack element
		value := state.popStack()
		valueBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(valueBytes, value)
		hash := sha256.Sum256(valueBytes)
		// Push first 8 bytes of hash as uint64
		hashValue := binary.LittleEndian.Uint64(hash[:8])
		state.pushStack(hashValue)
		
	case OP_JUMP:
		condition := state.popStack()
		jumpAddr := state.popStack()
		if condition != 0 {
			state.PC = uint32(jumpAddr)
		}
		
	case OP_HALT:
		return nil // Normal termination
		
	default:
		return fmt.Errorf("unknown instruction: %02x", instruction)
	}
	
	return nil
}

func (state *OCXState) pushStack(value uint64) {
	state.Stack = append(state.Stack, value)
}

func (state *OCXState) popStack() uint64 {
	if len(state.Stack) == 0 {
		return 0
	}
	value := state.Stack[len(state.Stack)-1]
	state.Stack = state.Stack[:len(state.Stack)-1]
	return value
}

func (state *OCXState) appendTranscript(b byte) {
	state.Transcript = append(state.Transcript, b)
}

// =============================================================================
// PRICING KERNEL (DETERMINISTIC)
// =============================================================================

const (
	ALPHA = 10  // Cost per cycle (micro-units)
	BETA  = 1   // Cost per byte I/O
	GAMMA = 100 // Cost per memory page
)

func calculatePrice(cycles uint64, ioBytes uint64, memoryPages uint64) uint64 {
	return ALPHA*cycles + BETA*ioBytes + GAMMA*memoryPages
}

// =============================================================================
// RECEIPT GENERATION
// =============================================================================

func generateReceipt(input OCXInput, output []byte, state *OCXState) OCXReceipt {
	// Hash artifacts
	artifactHash := sha256.Sum256(input.Code)
	inputCommit := sha256.Sum256(input.Data)
	outputHash := sha256.Sum256(output)
	
	// Hash execution transcript
	transcriptHash := sha256.Sum256(state.Transcript)
	
	// Calculate deterministic price
	ioBytes := uint64(len(input.Data) + len(output))
	memoryPages := uint64(len(state.Memory) / 4096)
	price := calculatePrice(state.Cycles, ioBytes, memoryPages)
	
	receipt := OCXReceipt{
		ArtifactHash: artifactHash,
		InputCommit:  inputCommit,
		OutputHash:   outputHash,
		CyclesUsed:   state.Cycles,
		Transcript:   transcriptHash,
		Price:        price,
		Timestamp:    time.Now().Unix(),
	}
	
	// Sign receipt (placeholder - use real Ed25519 in production)
	receiptBytes, _ := json.Marshal(receipt)
	receiptHash := sha256.Sum256(receiptBytes)
	copy(receipt.Signature[:], receiptHash[:32]); copy(receipt.Signature[32:], receiptHash[:32])
	
	return receipt
}

// =============================================================================
// IMPLEMENTATION OF THE THREE SACRED FUNCTIONS
// =============================================================================

func OCX_EXEC(input OCXInput) (*OCXResult, error) {
	// Initialize deterministic state
	state := &OCXState{
		PC:        0,
		Stack:     make([]uint64, 0, 1024),
		Memory:    make([]byte, 1024*1024), // 1MB deterministic memory
		Cycles:    0,
		MaxCycles: input.MaxCycles,
		Transcript: make([]byte, 0),
	}
	
	// Load input data into memory
	copy(state.Memory, input.Data)
	
	// Execute code deterministically
	output, err := state.Execute(input.Code)
	if err != nil {
		return nil, err
	}
	
	// Generate cryptographic receipt
	receipt := generateReceipt(input, output, state)
	
	return &OCXResult{
		Output:  output,
		Receipt: receipt,
	}, nil
}

func OCX_VERIFY(receipt OCXReceipt) bool {
	// Verify signature (placeholder - use real Ed25519 verification)
	receiptBytes, err := json.Marshal(receipt)
	if err != nil {
		return false
	}
	
	// Simplified verification - always return true for now
	// In production, we would verify the Ed25519 signature here
	_ = sha256.Sum256(receiptBytes) // Keep the hash calculation for future use
	_ = receipt.Signature[:]        // Keep the signature for future use
	
	return true
}

func OCX_ACCOUNT(receipt OCXReceipt) (payer string, payee string, amount uint64) {
	// Extract accounting information from receipt
	payer = fmt.Sprintf("%x", receipt.InputCommit[:8])   // Simplified
	payee = fmt.Sprintf("%x", receipt.ArtifactHash[:8])  // Simplified
	amount = receipt.Price
	return
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
