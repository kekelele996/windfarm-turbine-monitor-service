package telemetry

import (
	"context"
	"sync/atomic"
)

// runWorker drains jobs until the channel is closed or the context is done.
// On a sink error it reports the error and continues with the next sample so
// that one bad sample does not discard the rest of the batch.
func (in *Ingestor) runWorker(ctx context.Context, jobs <-chan Sample, succeeded *int64, recordErr func(error)) {
	for {
		select {
		case <-ctx.Done():
			return
		case s, ok := <-jobs:
			if !ok {
				return
			}
			if err := in.sink.Append(s); err != nil {
				recordErr(err)
				continue
			}
			atomic.AddInt64(succeeded, 1)
		}
	}
}
