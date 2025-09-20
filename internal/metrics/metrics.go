package metrics

import (
    "sync"
    "time"
)

type Counter struct {
    mu    sync.RWMutex
    value int64
}

func (c *Counter) Inc() {
    c.mu.Lock()
    c.value++
    c.mu.Unlock()
}

func (c *Counter) Value() int64 {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.value
}

type Histogram struct {
    mu     sync.RWMutex
    values []float64
}

func (h *Histogram) Observe(value float64) {
    h.mu.Lock()
    h.values = append(h.values, value)
    h.mu.Unlock()
}

func (h *Histogram) P99() float64 {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    if len(h.values) == 0 {
        return 0
    }
    
    // Simple P99 calculation
    idx := int(float64(len(h.values)) * 0.99)
    if idx >= len(h.values) {
        idx = len(h.values) - 1
    }
    return h.values[idx]
}

var (
    ExecuteCounter   = &Counter{}
    VerifyCounter    = &Counter{}
    ExecuteLatency   = &Histogram{}
    VerifyLatency    = &Histogram{}
)

func RecordExecution(duration time.Duration) {
    ExecuteCounter.Inc()
    ExecuteLatency.Observe(duration.Seconds())
}

func RecordVerification(duration time.Duration) {
    VerifyCounter.Inc()
    VerifyLatency.Observe(duration.Seconds())
}
