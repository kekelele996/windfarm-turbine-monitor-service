package telemetry

import (
	"math"
	"sort"
)

// OutlierDetector flags samples that deviate far from their local neighbors
// using a simple median-absolute-deviation rule.
type OutlierDetector struct {
	Threshold float64
}

func DefaultOutlierDetector() OutlierDetector { return OutlierDetector{Threshold: 3.5} }

// IsOutlier reports whether value is an outlier within a window of values.
func (d OutlierDetector) IsOutlier(value float64, window []float64) bool {
	if len(window) < 5 {
		return false
	}
	median := medianOf(window)
	mad := medianAbsoluteDeviation(window, median)
	if mad == 0 {
		return math.Abs(value-median) > d.Threshold
	}
	score := 0.6745 * math.Abs(value-median) / mad
	return score > d.Threshold
}

// FilterOutliers removes outlier samples from a series, preserving order.
func (d OutlierDetector) FilterOutliers(samples []Sample) []Sample {
	out := make([]Sample, 0, len(samples))
	for i, s := range samples {
		start := i - 10
		if start < 0 {
			start = 0
		}
		window := make([]float64, 0, i-start)
		for _, w := range samples[start:i] {
			window = append(window, w.PowerOutput)
		}
		if d.IsOutlier(s.PowerOutput, window) {
			continue
		}
		out = append(out, s)
	}
	return out
}

func medianOf(values []float64) float64 {
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func medianAbsoluteDeviation(values []float64, median float64) float64 {
	deviations := make([]float64, len(values))
	for i, v := range values {
		deviations[i] = math.Abs(v - median)
	}
	sort.Float64s(deviations)
	n := len(deviations)
	if n%2 == 1 {
		return deviations[n/2]
	}
	return (deviations[n/2-1] + deviations[n/2]) / 2
}
