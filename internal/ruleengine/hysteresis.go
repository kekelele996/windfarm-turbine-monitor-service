package ruleengine

// Hysteresis prevents alarm flapping by requiring a threshold to be crossed
// with a margin before the alarm clears.
type Hysteresis struct {
	OnThreshold  float64
	OffThreshold float64
	active       map[string]bool
}

func NewHysteresis(on, off float64) *Hysteresis {
	return &Hysteresis{OnThreshold: on, OffThreshold: off, active: map[string]bool{}}
}

// Evaluate updates the latched state for a subject and reports whether the
// condition is currently active.
func (h *Hysteresis) Evaluate(subject string, value float64) bool {
	if h.active[subject] {
		if value <= h.OffThreshold {
			h.active[subject] = false
			return false
		}
		return true
	}
	if value >= h.OnThreshold {
		h.active[subject] = true
		return true
	}
	return false
}

// LatchedSubjects returns every subject currently held active.
func (h *Hysteresis) LatchedSubjects() []string {
	out := make([]string, 0, len(h.active))
	for s, active := range h.active {
		if active {
			out = append(out, s)
		}
	}
	return out
}
