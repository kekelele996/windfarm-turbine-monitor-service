package alarm

import "time"

// EscalationPolicy defines how an unacknowledged alarm escalates.
type EscalationPolicy struct {
	FirstReminder  time.Duration
	EscalateAfter  time.Duration
	EscalateLevels int
}

func DefaultEscalationPolicy() EscalationPolicy {
	return EscalationPolicy{FirstReminder: 10 * time.Minute, EscalateAfter: 30 * time.Minute, EscalateLevels: 3}
}

// LevelFor computes the current escalation level for an open alarm's age.
func (p EscalationPolicy) LevelFor(age time.Duration) int {
	if age < p.FirstReminder {
		return 0
	}
	if age < p.EscalateAfter {
		return 1
	}
	extra := int((age - p.EscalateAfter) / p.EscalateAfter)
	if extra >= p.EscalateLevels {
		return p.EscalateLevels
	}
	return 1 + extra
}
