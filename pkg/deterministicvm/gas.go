// Package deterministicvm provides deterministic gas metering for OCX Protocol
package deterministicvm

import (
	"fmt"
	"strings"
)

// GasMeter provides deterministic gas metering for VM execution
type GasMeter struct {
	gasUsed     uint64
	gasLimit    uint64
	opcodeCosts map[string]uint64
}

// GasCosts defines the deterministic cost table for different operations
var GasCosts = map[string]uint64{
	// Basic operations
	"OP_PUSH":  1,
	"OP_POP":   1,
	"OP_LOAD":  2,
	"OP_STORE": 2,
	"OP_ADD":   3,
	"OP_SUB":   3,
	"OP_MUL":   5,
	"OP_DIV":   5,
	"OP_MOD":   5,
	"OP_EQ":    2,
	"OP_NE":    2,
	"OP_LT":    2,
	"OP_LE":    2,
	"OP_GT":    2,
	"OP_GE":    2,
	"OP_AND":   2,
	"OP_OR":    2,
	"OP_NOT":   1,
	"OP_XOR":   2,
	"OP_SHL":   3,
	"OP_SHR":   3,
	"OP_HALT":  1,

	// Memory operations
	"OP_MEMCPY": 10,
	"OP_MEMSET": 8,
	"OP_MEMCMP": 6,

	// Control flow
	"OP_JUMP":   2,
	"OP_JUMPIF": 3,
	"OP_CALL":   5,
	"OP_RET":    2,

	// I/O operations (expensive)
	"OP_READ":  100,
	"OP_WRITE": 100,
	"OP_PRINT": 50,

	// System operations (very expensive)
	"OP_SYSCALL": 1000,
	"OP_NETWORK": 5000,
	"OP_FILE":    2000,
}

// NewGasMeter creates a new gas meter with the specified limit
func NewGasMeter(gasLimit uint64) *GasMeter {
	return &GasMeter{
		gasUsed:     0,
		gasLimit:    gasLimit,
		opcodeCosts: GasCosts,
	}
}

// ConsumeGas consumes the specified amount of gas
func (gm *GasMeter) ConsumeGas(amount uint64) error {
	if gm.gasUsed+amount > gm.gasLimit {
		return &ExecutionError{
			Code:    ErrorCodeCycleLimitExceeded,
			Message: fmt.Sprintf("gas limit exceeded: used %d, limit %d", gm.gasUsed+amount, gm.gasLimit),
		}
	}
	gm.gasUsed += amount
	return nil
}

// ConsumeOpcodeGas consumes gas for a specific opcode
func (gm *GasMeter) ConsumeOpcodeGas(opcode string) error {
	cost, exists := gm.opcodeCosts[opcode]
	if !exists {
		// Default cost for unknown opcodes
		cost = 10
	}
	return gm.ConsumeGas(cost)
}

// ConsumeInstructionGas consumes gas for a complete instruction
func (gm *GasMeter) ConsumeInstructionGas(instruction []byte) error {
	if len(instruction) == 0 {
		return nil
	}

	// Parse opcode from instruction
	opcode := gm.parseOpcode(instruction)
	return gm.ConsumeOpcodeGas(opcode)
}

// parseOpcode extracts the opcode from an instruction
func (gm *GasMeter) parseOpcode(instruction []byte) string {
	if len(instruction) == 0 {
		return "UNKNOWN"
	}

	// Simple opcode parsing - in a real implementation, this would be more sophisticated
	switch instruction[0] {
	case 0x01:
		return "OP_PUSH"
	case 0x02:
		return "OP_POP"
	case 0x03:
		return "OP_LOAD"
	case 0x04:
		return "OP_STORE"
	case 0x05:
		return "OP_ADD"
	case 0x06:
		return "OP_SUB"
	case 0x07:
		return "OP_MUL"
	case 0x08:
		return "OP_DIV"
	case 0x09:
		return "OP_MOD"
	case 0x0A:
		return "OP_EQ"
	case 0x0B:
		return "OP_NE"
	case 0x0C:
		return "OP_LT"
	case 0x0D:
		return "OP_LE"
	case 0x0E:
		return "OP_GT"
	case 0x0F:
		return "OP_GE"
	case 0x10:
		return "OP_AND"
	case 0x11:
		return "OP_OR"
	case 0x12:
		return "OP_NOT"
	case 0x13:
		return "OP_XOR"
	case 0x14:
		return "OP_SHL"
	case 0x15:
		return "OP_SHR"
	case 0x16:
		return "OP_JUMP"
	case 0x17:
		return "OP_JUMPIF"
	case 0x18:
		return "OP_CALL"
	case 0x19:
		return "OP_RET"
	case 0x1A:
		return "OP_HALT"
	case 0x1B:
		return "OP_MEMCPY"
	case 0x1C:
		return "OP_MEMSET"
	case 0x1D:
		return "OP_MEMCMP"
	case 0x1E:
		return "OP_READ"
	case 0x1F:
		return "OP_WRITE"
	case 0x20:
		return "OP_PRINT"
	case 0x21:
		return "OP_SYSCALL"
	case 0x22:
		return "OP_NETWORK"
	case 0x23:
		return "OP_FILE"
	default:
		return "UNKNOWN"
	}
}

