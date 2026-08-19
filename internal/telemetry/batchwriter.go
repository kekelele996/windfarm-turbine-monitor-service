package telemetry

import (
	"sync"
	"time"
)

// BatchWriter buffers samples and flushes them to a sink in batches, keeping
// the hot path cheap.
type BatchWriter struct {
	mu      sync.Mutex
	pending []Sample
	limit   int
	flushFn func([]Sample) error
}

func NewBatchWriter(limit int, flushFn func([]Sample) error) *BatchWriter {
	if limit <= 0 {
		limit = 100
	}
	return &BatchWriter{limit: limit, flushFn: flushFn}
}

// Add buffers one sample and flushes when the batch limit is reached.
func (w *BatchWriter) Add(s Sample) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pending = append(w.pending, s)
	if len(w.pending) >= w.limit {
		return w.flush()
	}
	return nil
}

// Flush writes every pending sample to the sink.
func (w *BatchWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flush()
}

// flush writes every pending sample to the sink. If the sink fails the batch
// is put back at the front of the pending buffer so it can be retried on the
// next flush instead of being dropped.
func (w *BatchWriter) flush() error {
	if len(w.pending) == 0 {
		return nil
	}
	batch := w.pending
	w.pending = nil
	if err := w.flushFn(batch); err != nil {
		// Return the failed batch for retry; keep any samples appended since
		// we snapshotted (none here, but defensive if this changes).
		w.pending = append(batch, w.pending...)
		return err
	}
	return nil
}

// Pending returns how many samples are currently buffered.
func (w *BatchWriter) Pending() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.pending)
}

// PeriodicallyFlush calls Flush at the given interval until stop is closed.
// It performs a final flush before returning so buffered samples are not lost
// on shutdown.
func (w *BatchWriter) PeriodicallyFlush(interval time.Duration, stop <-chan struct{}) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			_ = w.Flush()
			return
		case <-t.C:
			_ = w.Flush()
		}
	}
}
