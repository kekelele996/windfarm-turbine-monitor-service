package telemetry

import "context"

// runWorker drains jobs until the channel is closed or the context is done,
// reporting sink failures through errCh.
func (in *Ingestor) runWorker(ctx context.Context, jobs <-chan Sample, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		case s, ok := <-jobs:
			if !ok {
				return
			}
			if err := in.sink.Append(s); err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
		}
	}
}
