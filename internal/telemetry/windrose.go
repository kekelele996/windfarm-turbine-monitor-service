package telemetry

import "math"

// WindDirectionBucket buckets wind direction into 16 compass sectors.
type WindDirectionBucket struct {
	Sector   string
	Count    int
	MeanWind float64
}

// WindRose aggregates samples into direction sectors.
func WindRose(samples []Sample) []WindDirectionBucket {
	buckets := map[string]*WindDirectionBucket{}
	labels := [16]string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	for _, s := range samples {
		idx := int((s.YawAngle+22.5)/45.0) % 16
		if idx < 0 {
			idx += 16
		}
		label := labels[idx]
		b := buckets[label]
		if b == nil {
			b = &WindDirectionBucket{Sector: label}
			buckets[label] = b
		}
		b.Count++
		b.MeanWind += s.WindSpeed
	}
	out := make([]WindDirectionBucket, 0, len(buckets))
	for _, b := range buckets {
		if b.Count > 0 {
			b.MeanWind /= float64(b.Count)
		}
		out = append(out, *b)
	}
	return out
}

// PrevailingDirection returns the sector with the most samples.
func PrevailingDirection(samples []Sample) string {
	rose := WindRose(samples)
	best := ""
	bestN := -1
	for _, b := range rose {
		if b.Count > bestN {
			best = b.Sector
			bestN = b.Count
		}
	}
	return best
}

// TurbulenceIntensity approximates turbulence from wind speed variance.
func TurbulenceIntensity(samples []Sample) float64 {
	if len(samples) < 2 {
		return 0
	}
	mean := 0.0
	for _, s := range samples {
		mean += s.WindSpeed
	}
	mean /= float64(len(samples))
	if mean == 0 {
		return 0
	}
	var variance float64
	for _, s := range samples {
		d := s.WindSpeed - mean
		variance += d * d
	}
	return math.Sqrt(variance/float64(len(samples))) / mean
}
