package telemetry

import "context"

// runWorker drains jobs until the channel is closed or the context is done.
func (in *Ingestor) runWorker(ctx context.Context, jobs <-chan Sample) {
	for {
		select {
		case <-ctx.Done():
			return
		case s, ok := <-jobs:
			if !ok {
				return
			}
			if in.sink.Append(s) != nil {
				return
			}
		}
	}
}
