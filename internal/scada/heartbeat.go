package scada

import "time"

// Heartbeat tracks liveness of SCADA connections.
type Heartbeat struct {
	lastSeen map[string]time.Time
}

func NewHeartbeat() *Heartbeat { return &Heartbeat{lastSeen: map[string]time.Time{}} }

func (h *Heartbeat) Seen(turbineID string, at time.Time) { h.lastSeen[turbineID] = at }

// Stale reports turbines whose last heartbeat is older than the timeout.
func (h *Heartbeat) Stale(timeout time.Duration, now time.Time) []string {
	var out []string
	for id, seen := range h.lastSeen {
		if now.Sub(seen) > timeout {
			out = append(out, id)
		}
	}
	return out
}

// Age returns how long ago a turbine last heartbeated.
func (h *Heartbeat) Age(turbineID string, now time.Time) (time.Duration, bool) {
	seen, ok := h.lastSeen[turbineID]
	if !ok {
		return 0, false
	}
	return now.Sub(seen), true
}
