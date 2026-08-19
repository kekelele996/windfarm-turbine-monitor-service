package audit

import (
	"sync"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Stream is an in-memory fan-out event stream.
type Stream struct {
	mu      sync.RWMutex
	subs    map[string]chan Event
	history []Event
}

func NewStream() *Stream { return &Stream{subs: map[string]chan Event{}} }

func (s *Stream) Subscribe(id string) chan Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan Event, 256)
	s.subs[id] = ch
	return ch
}

func (s *Stream) Unsubscribe(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ch, ok := s.subs[id]; ok {
		delete(s.subs, id)
		_ = ch
	}
}

// Publish appends an event and fans it out to every subscriber without blocking.
func (s *Stream) Publish(typ EventType, turbineID, payload string) Event {
	ev := Event{ID: platform.NewID("evt"), Type: typ, TurbineID: turbineID, Payload: payload}
	s.mu.Lock()
	s.history = append(s.history, ev)
	chans := make([]chan Event, 0, len(s.subs))
	for _, ch := range s.subs {
		chans = append(chans, ch)
	}
	s.mu.Unlock()
	for _, ch := range chans {
		select {
		case ch <- ev:
		default:
		}
	}
	return ev
}

func (s *Stream) History() []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Event, len(s.history))
	copy(out, s.history)
	return out
}

// DeliverBatch invokes handler for every stored event, releasing the cursor
// after the whole batch.
func (s *Stream) DeliverBatch(handler func(Event) error) error {
	history := s.History()
	for _, e := range history {
		defer func(ev Event) {
			_ = handler(ev)
		}(e)
	}
	return nil
}
