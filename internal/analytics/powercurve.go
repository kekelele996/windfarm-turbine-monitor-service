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
