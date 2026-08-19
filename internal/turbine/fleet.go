package turbine

import "sort"

// FleetStats is a small aggregate over the current fleet.
type FleetStats struct {
	Total       int            `json:"total"`
	Active      int            `json:"active"`
	Maintenance int            `json:"maintenance"`
	BySite      map[string]int `json:"by_site"`
	Models      map[string]int `json:"models"`
}

func (r *Registry) Stats() FleetStats {
	s := FleetStats{Total: len(r.list), BySite: map[string]int{}, Models: map[string]int{}}
	for _, t := range r.list {
		s.BySite[t.Site]++
		s.Models[t.Model]++
		switch t.State {
		case StateRunning, StateIdle:
			s.Active++
		case StateMaintenance:
			s.Maintenance++
		}
	}
	return s
}

// Sites returns the distinct site codes in the fleet, sorted.
func (r *Registry) Sites() []string {
	seen := map[string]struct{}{}
	for _, t := range r.list {
		seen[t.Site] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
