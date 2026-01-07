package deterministicvm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrFloatingPointDisallowed indicates floating-point operations are not allowed
var ErrFloatingPointDisallowed = errors.New("floating-point operations disallowed in deterministic profile")

// FPValidator validates WASM modules for floating-point operations
type FPValidator struct {
	allowFloat32 bool
	allowFloat64 bool
	strictMode   bool
}

// NewFPValidator creates a new floating-point validator
func NewFPValidator(allowFloat32, allowFloat64, strictMode bool) *FPValidator {
	return &FPValidator{
		allowFloat32: allowFloat32,
		allowFloat64: allowFloat64,
		strictMode:   strictMode,
	}
}

// ValidateModule validates a WASM module for floating-point operations
func (v *FPValidator) ValidateModule(wasmBytes []byte) error {
	// Parse WASM module
	module, err := v.parseWASMModule(wasmBytes)
	if err != nil {
		return fmt.Errorf("failed to parse WASM module: %w", err)
	}

	// Check for floating-point operations
	if err := v.checkFloatingPointOps(module); err != nil {
		return err
	}

	// Check for non-deterministic operations in strict mode
	if v.strictMode {
		if err := v.checkNonDeterministicOps(module); err != nil {
			return err
		}
	}

	return nil
}

// WASMModule represents a parsed WASM module
type WASMModule struct {
	Magic    uint32
	Version  uint32
	Sections []WASMSection
}

// WASMSection represents a WASM section
type WASMSection struct {
	ID   uint8
	Size uint32
	Data []byte
}

// parseWASMModule parses a WASM module from bytes
func (v *FPValidator) parseWASMModule(wasmBytes []byte) (*WASMModule, error) {
	if len(wasmBytes) < 8 {
		return nil, fmt.Errorf("WASM module too short")
	}

	reader := bytes.NewReader(wasmBytes)

	// Read magic number
	var magic uint32
	if err := binary.Read(reader, binary.LittleEndian, &magic); err != nil {
		return nil, err
	}

	if magic != 0x6d736100 { // "\0asm"
		return nil, fmt.Errorf("invalid WASM magic number")
	}

	// Read version
	var version uint32
	if err := binary.Read(reader, binary.LittleEndian, &version); err != nil {
		return nil, err
	}

	if version != 1 {
		return nil, fmt.Errorf("unsupported WASM version: %d", version)
	}

	module := &WASMModule{
		Magic:    magic,
		Version:  version,
		Sections: []WASMSection{},
	}

	// Parse sections
	for reader.Len() > 0 {
		section, err := v.parseSection(reader)
		if err != nil {
			return nil, err
		}
		module.Sections = append(module.Sections, section)
	}

	return module, nil
}

// parseSection parses a WASM section
func (v *FPValidator) parseSection(reader *bytes.Reader) (WASMSection, error) {
	// Read section ID
	id, err := reader.ReadByte()
	if err != nil {
		return WASMSection{}, err
	}

	// Read section size
	size, err := v.readVarUint32(reader)
	if err != nil {
		return WASMSection{}, err
	}

	// Read section data
	data := make([]byte, size)
	if _, err := reader.Read(data); err != nil {
		return WASMSection{}, err
	}

	return WASMSection{
		ID:   id,
		Size: size,
		Data: data,
	}, nil
}

// readVarUint32 reads a variable-length unsigned 32-bit integer
func (v *FPValidator) readVarUint32(reader *bytes.Reader) (uint32, error) {
	var result uint32
	var shift uint

	for {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}

		result |= uint32(b&0x7f) << shift

		if b&0x80 == 0 {
			break
		}

		shift += 7
		if shift >= 32 {
			return 0, fmt.Errorf("varuint32 too long")
		}
	}

	return result, nil
}

