package deterministicvm

import (
	"crypto/sha256"
	"encoding/binary"
	"io"
	"sync"
)

// DeterministicRNG provides deterministic random number generation
type DeterministicRNG struct {
	state        [32]byte
	initialState [32]byte // Store initial state for reset
	counter      uint64
	mu           sync.Mutex
}

// NewDeterministicRNG creates a new deterministic RNG with the given seed
func NewDeterministicRNG(seed []byte) *DeterministicRNG {
	rng := &DeterministicRNG{
		counter: 0,
	}
	
	// Initialize state from seed
	if len(seed) > 0 {
		hash := sha256.Sum256(seed)
		copy(rng.state[:], hash[:])
		copy(rng.initialState[:], hash[:])
	} else {
		// Default seed
		defaultSeed := []byte("ocx-deterministic-rng-default-seed")
		hash := sha256.Sum256(defaultSeed)
		copy(rng.state[:], hash[:])
		copy(rng.initialState[:], hash[:])
	}
	
	return rng
}

// NewDeterministicRNGFromExecution creates a deterministic RNG from execution context
func NewDeterministicRNGFromExecution(executionID [32]byte, timestamp uint64, artifactHash []byte) *DeterministicRNG {
	// Create seed from execution context
	h := sha256.New()
	h.Write(executionID[:])
	binary.Write(h, binary.BigEndian, timestamp)
	h.Write(artifactHash)
	
	seed := h.Sum(nil)
	return NewDeterministicRNG(seed)
}

// Read implements io.Reader interface
func (d *DeterministicRNG) Read(p []byte) (n int, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	n = len(p)
	for i := 0; i < n; i++ {
		p[i] = d.nextByte()
	}
	
	return n, nil
}

// nextByte generates the next byte
func (d *DeterministicRNG) nextByte() byte {
	// Use counter to generate deterministic output
	h := sha256.New()
	h.Write(d.state[:])
	binary.Write(h, binary.BigEndian, d.counter)
	
	output := h.Sum(nil)
	d.counter++
	
	// Update state for next call
	copy(d.state[:], output)
	
	return output[0]
}

// Uint32 generates a deterministic uint32
func (d *DeterministicRNG) Uint32() uint32 {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	h := sha256.New()
	h.Write(d.state[:])
	binary.Write(h, binary.BigEndian, d.counter)
	
	output := h.Sum(nil)
	d.counter++
	
	// Update state
	copy(d.state[:], output)
	
	return binary.BigEndian.Uint32(output[:4])
}

// Uint64 generates a deterministic uint64
func (d *DeterministicRNG) Uint64() uint64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	h := sha256.New()
	h.Write(d.state[:])
	binary.Write(h, binary.BigEndian, d.counter)
	
	output := h.Sum(nil)
	d.counter++
	
	// Update state
	copy(d.state[:], output)
	
	return binary.BigEndian.Uint64(output[:8])
}

// Int32 generates a deterministic int32
func (d *DeterministicRNG) Int32() int32 {
	return int32(d.Uint32())
}

// Int64 generates a deterministic int64
func (d *DeterministicRNG) Int64() int64 {
	return int64(d.Uint64())
}

// Float32 generates a deterministic float32 in range [0, 1)
func (d *DeterministicRNG) Float32() float32 {
	u := d.Uint32()
	return float32(u) / float32(^uint32(0))
}

// Float64 generates a deterministic float64 in range [0, 1)
func (d *DeterministicRNG) Float64() float64 {
	u := d.Uint64()
	return float64(u) / float64(^uint64(0))
}

// Intn generates a deterministic int in range [0, n)
func (d *DeterministicRNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	
	u := d.Uint32()
	return int(u % uint32(n))
}

// IntRange generates a deterministic int in range [min, max)
func (d *DeterministicRNG) IntRange(min, max int) int {
	if min >= max {
		return min
	}
	
	return min + d.Intn(max-min)
}

// Shuffle shuffles a slice deterministically
func (d *DeterministicRNG) Shuffle(n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		j := d.Intn(i + 1)
		swap(i, j)
	}
}

// Reset resets the RNG to its initial state
func (d *DeterministicRNG) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.counter = 0
	// Reset state to initial state
	copy(d.state[:], d.initialState[:])
}

// GetState returns the current state (for debugging)
func (d *DeterministicRNG) GetState() (state [32]byte, counter uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	return d.state, d.counter
}

