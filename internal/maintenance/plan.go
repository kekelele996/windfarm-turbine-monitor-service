package maintenance

import "time"

// Plan describes a recurring maintenance routine.
type Plan struct {
	ID            string
	TurbineModel  string
	IntervalDays  int
	DurationHours float64
	Checklist     []string
}

// NextDue computes the next due date after the last service.
func (p Plan) NextDue(lastService time.Time) time.Time {
	return lastService.Add(time.Duration(p.IntervalDays) * 24 * time.Hour)
}

// Overdue reports whether a turbine is past its next due date.
func (p Plan) Overdue(lastService, now time.Time) bool {
	return now.After(p.NextDue(lastService))
}

// DaysOverdue returns how many days a turbine is overdue (0 when not).
func (p Plan) DaysOverdue(lastService, now time.Time) int {
	due := p.NextDue(lastService)
	if !now.After(due) {
		return 0
	}
	return int(now.Sub(due).Hours() / 24)
}
