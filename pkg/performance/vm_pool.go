package performance

import (
	"context"
	"errors"
	"sync"
	"time"

	"ocx.local/pkg/deterministicvm"
)

// VMPool provides a pool of pre-created VM instances for high-performance execution
type VMPool struct {
	// Pool of available VMs
	vms chan *PooledVM
	
	// Configuration for new VMs
	config deterministicvm.VMConfig
	
	// Pool statistics
	stats *VMPoolStats
	
	// Pool management
	maxSize    int
	currentSize int
	mutex      sync.RWMutex
	
	// Context for pool lifecycle
	ctx    context.Context
	cancel context.CancelFunc
}

// PooledVM wraps a VM with pool metadata
type PooledVM struct {
	VM           *deterministicvm.OSProcessVM
	CreatedAt    time.Time
	LastUsedAt   time.Time
	UseCount     int64
	IsHealthy    bool
}

// VMPoolStats tracks pool performance
type VMPoolStats struct {
	TotalCreated    int64
	TotalDestroyed  int64
	TotalExecutions int64
	AverageWaitTime time.Duration
	PoolHits        int64
	PoolMisses      int64
}

// NewVMPool creates a new VM pool
func NewVMPool(config deterministicvm.VMConfig, maxSize int) *VMPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &VMPool{
		vms:      make(chan *PooledVM, maxSize),
		config:   config,
		stats:    &VMPoolStats{},
		maxSize:  maxSize,
		ctx:      ctx,
		cancel:   cancel,
	}
	
	// Pre-create some VMs for warm startup
	go pool.warmup()
	
	return pool
}

// GetVM returns a VM from the pool or creates a new one
func (p *VMPool) GetVM() (*PooledVM, error) {
	start := time.Now()
	
	select {
	case vm := <-p.vms:
		// Reuse existing VM
		vm.LastUsedAt = time.Now()
		vm.UseCount++
		p.stats.PoolHits++
		p.stats.AverageWaitTime = time.Since(start)
		return vm, nil
		
	case <-time.After(100 * time.Millisecond):
		// Create new VM if pool is empty
		return p.createNewVM()
		
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	}
}

// PutVM returns a VM to the pool
func (p *VMPool) PutVM(vm *PooledVM) {
	if vm == nil || !vm.IsHealthy {
		p.destroyVM(vm)
		return
	}
	
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if p.currentSize < p.maxSize {
		select {
		case p.vms <- vm:
			// VM returned to pool
		default:
			// Pool is full, destroy VM
			p.destroyVM(vm)
		}
	} else {
		// Pool is at capacity, destroy VM
		p.destroyVM(vm)
	}
}

// createNewVM creates a new VM instance
func (p *VMPool) createNewVM() (*PooledVM, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if p.currentSize >= p.maxSize {
		p.stats.PoolMisses++
		return nil, ErrPoolExhausted
	}
	
	// Create new VM
	vm := &deterministicvm.OSProcessVM{}
	
	pooledVM := &PooledVM{
		VM:           vm,
		CreatedAt:    time.Now(),
		LastUsedAt:   time.Now(),
		UseCount:     1,
		IsHealthy:    true,
	}
	
	p.currentSize++
	p.stats.TotalCreated++
	p.stats.PoolMisses++
	
	return pooledVM, nil
}

// destroyVM destroys a VM instance
func (p *VMPool) destroyVM(vm *PooledVM) {
	if vm == nil {
		return
	}
	
	// Clean up VM resources
	if vm.VM != nil {
		// VM cleanup would go here
	}
	
	p.mutex.Lock()
	p.currentSize--
	p.stats.TotalDestroyed++
	p.mutex.Unlock()
}

// warmup pre-creates VMs for better performance
func (p *VMPool) warmup() {
	// Pre-create 25% of max pool size
	warmupCount := p.maxSize / 4
	if warmupCount < 1 {
		warmupCount = 1
	}
	
	for i := 0; i < warmupCount; i++ {
		vm, err := p.createNewVM()
		if err != nil {
			break
		}
		
		select {
		case p.vms <- vm:
			// VM added to pool
		default:
			// Pool is full, destroy VM
			p.destroyVM(vm)
		}
	}
}

// Execute runs an artifact using a pooled VM
func (p *VMPool) Execute(ctx context.Context, artifactPath string) (*deterministicvm.ExecutionResult, error) {
	vm, err := p.GetVM()
	if err != nil {
		return nil, err
	}
	
	// Execute artifact
	result, err := vm.VM.Run(ctx, deterministicvm.VMConfig{
		ArtifactPath: artifactPath,
		WorkingDir:   p.config.WorkingDir,
		Env:          p.config.Env,
		Timeout:      p.config.Timeout,
		StrictMode:   p.config.StrictMode,
	})
	
	// Return VM to pool
	p.PutVM(vm)
	
	if err != nil {
		return nil, err
	}
	
	p.stats.TotalExecutions++
	return result, nil
}

// GetStats returns pool statistics
func (p *VMPool) GetStats() VMPoolStats {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	stats := *p.stats
	stats.AverageWaitTime = p.stats.AverageWaitTime
	return stats
}

// Close closes the VM pool and cleans up resources
func (p *VMPool) Close() error {
	p.cancel()
	
	// Drain and destroy all VMs
	for {
		select {
		case vm := <-p.vms:
			p.destroyVM(vm)
		default:
			return nil
		}
	}
}

// HealthCheck checks the health of VMs in the pool
func (p *VMPool) HealthCheck() error {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	// Check if pool is healthy
	if p.currentSize == 0 {
		return ErrPoolEmpty
	}
	
	return nil
}

// Resize changes the maximum pool size
func (p *VMPool) Resize(newMaxSize int) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if newMaxSize < 1 {
		return ErrInvalidPoolSize
	}
	
	oldMaxSize := p.maxSize
	p.maxSize = newMaxSize
	
	// If new size is smaller, destroy excess VMs
	if newMaxSize < oldMaxSize {
		excess := p.currentSize - newMaxSize
		for i := 0; i < excess; i++ {
			select {
			case vm := <-p.vms:
				p.destroyVM(vm)
			default:
				break
			}
		}
	}
	
	return nil
}

// Pool errors
var (
	ErrPoolExhausted   = errors.New("VM pool exhausted")
	ErrPoolEmpty       = errors.New("VM pool is empty")
	ErrInvalidPoolSize = errors.New("invalid pool size")
)

// DefaultVMPoolConfig returns a default VM pool configuration
func DefaultVMPoolConfig() deterministicvm.VMConfig {
	return deterministicvm.VMConfig{
		WorkingDir: "/tmp",
		Timeout:    30 * time.Second,
		StrictMode: false,
		Env: []string{
			"PATH=/usr/bin:/bin",
			"HOME=/tmp",
		},
	}
}
