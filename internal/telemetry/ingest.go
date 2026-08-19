package telemetry

import (
	"context"
	"fmt"
	"sync"
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
func (in *Ingestor) Ingest(ctx context.Context, samples []Sample) (int, error) {
	if len(samples) == 0 {
		return 0, nil
	}
	jobs := make(chan Sample, len(samples))
	errCh := make(chan error, in.workers)
	var wg sync.WaitGroup
	wg.Add(in.workers)
	for w := 0; w < in.workers; w++ {
		go func() {
			defer wg.Done()
			in.runWorker(ctx, jobs, errCh)
		}()
	}
	for _, s := range samples {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return 0, fmt.Errorf("ingest cancelled: %w", ctx.Err())
		case err := <-errCh:
			close(jobs)
			wg.Wait()
			return 0, fmt.Errorf("sink append: %w", err)
		case jobs <- s:
		}
	}
	close(jobs)
	wg.Wait()
	select {
	case err := <-errCh:
		return 0, fmt.Errorf("sink append: %w", err)
	default:
	}
	return len(samples), nil
}
