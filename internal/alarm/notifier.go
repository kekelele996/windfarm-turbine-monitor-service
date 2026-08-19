package alarm

import "sync"

// Notifier fans alarm notifications out to registered channels.
type Notifier struct {
	mu    sync.RWMutex
	sinks []chan Alarm
}

func NewNotifier() *Notifier { return &Notifier{} }

func (n *Notifier) Register(ch chan Alarm) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sinks = append(n.sinks, ch)
}

func (n *Notifier) Notify(a Alarm) int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	sent := 0
	for _, ch := range n.sinks {
		select {
		case ch <- a:
			sent++
		default:
		}
	}
	return sent
}

func (n *Notifier) SinkCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.sinks)
}
