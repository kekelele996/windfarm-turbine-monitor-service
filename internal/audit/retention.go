package audit

import "time"

// Retention trims stream history to a time window.
type Retention struct {
	window time.Duration
}

func NewRetention(window time.Duration) *Retention {
	if window <= 0 {
		window = 24 * time.Hour
	}
	return &Retention{window: window}
}

// Trim removes events older than the window from history, preserving order.
func (r *Retention) Trim(history []Event, now time.Time) []Event {
	cutoff := now.Add(-r.window)
	out := history[:0]
	for _, e := range history {
		if !e.At.Before(cutoff) {
			out = append(out, e)
		}
	}
	return out
}
