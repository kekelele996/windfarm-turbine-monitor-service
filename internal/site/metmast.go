package site

import "time"

// MetMast is a meteorological reference mast measurement.
type MetMast struct {
	ID        string
	Site      string
	Height    float64
	WindSpeed float64
	WindDir   float64
	AirTemp   float64
	Recorded  time.Time
}

// MetMastStore keeps recent met mast readings.
type MetMastStore struct {
	readings map[string][]MetMast
}

func NewMetMastStore() *MetMastStore { return &MetMastStore{readings: map[string][]MetMast{}} }

func (s *MetMastStore) Add(m MetMast) { s.readings[m.ID] = append(s.readings[m.ID], m) }

func (s *MetMastStore) Latest(id string) (MetMast, bool) {
	rs := s.readings[id]
	if len(rs) == 0 {
		return MetMast{}, false
	}
	return rs[len(rs)-1], true
}

// MeanWind returns the average wind across all masts for a site.
func (s *MetMastStore) MeanWind(site string) float64 {
	var sum float64
	n := 0
	for _, rs := range s.readings {
		if len(rs) == 0 || rs[0].Site != site {
			continue
		}
		for _, m := range rs {
			sum += m.WindSpeed
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
