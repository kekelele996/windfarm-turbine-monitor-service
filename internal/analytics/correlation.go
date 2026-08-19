package analytics

import (
	"math"

	"windfarm-turbine-monitor-service/internal/telemetry"
)

// PearsonCorrelation computes the Pearson correlation between wind speed and
// power output over paired samples.
func PearsonCorrelation(samples []telemetry.Sample) float64 {
	n := float64(len(samples))
	if n < 2 {
		return 0
	}
	var sx, sy, sxx, syy, sxy float64
	for _, s := range samples {
		sx += s.WindSpeed
		sy += s.PowerOutput
		sxx += s.WindSpeed * s.WindSpeed
		syy += s.PowerOutput * s.PowerOutput
		sxy += s.WindSpeed * s.PowerOutput
	}
	num := n*sxy - sx*sy
	den := math.Sqrt((n*sxx - sx*sx) * (n*syy - sy*sy))
	if den == 0 {
		return 0
	}
	return num / den
}

// Underperformance flags samples where power is far below the wind-speed curve.
func Underperformance(samples []telemetry.Sample, threshold float64) int {
	count := 0
	for _, s := range samples {
		expected := 0.0
		if s.WindSpeed >= 12 {
			expected = 2000
		} else if s.WindSpeed >= 6 {
			expected = 2000 * (s.WindSpeed - 6) / 6
		}
		if expected > 0 && s.PowerOutput < expected*threshold {
			count++
		}
	}
	return count
}
