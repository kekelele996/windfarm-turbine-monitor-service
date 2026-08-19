package maintenance

import (
	"sort"
	"time"
)

// ServiceRecord is one completed maintenance event.
type ServiceRecord struct {
	TurbineID string
	At        time.Time
	Hours     float64
}

// Scheduler plans service windows for a fleet.
type Scheduler struct {
	records map[string][]ServiceRecord
}

func NewScheduler() *Scheduler { return &Scheduler{records: map[string][]ServiceRecord{}} }

func (s *Scheduler) Record(r ServiceRecord) {
	s.records[r.TurbineID] = append(s.records[r.TurbineID], r)
}

// LastService returns the most recent service record for a turbine.
func (s *Scheduler) LastService(turbineID string) (ServiceRecord, bool) {
	rs := s.records[turbineID]
	if len(rs) == 0 {
		return ServiceRecord{}, false
	}
	latest := rs[0]
	for _, r := range rs[1:] {
		if r.At.After(latest.At) {
			latest = r
		}
	}
	return latest, true
}

// Upcoming returns turbines due for service within the next days.
func (s *Scheduler) Upcoming(plan Plan, now time.Time, days int) []string {
	var out []string
	for id, rs := range s.records {
		last, ok := s.last(rs)
		if !ok {
			out = append(out, id)
			continue
		}
		due := plan.NextDue(last)
		if now.Add(time.Duration(days) * 24 * time.Hour).After(due) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Scheduler) last(rs []ServiceRecord) (time.Time, bool) {
	if len(rs) == 0 {
		return time.Time{}, false
	}
	latest := rs[0].At
	for _, r := range rs[1:] {
		if r.At.After(latest) {
			latest = r.At
		}
	}
	return latest, true
}
