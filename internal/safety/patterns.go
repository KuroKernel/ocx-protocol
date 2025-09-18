package safety

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// SafeLoop provides safe loop patterns with limits
type SafeLoop struct {
	maxIterations int
	timeout       time.Duration
	startTime     time.Time
	count         int
}

// NewSafeLoop creates a new safe loop
func NewSafeLoop(maxIterations int, timeout time.Duration) *SafeLoop {
	return &SafeLoop{
		maxIterations: maxIterations,
		timeout:       timeout,
		startTime:     time.Now(),
	}
}

// Next checks if the loop should continue
func (sl *SafeLoop) Next() bool {
	sl.count++
	if sl.count > sl.maxIterations {
		return false
	}
	if time.Since(sl.startTime) > sl.timeout {
		return false
	}
	return true
}

// GetCount returns the current iteration count
func (sl *SafeLoop) GetCount() int {
	return sl.count
}

// SafeSlice provides safe slice operations with capacity limits
type SafeSlice[T any] struct {
	data     []T
	capacity int
	mu       sync.RWMutex
}

// NewSafeSlice creates a new safe slice
func NewSafeSlice[T any](capacity int) *SafeSlice[T] {
	return &SafeSlice[T]{
		data:     make([]T, 0, capacity),
		capacity: capacity,
	}
}

// Append safely appends an item to the slice
func (ss *SafeSlice[T]) Append(item T) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	if len(ss.data) >= ss.capacity {
		return fmt.Errorf("capacity exceeded: %d", ss.capacity)
	}
	
	ss.data = append(ss.data, item)
	return nil
}

// Get safely retrieves an item from the slice
func (ss *SafeSlice[T]) Get(index int) (T, error) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	
	var zero T
	if index < 0 || index >= len(ss.data) {
		return zero, fmt.Errorf("index out of range: %d", index)
	}
	
	return ss.data[index], nil
}

// Length returns the current length of the slice
func (ss *SafeSlice[T]) Length() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return len(ss.data)
}

// SafeMap provides safe map operations with size limits
type SafeMap[K comparable, V any] struct {
	data    map[K]V
	maxSize int
	mu      sync.RWMutex
}

// NewSafeMap creates a new safe map
func NewSafeMap[K comparable, V any](maxSize int) *SafeMap[K, V] {
	return &SafeMap[K, V]{
		data:    make(map[K]V),
		maxSize: maxSize,
	}
}

// Set safely sets a key-value pair in the map
func (sm *SafeMap[K, V]) Set(key K, value V) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if len(sm.data) >= sm.maxSize {
		return fmt.Errorf("map size exceeded: %d", sm.maxSize)
	}
	
	sm.data[key] = value
	return nil
}

// Get safely retrieves a value from the map
func (sm *SafeMap[K, V]) Get(key K) (V, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	val, ok := sm.data[key]
	return val, ok
}

// Delete safely deletes a key from the map
func (sm *SafeMap[K, V]) Delete(key K) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	delete(sm.data, key)
}

// Keys returns all keys in the map
func (sm *SafeMap[K, V]) Keys() []K {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	keys := make([]K, 0, len(sm.data))
	for k := range sm.data {
		keys = append(keys, k)
	}
	return keys
}

// SafeRecursion provides safe recursion patterns
type SafeRecursion struct {
	maxDepth     int
	currentDepth int
	mu           sync.Mutex
}

// NewSafeRecursion creates a new safe recursion instance
func NewSafeRecursion(maxDepth int) *SafeRecursion {
	return &SafeRecursion{
		maxDepth: maxDepth,
	}
}

// Execute executes a recursive function safely
func (sr *SafeRecursion) Execute(fn func() error) error {
	sr.mu.Lock()
	if sr.currentDepth >= sr.maxDepth {
		sr.mu.Unlock()
		return fmt.Errorf("maximum recursion depth %d exceeded", sr.maxDepth)
	}
	sr.currentDepth++
	sr.mu.Unlock()
	
	defer func() {
		sr.mu.Lock()
		sr.currentDepth--
		sr.mu.Unlock()
	}()
	
	return fn()
}

// SafeChannel provides safe channel operations
type SafeChannel[T any] struct {
	ch     chan T
	mu     sync.RWMutex
	closed bool
}

// NewSafeChannel creates a new safe channel
func NewSafeChannel[T any](bufferSize int) *SafeChannel[T] {
	return &SafeChannel[T]{
		ch: make(chan T, bufferSize),
	}
}

// Send safely sends a value to the channel
func (sc *SafeChannel[T]) Send(value T) error {
	sc.mu.RLock()
	if sc.closed {
		sc.mu.RUnlock()
		return fmt.Errorf("cannot send to closed channel")
	}
	sc.mu.RUnlock()
	
	select {
	case sc.ch <- value:
		return nil
	case <-time.After(100 * time.Millisecond):
		return fmt.Errorf("send timed out")
	}
}

