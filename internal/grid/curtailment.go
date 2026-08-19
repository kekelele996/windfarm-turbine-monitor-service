package grid

import "time"

// Curtailment is a grid-operator instruction to reduce output.
type Curtailment struct {
	ID      string
	Site    string
	StartAt time.Time
	EndAt   time.Time
	LimitKW float64
	Active  bool
}

// Schedule tracks active and upcoming curtailment instructions.
type Schedule struct {
	items []Curtailment
}

func NewSchedule() *Schedule { return &Schedule{} }

func (s *Schedule) Add(c Curtailment) { s.items = append(s.items, c) }

// ActiveFor returns the active curtailment limit for a site at a time.
func (s *Schedule) ActiveFor(site string, at time.Time) (float64, bool) {
	for _, c := range s.items {
		if c.Site == site && c.Active && !at.Before(c.StartAt) && at.Before(c.EndAt) {
			return c.LimitKW, true
		}
	}
	return 0, false
}

// Expire marks any curtailment whose window has passed as inactive.
func (s *Schedule) Expire(at time.Time) int {
	expired := 0
	for i := range s.items {
		if s.items[i].Active && !at.Before(s.items[i].EndAt) {
			s.items[i].Active = false
			expired++
		}
	}
	return expired
}
