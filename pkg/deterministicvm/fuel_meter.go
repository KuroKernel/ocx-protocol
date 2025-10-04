package deterministicvm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// FuelMeter provides deterministic resource metering for WASM execution
type FuelMeter struct {
	limit     uint64
	used      uint64
	mu        sync.Mutex
	startTime time.Time
}

// NewFuelMeter creates a new fuel meter with the specified limit
func NewFuelMeter(limit uint64) *FuelMeter {
	return &FuelMeter{
		limit:     limit,
		used:      0,
		startTime: time.Now(),
	}
}

// Consume consumes the specified amount of fuel
func (f *FuelMeter) Consume(amount uint64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.used += amount
	if f.used > f.limit {
		return fmt.Errorf("fuel limit exceeded: %d > %d", f.used, f.limit)
	}
	
	return nil
}

// GetUsed returns the amount of fuel used
func (f *FuelMeter) GetUsed() uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.used
}

// GetRemaining returns the amount of fuel remaining
func (f *FuelMeter) GetRemaining() uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.used >= f.limit {
		return 0
	}
	return f.limit - f.used
}

// GetLimit returns the fuel limit
func (f *FuelMeter) GetLimit() uint64 {
	return f.limit
}

// Reset resets the fuel meter
func (f *FuelMeter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.used = 0
	f.startTime = time.Now()
}

// GetDuration returns the execution duration
func (f *FuelMeter) GetDuration() time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	return time.Since(f.startTime)
}

// FuelCost represents the cost of different WASM operations
type FuelCost struct {
	// Basic operations
	Const     uint64 // Constant operations
	LocalGet  uint64 // Local variable access
	LocalSet  uint64 // Local variable assignment
	GlobalGet uint64 // Global variable access
	GlobalSet uint64 // Global variable assignment
	
	// Memory operations
	Load      uint64 // Memory load
	Store     uint64 // Memory store
	MemoryGrow uint64 // Memory growth
	
	// Control flow
	Call      uint64 // Function call
	Return    uint64 // Function return
	Branch    uint64 // Branch operation
	BranchIf  uint64 // Conditional branch
	
	// Arithmetic operations
	Add       uint64 // Addition
	Sub       uint64 // Subtraction
	Mul       uint64 // Multiplication
	Div       uint64 // Division
	Rem       uint64 // Remainder
	
	// Comparison operations
	Eq        uint64 // Equality
	Ne        uint64 // Inequality
	Lt        uint64 // Less than
	Le        uint64 // Less than or equal
	Gt        uint64 // Greater than
	Ge        uint64 // Greater than or equal
	
	// Logical operations
	And       uint64 // Logical AND
	Or        uint64 // Logical OR
	Xor       uint64 // Logical XOR
	Not       uint64 // Logical NOT
	
	// Conversion operations
	Convert   uint64 // Type conversion
	
	// Other operations
	Drop      uint64 // Drop value
	Select    uint64 // Select value
	Unreachable uint64 // Unreachable instruction
}

// DefaultFuelCosts returns the default fuel costs for WASM operations
func DefaultFuelCosts() *FuelCost {
	return &FuelCost{
		// Basic operations
		Const:     1,
		LocalGet:  1,
		LocalSet:  1,
		GlobalGet: 1,
		GlobalSet: 1,
		
		// Memory operations
		Load:      2,
		Store:     2,
		MemoryGrow: 100,
		
		// Control flow
		Call:      5,
		Return:    1,
		Branch:    2,
		BranchIf:  2,
		
		// Arithmetic operations
		Add:       1,
		Sub:       1,
		Mul:       2,
		Div:       3,
		Rem:       3,
		
		// Comparison operations
		Eq:        1,
		Ne:        1,
		Lt:        1,
		Le:        1,
		Gt:        1,
		Ge:        1,
		
		// Logical operations
		And:       1,
		Or:        1,
		Xor:       1,
		Not:       1,
		
		// Conversion operations
		Convert:   2,
		
		// Other operations
		Drop:      1,
		Select:    1,
		Unreachable: 1,
	}
}

// FuelMeteredWASMEngine extends WASMEngine with fuel metering
type FuelMeteredWASMEngine struct {
	*WASMEngine
	fuelMeter *FuelMeter
	fuelCosts *FuelCost
}

// NewFuelMeteredWASMEngine creates a new fuel-metered WASM engine
func NewFuelMeteredWASMEngine(fuelLimit uint64) *FuelMeteredWASMEngine {
	return &FuelMeteredWASMEngine{
		WASMEngine: NewWASMEngine().(*WASMEngine),
		fuelMeter:  NewFuelMeter(fuelLimit),
		fuelCosts:  DefaultFuelCosts(),
	}
}

// Run executes a WASM module with fuel metering
func (f *FuelMeteredWASMEngine) Run(ctx context.Context, config VMConfig) (*ExecutionResult, error) {
	// Reset fuel meter for new execution
	f.fuelMeter.Reset()
	
	// Execute with fuel monitoring
	result, err := f.WASMEngine.Run(ctx, config)
	if err != nil {
		return nil, err
	}
	
	// Update result with fuel usage
	result.GasUsed = f.fuelMeter.GetUsed()
	
	// Check fuel limit
	if f.fuelMeter.GetUsed() > f.fuelMeter.GetLimit() {
		return nil, &ExecutionError{
			Code:    ErrorCodeCycleLimitExceeded,
			Message: fmt.Sprintf("Fuel limit exceeded: %d > %d", f.fuelMeter.GetUsed(), f.fuelMeter.GetLimit()),
			Context: map[string]interface{}{
				"fuel_used":  f.fuelMeter.GetUsed(),
				"fuel_limit": f.fuelMeter.GetLimit(),
			},
		}
	}
	
	return result, nil
}

// GetFuelMeter returns the fuel meter
func (f *FuelMeteredWASMEngine) GetFuelMeter() *FuelMeter {
	return f.fuelMeter
}

// GetFuelCosts returns the fuel costs
func (f *FuelMeteredWASMEngine) GetFuelCosts() *FuelCost {
	return f.fuelCosts
}

// SetFuelCosts sets the fuel costs
func (f *FuelMeteredWASMEngine) SetFuelCosts(costs *FuelCost) {
	f.fuelCosts = costs
}
