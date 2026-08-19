package alarm

import (
	"sync"
	"time"

	"windfarm-turbine-monitor-service/internal/platform"
)

// Dispatcher stores alarms and supports ack/resolve transitions.
type Dispatcher struct {
	mu     sync.RWMutex
	alarms map[string]Alarm
	order  []string
	dedup  *Deduplicator
}

func NewDispatcher(dedup *Deduplicator) *Dispatcher {
	return &Dispatcher{dedup: dedup}
}

func (d *Dispatcher) Dispatch(a Alarm) (Alarm, bool, error) {
	if a.TurbineID == "" || a.RuleID == "" {
		return Alarm{}, false, platform.ErrInvalid
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	open := d.openLocked()
	if d.dedup.Suppress(a, open) {
		return Alarm{}, false, nil
	}
	if a.ID == "" {
		a.ID = platform.NewID("alarm")
	}
	if a.OccurredAt.IsZero() {
		a.OccurredAt = time.Now()
	}
	a.UpdatedAt = time.Now()
	a.Status = StatusOpen
	d.alarms[a.ID] = a
	d.order = append(d.order, a.ID)
	return a, true, nil
}

func (d *Dispatcher) Ack(id string) (Alarm, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	a, ok := d.alarms[id]
	if !ok {
		return Alarm{}, platform.ErrNotFound
	}
	a.Status = StatusAck
	a.UpdatedAt = time.Now()
	d.alarms[id] = a
	return a, nil
}

func (d *Dispatcher) Resolve(id string) (Alarm, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	a, ok := d.alarms[id]
	if !ok {
		return Alarm{}, platform.ErrNotFound
	}
	a.Status = StatusResolved
	a.UpdatedAt = time.Now()
	d.alarms[id] = a
	return a, nil
}

func (d *Dispatcher) ListOpen() []Alarm {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.openLocked()
}

func (d *Dispatcher) openLocked() []Alarm {
	var out []Alarm
	for _, id := range d.order {
		a := d.alarms[id]
		if a.Status == StatusOpen {
			out = append(out, a)
		}
	}
	return out
}
