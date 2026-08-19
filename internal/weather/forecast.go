package weather

import "time"

// ForecastPoint is one forecasted weather value.
type ForecastPoint struct {
	At        time.Time
	WindSpeed float64
	AirTemp   float64
}

// Forecast is a time series of forecast points.
type Forecast struct {
	Points []ForecastPoint
}

// WindAt returns the forecasted wind speed nearest to a time.
func (f Forecast) WindAt(t time.Time) (float64, bool) {
	if len(f.Points) == 0 {
		return 0, false
	}
	best := f.Points[0]
	bestDiff := absDur(t.Sub(best.At))
	for _, p := range f.Points[1:] {
		if d := absDur(t.Sub(p.At)); d < bestDiff {
			best = p
			bestDiff = d
		}
	}
	return best.WindSpeed, true
}

// Coldest returns the minimum forecasted temperature and its time.
func (f Forecast) Coldest() (float64, time.Time, bool) {
	if len(f.Points) == 0 {
		return 0, time.Time{}, false
	}
	cold := f.Points[0]
	for _, p := range f.Points[1:] {
		if p.AirTemp < cold.AirTemp {
			cold = p
		}
	}
	return cold.AirTemp, cold.At, true
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
