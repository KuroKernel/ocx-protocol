package deterministicvm

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool manages parallel execution of WASM/script artifacts
// Provides bounded concurrency with work stealing for optimal throughput
type WorkerPool struct {
	workers    int
	vm         VM
	jobQueue   chan *ExecutionJob
	resultPool sync.Pool
	running    int32
	shutdown   chan struct{}
	wg         sync.WaitGroup
	metrics    *PoolMetrics
}

// ExecutionJob represents a single execution request
type ExecutionJob struct {
	ID       string
	Config   VMConfig
	Result   chan *ExecutionJobResult
	ctx      context.Context
	priority int // Higher = more urgent
}

// ExecutionJobResult contains the result of an execution job
type ExecutionJobResult struct {
	ID       string
	Result   *ExecutionResult
	Error    error
	Duration time.Duration
}

// PoolMetrics contains pool performance metrics
type PoolMetrics struct {
	JobsSubmitted   int64
	JobsCompleted   int64
	JobsFailed      int64
	TotalLatency    int64 // nanoseconds
	ActiveWorkers   int32
	QueueDepth      int32
}

// WorkerPoolConfig contains configuration for WorkerPool
type WorkerPoolConfig struct {
	Workers   int    // Number of workers (default: NumCPU)
	QueueSize int    // Job queue size (default: Workers * 10)
	VMType    VMType // VM type to use
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(cfg WorkerPoolConfig) (*WorkerPool, error) {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = cfg.Workers * 10
	}

	var vm VM
	switch cfg.VMType {
	case VMTypeWASM:
		vm = NewWASMEngine()
	case VMTypeOSProcess:
		vm = &OSProcessVM{}
	default:
		vm = &OSProcessVM{}
	}

	pool := &WorkerPool{
		workers:  cfg.Workers,
		vm:       vm,
		jobQueue: make(chan *ExecutionJob, cfg.QueueSize),
		resultPool: sync.Pool{
			New: func() interface{} {
				return &ExecutionJobResult{}
			},
		},
		shutdown: make(chan struct{}),
		metrics:  &PoolMetrics{},
	}

	// Start workers
	pool.start()

	return pool, nil
}

// start spawns all worker goroutines
func (p *WorkerPool) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker is a single worker goroutine
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.shutdown:
			return
		case job := <-p.jobQueue:
			if job == nil {
				return
			}
			atomic.AddInt32(&p.metrics.ActiveWorkers, 1)
			atomic.AddInt32(&p.metrics.QueueDepth, -1)

			p.executeJob(job)

			atomic.AddInt32(&p.metrics.ActiveWorkers, -1)
		}
	}
}

// executeJob runs a single job and sends result
func (p *WorkerPool) executeJob(job *ExecutionJob) {
	start := time.Now()

	// Execute with context
	result, err := p.vm.Run(job.ctx, job.Config)

	duration := time.Since(start)
	atomic.AddInt64(&p.metrics.TotalLatency, int64(duration))
	atomic.AddInt64(&p.metrics.JobsCompleted, 1)

	if err != nil {
		atomic.AddInt64(&p.metrics.JobsFailed, 1)
	}

	// Send result
	jobResult := &ExecutionJobResult{
		ID:       job.ID,
		Result:   result,
		Error:    err,
		Duration: duration,
	}

	select {
	case job.Result <- jobResult:
	case <-job.ctx.Done():
		// Context cancelled, result dropped
	}
}

// Submit submits a job to the pool
func (p *WorkerPool) Submit(ctx context.Context, id string, config VMConfig) (<-chan *ExecutionJobResult, error) {
	if atomic.LoadInt32(&p.running) == 0 {
		return nil, fmt.Errorf("pool is not running")
	}

	resultCh := make(chan *ExecutionJobResult, 1)
	job := &ExecutionJob{
		ID:     id,
		Config: config,
		Result: resultCh,
		ctx:    ctx,
	}

	select {
	case p.jobQueue <- job:
		atomic.AddInt64(&p.metrics.JobsSubmitted, 1)
		atomic.AddInt32(&p.metrics.QueueDepth, 1)
		return resultCh, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, fmt.Errorf("job queue is full")
	}
}

// SubmitBatch submits multiple jobs and returns when all complete
func (p *WorkerPool) SubmitBatch(ctx context.Context, configs []VMConfig) ([]*ExecutionJobResult, error) {
	results := make([]*ExecutionJobResult, len(configs))
	resultChans := make([]<-chan *ExecutionJobResult, len(configs))

	// Submit all jobs
	for i, cfg := range configs {
		id := fmt.Sprintf("batch-%d", i)
		ch, err := p.Submit(ctx, id, cfg)
		if err != nil {
			// Cancel remaining
			return nil, fmt.Errorf("failed to submit job %d: %w", i, err)
		}
		resultChans[i] = ch
	}

	// Collect results
	for i, ch := range resultChans {
		select {
		case result := <-ch:
			results[i] = result
		case <-ctx.Done():
			return results, ctx.Err()
		}
	}

	return results, nil
}

// SubmitAsync submits a job and returns immediately
// The returned channel will receive the result when complete
func (p *WorkerPool) SubmitAsync(ctx context.Context, config VMConfig) (<-chan *ExecutionJobResult, error) {
	id := fmt.Sprintf("async-%d", time.Now().UnixNano())
	return p.Submit(ctx, id, config)
}

// Shutdown gracefully shuts down the pool
func (p *WorkerPool) Shutdown(ctx context.Context) error {
	close(p.shutdown)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Metrics returns current pool metrics
func (p *WorkerPool) Metrics() PoolMetrics {
	return PoolMetrics{
		JobsSubmitted: atomic.LoadInt64(&p.metrics.JobsSubmitted),
		JobsCompleted: atomic.LoadInt64(&p.metrics.JobsCompleted),
		JobsFailed:    atomic.LoadInt64(&p.metrics.JobsFailed),
		TotalLatency:  atomic.LoadInt64(&p.metrics.TotalLatency),
		ActiveWorkers: atomic.LoadInt32(&p.metrics.ActiveWorkers),
		QueueDepth:    atomic.LoadInt32(&p.metrics.QueueDepth),
	}
}

// AvgLatency returns average job latency
func (p *WorkerPool) AvgLatency() time.Duration {
	completed := atomic.LoadInt64(&p.metrics.JobsCompleted)
	if completed == 0 {
		return 0
	}
	totalNs := atomic.LoadInt64(&p.metrics.TotalLatency)
	return time.Duration(totalNs / completed)
}

// Throughput returns jobs per second
func (p *WorkerPool) Throughput() float64 {
	completed := atomic.LoadInt64(&p.metrics.JobsCompleted)
	totalNs := atomic.LoadInt64(&p.metrics.TotalLatency)
	if totalNs == 0 {
		return 0
	}
	seconds := float64(totalNs) / float64(time.Second)
	return float64(completed) / seconds
}

// Start marks the pool as running
func (p *WorkerPool) Start() {
	atomic.StoreInt32(&p.running, 1)
}

// IsRunning returns true if pool is accepting jobs
func (p *WorkerPool) IsRunning() bool {
	return atomic.LoadInt32(&p.running) == 1
}

// QueueSize returns current queue depth
func (p *WorkerPool) QueueSize() int {
	return int(atomic.LoadInt32(&p.metrics.QueueDepth))
}

// ActiveWorkers returns number of workers currently executing
func (p *WorkerPool) ActiveWorkers() int {
	return int(atomic.LoadInt32(&p.metrics.ActiveWorkers))
}
