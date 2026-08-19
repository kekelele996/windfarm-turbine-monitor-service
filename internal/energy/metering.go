package energy

import (
	"time"

	"windfarm-turbine-monitor-service/internal/telemetry"
)

// MeterReading is a cumulative energy counter at a point in time.
type MeterReading struct {
	TurbineID string
	At        time.Time
	EnergyKWh float64
}

// Meter aggregates sample power into energy over a fixed interval.
type Meter struct {
	interval time.Duration
}

func NewMeter(interval time.Duration) *Meter {
	if interval <= 0 {
		interval = time.Hour
	}
	return &Meter{interval: interval}
}

// ComputeEnergy converts average power over the interval into energy (kWh).
func (m *Meter) ComputeEnergy(samples []telemetry.Sample) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		sum += s.PowerOutput
	}
	mean := sum / float64(len(samples))
	return mean * m.interval.Hours()
}

// ReadingsFor groups samples by turbine and computes one reading each.
func (m *Meter) ReadingsFor(samples []telemetry.Sample) []MeterReading {
	grouped := map[string][]telemetry.Sample{}
	for _, s := range samples {
		grouped[s.TurbineID] = append(grouped[s.TurbineID], s)
	}
	out := make([]MeterReading, 0, len(grouped))
	for id, ss := range grouped {
		out = append(out, MeterReading{
			TurbineID: id,
			EnergyKWh: m.ComputeEnergy(ss),
		})
	}
	return out
}
