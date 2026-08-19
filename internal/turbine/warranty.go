package turbine

import "time"

// Warranty describes the manufacturer warranty for a turbine.
type Warranty struct {
	TurbineID string
	Start     time.Time
	Years     int
}

// ExpiresAt returns the warranty end date.
func (w Warranty) ExpiresAt() time.Time {
	return w.Start.AddDate(w.Years, 0, 0)
}

// ActiveAt reports whether the warranty covers a given time.
func (w Warranty) ActiveAt(at time.Time) bool {
	return !at.After(w.ExpiresAt())
}

// RemainingDays returns the warranty days remaining at a time.
func (w Warranty) RemainingDays(at time.Time) int {
	exp := w.ExpiresAt()
	if at.After(exp) {
		return 0
	}
	return int(exp.Sub(at).Hours() / 24)
}

// Coverage maps turbine IDs to warranties.
type Coverage struct {
	items map[string]Warranty
}

func NewCoverage() *Coverage { return &Coverage{items: map[string]Warranty{}} }

func (c *Coverage) Add(w Warranty) { c.items[w.TurbineID] = w }

func (c *Coverage) Covers(turbineID string, at time.Time) bool {
	w, ok := c.items[turbineID]
	if !ok {
		return false
	}
	return w.ActiveAt(at)
}
