package alarm

import "time"

// Throttle limits how many alarms a single turbine may raise per window.
type Throttle struct {
	window  time.Duration
	limit   int
	buckets map[string][]time.Time
}

func NewThrottle(window time.Duration, limit int) *Throttle {
	if window <= 0 {
		window = time.Minute
	}
	return &Throttle{window: window, limit: limit, buckets: map[string][]time.Time{}}
}

// Allow records a dispatch attempt and reports whether it is within budget.
func (t *Throttle) Allow(turbineID string, now time.Time) bool {
	times := t.buckets[turbineID]
	cutoff := now.Add(-t.window)
	kept := times[:0]
	for _, ts := range times {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= t.limit {
		t.buckets[turbineID] = kept
		return false
	}
	kept = append(kept, now)
	t.buckets[turbineID] = kept
	return true
}