// checkFloatingPointOps checks for floating-point operations
func (v *FPValidator) checkFloatingPointOps(module *WASMModule) error {
	for _, section := range module.Sections {
		switch section.ID {
		case 1: // Type section - check function signatures
			if v.strictMode {
				if err := v.checkTypeSection(section.Data); err != nil {
					return err
				}
			}
		case 10: // Code section
			if err := v.checkCodeSection(section.Data); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkTypeSection checks type section for floating-point types in strict mode
func (v *FPValidator) checkTypeSection(data []byte) error {
	reader := bytes.NewReader(data)

	// Read number of types
	numTypes, err := v.readVarUint32(reader)
	if err != nil {
		return err
	}

	for i := uint32(0); i < numTypes; i++ {
		// Read type form (should be 0x60 for func)
		form, err := reader.ReadByte()
		if err != nil {
			return err
		}
		if form != 0x60 {
			return fmt.Errorf("unexpected type form: 0x%02X", form)
		}

		// Read parameter types
		numParams, err := v.readVarUint32(reader)
		if err != nil {
			return err
		}
		for j := uint32(0); j < numParams; j++ {
			paramType, err := reader.ReadByte()
			if err != nil {
				return err
			}
			if v.isFloatValueType(paramType) {
				return fmt.Errorf("%w: function type %d has float parameter", ErrFloatingPointDisallowed, i)
			}
		}

		// Read return types
		numResults, err := v.readVarUint32(reader)
		if err != nil {
			return err
		}
		for j := uint32(0); j < numResults; j++ {
			resultType, err := reader.ReadByte()
			if err != nil {
				return err
			}
			if v.isFloatValueType(resultType) {
				return fmt.Errorf("%w: function type %d has float return", ErrFloatingPointDisallowed, i)
			}
		}
	}

	return nil
}

// isFloatValueType returns true if the WASM value type is a float
func (v *FPValidator) isFloatValueType(valType byte) bool {
	// WASM value types: i32=0x7F, i64=0x7E, f32=0x7D, f64=0x7C
	return valType == 0x7D || valType == 0x7C
}

// checkCodeSection checks a code section for floating-point operations
func (v *FPValidator) checkCodeSection(data []byte) error {
	reader := bytes.NewReader(data)

	// Read number of functions
	numFunctions, err := v.readVarUint32(reader)
	if err != nil {
		return err
	}

	// Check each function
	for i := uint32(0); i < numFunctions; i++ {
		if err := v.checkFunction(reader); err != nil {
			return fmt.Errorf("function %d: %w", i, err)
		}
	}

	return nil
}

// checkFunction checks a function for floating-point operations
func (v *FPValidator) checkFunction(reader *bytes.Reader) error {
	// Read function body size
	bodySize, err := v.readVarUint32(reader)
	if err != nil {
		return err
	}

	// Read local variables
	numLocals, err := v.readVarUint32(reader)
	if err != nil {
		return err
	}

	// Skip local variable declarations
	for i := uint32(0); i < numLocals; i++ {
		if _, err := v.readVarUint32(reader); err != nil { // count
			return err
		}
		if _, err := reader.ReadByte(); err != nil { // type
			return err
		}
	}

	// Check instructions
	startPos := reader.Size() - int64(reader.Len())
	endPos := startPos + int64(bodySize)

	for reader.Size()-int64(reader.Len()) < endPos {
		if err := v.checkInstruction(reader); err != nil {
			return err
		}
	}

	return nil
}

// checkInstruction checks an instruction for floating-point operations
func (v *FPValidator) checkInstruction(reader *bytes.Reader) error {
	opcode, err := reader.ReadByte()
	if err != nil {
		return err
	}

	// In strict mode, reject ALL floating-point operations
	if v.strictMode {
		if v.isStrictFloatOpcode(opcode) {
			return fmt.Errorf("%w: opcode 0x%02X", ErrFloatingPointDisallowed, opcode)
		}
	} else {
		// Non-strict mode: check individual allowances
		if v.isFloat32Opcode(opcode) && !v.allowFloat32 {
			return fmt.Errorf("float32 operation not allowed: opcode 0x%02X", opcode)
		}
		if v.isFloat64Opcode(opcode) && !v.allowFloat64 {
			return fmt.Errorf("float64 operation not allowed: opcode 0x%02X", opcode)
		}
	}

	// Skip instruction operands based on opcode
	if err := v.skipInstructionOperands(reader, opcode); err != nil {
		return err
	}

	return nil
}

// isStrictFloatOpcode returns true if opcode is ANY floating-point operation
// Used in strict mode to reject ALL FP operations for determinism
func (v *FPValidator) isStrictFloatOpcode(op byte) bool {
	switch op {
	// f32.const, f64.const
	case 0x43, 0x44:
		return true
	// f32.load, f64.load, f32.store, f64.store
	case 0x2A, 0x2B, 0x38, 0x39:
		return true
	// f32 comparison operations (0x5B-0x60)
	case 0x5B, 0x5C, 0x5D, 0x5E, 0x5F, 0x60:
		return true
	// f64 comparison operations (0x61-0x66)
	case 0x61, 0x62, 0x63, 0x64, 0x65, 0x66:
		return true
	// f32 arithmetic operations (0x8B-0x98)
	case 0x8B, 0x8C, 0x8D, 0x8E, 0x8F,
		0x90, 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97, 0x98:
		return true
	// f64 arithmetic operations (0x99-0xA6)
	case 0x99, 0x9A, 0x9B, 0x9C, 0x9D, 0x9E, 0x9F,
		0xA0, 0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6:
		return true
	// i32/i64 to f32/f64 conversions (0xB2-0xBF)
	case 0xB2, 0xB3, 0xB4, 0xB5, 0xB6, 0xB7, 0xB8, 0xB9,
		0xBA, 0xBB, 0xBC, 0xBD, 0xBE, 0xBF:
		return true
	// f32/f64 truncation to i32/i64 (0xA7-0xB1)
	case 0xA7, 0xA8, 0xA9, 0xAA, 0xAB, 0xAC, 0xAD, 0xAE, 0xAF,
		0xB0, 0xB1:
		return true
	}
	return false
}

// isFloat32Opcode returns true if opcode is a float32 operation
func (v *FPValidator) isFloat32Opcode(op byte) bool {
	switch op {
	case 0x43, // f32.const
		0x2A, 0x38, // f32.load, f32.store
		0x5B, 0x5C, 0x5D, 0x5E, 0x5F, 0x60, // f32 comparisons
		0x8B, 0x8C, 0x8D, 0x8E, 0x8F, // f32 abs/neg/ceil/floor/trunc
		0x90, 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97, 0x98, // f32 ops
		0xB2, 0xB3, 0xB4, 0xB5, // i32/i64 -> f32 conversions
		0xA8, 0xA9, // f32 -> i32/i64 truncations
		0xB6, 0xBE: // f64 -> f32, reinterpret
		return true
	}
	return false
}

// isFloat64Opcode returns true if opcode is a float64 operation
func (v *FPValidator) isFloat64Opcode(op byte) bool {
	switch op {
	case 0x44, // f64.const
		0x2B, 0x39, // f64.load, f64.store
		0x61, 0x62, 0x63, 0x64, 0x65, 0x66, // f64 comparisons
		0x99, 0x9A, 0x9B, 0x9C, 0x9D, 0x9E, 0x9F, // f64 abs/neg/ceil/floor/trunc/nearest/sqrt
		0xA0, 0xA1, 0xA2, 0xA3, 0xA4, 0xA5, 0xA6, // f64 add/sub/mul/div/min/max/copysign
		0xB7, 0xB8, 0xB9, 0xBA, 0xBB, // i32/i64 -> f64 conversions
		0xAA, 0xAB, 0xAC, 0xAD, 0xAE, 0xAF, // f64 -> i32/i64 truncations
		0xBC, 0xBF: // f32 -> f64, reinterpret
		return true
	}
	return false
}

// skipInstructionOperands skips instruction operands
func (v *FPValidator) skipInstructionOperands(reader *bytes.Reader, opcode byte) error {
	switch opcode {
	case 0x0b: // end
		return nil
	case 0x0c: // else
		return nil
	case 0x0d: // return
		return nil
	case 0x0e: // unreachable
		return nil
	case 0x0f: // nop
		return nil
	case 0x10: // block
		_, err := reader.ReadByte() // block type
		return err
	case 0x11: // loop
		_, err := reader.ReadByte() // block type
		return err
	case 0x12: // if
		_, err := reader.ReadByte() // block type
		return err
	case 0x20: // local.get
		_, err := v.readVarUint32(reader)
		return err
	case 0x21: // local.set
		_, err := v.readVarUint32(reader)
		return err
	case 0x22: // local.tee
		_, err := v.readVarUint32(reader)
		return err
	case 0x23: // global.get
		_, err := v.readVarUint32(reader)
		return err
	case 0x24: // global.set
		_, err := v.readVarUint32(reader)
		return err
	case 0x28: // i32.load
		_, err := v.readVarUint32(reader) // align
		if err != nil {
			return err
		}
		_, err = v.readVarUint32(reader) // offset
		return err
	case 0x29: // i64.load
		_, err := v.readVarUint32(reader) // align
		if err != nil {
			return err
		}
		_, err = v.readVarUint32(reader) // offset
		return err
	case 0x2a: // f32.load
		_, err := v.readVarUint32(reader) // align
		if err != nil {
			return err
		}
		_, err = v.readVarUint32(reader) // offset
		return err
	case 0x2b: // f64.load
		_, err := v.readVarUint32(reader) // align
		if err != nil {
			return err
		}
		_, err = v.readVarUint32(reader) // offset
		return err
	case 0x36: // i32.store
		_, err := v.readVarUint32(reader) // align
		if err != nil {
			return err
		}
		_, err = v.readVarUint32(reader) // offset
		return err
	case 0x37: // i64.store
		_, err := v.readVarUint32(reader) // align
		if err != nil {
			return err
		}
		_, err = v.readVarUint32(reader) // offset
		return err
	case 0x38: // f32.store
		_, err := v.readVarUint32(reader) // align
		if err != nil {
			return err
		}
		_, err = v.readVarUint32(reader) // offset
		return err
	case 0x39: // f64.store
		_, err := v.readVarUint32(reader) // align
		if err != nil {
			return err
		}
		_, err = v.readVarUint32(reader) // offset
		return err
	case 0x3a: // memory.size
		_, err := reader.ReadByte() // memory index
		return err
	case 0x3b: // memory.grow
		_, err := reader.ReadByte() // memory index
		return err
	case 0x41: // i32.const
		_, err := v.readVarInt32(reader)
		return err
	case 0x42: // i64.const
		_, err := v.readVarInt64(reader)
		return err
	case 0x43: // f32.const
		_, err := reader.Read(make([]byte, 4))
		return err
	case 0x44: // f64.const
		_, err := reader.Read(make([]byte, 8))
		return err
	case 0xfc: // Prefix for extended instructions
		// Read extended opcode
		extOpcode, err := v.readVarUint32(reader)
		if err != nil {
			return err
		}

		// Handle specific extended instructions
		switch extOpcode {
		case 0x00: // i32.trunc_sat_f32_s
			return nil
		case 0x01: // i32.trunc_sat_f32_u
			return nil
		case 0x02: // i32.trunc_sat_f64_s
			return nil
		case 0x03: // i32.trunc_sat_f64_u
			return nil
		case 0x04: // i64.trunc_sat_f32_s
			return nil
		case 0x05: // i64.trunc_sat_f32_u
			return nil
		case 0x06: // i64.trunc_sat_f64_s
			return nil
		case 0x07: // i64.trunc_sat_f64_u
			return nil
		case 0x08: // memory.init
			_, err := reader.ReadByte() // memory index
			if err != nil {
				return err
			}
			_, err = reader.ReadByte() // data index
			return err
		case 0x09: // data.drop
			_, err := reader.ReadByte() // data index
			return err
		case 0x0a: // memory.copy
			_, err := reader.ReadByte() // dest memory index
			if err != nil {
				return err
			}
			_, err = reader.ReadByte() // src memory index
			return err
		case 0x0b: // memory.fill
			_, err := reader.ReadByte() // memory index
			return err
		case 0x0c: // table.init
			_, err := reader.ReadByte() // table index
			if err != nil {
				return err
			}
			_, err = reader.ReadByte() // element index
			return err
		case 0x0d: // elem.drop
			_, err := reader.ReadByte() // element index
			return err
		case 0x0e: // table.copy
			_, err := reader.ReadByte() // dest table index
			if err != nil {
				return err
			}
			_, err = reader.ReadByte() // src table index
			return err
		case 0x0f: // table.grow
			_, err := reader.ReadByte() // table index
			return err
		case 0x10: // table.size
			_, err := reader.ReadByte() // table index
			return err
		case 0x11: // table.fill
			_, err := reader.ReadByte() // table index
			return err
		}
	}

	return nil
}

// readVarInt32 reads a variable-length signed 32-bit integer
func (v *FPValidator) readVarInt32(reader *bytes.Reader) (int32, error) {
	var result uint32
	var shift uint

	for {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}

		result |= uint32(b&0x7f) << shift

		if b&0x80 == 0 {
			break
		}

		shift += 7
		if shift >= 32 {
			return 0, fmt.Errorf("varint32 too long")
		}
	}

	// Sign extend - for 32-bit values, we don't need to extend beyond 32 bits
	// The cast to int32 will handle the sign extension

	return int32(result), nil
}

// readVarInt64 reads a variable-length signed 64-bit integer
func (v *FPValidator) readVarInt64(reader *bytes.Reader) (int64, error) {
	var result uint64
	var shift uint

	for {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}

		result |= uint64(b&0x7f) << shift

		if b&0x80 == 0 {
			break
		}

		shift += 7
		if shift >= 64 {
			return 0, fmt.Errorf("varint64 too long")
		}
	}

	// Sign extend - for 64-bit values, the cast to int64 will handle the sign extension

	return int64(result), nil
}

// checkNonDeterministicOps checks for non-deterministic operations in strict mode
func (v *FPValidator) checkNonDeterministicOps(module *WASMModule) error {
	// In strict mode, we check for operations that might be non-deterministic
	// Check for time-related operations, random number generation, etc.

	for _, section := range module.Sections {
		if section.ID == 10 { // Code section
			if err := v.checkCodeSectionForNonDeterminism(section.Data); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkCodeSectionForNonDeterminism checks code section for non-deterministic operations
func (v *FPValidator) checkCodeSectionForNonDeterminism(data []byte) error {
	reader := bytes.NewReader(data)

	// Read number of functions
	numFunctions, err := v.readVarUint32(reader)
	if err != nil {
		return err
	}

	// Check each function for non-deterministic operations
	for i := uint32(0); i < numFunctions; i++ {
		if err := v.checkFunctionForNonDeterminism(reader); err != nil {
			return fmt.Errorf("function %d: %w", i, err)
		}
	}

	return nil
}

// checkFunctionForNonDeterminism checks a function for non-deterministic operations
func (v *FPValidator) checkFunctionForNonDeterminism(reader *bytes.Reader) error {
	// Read function body size
	bodySize, err := v.readVarUint32(reader)
	if err != nil {
		return err
	}

	// Read local variables
	numLocals, err := v.readVarUint32(reader)
	if err != nil {
		return err
	}

	// Skip local variable declarations
	for i := uint32(0); i < numLocals; i++ {
		if _, err := v.readVarUint32(reader); err != nil { // count
			return err
		}
		if _, err := reader.ReadByte(); err != nil { // type
			return err
		}
	}

	// Check instructions for non-deterministic operations
	startPos := reader.Size() - int64(reader.Len())
	endPos := startPos + int64(bodySize)

	for reader.Size()-int64(reader.Len()) < endPos {
		if err := v.checkInstructionForNonDeterminism(reader); err != nil {
			return err
		}
	}

	return nil
}

// checkInstructionForNonDeterminism checks an instruction for non-deterministic operations
func (v *FPValidator) checkInstructionForNonDeterminism(reader *bytes.Reader) error {
	opcode, err := reader.ReadByte()
	if err != nil {
		return err
	}

	// Check for non-deterministic operations
	// These are operations that might produce different results on different runs
	switch opcode {
	case 0xfc: // Extended instructions
		extOpcode, err := v.readVarUint32(reader)
		if err != nil {
			return err
		}

		// Check for non-deterministic extended instructions
		switch extOpcode {
		case 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07: // Truncation operations
			// These can be non-deterministic due to rounding differences
			return fmt.Errorf("non-deterministic truncation operation detected")
		}
	}

	// Skip instruction operands
	if err := v.skipInstructionOperands(reader, opcode); err != nil {
		return err
	}

	return nil
}

// ValidateNoFloatingPoint validates that an artifact contains no floating-point operations
// This enforces a deterministic policy by disallowing FP operations
func ValidateNoFloatingPoint(art *ArtifactInfo) error {
	switch strings.ToLower(art.Format) {
	case "wasm":
		return validateWASMNoFloat(art)
	case "elf":
		return validateELFNoFloat(art)
	default:
		// For shell/script artifacts, rely on VM config (no FP)
		return nil
	}
}

// validateWASMNoFloat checks WASM artifacts for floating-point opcodes
func validateWASMNoFloat(art *ArtifactInfo) error {
	// Read the WASM file and check for float opcodes
	data, err := readArtifactData(art)
	if err != nil {
		return err
	}

	// Parse WASM and check for float opcodes
	module, err := parseWASM(data)
	if err != nil {
		return err
	}

	// Check for floating-point opcodes in code section
	for _, section := range module.Sections {
		if section.ID == 10 { // Code section
			if err := checkWASMFloatOpcodes(section.Data); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateELFNoFloat checks ELF artifacts for floating-point instructions
func validateELFNoFloat(art *ArtifactInfo) error {
	// Read ELF file and check for floating-point instructions
	data, err := readArtifactData(art)
	if err != nil {
		return err
	}

	// Check ELF magic number
	if len(data) < 4 || string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("not a valid ELF file")
	}

	// Check for common FP instruction patterns in x86/x64
	if containsFloatInstructions(data) {
		return ErrFloatingPointDisallowed
	}

	// Check for FP-related sections in ELF
	if containsFloatSections(data) {
		return ErrFloatingPointDisallowed
	}

	return nil
}

// containsFloatSections checks for floating-point related sections in ELF
func containsFloatSections(data []byte) bool {
	// Look for common FP-related section names in ELF
	fpSectionNames := []string{
		".fini",   // Finalization code (might contain FP)
		".init",   // Initialization code (might contain FP)
		".plt",    // Procedure linkage table (might contain FP calls)
		".got",    // Global offset table (might contain FP addresses)
		".rodata", // Read-only data (might contain FP constants)
		".data",   // Data section (might contain FP data)
		".bss",    // Uninitialized data (might contain FP variables)
	}

	// Convert data to string for searching
	dataStr := string(data)

	for _, sectionName := range fpSectionNames {
		if strings.Contains(dataStr, sectionName) {
			return true
		}
	}

	return false
}

// checkWASMFloatOpcodes checks for floating-point opcodes in WASM code
func checkWASMFloatOpcodes(codeData []byte) error {
	// WASM float opcodes range samples:
	// f32.add(0x92) f32.sub(0x93) f64.add(0xA0) etc.
	for _, op := range codeData {
		if isFloatOpcode(op) {
			return ErrFloatingPointDisallowed
		}
	}
	return nil
}

// isFloatOpcode checks if a WASM opcode is a floating-point operation
func isFloatOpcode(op byte) bool {
	// WASM float opcodes range samples:
	// f32.add(0x92) f32.sub(0x93) f64.add(0xA0) etc.
	// Comprehensive set for production use
	switch op {
	// f32 operations
	case 0x92, 0x93, 0x94, 0x95, // f32 add/sub/mul/div
		0x96, 0x99, // f32 sqrt/copysign
		0x9A, 0x9B, 0x9C, 0x9D, // f32 abs/neg/ceil/floor
		0x9E, 0x9F, // f32 trunc/nearest

		// f64 operations
		0xA0, 0xA1, 0xA2, 0xA3, // f64 add/sub/mul/div
		0xA4, 0xA5, 0xA6, 0xA7, // f64 sqrt/min/max/copysign
		0xA8, 0xA9, 0xAA, 0xAB, // f64 abs/neg/ceil/floor
		0xAC, 0xAD, // f64 trunc/nearest

		// f32/f64 conversions
		0xB2, 0xB3, 0xB4, 0xB5, // f32 convert operations
		0xB6, 0xB7, 0xB8, 0xB9, // f64 convert operations
		0xBA, 0xBB, 0xBC, 0xBD, // f32/f64 demote/promote
		0xBE, 0xBF, // f32/f64 reinterpret

		// f32/f64 constants and memory operations
		0x43, 0x44, // f32/f64 const
		0x8B, 0x8C, // f32/f64 load
		0x97, 0x98: // f32/f64 store
		return true
	}
	return false
}

// containsFloatInstructions checks for floating-point instructions in binary data
func containsFloatInstructions(data []byte) bool {
	// Look for common x86/x64 FP instruction patterns
	fpPatterns := [][]byte{
		// x87 FPU instructions
		{0xD8, 0x00}, // fadd
		{0xD8, 0x28}, // fsub
		{0xD8, 0x08}, // fmul
		{0xD8, 0x38}, // fdiv
		{0xD9, 0x00}, // fld
		{0xDD, 0x18}, // fstp
		{0xD9, 0xE8}, // fld1
		{0xD9, 0xE9}, // fldl2t
		{0xD9, 0xEA}, // fldl2e
		{0xD9, 0xEB}, // fldpi
		{0xD9, 0xEC}, // fldlg2
		{0xD9, 0xED}, // fldln2
		{0xD9, 0xEE}, // fldz
		{0xD9, 0xF0}, // f2xm1
		{0xD9, 0xF1}, // fyl2x
		{0xD9, 0xF2}, // fptan
		{0xD9, 0xF3}, // fpatan
		{0xD9, 0xF4}, // fxtract
		{0xD9, 0xF5}, // fprem1
		{0xD9, 0xF6}, // fdecstp
		{0xD9, 0xF7}, // fincstp
		{0xD9, 0xF8}, // fprem
		{0xD9, 0xF9}, // fyl2xp1
		{0xD9, 0xFA}, // fsqrt
		{0xD9, 0xFB}, // fsincos
		{0xD9, 0xFC}, // frndint
		{0xD9, 0xFD}, // fscale
		{0xD9, 0xFE}, // fsin
		{0xD9, 0xFF}, // fcos

		// SSE/SSE2 instructions
		{0xF3, 0x0F, 0x58}, // addss
		{0xF3, 0x0F, 0x5C}, // subss
		{0xF3, 0x0F, 0x59}, // mulss
		{0xF3, 0x0F, 0x5E}, // divss
		{0xF2, 0x0F, 0x58}, // addsd
		{0xF2, 0x0F, 0x5C}, // subsd
		{0xF2, 0x0F, 0x59}, // mulsd
		{0xF2, 0x0F, 0x5E}, // divsd

		// AVX instructions
		{0xC5, 0xF8, 0x58}, // vaddps
		{0xC5, 0xF8, 0x5C}, // vsubps
		{0xC5, 0xF8, 0x59}, // vmulps
		{0xC5, 0xF8, 0x5E}, // vdivps
		{0xC5, 0xF9, 0x58}, // vaddpd
		{0xC5, 0xF9, 0x5C}, // vsubpd
		{0xC5, 0xF9, 0x59}, // vmulpd
		{0xC5, 0xF9, 0x5E}, // vdivpd
	}

	for _, pattern := range fpPatterns {
		if bytes.Contains(data, pattern) {
			return true
		}
	}
	return false
}

// readArtifactData reads the artifact data from disk
func readArtifactData(art *ArtifactInfo) ([]byte, error) {
	// Read the actual file data from the artifact path
	if art.Path == "" {
		return nil, fmt.Errorf("artifact path is empty")
	}

	data, err := os.ReadFile(art.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact file %s: %w", art.Path, err)
	}

	return data, nil
}

// parseWASM parses WASM binary data into a module structure
func parseWASM(data []byte) (*WASMModule, error) {
	// Use the existing WASM parser from the FPValidator
	validator := NewFPValidator(false, false, false) // Create validator for parsing
	return validator.parseWASMModule(data)
}
