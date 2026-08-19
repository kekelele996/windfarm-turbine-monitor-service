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
// error when the subscriber is blocked.
func (d *Delivery) Deliver(subID string, ch chan Event) (err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	defer func() { err = nil }()
	offset := d.checkpoint.Get(subID)
	history := d.stream.History()
	for i := offset; i < len(history); i++ {
		select {
		case ch <- history[i]:
			if aErr := d.checkpoint.Advance(subID, i+1); aErr != nil {
				return aErr
			}
		default:
			return fmt.Errorf("subscriber blocked: %w", platform.ErrUnavailable)
		}
	}
	return nil
}
