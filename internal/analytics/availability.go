package analytics

import "windfarm-turbine-monitor-service/internal/telemetry"

// AvailabilityWindow computes turbine availability over a series of samples.
func AvailabilityWindow(samples []telemetry.Sample, minPower float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	up := 0
	for _, s := range samples {
		if s.PowerOutput >= minPower {
			up++
		}
	}
	return float64(up) / float64(len(samples))
}

// CapacityFactor compares mean output to a rated power.
func CapacityFactor(samples []telemetry.Sample, ratedPower float64) float64 {
	if ratedPower <= 0 {
		return 0
	}
	agg := AggregateSamples(samples, 3600)
	return agg.MeanPower / ratedPower
}
