package ops

import (
	"context"
	"sync"
)

// BackpressureGate limits concurrent processing to a fixed number of slots.
type BackpressureGate struct {
	slots chan struct{}
}

func NewBackpressureGate(limit int) *BackpressureGate {
	if limit <= 0 {
		limit = 8
	}
	return &BackpressureGate{slots: make(chan struct{}, limit)}
}

// Acquire blocks until a slot is available.
func (g *BackpressureGate) Acquire() { g.slots <- struct{}{} }

// Release returns one slot.
func (g *BackpressureGate) Release() { <-g.slots }

// TryAcquire acquires a slot without blocking.
func (g *BackpressureGate) TryAcquire() bool {
	select {
	case g.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

// Available returns how many slots are currently free.
func (g *BackpressureGate) Available() int { return cap(g.slots) - len(g.slots) }

// Run executes fn under a slot and releases it afterwards.
func (g *BackpressureGate) Run(fn func()) {
	g.Acquire()
	defer g.Release()
	fn()
}

// WaitGroupGate is a simpler cooperative gate using a WaitGroup.
type WaitGroupGate struct {
	mu sync.Mutex
	wg sync.WaitGroup
	n  int
}

func (g *WaitGroupGate) Inc() {
	g.mu.Lock()
	g.n++
	g.wg.Add(1)
	g.mu.Unlock()
}

func (g *WaitGroupGate) Done() {
	g.mu.Lock()
	g.n--
	g.wg.Done()
	g.mu.Unlock()
}

func (g *WaitGroupGate) Wait() { g.wg.Wait() }

func (g *WaitGroupGate) Count() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.n
}

// RunContext executes fn under a slot unless the context is already cancelled.
func (g *BackpressureGate) RunContext(ctx context.Context, fn func()) {
	if ctx.Err() != nil {
		return
	}
	g.Acquire()
	defer g.Release()
	fn()
}
