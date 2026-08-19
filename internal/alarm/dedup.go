package alarm

import "time"

// Deduplicator decides whether a fresh evaluation should be suppressed because
// an identical alarm is still open.
type Deduplicator struct {
	window time.Duration
}

func NewDeduplicator(window time.Duration) *Deduplicator {
	if window <= 0 {
		window = 5 * time.Minute
	}
	return &Deduplicator{window: window}
}

// Suppress reports whether the candidate duplicates an existing open alarm.
func (d *Deduplicator) Suppress(candidate Alarm, existing []Alarm) bool {
	for _, a := range existing {
		if a.Key() == candidate.Key() && a.Status != StatusResolved {
			return true
		}
	}
	return false
}
