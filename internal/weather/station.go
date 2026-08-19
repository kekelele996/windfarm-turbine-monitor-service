package weather

import "time"

// Reading is a single weather station observation.
type Reading struct {
	StationID string
	At        time.Time
	AirTemp   float64
	Humidity  float64
	Pressure  float64
	WindSpeed float64
}

// Station aggregates readings for one weather station.
type Station struct {
	ID       string
	readings []Reading
}

func NewStation(id string) *Station { return &Station{ID: id} }

func (s *Station) Add(r Reading) { s.readings = append(s.readings, r) }

func (s *Station) Latest() (Reading, bool) {
	if len(s.readings) == 0 {
		return Reading{}, false
	}
	return s.readings[len(s.readings)-1], true
}

// AverageHumidity computes mean humidity over the station's history.
func (s *Station) AverageHumidity() float64 {
	if len(s.readings) == 0 {
		return 0
	}
	var sum float64
	for _, r := range s.readings {
		sum += r.Humidity
	}
	return sum / float64(len(s.readings))
}
