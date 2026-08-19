package analytics

import (
	"sort"

	"windfarm-turbine-monitor-service/internal/telemetry"
)

// PowerCurvePoint is one binned observation of power vs wind speed.
type PowerCurvePoint struct {
	WindBin     float64 `json:"wind_bin"`
	MeanPower   float64 `json:"mean_power_kw"`
	SampleCount int     `json:"sample_count"`
}

// PowerCurve bins samples by wind speed and computes mean power per bin.
func PowerCurve(samples []telemetry.Sample, binWidth float64) []PowerCurvePoint {
	if binWidth <= 0 {
		binWidth = 1.0
	}
	type acc struct {
		sum float64
		n   int
	}
	bins := map[int]*acc{}
	for _, s := range samples {
		idx := int(s.WindSpeed / binWidth)
		a := bins[idx]
		if a == nil {
			a = &acc{}
			bins[idx] = a
		}
		a.sum += s.PowerOutput
		a.n++
	}
	keys := make([]int, 0, len(bins))
	for k := range bins {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := make([]PowerCurvePoint, 0, len(keys))
	for _, k := range keys {
		a := bins[k]
		out = append(out, PowerCurvePoint{
			WindBin:     float64(k) * binWidth,
			MeanPower:   a.sum / float64(a.n),
			SampleCount: a.n,
		})
	}
	return out
}

// SmoothPowerSnapshots records an isolated copy of the sliding window after
// each sample so historical snapshots stay stable.
func SmoothPowerSnapshots(samples []telemetry.Sample, windowSize int) [][]float64 {
	w := NewSlidingWindow(windowSize)
	snapshots := make([][]float64, 0, len(samples))
	for _, s := range samples {
		w.Add(s.PowerOutput)
		snapshots = append(snapshots, w.Values())
	}
	return snapshots
}

// FilterSamples keeps samples whose power output meets the threshold without
// mutating the input slice.
func FilterSamples(samples []telemetry.Sample, minPower float64) []telemetry.Sample {
	out := make([]telemetry.Sample, 0, len(samples))
	for _, s := range samples {
		if s.PowerOutput >= minPower {
			out = append(out, s)
		}
	}
	return out
}