// GetGasUsed returns the current gas usage
func (gm *GasMeter) GetGasUsed() uint64 {
	return gm.gasUsed
}

// GetGasRemaining returns the remaining gas
func (gm *GasMeter) GetGasRemaining() uint64 {
	if gm.gasUsed >= gm.gasLimit {
		return 0
	}
	return gm.gasLimit - gm.gasUsed
}

// Reset resets the gas meter
func (gm *GasMeter) Reset() {
	gm.gasUsed = 0
}

// SetGasLimit sets a new gas limit
func (gm *GasMeter) SetGasLimit(limit uint64) {
	gm.gasLimit = limit
}

// CalculateDeterministicGas calculates deterministic gas for shell script execution
func CalculateDeterministicGas(script string, input []byte) uint64 {
	gas := uint64(0)

	// Base cost for script execution
	gas += 100

	// Cost per character in script
	gas += uint64(len(script))

	// Cost per line in script
	lines := strings.Count(script, "\n") + 1
	gas += uint64(lines * 10)

	// Cost for input processing
	gas += uint64(len(input))

	// Cost for common shell operations
	if strings.Contains(script, "echo") {
		gas += 50
	}
	if strings.Contains(script, "cat") {
		gas += 30
	}
	if strings.Contains(script, "ls") {
		gas += 100
	}
	if strings.Contains(script, "grep") {
		gas += 200
	}
	if strings.Contains(script, "sed") {
		gas += 300
	}
	if strings.Contains(script, "awk") {
		gas += 500
	}
	if strings.Contains(script, "sort") {
		gas += 400
	}
	if strings.Contains(script, "uniq") {
		gas += 200
	}
	if strings.Contains(script, "wc") {
		gas += 100
	}
	if strings.Contains(script, "head") {
		gas += 50
	}
	if strings.Contains(script, "tail") {
		gas += 50
	}
	if strings.Contains(script, "cut") {
		gas += 150
	}
	if strings.Contains(script, "tr") {
		gas += 200
	}
	if strings.Contains(script, "xargs") {
		gas += 300
	}
	if strings.Contains(script, "find") {
		gas += 1000
	}
	if strings.Contains(script, "tar") {
		gas += 2000
	}
	if strings.Contains(script, "gzip") {
		gas += 1500
	}
	if strings.Contains(script, "curl") {
		gas += 5000
	}
	if strings.Contains(script, "wget") {
		gas += 5000
	}
	if strings.Contains(script, "ssh") {
		gas += 10000
	}
	if strings.Contains(script, "scp") {
		gas += 8000
	}

	// Cost for loops and conditionals
	loopCount := strings.Count(script, "for") + strings.Count(script, "while") + strings.Count(script, "until")
	gas += uint64(loopCount * 100)

	ifCount := strings.Count(script, "if")
	gas += uint64(ifCount * 50)

	// Cost for function definitions
	funcCount := strings.Count(script, "function") + strings.Count(script, "()")
	gas += uint64(funcCount * 200)

	return gas
}

// GasMeterConfig provides configuration for gas metering
type GasMeterConfig struct {
	// GasLimit is the maximum gas allowed for execution
	GasLimit uint64

	// EnableDetailedMetering enables detailed per-instruction gas metering
	EnableDetailedMetering bool

	// CustomCosts allows overriding default gas costs
	CustomCosts map[string]uint64
}

// DefaultGasMeterConfig returns default gas meter configuration
func DefaultGasMeterConfig() GasMeterConfig {
	return GasMeterConfig{
		GasLimit:               10_000_000, // 10M gas units
		EnableDetailedMetering: true,
		CustomCosts:            make(map[string]uint64),
	}
}
