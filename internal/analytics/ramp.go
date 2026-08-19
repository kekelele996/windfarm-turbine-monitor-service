package analytics

import "windfarm-turbine-monitor-service/internal/telemetry"

// RampEvent is a rapid change in power output between consecutive samples.
type RampEvent struct {
	TurbineID string
	From      float64
	To        float64
	Delta     float64
}

// DetectRamps finds power changes exceeding a threshold between consecutive
// samples, which can indicate grid events or control failures.
func DetectRamps(samples []telemetry.Sample, thresholdKW float64) []RampEvent {
	var events []RampEvent
	for i := 1; i < len(samples); i++ {
		delta := samples[i].PowerOutput - samples[i-1].PowerOutput
		if delta < 0 {
			delta = -delta
		}
		if delta >= thresholdKW {
			events = append(events, RampEvent{
				TurbineID: samples[i].TurbineID,
				From:      samples[i-1].PowerOutput,
				To:        samples[i].PowerOutput,
				Delta:     delta,
			})
		}
	}
	return events
}

// MaxRamp returns the largest power change in the series.
func MaxRamp(samples []telemetry.Sample) float64 {
	max := 0.0
	for i := 1; i < len(samples); i++ {
		d := samples[i].PowerOutput - samples[i-1].PowerOutput
		if d < 0 {
			d = -d
		}
		if d > max {
			max = d
		}
	}
	return max
}
