package audit

import "sync"

// Delivery replays unread events to a subscriber and advances its checkpoint.
type Delivery struct {
	stream     *Stream
	checkpoint *Checkpoint
	mu         sync.Mutex
}

func NewDelivery(stream *Stream, checkpoint *Checkpoint) *Delivery {
	return &Delivery{stream: stream, checkpoint: checkpoint}
}

// Deliver sends every event after the subscriber's checkpoint and returns how
// many were delivered.
func (d *Delivery) Deliver(subID string, ch chan Event) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	offset := d.checkpoint.Get(subID)
	history := d.stream.History()
	delivered := 0
	for i := offset; i < len(history); i++ {
		select {
		case ch <- history[i]:
			delivered++
			_ = d.checkpoint.Advance(subID, i+1)
		default:
			return delivered
		}
	}
	return delivered
}