// SetState sets the RNG state (for testing)
func (d *DeterministicRNG) SetState(state [32]byte, counter uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.state = state
	d.counter = counter
}

// RNGProvider provides deterministic RNG instances
type RNGProvider struct {
	mu sync.RWMutex
	rngs map[string]*DeterministicRNG
}

// NewRNGProvider creates a new RNG provider
func NewRNGProvider() *RNGProvider {
	return &RNGProvider{
		rngs: make(map[string]*DeterministicRNG),
	}
}

// GetRNG gets or creates an RNG for the given domain
func (p *RNGProvider) GetRNG(domain string, seed []byte) *DeterministicRNG {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if rng, exists := p.rngs[domain]; exists {
		return rng
	}
	
	// Create new RNG for this domain
	rng := NewDeterministicRNG(seed)
	p.rngs[domain] = rng
	
	return rng
}

// GetRNGFromExecution gets or creates an RNG from execution context
func (p *RNGProvider) GetRNGFromExecution(domain string, executionID [32]byte, timestamp uint64, artifactHash []byte) *DeterministicRNG {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if rng, exists := p.rngs[domain]; exists {
		return rng
	}
	
	// Create new RNG for this domain
	rng := NewDeterministicRNGFromExecution(executionID, timestamp, artifactHash)
	p.rngs[domain] = rng
	
	return rng
}

// Clear clears all RNGs
func (p *RNGProvider) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.rngs = make(map[string]*DeterministicRNG)
}

// GetDomains returns all RNG domains
func (p *RNGProvider) GetDomains() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	domains := make([]string, 0, len(p.rngs))
	for domain := range p.rngs {
		domains = append(domains, domain)
	}
	
	return domains
}

// RNGDomains defines common RNG domains
var RNGDomains = struct {
	Execution    string
	Memory       string
	Timing       string
	Network      string
	Filesystem   string
	Verification string
}{
	Execution:    "execution",
	Memory:       "memory",
	Timing:       "timing",
	Network:      "network",
	Filesystem:   "filesystem",
	Verification: "verification",
}

// MockRNG is a mock RNG for testing
type MockRNG struct {
	values []byte
	index  int
}

// NewMockRNG creates a mock RNG with predefined values
func NewMockRNG(values []byte) *MockRNG {
	return &MockRNG{
		values: values,
		index:  0,
	}
}

// Read implements io.Reader interface
func (m *MockRNG) Read(p []byte) (n int, err error) {
	n = len(p)
	for i := 0; i < n; i++ {
		if m.index >= len(m.values) {
			return i, io.EOF
		}
		p[i] = m.values[m.index]
		m.index++
	}
	return n, nil
}

// Uint32 generates a mock uint32
func (m *MockRNG) Uint32() uint32 {
	if m.index+4 > len(m.values) {
		return 0
	}
	
	val := binary.BigEndian.Uint32(m.values[m.index : m.index+4])
	m.index += 4
	return val
}

// Uint64 generates a mock uint64
func (m *MockRNG) Uint64() uint64 {
	if m.index+8 > len(m.values) {
		return 0
	}
	
	val := binary.BigEndian.Uint64(m.values[m.index : m.index+8])
	m.index += 8
	return val
}

// Int32 generates a mock int32
func (m *MockRNG) Int32() int32 {
	return int32(m.Uint32())
}

// Int64 generates a mock int64
func (m *MockRNG) Int64() int64 {
	return int64(m.Uint64())
}

// Float32 generates a mock float32
func (m *MockRNG) Float32() float32 {
	return float32(m.Uint32()) / float32(^uint32(0))
}

// Float64 generates a mock float64
func (m *MockRNG) Float64() float64 {
	return float64(m.Uint64()) / float64(^uint64(0))
}

// Intn generates a mock int in range [0, n)
func (m *MockRNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	
	u := m.Uint32()
	return int(u % uint32(n))
}

// IntRange generates a mock int in range [min, max)
func (m *MockRNG) IntRange(min, max int) int {
	if min >= max {
		return min
	}
	
	return min + m.Intn(max-min)
}

// Shuffle shuffles a slice using mock RNG
func (m *MockRNG) Shuffle(n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		j := m.Intn(i + 1)
		swap(i, j)
	}
}

// Reset resets the mock RNG
func (m *MockRNG) Reset() {
	m.index = 0
}

// GetState returns the current state
func (m *MockRNG) GetState() (index int) {
	return m.index
}

// SetState sets the RNG state
func (m *MockRNG) SetState(index int) {
	m.index = index
}
