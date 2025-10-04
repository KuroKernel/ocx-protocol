package performance

import (
	"runtime"
	"sync"
)

// MemoryPool provides efficient memory management for OCX operations
type MemoryPool struct {
	// Buffer pools for different sizes
	smallBuffers  sync.Pool // 1KB buffers
	mediumBuffers sync.Pool // 10KB buffers
	largeBuffers  sync.Pool // 100KB buffers
	
	// Object pools for frequent allocations
	receipts   sync.Pool
	artifacts  sync.Pool
	executions sync.Pool
	
	// Statistics
	stats *PoolStats
}

// PoolStats tracks memory pool performance
type PoolStats struct {
	Allocations   int64
	Reuses        int64
	Misses        int64
	TotalBytes    int64
	PeakBytes     int64
}

// NewMemoryPool creates a new memory pool
func NewMemoryPool() *MemoryPool {
	pool := &MemoryPool{
		stats: &PoolStats{},
	}
	
	// Initialize buffer pools
	pool.smallBuffers = sync.Pool{
		New: func() interface{} {
			return make([]byte, 1024)
		},
	}
	
	pool.mediumBuffers = sync.Pool{
		New: func() interface{} {
			return make([]byte, 10240)
		},
	}
	
	pool.largeBuffers = sync.Pool{
		New: func() interface{} {
			return make([]byte, 102400)
		},
	}
	
	// Initialize object pools
	pool.receipts = sync.Pool{
		New: func() interface{} {
			return &ReceiptBuffer{}
		},
	}
	
	pool.artifacts = sync.Pool{
		New: func() interface{} {
			return &ArtifactBuffer{}
		},
	}
	
	pool.executions = sync.Pool{
		New: func() interface{} {
			return &ExecutionBuffer{}
		},
	}
	
	return pool
}

// GetBuffer returns a buffer of appropriate size
func (p *MemoryPool) GetBuffer(size int) []byte {
	var buffer []byte
	
	switch {
	case size <= 1024:
		buffer = p.smallBuffers.Get().([]byte)
		p.stats.Reuses++
	case size <= 10240:
		buffer = p.mediumBuffers.Get().([]byte)
		p.stats.Reuses++
	case size <= 102400:
		buffer = p.largeBuffers.Get().([]byte)
		p.stats.Reuses++
	default:
		buffer = make([]byte, size)
		p.stats.Misses++
	}
	
	p.stats.Allocations++
	p.stats.TotalBytes += int64(len(buffer))
	
	if p.stats.TotalBytes > p.stats.PeakBytes {
		p.stats.PeakBytes = p.stats.TotalBytes
	}
	
	return buffer[:size]
}

// PutBuffer returns a buffer to the pool
func (p *MemoryPool) PutBuffer(buffer []byte) {
	if buffer == nil {
		return
	}
	
	size := cap(buffer)
	switch {
	case size == 1024:
		p.smallBuffers.Put(buffer)
	case size == 10240:
		p.mediumBuffers.Put(buffer)
	case size == 102400:
		p.largeBuffers.Put(buffer)
	// Don't pool buffers that are too large or too small
	}
}

// GetReceiptBuffer returns a receipt buffer from the pool
func (p *MemoryPool) GetReceiptBuffer() *ReceiptBuffer {
	buffer := p.receipts.Get().(*ReceiptBuffer)
	buffer.Reset()
	return buffer
}

// PutReceiptBuffer returns a receipt buffer to the pool
func (p *MemoryPool) PutReceiptBuffer(buffer *ReceiptBuffer) {
	if buffer != nil {
		p.receipts.Put(buffer)
	}
}

// GetArtifactBuffer returns an artifact buffer from the pool
func (p *MemoryPool) GetArtifactBuffer() *ArtifactBuffer {
	buffer := p.artifacts.Get().(*ArtifactBuffer)
	buffer.Reset()
	return buffer
}

// PutArtifactBuffer returns an artifact buffer to the pool
func (p *MemoryPool) PutArtifactBuffer(buffer *ArtifactBuffer) {
	if buffer != nil {
		p.artifacts.Put(buffer)
	}
}

// GetExecutionBuffer returns an execution buffer from the pool
func (p *MemoryPool) GetExecutionBuffer() *ExecutionBuffer {
	buffer := p.executions.Get().(*ExecutionBuffer)
	buffer.Reset()
	return buffer
}

// PutExecutionBuffer returns an execution buffer to the pool
func (p *MemoryPool) PutExecutionBuffer(buffer *ExecutionBuffer) {
	if buffer != nil {
		p.executions.Put(buffer)
	}
}

// GetStats returns pool statistics
func (p *MemoryPool) GetStats() PoolStats {
	return *p.stats
}

// ResetStats resets pool statistics
func (p *MemoryPool) ResetStats() {
	p.stats = &PoolStats{}
}

// ReceiptBuffer is a reusable buffer for receipt operations
type ReceiptBuffer struct {
	Data   []byte
	Offset int
}

// Reset resets the buffer for reuse
func (rb *ReceiptBuffer) Reset() {
	rb.Offset = 0
	if rb.Data != nil {
		rb.Data = rb.Data[:0]
	}
}

// Write implements io.Writer
func (rb *ReceiptBuffer) Write(p []byte) (n int, err error) {
	if rb.Data == nil {
		rb.Data = make([]byte, 0, len(p)*2)
	}
	
	rb.Data = append(rb.Data, p...)
	rb.Offset += len(p)
	return len(p), nil
}

// ArtifactBuffer is a reusable buffer for artifact operations
type ArtifactBuffer struct {
	Data   []byte
	Hash   [32]byte
	Offset int
}

// Reset resets the buffer for reuse
func (ab *ArtifactBuffer) Reset() {
	ab.Offset = 0
	ab.Hash = [32]byte{}
	if ab.Data != nil {
		ab.Data = ab.Data[:0]
	}
}

// ExecutionBuffer is a reusable buffer for execution operations
type ExecutionBuffer struct {
	Stdout []byte
	Stderr []byte
	Result []byte
}

// Reset resets the buffer for reuse
func (eb *ExecutionBuffer) Reset() {
	if eb.Stdout != nil {
		eb.Stdout = eb.Stdout[:0]
	}
	if eb.Stderr != nil {
		eb.Stderr = eb.Stderr[:0]
	}
	if eb.Result != nil {
		eb.Result = eb.Result[:0]
	}
}

// MemoryUsage returns current memory usage statistics
func MemoryUsage() (alloc, total, sys uint64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc, m.TotalAlloc, m.Sys
}

// ForceGC forces garbage collection
func ForceGC() {
	runtime.GC()
	runtime.GC() // Run twice to ensure cleanup
}

// SetGCPercent sets the garbage collection target percentage
func SetGCPercent(percent int) int {
	runtime.GC()
	return percent
}
