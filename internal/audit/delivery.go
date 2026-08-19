package audit

import (
	"fmt"
	"sync"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Delivery replays unread events to a subscriber and advances its checkpoint.
type Delivery struct {
	stream     *Stream
	checkpoint *Checkpoint
	mu         sync.Mutex
}

func NewDelivery(stream *Stream, checkpoint *Checkpoint) *Delivery {
	return &Delivery{stream: stream, checkpoint: checkpoint}
}

// Deliver sends every event after the subscriber's checkpoint and returns an
// error when the subscriber is blocked or a checkpoint cannot be advanced.
func (d *Delivery) Deliver(subID string, ch chan Event) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	offset := d.checkpoint.Get(subID)
	history := d.stream.History()
	delivered := 0
	for i := offset; i < len(history); i++ {
		select {
		case ch <- history[i]:
			if err := d.checkpoint.Advance(subID, i+1); err != nil {
				return fmt.Errorf("advance checkpoint: %w", err)
			}
			delivered++
		default:
			if delivered == 0 {
				return fmt.Errorf("subscriber blocked: %w", platform.ErrUnavailable)
			}
			return fmt.Errorf("subscriber blocked after %d events: %w", delivered, platform.ErrUnavailable)
		}
	}
	return nil
}