// Receive safely receives a value from the channel
func (sc *SafeChannel[T]) Receive() (T, error) {
	sc.mu.RLock()
	if sc.closed {
		sc.mu.RUnlock()
		var zero T
		return zero, fmt.Errorf("cannot receive from closed channel")
	}
	sc.mu.RUnlock()
	
	select {
	case value := <-sc.ch:
		return value, nil
	case <-time.After(100 * time.Millisecond):
		var zero T
		return zero, fmt.Errorf("receive timed out")
	}
}

// Close safely closes the channel
func (sc *SafeChannel[T]) Close() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	if !sc.closed {
		close(sc.ch)
		sc.closed = true
	}
}

// SafeExecution provides safe execution with timeout and memory limits
type SafeExecution struct {
	timeout          time.Duration
	maxHeapAllocation uint64
	config           *SafetyConfig
}

// NewSafeExecution creates a new safe execution instance
func NewSafeExecution(timeout time.Duration, maxHeapAllocation uint64) *SafeExecution {
	return &SafeExecution{
		timeout:          timeout,
		maxHeapAllocation: maxHeapAllocation,
		config:           DefaultSafetyConfig(),
	}
}

// Execute executes a function safely with timeout and memory limits
func (se *SafeExecution) Execute(fn func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), se.timeout)
	defer cancel()
	
	// Start memory monitoring
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	initialHeap := memStats.HeapAlloc
	
	// Create a channel to receive the result
	resultChan := make(chan error, 1)
	
	// Execute the function in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- fmt.Errorf("panic recovered: %v", r)
			}
		}()
		
		// Check memory usage during execution
		runtime.ReadMemStats(&memStats)
		allocated := memStats.HeapAlloc - initialHeap
		if allocated > uint64(se.maxHeapAllocation) {
			resultChan <- fmt.Errorf("heap allocation exceeded: %d bytes", allocated)
			return
		}
		
		resultChan <- fn()
	}()
	
	// Wait for completion or timeout
	select {
	case err := <-resultChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("execution timed out after %v", se.timeout)
	}
}

// SafeContext provides safe context operations
type SafeContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewSafeContext creates a new safe context with timeout
func NewSafeContext(timeout time.Duration) *SafeContext {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &SafeContext{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Done returns the context's done channel
func (sc *SafeContext) Done() <-chan struct{} {
	return sc.ctx.Done()
}

// Err returns the context's error
func (sc *SafeContext) Err() error {
	return sc.ctx.Err()
}

// Cancel cancels the context
func (sc *SafeContext) Cancel() {
	sc.cancel()
}

// Close cancels the context (alias for Cancel)
func (sc *SafeContext) Close() {
	sc.cancel()
}

// SafeWorkerPool provides safe worker pool operations
type SafeWorkerPool struct {
	workers    int
	jobQueue   chan func()
	workerPool chan chan func()
	quit       chan bool
	wg         sync.WaitGroup
}

// NewSafeWorkerPool creates a new safe worker pool
func NewSafeWorkerPool(workers, queueSize int) *SafeWorkerPool {
	return &SafeWorkerPool{
		workers:    workers,
		jobQueue:   make(chan func(), queueSize),
		workerPool: make(chan chan func(), workers),
		quit:       make(chan bool),
	}
}

// Start starts the worker pool
func (swp *SafeWorkerPool) Start() {
	for i := 0; i < swp.workers; i++ {
		worker := NewSafeWorker(swp.workerPool, swp.jobQueue, swp.quit)
		swp.wg.Add(1)
		go worker.Start(&swp.wg)
	}
}

// Stop stops the worker pool
func (swp *SafeWorkerPool) Stop() {
	close(swp.quit)
	swp.wg.Wait()
}

// Submit submits a job to the worker pool
func (swp *SafeWorkerPool) Submit(job func()) error {
	select {
	case swp.jobQueue <- job:
		return nil
	case <-time.After(100 * time.Millisecond):
		return fmt.Errorf("job queue is full")
	}
}

// SafeWorker represents a single worker in the pool
type SafeWorker struct {
	workerPool chan chan func()
	jobChannel chan func()
	quit       chan bool
}

// NewSafeWorker creates a new safe worker
func NewSafeWorker(workerPool chan chan func(), jobQueue chan func(), quit chan bool) *SafeWorker {
	return &SafeWorker{
		workerPool: workerPool,
		jobChannel: make(chan func()),
		quit:       quit,
	}
}

// Start starts the worker
func (sw *SafeWorker) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		sw.workerPool <- sw.jobChannel
		select {
		case job := <-sw.jobChannel:
			job()
		case <-sw.quit:
			return
		}
	}
}
