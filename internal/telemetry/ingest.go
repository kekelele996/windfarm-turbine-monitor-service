package telemetry

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// Sink receives normalized samples and persists them.
type Sink interface {
	Append(s Sample) error
}

// Ingestor fans out batches to a bounded pool of workers.
type Ingestor struct {
	sink    Sink
	workers int
}

func NewIngestor(sink Sink, workers int) *Ingestor {
	if workers <= 0 {
		workers = 4
	}
	return &Ingestor{sink: sink, workers: workers}
}

// Ingest processes the given samples concurrently and waits for completion.
// It returns the number of samples successfully persisted and the first error
// encountered. If the context is cancelled, queued but unprocessed samples
// are abandoned and the cancellation error is returned.
func (in *Ingestor) Ingest(ctx context.Context, samples []Sample) (int, error) {
	if len(samples) == 0 {
		return 0, nil
	}
	jobs := make(chan Sample, len(samples))
	var (
		wg        sync.WaitGroup
		succeeded int64
		mu        sync.Mutex
		firstErr  error
	)
	recordErr := func(err error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
	}
	for w := 0; w < in.workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			in.runWorker(ctx, jobs, &succeeded, recordErr)
		}()
	}
	for _, s := range samples {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return int(atomic.LoadInt64(&succeeded)), fmt.Errorf("ingest cancelled: %w", ctx.Err())
		case jobs <- s:
		}
	}
	close(jobs)
	wg.Wait()
	mu.Lock()
	err := firstErr
	mu.Unlock()
	return int(atomic.LoadInt64(&succeeded)), err
}
