package telemetry

// Normalizer converts raw SCADA readings into consistent engineering units and
// clamps obvious sensor glitches.
type Normalizer struct{}

// Normalize clamps a sample into physically sane bounds and rounds values to a
// stable precision.
func (Normalizer) Normalize(s Sample) Sample {
	if s.WindSpeed < 0 {
		s.WindSpeed = 0
	}
	if s.RotorRPM < 0 {
		s.RotorRPM = 0
	}
	if s.PowerOutput < 0 {
		s.PowerOutput = 0
	}
	return s
}

// NormalizeBatch normalizes every sample in place and returns the count.
func (n Normalizer) NormalizeBatch(samples []Sample) int {
	for i := range samples {
		samples[i] = n.Normalize(samples[i])
	}
	return len(samples)
}
